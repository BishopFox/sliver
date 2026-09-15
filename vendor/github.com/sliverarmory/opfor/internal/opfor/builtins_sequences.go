package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// sleepSequenceFunctions returns the iterable, set, and functional tranche of
// the portable Sleep runtime. The canonical functions follow Sleep 2.1's
// BasicUtilities, BasicNumbers, and BasicStrings bridges. Evidence-gated
// conveniences remain here until coreFunctions removes them from the stock
// namespace.
func (r *Runtime) sleepSequenceFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"concat":      builtinConcat,
		"sublist":     builtinSublist,
		"subarray":    builtinSublist,
		"addAll":      builtinAddAll,
		"removeAll":   builtinRemoveAll,
		"retainAll":   builtinRetainAll,
		"contains":    builtinContains,
		"containsAll": builtinContainsAll,
		"clear":       builtinClear,
		"isEmpty":     builtinIsEmpty,
		"map":         builtinMap,
		"filter":      builtinFilter,
		"grep":        builtinGrep,
		"reduce":      builtinReduce,
		"sum":         builtinSum,
		"sort":        builtinSort,
		"sorta":       builtinSortAscending,
		"sortn":       builtinSortNumeric,
		"sortd":       builtinSortDouble,
		"reverse":     builtinSequenceReverse,
		"zip":         builtinZip,
		"mapValues":   builtinMapValues,
	}
}

// aggressorSequenceFunctions returns the documented Aggressor sequence
// helpers which are not part of Sleep's stock bridge namespace.
func aggressorSequenceFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"range": builtinRange,
	}
}

func builtinConcat(ctx context.Context, invocation Invocation) (Value, error) {
	values := make([]Value, 0)
	for _, argument := range invocation.Arguments {
		value := argument.Resolve()
		if array, ok := value.Array(); ok {
			if array.backend != nil {
				iterator := array.backend.iterator(value)
				for {
					item, present, err := iterator.next(ctx)
					if err != nil {
						return Null(), err
					}
					if !present {
						break
					}
					if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
						return Null(), err
					}
					values = append(values, item.value)
				}
				continue
			}
			cells, err := array.snapshotCells()
			if err != nil {
				return Null(), err
			}
			if err := reserveCollectionEntries(invocation.Runtime, len(cells)); err != nil {
				return Null(), err
			}
			current := valuesFromCells(cells)
			values = append(values, current...)
			continue
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), err
		}
		values = append(values, value)
	}
	return ArrayValue(NewArray(values...)), nil
}

type sequenceCursor struct {
	values        []Value
	callable      Callable
	source        Value
	arrayIterator valueIterator
	javaIterator  *portableJavaIterator
	iterator      Iterator
	index         int
	done          bool
}

func emptySequenceCursor() *sequenceCursor { return &sequenceCursor{} }

// sleepSequenceClosure marks callbacks which have SleepClosure semantics.
// Sleep's BridgeUtilities.getFunction and SleepUtils.getIterator deliberately
// distinguish a SleepClosure scalar from an arbitrary native Function object.
// Keeping the marker private means importer-defined Go callbacks cannot cross
// that language boundary accidentally.
type sleepSequenceClosure interface {
	Callable
	isSleepSequenceClosure()
}

func (*scriptClosure) isSleepSequenceClosure() {}

func newSequenceCursor(value Value, name string) (*sequenceCursor, error) {
	if array, ok := value.Array(); ok {
		if array.backend != nil {
			return &sequenceCursor{source: value, arrayIterator: array.backend.iterator(value)}, nil
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return nil, err
		}
		return &sequenceCursor{values: valuesFromCells(cells)}, nil
	}
	if callable, ok := value.Function(); ok {
		if _, closure := callable.(sleepSequenceClosure); !closure {
			return nil, fmt.Errorf("&%s: expected iterator (@array or &closure)--received: %s",
				builtinName(name), sequenceClosureDescription(value))
		}
		return &sequenceCursor{callable: callable, source: value}, nil
	}
	if object, ok := value.Object(); ok {
		if iterator, ok := object.(*portableJavaIterator); ok && iterator != nil {
			return &sequenceCursor{javaIterator: iterator}, nil
		}
		if iterator, ok := object.(Iterator); ok && iterator != nil {
			return &sequenceCursor{iterator: iterator}, nil
		}
	}
	return nil, fmt.Errorf("&%s: expected iterator (@array or &closure)--received: %s",
		builtinName(name), value.Describe())
}

