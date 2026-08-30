package opfor

import (
	"context"
	"errors"
)

// HashAt performs Sleep's autovivifying hash access. Ordered-hash miss and
// removal policies are invoked with the supplied context. Read-only wrappers
// return a detached value and do not autovivify. Ordinary host code that only
// needs a non-autovivifying lookup should use Hash.Get.
func (h *Hash) HashAt(ctx context.Context, key string) (Value, error) {
	return h.HashAtValue(ctx, String(key))
}

// HashAtValue is HashAt with the original Sleep key value retained for an
// ordered-hash miss policy. Hash storage still uses Sleep's string-coerced key.
func (h *Hash) HashAtValue(ctx context.Context, key Value) (Value, error) {
	cell, err := h.EnsureValueContext(ctx, key)
	if err != nil {
		return Null(), err
	}
	return cell.Get(), nil
}

// EnsureContext returns the key's mutable cell using Sleep's ordered-hash
// policy behavior. Missing ordinary hashes are autovivified with $null;
// read-only wrappers return a detached cell.
func (h *Hash) EnsureContext(ctx context.Context, key string) (*Cell, error) {
	return h.EnsureValueContext(ctx, String(key))
}

// EnsureValueContext is EnsureContext with the original key retained for a
// miss-policy callback. Hash identity remains the key's string coercion.
func (h *Hash) EnsureValueContext(ctx context.Context, key Value) (*Cell, error) {
	return h.ensureValueAtExecution(ctx, executionMutationScript(ctx, Invocation{}), key)
}

