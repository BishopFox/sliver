package opfor

import (
	"context"
	"errors"
)

// executionMutationScript identifies the script whose evaluator currently
// owns ctx. Invocation metadata disambiguates native entries which do not yet
// have a current fiber; it is never sufficient without a live execution token.
// That prevents a retained or fabricated Invocation from reviving script-owned
// mutation after its execution lease has ended.
func executionMutationScript(ctx context.Context, invocation Invocation) *Script {
	if ctx == nil {
		return nil
	}
	if current := currentFiber(ctx); current != nil && current.closure != nil {
		if script := current.closure.script; script != nil {
			if _, owned := executionOwnsScript(ctx, script); owned {
				return script
			}
		}
	}

	token, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken)
	for token != nil {
		script := token.script
		if token.active.Load() && script != nil &&
			(invocation.Runtime == nil || script.runtime == invocation.Runtime) &&
			(invocation.Script == 0 || script.id == invocation.Script) {
			return script
		}
		token = token.parent
	}
	return nil
}

func executionMutationError(ctx context.Context, script *Script) error {
	if err := executionContextError(ctx); err != nil {
		return err
	}
	if script != nil && !script.active {
		return ErrScriptUnloaded
	}
	return nil
}

// mutateCellsAtExecution linearizes an evaluator-visible array commit with
// Script unload. The optimistic path takes the container first and only tries
// Script.mu, so a goroutine already blocked on the array cannot keep unload
// from publishing cancellation. Once a Script writer is pending, the fallback
// yields the array and reacquires locks in Script-then-array order.
func (a *Array) mutateCellsAtExecution(
	ctx context.Context,
	script *Script,
	structural bool,
	mutate func([]*Cell) ([]*Cell, error),
) error {
	return a.withMutationAtExecution(ctx, script, func(storage *arrayStorage, window *arrayWindow) error {
		return mutateArrayCellsLocked(storage, window, structural, mutate)
	})
}

// withMutationAtExecution linearizes a container operation with Script unload
// while leaving the operation free to use a specialized locked implementation.
func (a *Array) withMutationAtExecution(
	ctx context.Context,
	script *Script,
	mutate func(*arrayStorage, *arrayWindow) error,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if a != nil && a.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return err
		}
		return ErrReadOnlyArray
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return ErrIndexOutOfRange
	}

	storage.mu.Lock()
	if script == nil {
		err := executionMutationError(ctx, nil)
		if err == nil {
			err = mutate(storage, window)
		}
		storage.mu.Unlock()
		return err
	}
	if script.mu.TryRLock() {
		err := executionMutationError(ctx, script)
		if err == nil {
			err = mutate(storage, window)
		}
		script.mu.RUnlock()
		storage.mu.Unlock()
		return err
	}
	storage.mu.Unlock()

	script.mu.RLock()
	storage.mu.Lock()
	err := executionMutationError(ctx, script)
	if err == nil {
		err = mutate(storage, window)
	}
	storage.mu.Unlock()
	script.mu.RUnlock()
	return err
}

func (a *Array) mutateCellsForInvocation(
	ctx context.Context,
	invocation Invocation,
	structural bool,
	mutate func([]*Cell) ([]*Cell, error),
) error {
	return a.mutateCellsAtExecution(ctx, executionMutationScript(ctx, invocation), structural, mutate)
}

func (a *Array) ensureAtExecution(ctx context.Context, script *Script, index int) (*Cell, error) {
	if a == nil || index < 0 {
		return nil, ErrIndexOutOfRange
	}
	if a.backend != nil {
		cell, ok, err := a.cellAtExecution(ctx, script, index)
		if err != nil {
			return nil, err
		}
		if ok {
			return cell, nil
		}
		return nil, ErrIndexOutOfRange
	}
	var ensured *Cell
	err := a.mutateCellsAtExecution(ctx, script, true, func(cells []*Cell) ([]*Cell, error) {
		growth := index + 1 - len(cells)
		if err := reserveCollectionEntriesAtExecution(ctx, script, growth); err != nil {
			return nil, err
		}
		for len(cells) <= index {
			cells = append(cells, NewCell(Null()))
		}
		ensured = cells[index]
		return cells, nil
	})
	return ensured, err
}

// cellAtExecution is the error-bearing indexed access used by evaluator and
// builtin paths. Public Array.Cell retains its historical two-result API, but
// must not turn a wrapper cache admission failure into an ordinary index miss
// while an execution context is available.
func (a *Array) cellAtExecution(ctx context.Context, script *Script, index int) (*Cell, bool, error) {
	if a == nil {
		return nil, false, nil
	}
	if a.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return nil, false, err
		}
		return a.backend.cellContext(index)
	}
	cell, ok := a.Cell(index)
	return cell, ok, nil
}

func (a *Array) getAtExecution(ctx context.Context, script *Script, index int) (Value, bool, error) {
	if a == nil || a.backend == nil {
		value, ok := a.Get(index)
		return value, ok, nil
	}
	cell, ok, err := a.cellAtExecution(ctx, script, index)
	if err != nil || !ok {
		return Null(), false, err
	}
	return cell.Get(), true, nil
}