func invocationSequenceCursor(invocation Invocation, index int) (*sequenceCursor, error) {
	if index >= len(invocation.Arguments) {
		return emptySequenceCursor(), nil
	}
	return newSequenceCursor(invocation.Arg(index), invocation.Name)
}

func (cursor *sequenceCursor) next(ctx context.Context) (Value, bool, error) {
	if cursor == nil || cursor.done {
		return Null(), false, nil
	}
	if err := ctx.Err(); err != nil {
		return Null(), false, err
	}
	if cursor.arrayIterator != nil {
		item, present, err := cursor.arrayIterator.next(ctx)
		if err != nil || !present {
			cursor.done = !present
			return Null(), false, err
		}
		return item.value, true, nil
	}
	if cursor.callable != nil {
		value, err := invokeTracedClosure(ctx, cursor.source, "eval", nil, cursor.callable)
		if err != nil {
			return Null(), false, err
		}
		if value.IsNull() {
			cursor.done = true
			return Null(), false, nil
		}
		return value, true, nil
	}
	if cursor.javaIterator != nil {
		hasNext, _, err := cursor.javaIterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "hasNext"})
		if err != nil {
			return Null(), false, err
		}
		if !hasNext.Truth() {
			cursor.done = true
			return Null(), false, nil
		}
		value, _, err := cursor.javaIterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "next"})
		return value, err == nil, err
	}
	if cursor.iterator != nil {
		value, present, err := cursor.iterator.Next(ctx)
		if err != nil {
			err = preserveNativeBoundaryError(ctx, err)
		}
		if err != nil || !present {
			cursor.done = !present
			return Null(), false, err
		}
		return value, true, nil
	}
	if cursor.index >= len(cursor.values) {
		cursor.done = true
		return Null(), false, nil
	}
	value := cursor.values[cursor.index]
	cursor.index++
	return value, true, nil
}

func invocationCallable(invocation Invocation, index int) (Callable, error) {
	value := invocation.Arg(index)
	callable, ok := value.Function()
	if !ok {
		return nil, fmt.Errorf("&%s: expected a function, received: %s",
			builtinName(invocation.Name), value.Describe())
	}
	if _, closure := callable.(sleepSequenceClosure); !closure {
		return nil, fmt.Errorf("&%s: expected &closure--received: %s",
			builtinName(invocation.Name), sequenceClosureDescription(value))
	}
	return callable, nil
}

func invocationSequenceClosure(invocation Invocation, index int) (Callable, error) {
	value := invocation.Arg(index)
	callable, ok := value.Function()
	if !ok {
		return nil, fmt.Errorf("&%s: expected &closure--received: %s",
			builtinName(invocation.Name), sequenceClosureDescription(value))
	}
	if _, closure := callable.(sleepSequenceClosure); !closure {
		return nil, fmt.Errorf("&%s: expected &closure--received: %s",
			builtinName(invocation.Name), sequenceClosureDescription(value))
	}
	return callable, nil
}

func sequenceClosureDescription(value Value) string {
	callable, ok := value.Function()
	if !ok {
		return value.Describe()
	}
	// BasicStrings registers &asc with its func_asc bridge object. The JVM
	// appends an identity hash to this class name; OPFOR intentionally omits
	// that process-specific suffix while preserving the meaningful diagnostic.
	if native, ok := callable.(*runtimeCallable); ok &&
		strings.EqualFold(strings.TrimPrefix(native.name, "&"), "asc") {
		return "sleep.bridges.BasicStrings$func_asc"
	}
	return value.Describe()
}

