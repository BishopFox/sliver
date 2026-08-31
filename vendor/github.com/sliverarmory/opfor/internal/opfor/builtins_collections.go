package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
)

// collectionFunctions returns the first portable tranche of Sleep collection,
// conversion, and string functions. The semantic references are Sleep 2.1's
// src/sleep/bridges/{BasicUtilities,BasicNumbers,BasicStrings}.java.
func (r *Runtime) collectionFunctions() map[string]NativeFunc {
	state := &collectionBuiltinState{ordered: make(map[uintptr]struct{})}
	return map[string]NativeFunc{
		"@":                state.array,
		"%":                state.hash,
		"array":            state.array,
		"hash":             state.hash,
		"ohash":            state.hash,
		"ohasha":           state.hash,
		"scalar":           builtinScalar,
		"size":             builtinSize,
		"push":             builtinPush,
		"pop":              builtinPop,
		"shift":            builtinShift,
		"unshift":          builtinUnshift,
		"add":              builtinAdd,
		"remove":           builtinRemove,
		"setMissPolicy":    builtinSetMissPolicy,
		"setRemovalPolicy": builtinSetRemovalPolicy,
		"keys":             builtinKeys,
		"values":           builtinValues,
		"copy":             state.copy,
		"flatten":          builtinFlatten,
		"typeOf":           state.typeOf,
		"int":              builtinInt,
		"long":             builtinLong,
		"double":           builtinDouble,
		"strrep":           builtinStrrep,
		"substr":           builtinSubstr,
		"uc":               builtinUpper,
		"lc":               builtinLower,
		"trim":             builtinTrim,
	}
}

type collectionBuiltinState struct {
	orderedMu sync.RWMutex
	ordered   map[uintptr]struct{}
}

type sleepClass string

func (class sleepClass) String() string { return "class " + string(class) }

type sleepKeyValue struct {
	key   Value
	value Value
}

func (pair sleepKeyValue) String() string {
	return pair.key.String() + "=" + pair.value.String()
}

func (state *collectionBuiltinState) array(_ context.Context, invocation Invocation) (Value, error) {
	array, err := newRuntimeArray(invocation.Runtime, invocation.Values()...)
	if err != nil {
		return Null(), err
	}
	return ArrayValue(array), nil
}

func (state *collectionBuiltinState) hash(_ context.Context, invocation Invocation) (Value, error) {
	type entry struct {
		key   string
		value Value
	}
	entries := make([]entry, 0, len(invocation.Arguments))
	unique := make(map[string]struct{}, len(invocation.Arguments))
	for _, argument := range invocation.Arguments {
		key, value, err := hashArgument(argument)
		if err != nil {
			if invocation.Runtime != nil {
				invocation.Runtime.writeWarning(err.Error(), invocation.Span)
			}
			return Null(), err
		}
		entries = append(entries, entry{key: key, value: value})
		unique[sleepCanonicalString(String(key))] = struct{}{}
	}
	if err := reserveCollectionEntries(invocation.Runtime, len(unique)); err != nil {
		return Null(), err
	}

	hash := NewHash()
	if invocation.Name == "ohash" {
		hash = NewOrderedHash()
		state.markOrdered(hash)
	} else if invocation.Name == "ohasha" {
		hash = NewAccessOrderedHash()
		state.markOrdered(hash)
	}
	for _, entry := range entries {
		hash.Set(entry.key, entry.value)
	}
	return HashValue(hash), nil
}

func (state *collectionBuiltinState) markOrdered(hash *Hash) {
	identity := reflect.ValueOf(hash).Pointer()
	state.orderedMu.Lock()
	state.ordered[identity] = struct{}{}
	state.orderedMu.Unlock()
	goruntime.SetFinalizer(hash, func(finalized *Hash) {
		state.orderedMu.Lock()
		delete(state.ordered, reflect.ValueOf(finalized).Pointer())
		state.orderedMu.Unlock()
	})
}

func (state *collectionBuiltinState) isOrdered(hash *Hash) bool {
	state.orderedMu.RLock()
	_, ordered := state.ordered[reflect.ValueOf(hash).Pointer()]
	state.orderedMu.RUnlock()
	goruntime.KeepAlive(hash)
	return ordered
}

func hashArgument(argument Argument) (string, Value, error) {
	if key, value, ok := sleepNamedArgument(argument); ok {
		return key, value, nil
	}

	value := argument.Resolve()
	if value.Kind() != KindArray && value.Kind() != KindHash {
		legacy := value.String()
		if separator := strings.IndexByte(legacy, '='); separator >= 0 {
			return legacy[:separator], String(legacy[separator+1:]), nil
		}
	}
	description := value.Describe()
	if value.Kind() == KindString {
		description = value.String()
	}
	return "", Null(), fmt.Errorf("attempted to pass a malformed key value pair: %s", description)
}