func (a *Array) getForInvocation(ctx context.Context, invocation Invocation, index int) (Value, bool, error) {
	return a.getAtExecution(ctx, executionMutationScript(ctx, invocation), index)
}

func (a *Array) appendValuesAtExecution(ctx context.Context, invocation Invocation, values ...Value) error {
	if a == nil {
		return ErrIndexOutOfRange
	}
	script := executionMutationScript(ctx, invocation)
	return a.withMutationAtExecution(ctx, script, func(storage *arrayStorage, window *arrayWindow) error {
		if len(values) == 0 {
			return nil
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, len(values)); err != nil {
			return err
		}
		if canAppendRootArrayLocked(storage, window) {
			return appendRootArrayValuesLocked(storage, values)
		}
		return mutateArrayCellsLocked(storage, window, true, func(cells []*Cell) ([]*Cell, error) {
			for _, value := range values {
				cells = append(cells, NewCell(value))
			}
			return cells, nil
		})
	})
}

// mutateAtExecution is the hash counterpart to mutateCellsAtExecution. The
// closure runs with h.mu held and must not invoke script code. Policy callbacks
// therefore run between separately admitted mutation phases.
func (h *Hash) mutateAtExecution(ctx context.Context, script *Script, mutate func() error) error {
	if h == nil {
		return errors.New("opfor: hash is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if h.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return err
		}
		return ErrReadOnlyHash
	}

	h.mu.Lock()
	if script == nil {
		err := executionMutationError(ctx, nil)
		if err == nil {
			err = mutate()
		}
		h.mu.Unlock()
		return err
	}
	if script.mu.TryRLock() {
		err := executionMutationError(ctx, script)
		if err == nil {
			err = mutate()
		}
		script.mu.RUnlock()
		h.mu.Unlock()
		return err
	}
	h.mu.Unlock()

	script.mu.RLock()
	h.mu.Lock()
	err := executionMutationError(ctx, script)
	if err == nil {
		err = mutate()
	}
	h.mu.Unlock()
	script.mu.RUnlock()
	return err
}

func (h *Hash) mutateForInvocation(ctx context.Context, invocation Invocation, mutate func() error) error {
	return h.mutateAtExecution(ctx, executionMutationScript(ctx, invocation), mutate)
}

func (h *Hash) getValueAtExecution(ctx context.Context, script *Script, key Value) (Value, bool, error) {
	if h == nil {
		return Null(), false, nil
	}
	if h.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return Null(), false, err
		}
		value, exists := h.backend.get(key)
		return value, exists, nil
	}
	keyText := sleepCanonicalString(key)
	var cell *Cell
	var exists bool
	err := h.mutateAtExecution(ctx, script, func() error {
		cell, exists = h.items[keyText]
		if exists && h.accessOrdered && h.moveToEndLocked(keyText) {
			h.modCount++
		}
		return nil
	})
	if err != nil || !exists {
		return Null(), false, err
	}
	return cell.Get(), true, nil
}

func (h *Hash) getValueForInvocation(ctx context.Context, invocation Invocation, key Value) (Value, bool, error) {
	return h.getValueAtExecution(ctx, executionMutationScript(ctx, invocation), key)
}

// ensureDirectAtExecution is the execution-aware form of Hash.EnsureValue. It
// deliberately does not invoke ordered-hash miss or removal policies.
func (h *Hash) ensureDirectAtExecution(ctx context.Context, script *Script, key Value) (*Cell, error) {
	if h == nil {
		return nil, errors.New("opfor: hash is nil")
	}
	if h.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return nil, err
		}
		return h.backend.ensure(key), nil
	}
	keyText, keyValue := sleepHashKey(key)
	var cell *Cell
	err := h.mutateAtExecution(ctx, script, func() error {
		if h.readOnly {
			if existing, ok := h.items[keyText]; ok {
				cell = NewCell(existing.Get())
			} else {
				cell = NewCell(Null())
			}
			return nil
		}
		if existing, ok := h.items[keyText]; ok {
			cell = existing
			if h.accessOrdered && h.moveToEndLocked(keyText) {
				h.modCount++
			}
			return nil
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, 1); err != nil {
			return err
		}
		cell = NewCell(Null())
		h.items[keyText] = cell
		h.rememberKeyLocked(keyText, keyValue)
		h.order = append(h.order, keyText)
		h.modCount++
		return nil
	})
	return cell, err
}

func setInvocationArgumentAtExecution(ctx context.Context, invocation Invocation, index int, value Value) error {
	if index < 0 || index >= len(invocation.Arguments) {
		return nil
	}
	reference := invocation.Arguments[index].Reference
	if reference == nil {
		return nil
	}
	return reference.setAtExecution(ctx, executionMutationScript(ctx, invocation), value, invocation.Span)
}