func sequenceInvocationError(ctx context.Context, invocation Invocation, err error) error {
	if err == nil || currentFiber(ctx) == nil {
		return err
	}
	message := strings.TrimPrefix(err.Error(), "&"+builtinName(invocation.Name)+": ")
	return sleepBridgeIllegalArgument(message)
}

func builtinSublist(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}

	length := array.Len()
	if array.backend == nil {
		cells, err := array.snapshotCells()
		if err != nil {
			return Null(), err
		}
		length = len(cells)
	}
	start, end := 0, length
	if len(invocation.Arguments) > 1 {
		start = int(sleepInt32(invocation.Arg(1)))
	}
	if len(invocation.Arguments) > 2 {
		end = int(sleepInt32(invocation.Arg(2)))
	}
	originalStart, originalEnd := start, end
	if start < 0 {
		start += length
	}
	if end < 0 {
		end += length
	}
	if end > length {
		end = length
	}
	if start < 0 || start > length || end < 0 || start > end {
		message := fmt.Sprintf("illegal subarray(%s, %d -> %d, %d -> %d)",
			ArrayValue(array).Describe(), originalStart, start, originalEnd, end)
		// BasicUtilities throws this bridge exception. Script execution turns an
		// uncaught bridge exception into a warning and exits the active block;
		// direct library invocation continues to receive an ordinary Go error.
		if currentFiber(ctx) != nil {
			return Null(), &uncaughtScriptWarning{err: errors.New(message)}
		}
		return Null(), fmt.Errorf("&%s: %s", builtinName(invocation.Name), message)
	}
	view, err := array.sublistAtExecution(ctx, invocation, start, end)
	if err != nil {
		return Null(), err
	}
	return ArrayValue(view), nil
}

func optionalArrayValues(ctx context.Context, invocation Invocation, index int) ([]Value, error) {
	if index >= len(invocation.Arguments) {
		return nil, nil
	}
	array, ok := invocation.Arg(index).Array()
	if !ok {
		return nil, nil
	}
	if array.backend != nil {
		return iteratorValues(ctx, invocation.Arg(index), invocation.Name)
	}
	cells, err := array.snapshotCells()
	if err != nil {
		return nil, err
	}
	return valuesFromCells(cells), nil
}

func builtinAddAll(ctx context.Context, invocation Invocation) (Value, error) {
	target, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}
	additions, err := optionalArrayValues(ctx, invocation, 1)
	if err != nil {
		return Null(), err
	}
	script := executionMutationScript(ctx, invocation)
	err = target.mutateCellsAtExecution(ctx, script, true, func(cells []*Cell) ([]*Cell, error) {
		initial := valuesFromCells(cells)
		growth := 0
		for _, value := range additions {
			if !matchesAnySleepValue(value, initial) {
				growth++
			}
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, growth); err != nil {
			return nil, err
		}
		for _, value := range additions {
			// Sleep deliberately compares against the original target only. Thus
			// equal source values are all appended when initially absent.
			if !matchesAnySleepValue(value, initial) {
				cells = append(cells, NewCell(value))
			}
		}
		return cells, nil
	})
	if err != nil {
		return Null(), err
	}
	return ArrayValue(target), nil
}

func builtinRemoveAll(ctx context.Context, invocation Invocation) (Value, error) {
	return retainOrRemoveAll(ctx, invocation, false)
}

func builtinRetainAll(ctx context.Context, invocation Invocation) (Value, error) {
	return retainOrRemoveAll(ctx, invocation, true)
}

func retainOrRemoveAll(ctx context.Context, invocation Invocation, retain bool) (Value, error) {
	target, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}
	set, err := optionalArrayValues(ctx, invocation, 1)
	if err != nil {
		return Null(), err
	}
	err = target.mutateCellsForInvocation(ctx, invocation, true, func(cells []*Cell) ([]*Cell, error) {
		kept := make([]*Cell, 0, len(cells))
		for _, cell := range cells {
			matched := matchesAnySleepValue(cell.Get(), set)
			if matched == retain {
				kept = append(kept, cell)
			}
		}
		return kept, nil
	})
	if err != nil {
		return Null(), err
	}
	return ArrayValue(target), nil
}