func builtinSize(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if array, ok := value.Array(); ok {
		if array.backend != nil {
			return Int(int32(array.Len())), nil
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return Null(), err
		}
		return Int(int32(len(cells))), nil
	}
	if hash, ok := value.Hash(); ok {
		keys, err := activeHashKeysForInvocation(ctx, invocation, hash, true)
		if err != nil {
			return Null(), err
		}
		return Int(int32(len(keys))), nil
	}
	return Null(), nil
}

func builtinPush(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		// BasicUtilities routes &push through expectArray, whose
		// IllegalArgumentException is consumed by the active Sleep Block.
		return Null(), sleepBridgeIllegalArgument(err.Error())
	}
	if len(invocation.Arguments) < 2 {
		return Null(), nil
	}
	values := make([]Value, 0, len(invocation.Arguments)-1)
	for _, argument := range invocation.Arguments[1:] {
		values = append(values, argument.Resolve())
	}
	if err := array.appendValuesAtExecution(ctx, invocation, values...); err != nil {
		return Null(), err
	}
	return values[len(values)-1], nil
}

func builtinPop(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			// BasicUtilities obtains an empty scalar and passes it through
			// expectArray, whose IllegalArgumentException is a block warning.
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}
	return removeArrayEnd(ctx, invocation, array, false)
}

func builtinShift(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		if currentFiber(ctx) != nil {
			// BridgeUtilities.getArray substitutes an empty array for a missing,
			// null, or non-array scalar. Removing index zero then supplies the
			// stable ListContainer index warning.
			return Null(), sleepBridgeInvalidIndex("Index: 0, Size: 0")
		}
		return Null(), err
	}
	return removeArrayEnd(ctx, invocation, array, true)
}

func removeArrayEnd(ctx context.Context, invocation Invocation, array *Array, front bool) (Value, error) {
	var removed *Cell
	err := array.mutateCellsForInvocation(ctx, invocation, true, func(cells []*Cell) ([]*Cell, error) {
		if len(cells) == 0 {
			if currentFiber(ctx) != nil {
				index := 0
				if !front {
					index = -1
				}
				return nil, sleepBridgeInvalidIndex(fmt.Sprintf("Index: %d, Size: 0", index))
			}
			return nil, fmt.Errorf("&%s: array is empty", builtinName(invocation.Name))
		}
		index := len(cells) - 1
		if front {
			index = 0
		}
		removed = cells[index]
		return append(cells[:index], cells[index+1:]...), nil
	})
	if err != nil {
		return Null(), err
	}
	return removed.Get(), nil
}

func builtinUnshift(ctx context.Context, invocation Invocation) (Value, error) {
	array, err := invocationArray(invocation, 0)
	if err != nil {
		return Null(), err
	}
	if len(invocation.Arguments) < 2 {
		return Null(), nil
	}
	values := make([]Value, 0, len(invocation.Arguments)-1)
	last := Null()
	for _, argument := range invocation.Arguments[1:] {
		last = argument.Resolve()
		values = append(values, last)
	}
	script := executionMutationScript(ctx, invocation)
	if err := array.mutateCellsAtExecution(ctx, script, true, func(current []*Cell) ([]*Cell, error) {
		if err := reserveCollectionEntriesAtExecution(ctx, script, len(values)); err != nil {
			return nil, err
		}
		cells := make([]*Cell, len(values))
		for index, value := range values {
			cells[index] = NewCell(value)
		}
		return append(cells, current...), nil
	}); err != nil {
		return Null(), err
	}
	return last, nil
}

func builtinAdd(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if array, ok := value.Array(); ok {
		item := invocation.Arg(1)
		index := int32(0)
		if len(invocation.Arguments) > 2 {
			index = sleepInt32(invocation.Arg(2))
		}
		if err := insertArray(ctx, invocation, array, int(index), item); err != nil {
			return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
		}
		return value, nil
	}
	if hash, ok := value.Hash(); ok {
		for _, argument := range invocation.Arguments[1:] {
			key, item, err := hashArgument(argument)
			if err != nil {
				return Null(), err
			}
			if err := hash.SetContext(ctx, key, item); err != nil {
				return Null(), err
			}
		}
		return value, nil
	}
	return Null(), nil
}

func builtinSetMissPolicy(ctx context.Context, invocation Invocation) (Value, error) {
	hash, ok := invocation.Arg(0).Hash()
	if !ok || hash == nil || !hash.ordered {
		err := fmt.Errorf("&%s: expected an ordered hash, received: %s", builtinName(invocation.Name), invocation.Arg(0).Describe())
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}
	policy, err := invocationCallable(invocation, 1)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + invocation.Arg(1).Describe())
		}
		return Null(), err
	}
	if err := hash.setMissPolicyAtExecution(ctx, invocation, policy); err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return Null(), nil
}