func removeIteratorAtExecution(ctx context.Context, invocation Invocation, iterator valueIterator) error {
	script := executionMutationScript(ctx, invocation)
	switch current := iterator.(type) {
	case *arrayIterator:
		return removeArrayIteratorAtExecution(ctx, script, current)
	case *hashIterator:
		return removeHashIteratorAtExecution(ctx, script, current)
	default:
		// Importer and Java iterator implementations own their backing storage.
		// They still receive the canceled execution context and retain their
		// established remove contract.
		return iterator.remove(ctx)
	}
}

func mutateArrayStorageAtExecution(
	ctx context.Context,
	script *Script,
	storage *arrayStorage,
	mutate func() error,
) error {
	if storage == nil {
		return errors.New("opfor: iterator does not support removal")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	storage.mu.Lock()
	if script == nil {
		err := executionMutationError(ctx, nil)
		if err == nil {
			err = mutate()
		}
		storage.mu.Unlock()
		return err
	}
	if script.mu.TryRLock() {
		err := executionMutationError(ctx, script)
		if err == nil {
			err = mutate()
		}
		script.mu.RUnlock()
		storage.mu.Unlock()
		return err
	}
	storage.mu.Unlock()

	script.mu.RLock()
	storage.mu.Lock()
	err := executionMutationError(ctx, script)
	if err == nil {
		err = mutate()
	}
	storage.mu.Unlock()
	script.mu.RUnlock()
	return err
}

func (a *Array) sublistAtExecution(ctx context.Context, invocation Invocation, start, end int) (*Array, error) {
	if a != nil && a.backend != nil {
		if err := executionMutationError(ctx, executionMutationScript(ctx, invocation)); err != nil {
			return nil, err
		}
		if backend, ok := a.backend.(*collectionWrapperArrayBackend); ok {
			return backend.sublistForRuntime(invocation.Runtime, start, end)
		}
		return a.backend.sublist(start, end)
	}
	storage, parent := a.arrayStorage()
	if storage == nil || parent == nil {
		return nil, ErrIndexOutOfRange
	}
	var view *Array
	err := mutateArrayStorageAtExecution(ctx, executionMutationScript(ctx, invocation), storage, func() error {
		var err error
		view, err = sublistArrayLocked(storage, parent, start, end)
		return err
	})
	return view, err
}

func removeArrayIteratorAtExecution(ctx context.Context, script *Script, iterator *arrayIterator) error {
	if iterator == nil || iterator.storage == nil || iterator.window == nil {
		return errors.New("opfor: iterator does not support removal")
	}
	return mutateArrayStorageAtExecution(ctx, script, iterator.storage, func() error {
		if !iterator.window.valid {
			return unsafeArrayViewError()
		}
		if iterator.expectedModCount != iterator.storage.modCount {
			return ErrArrayChangedDuringIteration
		}
		if !iterator.canRemove || iterator.lastIndex < 0 ||
			iterator.lastIndex >= iterator.window.end-iterator.window.start {
			return errors.New("opfor: iterator has no current element")
		}

		absolute := iterator.window.start + iterator.lastIndex
		copy(iterator.storage.items[absolute:], iterator.storage.items[absolute+1:])
		iterator.storage.items[len(iterator.storage.items)-1] = nil
		iterator.storage.items = iterator.storage.items[:len(iterator.storage.items)-1]
		iterator.storage.modCount++
		for candidate := range iterator.storage.views {
			if candidate != iterator.storage.root && candidate != iterator.window {
				candidate.valid = false
			}
		}
		iterator.window.end--
		iterator.storage.root.start = 0
		iterator.storage.root.end = len(iterator.storage.items)
		iterator.storage.root.valid = true
		iterator.storage.syncCachesLocked()

		iterator.expectedModCount = iterator.storage.modCount
		iterator.nextIndex--
		iterator.count--
		iterator.lastIndex = -1
		iterator.canRemove = false
		return nil
	})
}

func removeHashIteratorAtExecution(ctx context.Context, script *Script, iterator *hashIterator) error {
	if iterator == nil || iterator.hash == nil || !iterator.canRemove {
		return errors.New("opfor: iterator has no current element")
	}
	return iterator.hash.mutateAtExecution(ctx, script, func() error {
		if iterator.hash.readOnly {
			// MapWrapper iteration removes from a detached snapshot only.
			iterator.canRemove = false
			return nil
		}
		if iterator.expectedModCount != iterator.hash.modCount {
			iterator.stopped = true
			return ErrHashChangedDuringIteration
		}
		if _, ok := iterator.hash.items[iterator.current]; !ok {
			return errors.New("opfor: hash changed during iteration")
		}
		delete(iterator.hash.items, iterator.current)
		delete(iterator.hash.keyValues, iterator.current)
		for index, candidate := range iterator.hash.order {
			if candidate == iterator.current {
				iterator.hash.order = append(iterator.hash.order[:index], iterator.hash.order[index+1:]...)
				break
			}
		}
		iterator.hash.modCount++
		iterator.expectedModCount = iterator.hash.modCount
		iterator.canRemove = false
		return nil
	})
}