func builtinContains(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		return Null(), err
	}
	if array.backend != nil {
		values, err := iteratorValues(ctx, invocation.Arg(0), invocation.Name)
		if err != nil {
			return Null(), err
		}
		return Bool(matchesAnySleepValue(invocation.Arg(1), values)), nil
	}
	cells, err := array.snapshotCells()
	if err != nil {
		return Null(), err
	}
	return Bool(matchesAnySleepValue(invocation.Arg(1), valuesFromCells(cells))), nil
}

func builtinContainsAll(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		return Null(), err
	}
	var values []Value
	if array.backend != nil {
		values, err = iteratorValues(ctx, invocation.Arg(0), invocation.Name)
	} else {
		var cells []*Cell
		cells, err = array.snapshotCells()
		values = valuesFromCells(cells)
	}
	if err != nil {
		return Null(), err
	}
	cursor, err := invocationSequenceCursor(invocation, 1)
	if err != nil {
		if currentFiber(ctx) != nil {
			message := strings.TrimPrefix(err.Error(), "&"+builtinName(invocation.Name)+": ")
			return Null(), &uncaughtScriptWarning{err: fmt.Errorf("%s", message)}
		}
		return Null(), err
	}
	for {
		value, ok, err := cursor.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !ok {
			return Bool(true), nil
		}
		if !matchesAnySleepValue(value, values) {
			return Bool(false), nil
		}
	}
}

func builtinClear(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return Null(), nil
	}
	value := invocation.Arg(0)
	if array, ok := value.Array(); ok {
		if err := array.mutateCellsForInvocation(ctx, invocation, true, func([]*Cell) ([]*Cell, error) { return nil, nil }); err != nil {
			return Null(), err
		}
		return Null(), nil
	}
	if _, ok := value.Hash(); ok {
		// BasicUtilities.clear replaces the containing Scalar with a fresh
		// ScalarHash instead of mutating the old hash. Consequently aliases to
		// the old hash retain their entries while the cleared reference detaches.
		// A non-reference temporary is simply discarded, matching Sleep.
		if err := setInvocationArgumentAtExecution(ctx, invocation, 0, HashValue(NewHash())); err != nil {
			return Null(), err
		}
		return Null(), nil
	}
	if err := setInvocationArgumentAtExecution(ctx, invocation, 0, Null()); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func builtinIsEmpty(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	switch value.Kind() {
	case KindNull:
		return Bool(true), nil
	case KindString:
		return Bool(value.String() == ""), nil
	case KindArray:
		array, _ := value.Array()
		if array.backend != nil {
			return Bool(array.Len() == 0), nil
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return Null(), err
		}
		return Bool(len(cells) == 0), nil
	case KindHash:
		hash, _ := value.Hash()
		for _, key := range hash.KeyValues() {
			item, exists, err := hash.getValueForInvocation(ctx, invocation, key)
			if err != nil {
				return Null(), err
			}
			if exists && !item.IsNull() {
				return Bool(false), nil
			}
		}
		return Bool(true), nil
	default:
		return Bool(false), nil
	}
}

func builtinMap(ctx context.Context, invocation Invocation) (Value, error) {
	return transformSequence(ctx, invocation, "map")
}

func builtinFilter(ctx context.Context, invocation Invocation) (Value, error) {
	return transformSequence(ctx, invocation, "filter")
}

func builtinGrep(ctx context.Context, invocation Invocation) (Value, error) {
	return transformSequence(ctx, invocation, "grep")
}