func builtinSetRemovalPolicy(ctx context.Context, invocation Invocation) (Value, error) {
	hash, ok := invocation.Arg(0).Hash()
	if !ok || hash == nil || !hash.ordered {
		err := fmt.Errorf("&%s: expected an ordered hash, received: %s", builtinName(invocation.Name), invocation.Arg(0).Describe())
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		return Null(), err
	}
	policy, err := invocationCallable(invocation, 1)
	if err != nil {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + invocation.Arg(1).Describe())
		}
		return Null(), err
	}
	if err := hash.setRemovalPolicyAtExecution(ctx, invocation, policy); err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return Null(), nil
}

func insertArray(ctx context.Context, invocation Invocation, array *Array, index int, value Value) error {
	script := executionMutationScript(ctx, invocation)
	return array.mutateCellsAtExecution(ctx, script, true, func(cells []*Cell) ([]*Cell, error) {
		position := index
		if position < 0 {
			position += len(cells) + 1
		}
		if position < 0 || position > len(cells) {
			return nil, fmt.Errorf("index %d out of range for array of size %d", position, len(cells))
		}
		if err := reserveCollectionEntriesAtExecution(ctx, script, 1); err != nil {
			return nil, err
		}
		cells = append(cells, nil)
		copy(cells[position+1:], cells[position:])
		cells[position] = NewCell(value)
		return cells, nil
	})
}

func builtinRemove(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		fiber := currentFiber(ctx)
		iterator := activeForeachIterator(fiber)
		if iterator != nil {
			if err := removeIteratorAtExecution(ctx, invocation, iterator); err != nil {
				if errors.Is(err, errReadOnlyIterator) && fiber != nil {
					return Null(), &uncaughtScriptWarning{err: errors.New("iterator is read-only")}
				}
				return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
			}
			return iterator.sourceValue(), nil
		}
		err := fmt.Errorf("&%s: no active foreach loop to remove element from", builtinName(invocation.Name))
		if fiber != nil {
			return Null(), &uncaughtScriptWarning{err: err}
		}
		return Null(), err
	}
	value := invocation.Arg(0)
	targets := make([]Value, 0, len(invocation.Arguments)-1)
	for _, argument := range invocation.Arguments[1:] {
		targets = append(targets, argument.Resolve())
	}
	if array, ok := value.Array(); ok {
		if err := removeArrayValuesAtExecution(ctx, invocation, array, targets); err != nil {
			return Null(), err
		}
	} else if hash, ok := value.Hash(); ok {
		if err := removeHashValuesAtExecution(ctx, invocation, hash, targets); err != nil {
			return Null(), err
		}
	}
	return value, nil
}

func activeForeachIterator(fiber *fiber) valueIterator {
	for current := fiber; current != nil; current = current.caller {
		if len(current.iterators) != 0 {
			return current.iterators[len(current.iterators)-1]
		}
	}
	return nil
}

func removeArrayValues(array *Array, targets []Value) error {
	return array.mutateCells(true, func(cells []*Cell) ([]*Cell, error) {
		kept := make([]*Cell, 0, len(cells))
		for _, cell := range cells {
			if !matchesAnySleepValue(cell.Get(), targets) {
				kept = append(kept, cell)
			}
		}
		return kept, nil
	})
}

func removeArrayValuesAtExecution(ctx context.Context, invocation Invocation, array *Array, targets []Value) error {
	return array.mutateCellsForInvocation(ctx, invocation, true, func(cells []*Cell) ([]*Cell, error) {
		kept := make([]*Cell, 0, len(cells))
		for _, cell := range cells {
			if !matchesAnySleepValue(cell.Get(), targets) {
				kept = append(kept, cell)
			}
		}
		return kept, nil
	})
}

func removeHashValues(hash *Hash, targets []Value) error {
	return removeHashValuesAtExecution(context.Background(), Invocation{}, hash, targets)
}

func removeHashValuesAtExecution(ctx context.Context, invocation Invocation, hash *Hash, targets []Value) error {
	return hash.mutateForInvocation(ctx, invocation, func() error {
		if hash.readOnly {
			return ErrReadOnlyHash
		}
		kept := make([]string, 0, len(hash.order))
		removed := false
		for _, key := range hash.order {
			cell, exists := hash.items[key]
			if !exists || matchesAnySleepValue(cell.Get(), targets) {
				if exists {
					delete(hash.items, key)
					delete(hash.keyValues, key)
					removed = true
				}
				continue
			}
			kept = append(kept, key)
		}
		hash.order = kept
		if removed {
			hash.modCount++
		}
		return nil
	})
}

