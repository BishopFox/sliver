package opfor

import (
	"context"
	"errors"
	"fmt"
)

// mutationFunctions returns Sleep's positional collection mutators. They are
// kept separate from the broader collection tranche so their structural
// semantics can be evolved together when Array gains true sublist views.
func (*Runtime) mutationFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"removeAt": builtinRemoveAt,
		"splice":   builtinSplice,
	}
}

// wrapAddMutation mirrors Block's translation of the
// IndexOutOfBoundsException raised by List.add. BridgeUtilities.normalize
// adds size+1 exactly once for insertion positions; it does not wrap modulo.
// The wrapper is installed only on the runtime's portable default so direct
// builtin calls and importer-provided replacements retain ordinary Go errors.
func wrapAddMutation(next NativeFunc) NativeFunc {
	return func(ctx context.Context, invocation Invocation) (Value, error) {
		if currentFiber(ctx) != nil && len(invocation.Arguments) > 2 {
			if array, ok := invocation.Arg(0).Array(); ok {
				cells, err := array.snapshotCells()
				if err == nil {
					position := int(sleepInt32(invocation.Arg(2)))
					if position < 0 {
						position += len(cells) + 1
					}
					if position < 0 || position > len(cells) {
						warning := fmt.Errorf("attempted an invalid index: Index: %d, Size: %d", position, len(cells))
						return Null(), &uncaughtScriptWarning{err: warning}
					}
				}
			}
		}
		return next(ctx, invocation)
	}
}

func builtinRemoveAt(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		// BasicUtilities.removeAt performs a direct Stack.pop before it checks
		// the aggregate type. Sleep consumes that EmptyStackException at the
		// active Block boundary.
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeEmptyStack()
		}
		return Null(), errors.New("&removeAt: missing aggregate argument")
	}
	target := invocation.Arg(0)
	if array, ok := target.Array(); ok {
		for index := 1; index < len(invocation.Arguments); index++ {
			requested := int(sleepInt32(invocation.Arg(index)))
			if err := removeArrayAtExecution(ctx, invocation, array, requested); err != nil {
				if currentFiber(ctx) != nil {
					return Null(), &uncaughtScriptWarning{err: err}
				}
				return Null(), err
			}
		}
		return Null(), nil
	}

	if hash, ok := target.Hash(); ok {
		script := executionMutationScript(ctx, invocation)
		for index := 1; index < len(invocation.Arguments); index++ {
			// Sleep removes a hash entry by assigning its backing scalar $null.
			// keys(), values(), and size() subsequently purge null entries.
			cell, err := hash.ensureDirectAtExecution(ctx, script, invocation.Arg(index))
			if err != nil {
				return Null(), err
			}
			if err := cell.setAtExecution(ctx, script, Null(), invocation.Span); err != nil {
				return Null(), err
			}
		}
	}

	// Sleep silently ignores non-collection targets and always returns $null.
	return Null(), nil
}

func removeArrayAt(array *Array, requested int) error {
	return array.mutateCells(true, func(cells []*Cell) ([]*Cell, error) {
		return removeArrayAtCells(cells, requested)
	})
}

func removeArrayAtExecution(ctx context.Context, invocation Invocation, array *Array, requested int) error {
	return array.mutateCellsForInvocation(ctx, invocation, true, func(cells []*Cell) ([]*Cell, error) {
		return removeArrayAtCells(cells, requested)
	})
}

func removeArrayAtCells(cells []*Cell, requested int) ([]*Cell, error) {
	index := requested
	if index < 0 {
		// BridgeUtilities.normalize adds the current size exactly once. This
		// is deliberately not Array's modulo indexing behavior.
		index += len(cells)
	}
	if index < 0 || index >= len(cells) {
		return nil, fmt.Errorf("attempted an invalid index: Index: %d, Size: %d", index, len(cells))
	}
	return append(cells[:index], cells[index+1:]...), nil
}

func builtinSplice(ctx context.Context, invocation Invocation) (Value, error) {
	targetValue := invocation.Arg(0)
	target, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}

	// BridgeUtilities.getArray treats a missing or non-array insertion value as
	// an empty array. Sleep's ListIterator adds the actual Scalar objects, so we
	// preserve the corresponding OPFOR cells rather than copying their values.
	var inserted []*Cell
	if insertion, ok := invocation.Arg(1).Array(); ok {
		inserted, err = insertion.snapshotCells()
		if err != nil {
			return Null(), err
		}
	}

	script := executionMutationScript(ctx, invocation)
	err = target.mutateCellsAtExecution(ctx, script, true, func(cells []*Cell) ([]*Cell, error) {
		start := 0
		if len(invocation.Arguments) > 2 {
			start = int(sleepInt32(invocation.Arg(2)))
		}
		if start < 0 {
			start += len(cells)
		}
		// The reference advances iterators while possible, clamping positions
		// to the nearest end after one-step negative normalization.
		if start < 0 {
			start = 0
		} else if start > len(cells) {
			start = len(cells)
		}

		removeCount := len(inserted)
		if len(invocation.Arguments) > 3 {
			removeCount = int(sleepInt32(invocation.Arg(3)))
		}
		if removeCount < 0 {
			removeCount = 0
		}
		if remaining := len(cells) - start; removeCount > remaining {
			removeCount = remaining
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, len(inserted)-removeCount); err != nil {
			return nil, err
		}

		replacement := make([]*Cell, 0, len(cells)-removeCount+len(inserted))
		replacement = append(replacement, cells[:start]...)
		replacement = append(replacement, inserted...)
		replacement = append(replacement, cells[start+removeCount:]...)
		return replacement, nil
	})
	if err != nil {
		return Null(), err
	}
	return targetValue, nil
}