func transformSequence(ctx context.Context, invocation Invocation, mode string) (Value, error) {
	callable, err := invocationSequenceClosure(invocation, 0)
	if err != nil {
		return Null(), sequenceInvocationError(ctx, invocation, err)
	}
	cursor, err := invocationSequenceCursor(invocation, 1)
	if err != nil {
		if currentFiber(ctx) != nil {
			message := strings.TrimPrefix(err.Error(), "&"+builtinName(invocation.Name)+": ")
			return Null(), &uncaughtScriptWarning{err: fmt.Errorf("%s", message)}
		}
		return Null(), err
	}
	result := NewArray()
	for {
		value, ok, err := cursor.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !ok {
			return ArrayValue(result), nil
		}
		mapped, err := invokeTracedClosure(ctx, invocation.Arg(0), "eval", []Value{value}, callable)
		if err != nil {
			return Null(), err
		}
		switch mode {
		case "map":
			if err := result.appendValuesAtExecution(ctx, invocation, mapped); err != nil {
				return Null(), err
			}
		case "filter":
			if !mapped.IsNull() {
				if err := result.appendValuesAtExecution(ctx, invocation, mapped); err != nil {
					return Null(), err
				}
			}
		case "grep":
			if mapped.Truth() {
				if err := result.appendValuesAtExecution(ctx, invocation, value); err != nil {
					return Null(), err
				}
			}
		}
	}
}

func builtinReduce(ctx context.Context, invocation Invocation) (Value, error) {
	// BasicUtilities first obtains a default empty scalar and only enters the
	// reduce branch when that scalar is a function. reduce() therefore returns
	// $null without attempting BridgeUtilities.getFunction.
	if len(invocation.Arguments) == 0 {
		return Null(), nil
	}
	callable, err := invocationSequenceClosure(invocation, 0)
	if err != nil {
		return Null(), sequenceInvocationError(ctx, invocation, err)
	}
	cursor, err := invocationSequenceCursor(invocation, 1)
	if err != nil {
		return Null(), err
	}
	a, ok, err := cursor.next(ctx)
	if err != nil {
		return Null(), err
	}
	if !ok {
		a = Null()
	}
	b, ok, err := cursor.next(ctx)
	if err != nil {
		return Null(), err
	}
	if !ok {
		b = Null()
	}
	// Sleep's initial reduce call receives ($1 = second, $2 = first).
	accumulator, err := callable.Invoke(ctx, b, a)
	if err != nil {
		return Null(), err
	}
	for {
		next, ok, err := cursor.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !ok {
			return accumulator, nil
		}
		// Subsequent calls receive ($1 = accumulator, $2 = next).
		accumulator, err = callable.Invoke(ctx, accumulator, next)
		if err != nil {
			return Null(), err
		}
	}
}

func builtinSum(ctx context.Context, invocation Invocation) (Value, error) {
	primary, err := invocationSequenceCursor(invocation, 0)
	if err != nil {
		return Null(), err
	}
	auxiliaryCapacity := len(invocation.Arguments) - 1
	if auxiliaryCapacity < 0 {
		auxiliaryCapacity = 0
	}
	auxiliary := make([]*sequenceCursor, 0, auxiliaryCapacity)
	for index := 1; index < len(invocation.Arguments); index++ {
		cursor, err := newSequenceCursor(invocation.Arg(index), invocation.Name)
		if err != nil {
			return Null(), err
		}
		auxiliary = append(auxiliary, cursor)
	}

	result := 0.0
	for {
		value, ok, err := primary.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !ok {
			return Double(result), nil
		}
		product := sleepFloat64(value)
		for _, cursor := range auxiliary {
			factor, ok, err := cursor.next(ctx)
			if err != nil {
				return Null(), err
			}
			if !ok {
				product = 0
				break
			}
			product *= sleepFloat64(factor)
		}
		result += product
	}
}