func matchesAnySleepValue(value Value, targets []Value) bool {
	for _, target := range targets {
		if sameSleepValue(value, target) {
			return true
		}
	}
	return false
}

func sameSleepValue(left, right Value) bool {
	if leftArray, ok := left.Array(); ok {
		rightArray, rightOK := right.Array()
		return rightOK && leftArray == rightArray
	}
	if _, ok := right.Array(); ok {
		return false
	}
	if leftHash, ok := left.Hash(); ok {
		rightHash, rightOK := right.Hash()
		return rightOK && leftHash == rightHash
	}
	if _, ok := right.Hash(); ok {
		return false
	}
	if isSleepReference(left) || isSleepReference(right) {
		if left.Kind() != right.Kind() {
			return false
		}
		return sameSleepReference(left.data, right.data)
	}
	return sleepIdentityString(left) == sleepIdentityString(right)
}

func sameSleepReference(left, right any) bool {
	leftValue, rightValue := reflect.ValueOf(left), reflect.ValueOf(right)
	if !leftValue.IsValid() || !rightValue.IsValid() || leftValue.Type() != rightValue.Type() {
		return false
	}
	switch leftValue.Kind() {
	case reflect.Chan, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return leftValue.Pointer() == rightValue.Pointer()
	case reflect.Func:
		// Go exposes a function's code address, not closure identity.
		return false
	default:
		return comparableEqual(left, right)
	}
}

func isSleepReference(value Value) bool {
	return value.Kind() == KindFunction || value.Kind() == KindObject
}

func sleepIdentityString(value Value) string {
	if value.Kind() != KindDouble {
		return value.String()
	}
	number := value.data.(float64)
	switch {
	case math.IsNaN(number):
		return "NaN"
	case math.IsInf(number, 1):
		return "Infinity"
	case math.IsInf(number, -1):
		return "-Infinity"
	}
	text := strconv.FormatFloat(number, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}

func builtinKeys(ctx context.Context, invocation Invocation) (Value, error) {
	hash, ok := invocation.Arg(0).Hash()
	if !ok {
		return Null(), nil
	}
	if hash.backend != nil {
		if err := executionMutationError(ctx, executionMutationScript(ctx, invocation)); err != nil {
			return Null(), err
		}
		// MapWrapper.keys returns a fresh CollectionWrapper around the map's
		// live keySet, including its independent lazy getAt cache.
		var fallback *runtimeResourceAccount
		if invocation.Runtime != nil {
			fallback = invocation.Runtime.resources
		}
		keys, err := hash.backend.keysArray(fallback)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(keys), nil
	}
	keys, err := activeHashKeysForInvocation(ctx, invocation, hash, true)
	if err != nil {
		return Null(), err
	}
	array, err := newRuntimeArray(invocation.Runtime, keys...)
	if err != nil {
		return Null(), err
	}
	return ArrayValue(array), nil
}

func builtinValues(ctx context.Context, invocation Invocation) (Value, error) {
	hash, ok := invocation.Arg(0).Hash()
	if !ok {
		return Null(), nil
	}
	if len(invocation.Arguments) == 1 {
		values, err := activeHashValuesForInvocation(ctx, invocation, hash, true)
		if err != nil {
			return Null(), err
		}
		array, err := newRuntimeArray(invocation.Runtime, values...)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	}

	keys, err := iteratorValuesForCollection(ctx, invocation.Runtime, invocation.Arg(1), invocation.Name)
	if err != nil {
		return Null(), err
	}
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		cell, accessErr := hash.EnsureValueContext(ctx, key)
		if accessErr != nil {
			return Null(), accessErr
		}
		values = append(values, cell.Get())
	}
	return ArrayValue(NewArray(values...)), nil
}

func activeHashKeys(hash *Hash, cleanup bool) []Value {
	keys, _ := activeHashKeysAtExecution(context.Background(), nil, hash, cleanup)
	return keys
}

func activeHashKeysForInvocation(ctx context.Context, invocation Invocation, hash *Hash, cleanup bool) ([]Value, error) {
	return activeHashKeysAtExecution(ctx, executionMutationScript(ctx, invocation), hash, cleanup)
}

func activeHashKeysAtExecution(ctx context.Context, script *Script, hash *Hash, cleanup bool) ([]Value, error) {
	if hash != nil && hash.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return nil, err
		}
		return hash.backend.keyValues(), nil
	}
	var keys []Value
	err := hash.mutateAtExecution(ctx, script, func() error {
		keys = activeHashKeysLocked(hash, cleanup)
		return nil
	})
	return keys, err
}