func (h *Hash) ensureValueAtExecution(ctx context.Context, script *Script, key Value) (*Cell, error) {
	if h == nil {
		return nil, errors.New("opfor: hash is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if h.backend != nil {
		return h.ensureDirectAtExecution(ctx, script, key)
	}
	keyText, keyValue := sleepHashKey(key)

	var (
		cell                                  *Cell
		missPolicy                            Callable
		policy                                Callable
		eldestKey                             string
		eldestKeyValue                        Value
		eldestCell                            *Cell
		returnImmediately, readOnly, inserted bool
	)
	err := h.mutateAtExecution(ctx, script, func() error {
		if h.readOnly {
			readOnly = true
			if existing, ok := h.items[keyText]; ok {
				cell = NewCell(existing.Get())
			} else {
				cell = NewCell(Null())
			}
			returnImmediately = true
			return nil
		}

		var exists bool
		cell, exists = h.items[keyText]
		if exists && h.accessOrdered && h.moveToEndLocked(keyText) {
			h.modCount++
		}
		missPolicy = h.missPolicy
		if exists && (missPolicy == nil || !cell.Get().IsNull()) {
			returnImmediately = true
			return nil
		}
		if h.ordered {
			h.cleanupOrderedLocked()
			cell, exists = h.items[keyText]
		}
		if missPolicy != nil {
			return nil
		}
		if exists {
			returnImmediately = true
			return nil
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, 1); err != nil {
			return err
		}
		cell = NewCell(Null())
		policy, eldestKey, eldestKeyValue, eldestCell = h.insertLocked(keyText, keyValue, cell)
		inserted = true
		returnImmediately = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if returnImmediately {
		if !readOnly && inserted {
			if err := h.applyRemovalPolicyAtExecution(ctx, script, policy, eldestKey, eldestKeyValue, eldestCell); err != nil {
				return nil, err
			}
		}
		return cell, nil
	}

	value, err := missPolicy.Invoke(ctx, HashValue(h), key)
	if err != nil {
		return nil, err
	}

	var updateExisting bool
	err = h.mutateAtExecution(ctx, script, func() error {
		if current, ok := h.items[keyText]; ok {
			cell = current
			updateExisting = true
			if h.accessOrdered && h.moveToEndLocked(keyText) {
				h.modCount++
			}
			return nil
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, 1); err != nil {
			return err
		}
		cell = NewCell(value)
		policy, eldestKey, eldestKeyValue, eldestCell = h.insertLocked(keyText, keyValue, cell)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if updateExisting {
		if err := cell.setAtExecution(ctx, script, value, Span{}); err != nil {
			return nil, err
		}
	}
	if err := h.applyRemovalPolicyAtExecution(ctx, script, policy, eldestKey, eldestKeyValue, eldestCell); err != nil {
		return nil, err
	}
	return cell, nil
}

// SetContext inserts or replaces a key while honoring ordered-hash eviction
// policies. Replacement counts as access for an access-ordered hash.
func (h *Hash) SetContext(ctx context.Context, key string, value Value) error {
	return h.SetValueContext(ctx, String(key), value)
}

// SetValueContext is SetContext with the original key retained for a
// miss-policy callback.
func (h *Hash) SetValueContext(ctx context.Context, key, value Value) error {
	script := executionMutationScript(ctx, Invocation{})
	cell, err := h.ensureValueAtExecution(ctx, script, key)
	if err != nil {
		return err
	}
	return cell.setAtExecution(ctx, script, value, Span{})
}

func (h *Hash) insertLocked(key string, keyValue Value, cell *Cell) (Callable, string, Value, *Cell) {
	h.items[key] = cell
	h.rememberKeyLocked(key, keyValue)
	h.order = append(h.order, key)
	h.modCount++
	if len(h.order) == 0 || h.removalPolicy == nil {
		return nil, "", Null(), nil
	}
	eldestKey := h.order[0]
	return h.removalPolicy, eldestKey, h.keyValueLocked(eldestKey), h.items[eldestKey]
}

func (h *Hash) applyRemovalPolicy(
	ctx context.Context,
	policy Callable,
	eldestKey string,
	eldestKeyValue Value,
	eldestCell *Cell,
) error {
	return h.applyRemovalPolicyAtExecution(
		ctx,
		executionMutationScript(ctx, Invocation{}),
		policy,
		eldestKey,
		eldestKeyValue,
		eldestCell,
	)
}

func (h *Hash) applyRemovalPolicyAtExecution(
	ctx context.Context,
	script *Script,
	policy Callable,
	eldestKey string,
	eldestKeyValue Value,
	eldestCell *Cell,
) error {
	if policy == nil || eldestCell == nil {
		return nil
	}
	remove, err := policy.Invoke(ctx, HashValue(h), eldestKeyValue, eldestCell.Get())
	if err != nil {
		return err
	}
	if !remove.Truth() {
		return nil
	}
	return h.mutateAtExecution(ctx, script, func() error {
		if h.items[eldestKey] == eldestCell {
			delete(h.items, eldestKey)
			delete(h.keyValues, eldestKey)
			for index, key := range h.order {
				if key == eldestKey {
					h.order = append(h.order[:index], h.order[index+1:]...)
					break
				}
			}
			h.modCount++
		}
		return nil
	})
}

func (h *Hash) cleanupOrderedLocked() {
	if !h.shouldClean {
		return
	}
	kept := h.order[:0]
	removed := false
	for _, key := range h.order {
		cell, exists := h.items[key]
		if !exists || cell.Get().IsNull() {
			delete(h.items, key)
			delete(h.keyValues, key)
			removed = true
			continue
		}
		kept = append(kept, key)
	}
	h.order = kept
	if removed {
		h.modCount++
	}
	h.shouldClean = false
}

func (h *Hash) setMissPolicy(policy Callable) error {
	if h == nil || !h.ordered {
		return errors.New("expected an ordered hash")
	}
	if policy == nil {
		return errors.New("expected a function")
	}
	h.mu.Lock()
	h.missPolicy = policy
	h.mu.Unlock()
	return nil
}

func (h *Hash) setMissPolicyAtExecution(ctx context.Context, invocation Invocation, policy Callable) error {
	if h == nil || !h.ordered {
		return errors.New("expected an ordered hash")
	}
	if policy == nil {
		return errors.New("expected a function")
	}
	return h.mutateForInvocation(ctx, invocation, func() error {
		h.missPolicy = policy
		return nil
	})
}

func (h *Hash) setRemovalPolicy(policy Callable) error {
	if h == nil || !h.ordered {
		return errors.New("expected an ordered hash")
	}
	if policy == nil {
		return errors.New("expected a function")
	}
	h.mu.Lock()
	h.removalPolicy = policy
	h.mu.Unlock()
	return nil
}

func (h *Hash) setRemovalPolicyAtExecution(ctx context.Context, invocation Invocation, policy Callable) error {
	if h == nil || !h.ordered {
		return errors.New("expected an ordered hash")
	}
	if policy == nil {
		return errors.New("expected a function")
	}
	return h.mutateForInvocation(ctx, invocation, func() error {
		h.removalPolicy = policy
		return nil
	})
}