func builtinSort(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) != 2 {
		return Null(), sleepBridgeIllegalArgument("&sort requires a function to specify how to sort the data")
	}
	callable, err := invocationSequenceClosure(invocation, 0)
	if err != nil {
		return Null(), sequenceInvocationError(ctx, invocation, err)
	}
	array, err := invocationWorkableArray(ctx, invocation, 1)
	if err != nil {
		return Null(), err
	}
	flow := &sleepSortComparatorFlow{}
	result, err := sortSequenceArray(ctx, invocation.Name, array, func(left, right Value) (int, error) {
		// BasicStrings.CompareFunction invokes Sleep comparators with the
		// callback message "&sort". That name is observable as $0 even though
		// the compared values remain the only positional arguments.
		return flow.compare(ctx, func() (Value, error) {
			if named, ok := callable.(interface {
				invokeNamed(context.Context, string, ...Value) (Value, error)
			}); ok {
				return named.invokeNamed(ctx, "&sort", left, right)
			}
			return callable.Invoke(ctx, left, right)
		})
	})
	if err != nil {
		return Null(), err
	}
	if pending := flow.pendingThrow(); pending != nil {
		return result, pending
	}
	return result, nil
}

func builtinSortAscending(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return ArrayValue(NewArray()), nil
	}
	array, err := invocationWorkableArray(ctx, invocation, 0)
	if err != nil {
		return Null(), err
	}
	return sortSequenceArray(ctx, invocation.Name, array, func(left, right Value) (int, error) {
		return sleepStringCompareValues(left, right), nil
	})
}

func builtinSortNumeric(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return ArrayValue(NewArray()), nil
	}
	array, err := invocationWorkableArray(ctx, invocation, 0)
	if err != nil {
		return Null(), err
	}
	return sortSequenceArray(ctx, invocation.Name, array, func(left, right Value) (int, error) {
		// This deliberately mirrors Sleep's (int)(left.longValue() -
		// right.longValue()), including Java's narrowing behavior.
		return int(int32(sleepInt64(left) - sleepInt64(right))), nil
	})
}

func builtinSortDouble(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return ArrayValue(NewArray()), nil
	}
	array, err := invocationWorkableArray(ctx, invocation, 0)
	if err != nil {
		return Null(), err
	}
	return sortSequenceArray(ctx, invocation.Name, array, func(left, right Value) (int, error) {
		a, b := sleepFloat64(left), sleepFloat64(right)
		if a == b {
			return 0, nil
		}
		if a < b {
			return -1, nil
		}
		return 1, nil
	})
}

func sortSequenceArray(ctx context.Context, name string, array *Array, compare func(Value, Value) (int, error)) (Value, error) {
	original, err := array.snapshotCells()
	if err != nil {
		return Null(), err
	}
	sorted, err := sleepStableTimSort(original, func(left, right *Cell) (int, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		return compare(left.Get(), right.Get())
	})
	if err != nil {
		if errors.Is(err, errSleepComparatorContract) {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}

	err = array.mutateCellsAtExecution(ctx, executionMutationScript(ctx, Invocation{}), false, func(current []*Cell) ([]*Cell, error) {
		if !sameCellSequence(current, original) {
			return nil, fmt.Errorf("&%s: array changed while it was being sorted", builtinName(name))
		}
		return sorted, nil
	})
	if err != nil {
		return Null(), err
	}
	return ArrayValue(array), nil
}

// invocationWorkableArray mirrors BridgeUtilities.getWorkableArray: sorting
// an array backed by a read-only Java CollectionWrapper operates on a mutable
// copy, while ordinary Sleep arrays are sorted in place.
func invocationWorkableArray(ctx context.Context, invocation Invocation, index int) (*Array, error) {
	array, err := invocationArray(invocation, index)
	if err != nil {
		return nil, err
	}
	if array.isReadOnly() {
		if array.backend != nil {
			values := make([]Value, 0)
			iterator := array.backend.iterator(ArrayValue(array))
			for {
				item, present, err := iterator.next(ctx)
				if err != nil {
					return nil, err
				}
				if !present {
					return NewArray(values...), nil
				}
				if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
					return nil, err
				}
				values = append(values, item.value)
			}
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return nil, err
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(cells)); err != nil {
			return nil, err
		}
		return NewArray(valuesFromCells(cells)...), nil
	}
	return array, nil
}