func activeHashKeysLocked(hash *Hash, cleanup bool) []Value {
	cleanup = cleanup && !hash.readOnly
	keys := make([]Value, 0, len(hash.order))
	if hash.ordered {
		for _, key := range hash.compatibleKeysLocked() {
			cell, exists := hash.items[key]
			if exists && !cell.Get().IsNull() {
				keys = append(keys, hash.keyValueLocked(key))
			}
		}
		if cleanup && len(hash.items) > len(keys)+1 {
			hash.shouldClean = true
		}
		return keys
	}
	kept := make([]string, 0, len(hash.order))
	removed := false
	for _, key := range hash.compatibleKeysLocked() {
		cell, exists := hash.items[key]
		if !exists || cell.Get().IsNull() {
			if cleanup {
				if exists {
					delete(hash.items, key)
					delete(hash.keyValues, key)
					removed = true
				}
			} else if exists {
				kept = append(kept, key)
			}
			continue
		}
		keys = append(keys, hash.keyValueLocked(key))
	}
	if cleanup {
		for _, key := range hash.order {
			if _, exists := hash.items[key]; exists {
				kept = append(kept, key)
			}
		}
		hash.order = kept
		if removed {
			hash.modCount++
		}
	}
	return keys
}

func activeHashValues(hash *Hash, cleanup bool) []Value {
	values, _ := activeHashValuesAtExecution(context.Background(), nil, hash, cleanup)
	return values
}

func activeHashValuesForInvocation(ctx context.Context, invocation Invocation, hash *Hash, cleanup bool) ([]Value, error) {
	return activeHashValuesAtExecution(ctx, executionMutationScript(ctx, invocation), hash, cleanup)
}

func activeHashValuesAtExecution(ctx context.Context, script *Script, hash *Hash, cleanup bool) ([]Value, error) {
	if hash != nil && hash.backend != nil {
		if err := executionMutationError(ctx, script); err != nil {
			return nil, err
		}
		snapshot, err := hash.backend.dataSnapshotReserved(func(count int) error {
			return reserveCollectionEntriesAtExecution(ctx, script, count)
		})
		if err != nil {
			return nil, err
		}
		snapshot.mu.RLock()
		values := activeHashValuesLocked(snapshot, false)
		snapshot.mu.RUnlock()
		return values, nil
	}
	var values []Value
	err := hash.mutateAtExecution(ctx, script, func() error {
		values = activeHashValuesLocked(hash, cleanup)
		return nil
	})
	return values, err
}

func activeHashValuesLocked(hash *Hash, cleanup bool) []Value {
	cleanup = cleanup && !hash.readOnly
	values := make([]Value, 0, len(hash.order))
	if hash.ordered {
		for _, key := range hash.compatibleKeysLocked() {
			if cell, exists := hash.items[key]; exists && !cell.Get().IsNull() {
				values = append(values, cell.Get())
			}
		}
		return values
	}
	kept := make([]string, 0, len(hash.order))
	removed := false
	for _, key := range hash.compatibleKeysLocked() {
		cell, exists := hash.items[key]
		if !exists {
			continue
		}
		value := cell.Get()
		if value.IsNull() {
			if cleanup {
				delete(hash.items, key)
				delete(hash.keyValues, key)
				removed = true
			} else {
				kept = append(kept, key)
			}
			continue
		}
		values = append(values, value)
	}
	if cleanup {
		for _, key := range hash.order {
			if _, exists := hash.items[key]; exists {
				kept = append(kept, key)
			}
		}
		hash.order = kept
		if removed {
			hash.modCount++
		}
	}
	return values
}

func (state *collectionBuiltinState) copy(ctx context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	if array, ok := value.Array(); ok {
		if array.backend != nil {
			values, err := iteratorValuesForCollection(ctx, invocation.Runtime, value, invocation.Name)
			if err != nil {
				return Null(), err
			}
			return ArrayValue(NewArray(values...)), nil
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return Null(), err
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(cells)); err != nil {
			return Null(), err
		}
		return ArrayValue(NewArray(valuesFromCells(cells)...)), nil
	}
	if value.Kind() == KindFunction {
		values, err := iteratorValuesForCollection(ctx, invocation.Runtime, value, invocation.Name)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(NewArray(values...)), nil
	}
	if hash, ok := value.Hash(); ok {
		result := NewHash()
		keys, err := activeHashKeysForInvocation(ctx, invocation, hash, true)
		if err != nil {
			return Null(), err
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(keys)); err != nil {
			return Null(), err
		}
		for _, key := range keys {
			item, exists, accessErr := hash.getValueForInvocation(ctx, invocation, key)
			if accessErr != nil {
				return Null(), accessErr
			}
			if exists && !item.IsNull() {
				result.SetValue(key, item)
			}
		}
		return HashValue(result), nil
	}
	return value, nil
}