func builtinSequenceReverse(ctx context.Context, invocation Invocation) (Value, error) {
	cursor, err := invocationSequenceCursor(invocation, 0)
	if err != nil {
		return Null(), err
	}
	values := make([]Value, 0)
	for {
		value, ok, err := cursor.next(ctx)
		if err != nil {
			return Null(), err
		}
		if !ok {
			break
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), err
		}
		values = append(values, value)
	}
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
	return ArrayValue(NewArray(values...)), nil
}

var aggressorRangePart = regexp.MustCompile(`^([+-]?\d+)(?:\s*-\s*([+-]?\d+))?$`)

// builtinRange implements Aggressor's documented comma-separated range
// notation: range("2,4-6") produces @(2, 4, 5). Range ends are exclusive.
func builtinRange(ctx context.Context, invocation Invocation) (Value, error) {
	description := strings.TrimSpace(invocation.Arg(0).String())
	if description == "" {
		return ArrayValue(NewArray()), nil
	}
	values := make([]Value, 0)
	for _, raw := range strings.Split(description, ",") {
		part := strings.TrimSpace(raw)
		match := aggressorRangePart.FindStringSubmatch(part)
		if match == nil {
			return Null(), fmt.Errorf("&%s: invalid range %q", builtinName(invocation.Name), part)
		}
		start, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return Null(), fmt.Errorf("&%s: invalid range %q: %w", builtinName(invocation.Name), part, err)
		}
		if match[2] == "" {
			if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
				return Null(), err
			}
			values = append(values, sequenceInteger(start))
			continue
		}
		end, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return Null(), fmt.Errorf("&%s: invalid range %q: %w", builtinName(invocation.Name), part, err)
		}
		step := int64(1)
		if start > end {
			step = -1
		}
		for current := start; current != end; current += step {
			if err := ctx.Err(); err != nil {
				return Null(), err
			}
			if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
				return Null(), err
			}
			values = append(values, sequenceInteger(current))
			if (step > 0 && current == math.MaxInt64) || (step < 0 && current == math.MinInt64) {
				return Null(), fmt.Errorf("&%s: range %q overflows", builtinName(invocation.Name), part)
			}
		}
	}
	return ArrayValue(NewArray(values...)), nil
}

func sequenceInteger(value int64) Value {
	if value >= math.MinInt32 && value <= math.MaxInt32 {
		return Int(int32(value))
	}
	return Long(value)
}

func builtinZip(ctx context.Context, invocation Invocation) (Value, error) {
	cursors := make([]*sequenceCursor, len(invocation.Arguments))
	for index := range invocation.Arguments {
		cursor, err := newSequenceCursor(invocation.Arg(index), invocation.Name)
		if err != nil {
			return Null(), err
		}
		cursors[index] = cursor
	}
	result := NewArray()
	if len(cursors) == 0 {
		return ArrayValue(result), nil
	}
	for {
		row := make([]Value, len(cursors))
		for index, cursor := range cursors {
			value, ok, err := cursor.next(ctx)
			if err != nil {
				return Null(), err
			}
			if !ok {
				return ArrayValue(result), nil
			}
			row[index] = value
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(row)+1); err != nil {
			return Null(), err
		}
		result.Append(ArrayValue(NewArray(row...)))
	}
}

func builtinMapValues(ctx context.Context, invocation Invocation) (Value, error) {
	callable, err := invocationSequenceClosure(invocation, 0)
	if err != nil {
		return Null(), sequenceInvocationError(ctx, invocation, err)
	}
	hash, ok := invocation.Arg(1).Hash()
	if !ok {
		return Null(), fmt.Errorf("&%s: expected hash. received %s",
			builtinName(invocation.Name), invocation.Arg(1).Describe())
	}
	result := NewHash()
	for _, key := range hash.KeyValues() {
		value, exists, err := hash.getValueForInvocation(ctx, invocation, key)
		if err != nil {
			return Null(), err
		}
		if !exists || value.IsNull() {
			continue
		}
		mapped, err := callable.Invoke(ctx, value, key)
		if err != nil {
			return Null(), err
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), err
		}
		result.SetValue(key, mapped)
	}
	return HashValue(result), nil
}