func builtinFlatten(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return ArrayValue(NewArray()), nil
	}
	var result []Value
	visiting := make(map[*Array]bool)
	beforeAppend := func() error { return reserveCollectionEntries(invocation.Runtime, 1) }
	if err := visitIteratorValues(ctx, invocation.Arg(0), invocation.Name, func(value Value) error {
		return flattenValueAtExecution(ctx, value, &result, visiting, beforeAppend)
	}); err != nil {
		return Null(), err
	}
	return ArrayValue(NewArray(result...)), nil
}

func flattenValueAtExecution(
	ctx context.Context,
	value Value,
	result *[]Value,
	visiting map[*Array]bool,
	beforeAppend func() error,
) error {
	array, ok := value.Array()
	if !ok {
		if beforeAppend != nil {
			if err := beforeAppend(); err != nil {
				return err
			}
		}
		*result = append(*result, value)
		return nil
	}
	if visiting[array] {
		return fmt.Errorf("&flatten: cyclic array")
	}
	visiting[array] = true
	err := visitIteratorValues(ctx, value, "flatten", func(element Value) error {
		return flattenValueAtExecution(ctx, element, result, visiting, beforeAppend)
	})
	delete(visiting, array)
	return err
}

func visitIteratorValues(ctx context.Context, value Value, name string, visit func(Value) error) error {
	if array, ok := value.Array(); ok {
		if array.backend != nil {
			iterator := array.backend.iterator(value)
			for {
				item, present, err := iterator.next(ctx)
				if err != nil {
					return err
				}
				if !present {
					return nil
				}
				if err := visit(item.value); err != nil {
					return err
				}
			}
		}
		cells, err := array.snapshotCells()
		if err != nil {
			return err
		}
		for _, cell := range cells {
			if err := visit(cell.Get()); err != nil {
				return err
			}
		}
		return nil
	}
	if function, ok := value.Function(); ok {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			item, err := function.Invoke(ctx)
			if err != nil {
				return err
			}
			if item.IsNull() {
				return nil
			}
			if err := visit(item); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("&%s: expected iterator (@array or &closure)--received: %s", builtinName(name), value.Describe())
}

func iteratorValues(ctx context.Context, value Value, name string) ([]Value, error) {
	return iteratorValuesBeforeAppend(ctx, value, name, nil)
}

func iteratorValuesForCollection(ctx context.Context, runtime *Runtime, value Value, name string) ([]Value, error) {
	return iteratorValuesBeforeAppend(ctx, value, name, func(amount int) error {
		return reserveCollectionEntries(runtime, amount)
	})
}

func iteratorValuesBeforeAppend(
	ctx context.Context,
	value Value,
	name string,
	beforeAppend func(int) error,
) ([]Value, error) {
	var values []Value
	err := visitIteratorValues(ctx, value, name, func(item Value) error {
		if beforeAppend != nil {
			if err := beforeAppend(1); err != nil {
				return err
			}
		}
		values = append(values, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (state *collectionBuiltinState) typeOf(_ context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	name := "sleep.engine.types.ObjectValue"
	switch value.Kind() {
	case KindNull:
		name = "sleep.engine.types.NullValue"
	case KindInt:
		name = "sleep.engine.types.IntValue"
	case KindLong:
		name = "sleep.engine.types.LongValue"
	case KindDouble:
		name = "sleep.engine.types.DoubleValue"
	case KindString:
		name = "sleep.engine.types.StringValue"
	case KindArray:
		name = "sleep.engine.types.ListContainer"
	case KindHash:
		if hash, _ := value.Hash(); hash != nil {
			if state.isOrdered(hash) {
				name = "sleep.engine.types.OrderedHashContainer"
			} else {
				name = "sleep.engine.types.HashContainer"
			}
		}
	}
	return ObjectValue(sleepClass(name)), nil
}

func builtinScalar(_ context.Context, invocation Invocation) (Value, error) {
	value := invocation.Arg(0)
	object, ok := value.Object()
	if !ok {
		return value, nil
	}
	return scalarFromObjectAtRuntime(invocation.Runtime, object)
}

func scalarFromObject(object any) Value {
	value, _ := scalarFromObjectAtRuntime(nil, object)
	return value
}

func scalarFromObjectAtRuntime(runtime *Runtime, object any) (Value, error) {
	switch value := object.(type) {
	case nil:
		return Null(), nil
	case Value:
		return value, nil
	case *portableJavaPrimitive:
		if value == nil {
			return Null(), nil
		}
		return value.sleepValue(), nil
	case *portableJavaArray:
		if value == nil {
			return Null(), nil
		}
		return value.toSleepValueAtRuntime(runtime)
	case *serializedSleepScalar:
		if value == nil {
			return Null(), nil
		}
		return value.value, nil
	case *serializedJavaObject:
		if value == nil {
			return Null(), nil
		}
		return value.value, nil
	case bool:
		if value {
			return Int(1), nil
		}
		return Int(0), nil
	case int8:
		return Int(int32(value)), nil
	case int16:
		return Int(int32(value)), nil
	case int32:
		return Int(value), nil
	case int:
		return Int(int32(value)), nil
	case uint8:
		return Int(int32(value)), nil
	case uint16:
		return Int(int32(value)), nil
	case uint32:
		return Long(int64(value)), nil
	case uint:
		if uint64(value) <= math.MaxInt64 {
			return Long(int64(value)), nil
		}
	case int64:
		return Long(value), nil
	case uint64:
		if value <= math.MaxInt64 {
			return Long(int64(value)), nil
		}
	case float32:
		return Double(float64(value)), nil
	case float64:
		return Double(value), nil
	case string:
		return String(value), nil
	case []byte:
		return BinaryString(value), nil
	case []rune:
		return String(string(value)), nil
	}

	reflected := reflect.ValueOf(object)
	if (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface ||
		reflected.Kind() == reflect.Map || reflected.Kind() == reflect.Slice ||
		reflected.Kind() == reflect.Func) && reflected.IsNil() {
		return Null(), nil
	}
	if reflected.Kind() == reflect.Array || reflected.Kind() == reflect.Slice {
		values := make([]Value, reflected.Len())
		for index := range values {
			converted, err := scalarFromObjectAtRuntime(runtime, reflected.Index(index).Interface())
			if err != nil {
				return Null(), err
			}
			values[index] = converted
		}
		array, err := newRuntimeArray(runtime, values...)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	}
	return ObjectValue(object), nil
}

func builtinInt(_ context.Context, invocation Invocation) (Value, error) {
	return Int(sleepInt32(invocation.Arg(0))), nil
}

func builtinLong(_ context.Context, invocation Invocation) (Value, error) {
	return Long(sleepInt64(invocation.Arg(0))), nil
}

func builtinDouble(_ context.Context, invocation Invocation) (Value, error) {
	return Double(sleepFloat64(invocation.Arg(0))), nil
}

func sleepInt32(value Value) int32 {
	switch value.Kind() {
	case KindInt:
		return value.data.(int32)
	case KindLong:
		return int32(value.data.(int64))
	case KindDouble:
		return int32(value.data.(float64))
	case KindString:
		parsed, err := strconv.ParseInt(value.data.(string), 10, 32)
		if err == nil {
			return int32(parsed)
		}
	case KindObject, KindFunction:
		parsed, ok := decodeJavaInteger(value.String(), 32)
		if ok {
			return int32(parsed)
		}
	}
	return 0
}

func sleepInt64(value Value) int64 {
	switch value.Kind() {
	case KindInt:
		return int64(value.data.(int32))
	case KindLong:
		return value.data.(int64)
	case KindDouble:
		return int64(value.data.(float64))
	case KindString:
		parsed, ok := sleepParseLong(value)
		if ok {
			return parsed
		}
	case KindObject, KindFunction:
		parsed, ok := decodeJavaInteger(value.String(), 64)
		if ok {
			return parsed
		}
	}
	return 0
}

// sleepParseLong mirrors StringValue.longValue's Long.parseLong conversion.
// Java accepts Character.digit-compatible UTF-16 code units, an optional
// ASCII sign, and no radix prefix, whitespace, decimal point, or overflow.
func sleepParseLong(value Value) (int64, bool) {
	units := sleepStringUnits(value)
	if len(units) == 0 {
		return 0, false
	}

	negative := false
	index := 0
	switch units[0] {
	case '-':
		negative = true
		index = 1
	case '+':
		index = 1
	}
	if index == len(units) {
		return 0, false
	}

	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	var magnitude uint64
	for ; index < len(units); index++ {
		digit := sleepJavaDigit(units[index], 10)
		if digit < 0 || magnitude > (limit-uint64(digit))/10 {
			return 0, false
		}
		magnitude = magnitude*10 + uint64(digit)
	}
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return math.MinInt64, true
		}
		return -int64(magnitude), true
	}
	return int64(magnitude), true
}

func sleepFloat64(value Value) float64 {
	switch value.Kind() {
	case KindInt:
		return float64(value.data.(int32))
	case KindLong:
		return float64(value.data.(int64))
	case KindDouble:
		return value.data.(float64)
	case KindString:
		return parseJavaDouble(value.data.(string))
	case KindObject, KindFunction:
		text := value.String()
		if text == "true" {
			return 1
		}
		if text == "false" || text == "" {
			return 0
		}
		return parseJavaDouble(text)
	}
	return 0
}

func decodeJavaInteger(text string, bits int) (int64, bool) {
	if text == "true" {
		return 1, true
	}
	if text == "false" || text == "" {
		return 0, true
	}
	negative := false
	if text[0] == '+' || text[0] == '-' {
		negative = text[0] == '-'
		text = text[1:]
	}
	if text == "" {
		return 0, false
	}
	base := 10
	switch {
	case strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X"):
		base, text = 16, text[2:]
	case strings.HasPrefix(text, "#"):
		base, text = 16, text[1:]
	case len(text) > 1 && text[0] == '0':
		base, text = 8, text[1:]
	}
	if text == "" {
		return 0, false
	}
	magnitude, err := strconv.ParseUint(text, base, bits)
	if err != nil {
		return 0, false
	}
	if negative {
		limit := uint64(1) << (bits - 1)
		if magnitude > limit {
			return 0, false
		}
		if bits == 64 && magnitude == uint64(1)<<63 {
			return math.MinInt64, true
		}
		return -int64(magnitude), true
	}
	maximum := uint64(1)<<(bits-1) - 1
	if magnitude > maximum {
		return 0, false
	}
	return int64(magnitude), true
}

func parseJavaDouble(text string) float64 {
	text = strings.TrimSpace(text)
	switch text {
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func builtinStrrep(_ context.Context, invocation Invocation) (Value, error) {
	work := sleepStringCoercion(invocation.Arg(0))
	for index := 1; index < len(invocation.Arguments); index += 2 {
		oldText := sleepStringCoercion(invocation.Arg(index))
		newText := String("")
		if index+1 < len(invocation.Arguments) {
			newText = sleepStringCoercion(invocation.Arg(index + 1))
		}
		if sleepStringLength(oldText) != 0 {
			work = sleepStringReplaceAll(work, oldText, newText)
		}
	}
	return work, nil
}

func builtinSubstr(ctx context.Context, invocation Invocation) (Value, error) {
	// BasicStrings uses getString(..., "") and getInt(..., 0), so a wholly
	// absent argument stack is the empty string at index zero rather than an
	// EmptyStackException.
	value := String("")
	if len(invocation.Arguments) != 0 {
		value = sleepStringCoercion(invocation.Arg(0))
	}
	length := sleepStringLength(value)
	start := 0
	if len(invocation.Arguments) > 1 {
		start = int(sleepInt32(invocation.Arg(1)))
	}
	stop := length
	if len(invocation.Arguments) > 2 {
		stop = int(sleepInt32(invocation.Arg(2)))
	}
	originalStart, originalStop := start, stop
	if start < 0 {
		start += length
	}
	if stop < 0 {
		stop += length
	}
	if stop > length {
		stop = length
	}
	if start < 0 || start > length || stop < 0 || start > stop {
		err := fmt.Errorf("&%s: illegal substring(%s, %d -> %d, %d -> %d) indices",
			builtinName(invocation.Name), value.Describe(), originalStart, start, originalStop, stop)
		// BasicStrings throws a bridge exception here. Sleep turns an uncaught
		// bridge exception into a warning and aborts only the active block; the
		// caller continues. Direct library invocation still receives a Go error.
		if currentFiber(ctx) != nil {
			return Null(), &uncaughtScriptWarning{err: err}
		}
		return Null(), err
	}
	return sleepStringValueSlice(value, start, stop), nil
}

// BasicStrings.func_uc and func_lc read their sole argument with Stack.pop,
// so an omitted value takes Sleep's recoverable empty-stack warning path.
func builtinUpper(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	return sleepStringMapCase(value, true), nil
}

func builtinLower(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	return sleepStringMapCase(value, false), nil
}

func builtinTrim(_ context.Context, invocation Invocation) (Value, error) {
	value := sleepStringCoercion(invocation.Arg(0))
	units := sleepStringUnits(value)
	start, end := 0, len(units)
	for start < end && units[start] <= 0x20 {
		start++
	}
	for end > start && units[end-1] <= 0x20 {
		end--
	}
	return sleepStringValueSlice(value, start, end), nil
}

func invocationArray(invocation Invocation, index int) (*Array, error) {
	value := invocation.Arg(index)
	array, ok := value.Array()
	if !ok {
		return nil, fmt.Errorf("&%s: expected array. received %s", builtinName(invocation.Name), value.Describe())
	}
	return array, nil
}

func builtinName(name string) string {
	name = strings.TrimPrefix(name, "&")
	if name == "" {
		return "builtin"
	}
	return name
}
