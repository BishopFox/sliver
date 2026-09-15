package opfor

import (
	"context"
	"errors"
	"math"
	"math/bits"
	"sync"
)

const (
	portableCollectionsBinarySearchThreshold = 5000
	portableCollectionsShuffleThreshold      = 5
	portableCollectionsReverseThreshold      = 18
	portableCollectionsFillThreshold         = 25
	portableCollectionsCopyThreshold         = 10
	portableCollectionsRotateThreshold       = 100
	portableCollectionsReplaceAllThreshold   = 11
	portableCollectionsIndexOfThreshold      = 35
)

var (
	portableCollectionsDefaultRandomMu sync.Mutex
	portableCollectionsDefaultRandom   *portableJavaRandom

	portableCollectionsEmptyList = &portableJavaCollection{
		class: "Collections$EmptyList", readOnly: true,
	}
	portableCollectionsEmptySet = &portableJavaCollection{
		class: "Collections$EmptySet", readOnly: true,
	}
	portableCollectionsEmptyMap = newPortableCollectionsEmptyMap()

	portableCollectionsEmptyNavigableSet = &portableJavaCollection{
		class: portableCollectionsEmptyNavigableSetClass, readOnly: true,
	}
	portableCollectionsEmptySortedSet = portableCollectionsEmptyNavigableSet

	portableCollectionsEmptyNavigableMap = newPortableCollectionsEmptyNavigableMap()
	portableCollectionsEmptySortedMap    = portableCollectionsEmptyNavigableMap
	portableCollectionsReverseComparator = &portableJavaReverseComparator{}
	portableCollectionsNaturalComparator = &portableJavaNaturalComparator{}
)

// This file implements the no-callback portion of java.util.Collections at
// the portable Sleep bridge plus Comparator overloads backed by Sleep closures
// or portable newInstance proxies. Opaque importer-owned Comparator objects
// retain first refusal through ObjectHost.

func portableCollectionsShuffle(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	var random *portableJavaRandom
	if len(invocation.Arguments) == 1 {
		random = portableCollectionsProcessRandom()
	} else {
		if invocation.Arg(1).IsNull() {
			random = nil
		} else {
			object, objectOK := invocation.Arg(1).Object()
			var randomOK bool
			random, randomOK = object.(*portableJavaRandom)
			if !objectOK || !randomOK || random == nil {
				// A foreign Random implementation remains importer-owned.
				return Null(), false, nil
			}
		}
	}
	size, err := list.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if size < portableCollectionsShuffleThreshold || portableCollectionsRandomAccess(list) {
		for index := size; index > 1; index-- {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return Null(), true, err
			}
			other, err := portableCollectionsNextInt(random, index)
			if err != nil {
				return Null(), true, err
			}
			if err := portableCollectionsSwapAt(list, index-1, other); err != nil {
				return Null(), true, err
			}
		}
		return Null(), true, nil
	}

	values, err := list.snapshotChecked()
	if err != nil {
		return Null(), true, err
	}
	if len(values) != size {
		return Null(), true, errors.New("java.util.ConcurrentModificationException")
	}
	for index := size; index > 1; index-- {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		other, err := portableCollectionsNextInt(random, index)
		if err != nil {
			return Null(), true, err
		}
		values[index-1], values[other] = values[other], values[index-1]
	}
	_, expected, err := list.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	for index, value := range values {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		if _, present, err := list.iteratorValue(index, expected); err != nil {
			return Null(), true, err
		} else if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		if err := list.iteratorSet(index, expected, value); err != nil {
			return Null(), true, err
		}
	}
	return Null(), true, nil
}

func portableCollectionsProcessRandom() *portableJavaRandom {
	portableCollectionsDefaultRandomMu.Lock()
	defer portableCollectionsDefaultRandomMu.Unlock()
	if portableCollectionsDefaultRandom == nil {
		portableCollectionsDefaultRandom = newPortableJavaRandom(portableJavaRandomSeed())
	}
	return portableCollectionsDefaultRandom
}

func portableCollectionsNextInt(random *portableJavaRandom, bound int) (int, error) {
	if random == nil {
		return 0, errors.New("java.lang.NullPointerException")
	}
	random.mu.Lock()
	if random.random == nil {
		random.mu.Unlock()
		return 0, errors.New("java.lang.NullPointerException")
	}
	value, err := random.random.nextInt(int32(bound))
	random.mu.Unlock()
	return int(value), err
}

func portableCollectionsAddAll(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	collection, ok, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	elements, ok, err := portableCollectionsObjectArrayArgumentAtRuntime(invocation.Runtime, invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	changed := false
	for _, element := range elements {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		result, _, err := collection.add(ObjectInvocation{
			Runtime: invocation.Runtime, Script: invocation.Script,
			Op: ObjectInvoke, Message: "add", Arguments: []Argument{{Value: element}},
		})
		if err != nil {
			return Null(), true, err
		}
		changed = changed || result.Truth()
	}
	return portableJavaBooleanValue(changed), true, nil
}

func portableCollectionsObjectArrayArgument(value Value) ([]Value, bool, error) {
	return portableCollectionsObjectArrayArgumentAtRuntime(nil, value)
}

func portableCollectionsObjectArrayArgumentAtRuntime(runtime *Runtime, value Value) ([]Value, bool, error) {
	if value.IsNull() {
		return nil, true, errors.New("java.lang.NullPointerException")
	}
	if array, ok := value.Array(); ok && array != nil {
		values := array.Values()
		// Sleep ObjectUtilities infers a native array's component from its first
		// meaningful scalar. Primitive scalar arrays cannot bind to Object[].
		for _, element := range values {
			if element.IsNull() {
				continue
			}
			switch element.Kind() {
			case KindInt, KindLong, KindDouble:
				return nil, false, nil
			}
			break
		}
		return values, true, nil
	}
	object, ok := value.Object()
	if !ok {
		return nil, false, nil
	}
	array, ok := object.(*portableJavaArray)
	if !ok || array == nil {
		return nil, false, nil
	}
	typeInfo, dimensions, _ := array.snapshot()
	if typeInfo.primitive || len(dimensions) == 0 {
		return nil, false, nil
	}
	values := make([]Value, dimensions[0])
	for index := range values {
		value, err := array.getAtRuntime(runtime, index)
		if err != nil {
			return nil, true, err
		}
		values[index] = value
	}
	return values, true, nil
}

func portableCollectionsIndexOfSubList(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	source, sourceOK, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	target, targetOK, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !sourceOK || !targetOK {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	sourceSize, err := source.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	targetSize, err := target.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	maxCandidate := sourceSize - targetSize
	last := invocation.Message == "lastIndexOfSubList"
	indexed := sourceSize < portableCollectionsIndexOfThreshold ||
		portableCollectionsRandomAccess(source) && (!last && portableCollectionsRandomAccess(target) || last)
	if indexed {
		start, step := 0, 1
		if last {
			start, step = maxCandidate, -1
		}
		for candidate := start; candidate >= 0 && candidate <= maxCandidate; candidate += step {
			matched := true
			for index := 0; index < targetSize; index++ {
				if err := portableCollectionsLoopStep(ctx); err != nil {
					return Null(), true, err
				}
				targetValue, err := target.getAt(index)
				if err != nil {
					return Null(), true, err
				}
				sourceValue, err := source.getAt(candidate + index)
				if err != nil {
					return Null(), true, err
				}
				if !portableJavaEqual(targetValue, sourceValue) {
					matched = false
					break
				}
			}
			if matched {
				return Int(int32(candidate)), true, nil
			}
		}
		return Int(-1), true, nil
	}
	if maxCandidate < 0 {
		return Int(-1), true, nil
	}
	_, sourceExpected, err := source.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	sourceCursor := 0
	start, step := 0, 1
	if last {
		sourceCursor = maxCandidate
		start, step = maxCandidate, -1
	}
	for candidate := start; candidate >= 0 && candidate <= maxCandidate; candidate += step {
		_, targetExpected, err := target.iteratorBounds()
		if err != nil {
			return Null(), true, err
		}
		targetCursor := 0
		matched := true
		for index := 0; index < targetSize; index++ {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return Null(), true, err
			}
			targetValue, present, err := target.iteratorValue(targetCursor, targetExpected)
			if err != nil {
				return Null(), true, err
			}
			if !present {
				return Null(), true, errors.New("java.util.NoSuchElementException")
			}
			targetCursor++
			sourceValue, present, err := source.iteratorValue(sourceCursor, sourceExpected)
			if err != nil {
				return Null(), true, err
			}
			if !present {
				return Null(), true, errors.New("java.util.NoSuchElementException")
			}
			sourceCursor++
			if !portableJavaEqual(targetValue, sourceValue) {
				matched = false
				back := index
				if last && candidate != 0 {
					back = index + 2
				}
				for count := 0; count < back; count++ {
					if err := portableCollectionsLoopStep(ctx); err != nil {
						return Null(), true, err
					}
					sourceCursor--
					if _, present, err := source.iteratorValue(sourceCursor, sourceExpected); err != nil {
						return Null(), true, err
					} else if !present {
						return Null(), true, errors.New("java.util.NoSuchElementException")
					}
				}
				break
			}
		}
		if matched {
			return Int(int32(candidate)), true, nil
		}
	}
	return Int(-1), true, nil
}

func portableCollectionsEnumeration(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	collection, ok, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	_, expected, err := collection.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	return ObjectValue(&portableJavaIterator{
		collection: collection, last: -1, expectedMod: expected, enumeration: true,
	}), true, nil
}

func portableCollectionsList(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	if invocation.Arg(0).IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	if callable, ok := invocation.Arg(0).Function(); ok && callable != nil {
		values := make([]Value, 0)
		for {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return Null(), true, err
			}
			value, err := portableEnumerationClosureCall(ctx, invocation.Arg(0), callable, "hasMoreElements")
			if err != nil {
				return Null(), true, portableCollectionsListError(err)
			}
			if !value.Truth() {
				break
			}
			value, err = portableEnumerationClosureCall(ctx, invocation.Arg(0), callable, "nextElement")
			if err != nil {
				return Null(), true, portableCollectionsListError(err)
			}
			if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
				return Null(), true, err
			}
			values = append(values, value)
		}
		return ObjectValue(newPortableJavaCollection("ArrayList", values)), true, nil
	}
	object, ok := invocation.Arg(0).Object()
	iterator, ok := object.(*portableJavaIterator)
	if !ok || iterator == nil || !iterator.enumeration {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	values := make([]Value, 0)
	for {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		value, _, err := iterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "hasMoreElements"})
		if err != nil {
			return Null(), true, portableCollectionsListError(err)
		}
		if !value.Truth() {
			break
		}
		value, _, err = iterator.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "nextElement"})
		if err != nil {
			return Null(), true, portableCollectionsListError(err)
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), true, err
		}
		values = append(values, value)
	}
	return ObjectValue(newPortableJavaCollection("ArrayList", values)), true, nil
}

func portableCollectionsFactory(invocation ObjectInvocation) (Value, bool, error) {
	switch invocation.Message {
	case "emptyList":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptyList), true, nil
	case "emptySet":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptySet), true, nil
	case "emptyMap":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptyMap), true, nil
	case "emptySortedSet":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptySortedSet), true, nil
	case "emptyNavigableSet":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptyNavigableSet), true, nil
	case "emptySortedMap":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptySortedMap), true, nil
	case "emptyNavigableMap":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		return ObjectValue(portableCollectionsEmptyNavigableMap), true, nil
	case "singleton":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaCollection{
			class: "Collections$SingletonSet", values: []Value{invocation.Arg(0)}, readOnly: true,
		}), true, nil
	case "singletonList":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaCollection{
			class: "Collections$SingletonList", values: []Value{invocation.Arg(0)}, readOnly: true,
		}), true, nil
	case "singletonMap":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			return Null(), true, err
		}
		mapping := newPortableJavaMap("Collections$SingletonMap", nil)
		key, keyValue := sleepHashKey(invocation.Arg(0))
		mapping.keys = append(mapping.keys, key)
		mapping.keyValues[key] = keyValue
		mapping.values[key] = invocation.Arg(1)
		entry := &portableJavaMapEntry{
			mapping: mapping, key: key, keyValue: keyValue, value: invocation.Arg(1),
		}
		mapping.entries[key] = entry
		mapping.readOnly = true
		// SingletonMap lazily caches three SingletonSet-backed views. The map is
		// immutable, so eagerly creating the same stable identities is equivalent.
		mapping.keySetView = &portableJavaCollection{
			class: "Collections$SingletonSet", values: []Value{keyValue}, readOnly: true,
		}
		mapping.valuesView = &portableJavaCollection{
			class: "Collections$SingletonSet", values: []Value{invocation.Arg(1)}, readOnly: true,
		}
		mapping.entrySetView = &portableJavaCollection{
			class: "Collections$SingletonSet", values: []Value{ObjectValue(entry)}, readOnly: true,
		}
		return ObjectValue(mapping), true, nil
	case "nCopies":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
		}
		count := int(sleepInt32(invocation.Arg(0)))
		if count < 0 {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: List length = " + invocation.Arg(0).String())
		}
		if err := reserveCollectionEntries(invocation.Runtime, count); err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaCollection{
			class: "Collections$CopiesList", readOnly: true, copies: true,
			copiesCount: count, copiesValue: invocation.Arg(1),
		}), true, nil
	default:
		return Null(), false, nil
	}
}

func portableCollectionsReverseOrder(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) == 0 || len(invocation.Arguments) == 1 && invocation.Arg(0).IsNull() {
		return ObjectValue(portableCollectionsReverseComparator), true, nil
	}
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	comparator := invocation.Arg(0)
	if _, ok := comparator.Function(); ok {
		// Sleep's ObjectUtilities converts a function scalar to a fresh dynamic
		// Comparator proxy while adapting the Java method argument. The wrapper
		// retains that proxy, so reversing it again returns the proxy object rather
		// than the original function scalar.
		comparator = ObjectValue(invocation.Runtime.newPortableJavaProxy(
			comparator, []string{"java.util.Comparator"},
		))
	}
	if object, ok := comparator.Object(); ok {
		switch typed := object.(type) {
		case *portableJavaReverseComparator:
			if typed != nil {
				return ObjectValue(portableCollectionsNaturalComparator), true, nil
			}
		case *portableJavaNaturalComparator:
			if typed != nil {
				return ObjectValue(portableCollectionsReverseComparator), true, nil
			}
		case *portableJavaReverseComparator2:
			if typed != nil {
				return typed.comparator, true, nil
			}
		}
	}
	if !portableCollectionsComparatorValue(comparator) {
		// An opaque object may be an importer-supplied Comparator. ObjectHost
		// already had first refusal over the entire static call.
		return Null(), false, nil
	}
	return ObjectValue(&portableJavaReverseComparator2{comparator: comparator}), true, nil
}

func portableCollectionsComparatorValue(value Value) bool {
	if value.IsNull() {
		return true
	}
	if _, ok := value.Function(); ok {
		return true
	}
	object, ok := value.Object()
	if !ok {
		return false
	}
	switch typed := object.(type) {
	case *portableJavaReverseComparator:
		return typed != nil
	case *portableJavaNaturalComparator:
		return typed != nil
	case *portableJavaReverseComparator2:
		return typed != nil
	case *portableJavaProxy:
		return typed != nil && typed.implements("java.util.Comparator")
	default:
		return false
	}
}

func portableCollectionsCompareValues(
	ctx context.Context,
	invocation ObjectInvocation,
	comparator Value,
	left Value,
	right Value,
) (int, bool, error) {
	if comparator.IsNull() {
		comparison, err := portableCollectionsNaturalCompare(left, right)
		return comparison, true, err
	}
	if _, ok := comparator.Function(); ok {
		proxy := &portableJavaProxy{closure: comparator, interfaces: []string{"java.util.Comparator"}, runtime: invocation.Runtime}
		comparison, err := portableCollectionsProxyCompare(ctx, proxy, left, right)
		return comparison, true, err
	}
	object, ok := comparator.Object()
	if !ok {
		return 0, false, nil
	}
	switch typed := object.(type) {
	case *portableJavaReverseComparator:
		if typed == nil {
			return 0, false, nil
		}
		comparison, err := portableCollectionsNaturalCompare(right, left)
		return comparison, true, err
	case *portableJavaNaturalComparator:
		if typed == nil {
			return 0, false, nil
		}
		comparison, err := portableCollectionsNaturalCompare(left, right)
		return comparison, true, err
	case *portableJavaReverseComparator2:
		if typed == nil {
			return 0, false, nil
		}
		return portableCollectionsCompareValues(ctx, invocation, typed.comparator, right, left)
	case *portableJavaProxy:
		if typed == nil || !typed.implements("java.util.Comparator") {
			return 0, false, nil
		}
		comparison, err := portableCollectionsProxyCompare(ctx, typed, left, right)
		return comparison, true, err
	default:
		return 0, false, nil
	}
}

func portableCollectionsProxyCompare(ctx context.Context, proxy *portableJavaProxy, left, right Value) (int, error) {
	value, err := proxy.call(ctx, "compare", []Argument{{Value: left}, {Value: right}}, true)
	if err == nil {
		return int(sleepInt32(value)), nil
	}
	return 0, portableCollectionsProxyCallbackFailure(
		proxy, err,
		"public abstract int java.util.Comparator.compare(java.lang.Object,java.lang.Object)",
	)
}

func portableCollectionsProxyCallbackFailure(proxy *portableJavaProxy, err error, signature string) error {
	var thrown *scriptThrow
	if errors.As(err, &thrown) && thrown != nil {
		thrown.addFrame("   <Java>:-1 " + describeTraceValue(proxy.closure) + " as " + signature)
		return newPortableJavaException(err)
	}
	return &portableObjectCallbackError{cause: err}
}

func portableCollectionsBinarySearch(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 && len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	comparator := Null()
	if len(invocation.Arguments) == 3 {
		comparator = invocation.Arg(2)
		if !portableCollectionsComparatorValue(comparator) {
			return Null(), false, nil
		}
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, expected, err := list.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	key := invocation.Arg(1)
	low, high := 0, size-1
	cursor := 0
	for low <= high {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		mid := int(uint(low+high) >> 1)
		var middle Value
		if portableCollectionsRandomAccess(list) || size < portableCollectionsBinarySearchThreshold {
			middle, err = list.getAt(mid)
		} else {
			middle, err = portableCollectionsIteratorGet(list, &cursor, mid, expected)
		}
		if err != nil {
			return Null(), true, err
		}
		comparison, handled, err := portableCollectionsCompareValues(ctx, invocation, comparator, middle, key)
		if !handled {
			return Null(), false, nil
		}
		if err != nil {
			return Null(), true, err
		}
		switch {
		case comparison < 0:
			low = mid + 1
		case comparison > 0:
			high = mid - 1
		default:
			return Int(int32(mid)), true, nil
		}
	}
	return Int(int32(-(low + 1))), true, nil
}

func portableCollectionsSort(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	comparator := Null()
	if len(invocation.Arguments) == 2 {
		comparator = invocation.Arg(1)
		if !portableCollectionsComparatorValue(comparator) {
			return Null(), false, nil
		}
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	if list.listView == nil && (list.class == "Collections$EmptyList" || list.class == "Collections$SingletonList") {
		// These OpenJDK factory lists override List.sort as a no-op.
		return Null(), true, nil
	}
	size, expected, err := list.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	values, err := list.snapshotChecked()
	if err != nil {
		return Null(), true, err
	}
	if len(values) != size {
		return Null(), true, errors.New("java.util.ConcurrentModificationException")
	}
	if comparator.IsNull() {
		values, err = portableCollectionsStableNaturalSort(ctx, values)
	} else {
		values, err = portableCollectionsStableComparatorSort(ctx, invocation, values, comparator)
	}
	if err != nil {
		return Null(), true, err
	}
	for index, value := range values {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		if err := list.iteratorSet(index, expected, value); err != nil {
			return Null(), true, err
		}
	}
	if err := portableCollectionsFinishSort(list, expected); err != nil {
		return Null(), true, err
	}
	return Null(), true, nil
}

func portableCollectionsFinishSort(list *portableJavaCollection, expected uint64) error {
	// Collections.sort delegates to List.sort. OpenJDK 8 ArrayList overrides
	// that default and increments modCount after every successful sort, even
	// though element replacement is otherwise non-structural. AbstractList
	// sublists and LinkedList retain the default iterator/set behavior.
	if list.listView == nil && list.class == "ArrayList" {
		list.mu.Lock()
		defer list.mu.Unlock()
		if list.mod != expected {
			return errors.New("java.util.ConcurrentModificationException")
		}
		list.mod++
		return nil
	}
	_, revision, err := list.iteratorBounds()
	if err != nil {
		return err
	}
	if revision != expected {
		return errors.New("java.util.ConcurrentModificationException")
	}
	return nil
}

func portableCollectionsReverse(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, err := list.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if err := portableCollectionsReverseRange(ctx, list, 0, size, portableCollectionsRandomAccess(list)); err != nil {
		return Null(), true, err
	}
	return Null(), true, nil
}

func portableCollectionsReverseRange(ctx context.Context, list *portableJavaCollection, start, end int, randomAccess bool) error {
	size := end - start
	if randomAccess || size < portableCollectionsReverseThreshold {
		for left, right := start, end-1; left < start+size/2; left, right = left+1, right-1 {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return err
			}
			if err := portableCollectionsSwapAt(list, left, right); err != nil {
				return err
			}
		}
		return nil
	}
	_, expected, err := list.iteratorBounds()
	if err != nil {
		return err
	}
	for left, right := start, end-1; left < start+size/2; left, right = left+1, right-1 {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return err
		}
		leftValue, present, err := list.iteratorValue(left, expected)
		if err != nil {
			return err
		}
		if !present {
			return errors.New("java.util.NoSuchElementException")
		}
		rightValue, present, err := list.iteratorValue(right, expected)
		if err != nil {
			return err
		}
		if !present {
			return errors.New("java.util.NoSuchElementException")
		}
		// The sequential OpenJDK path calls fwd.set(rev.previous()) and
		// then rev.set(tmp), which is observably different from swap's
		// indexed set(j)/set(i) order under concurrent modification.
		if err := list.iteratorSet(left, expected, rightValue); err != nil {
			return err
		}
		if err := list.iteratorSet(right, expected, leftValue); err != nil {
			return err
		}
	}
	return nil
}

func portableCollectionsSwap(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	if err := portableCollectionsLoopStep(ctx); err != nil {
		return Null(), true, err
	}
	if err := portableCollectionsSwapAt(list, int(sleepInt32(invocation.Arg(1))), int(sleepInt32(invocation.Arg(2)))); err != nil {
		return Null(), true, err
	}
	return Null(), true, nil
}

// portableCollectionsSwapAt follows Collections.swap's observable evaluation
// order: get(i), set(j, valueAtI), then set(i, previousValueAtJ).
func portableCollectionsSwapAt(list *portableJavaCollection, left, right int) error {
	leftValue, err := list.getAt(left)
	if err != nil {
		return err
	}
	rightValue, err := list.setAt(right, leftValue)
	if err != nil {
		return err
	}
	_, err = list.setAt(left, rightValue)
	return err
}

func portableCollectionsFill(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, err := list.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	_, expected, err := list.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	for index := 0; index < size; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		if portableCollectionsRandomAccess(list) || size < portableCollectionsFillThreshold {
			if _, err := list.setAt(index, invocation.Arg(1)); err != nil {
				return Null(), true, err
			}
			continue
		}
		if _, present, err := list.iteratorValue(index, expected); err != nil {
			return Null(), true, err
		} else if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		if err := list.iteratorSet(index, expected, invocation.Arg(1)); err != nil {
			return Null(), true, err
		}
	}
	return Null(), true, nil
}

func portableCollectionsCopy(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	destination, destinationOK, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	source, sourceOK, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !destinationOK || !sourceOK {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	sourceSize, err := source.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	destinationSize, err := destination.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if sourceSize > destinationSize {
		return Null(), true, errors.New("java.lang.IndexOutOfBoundsException: Source does not fit in dest")
	}
	indexed := sourceSize < portableCollectionsCopyThreshold || portableCollectionsRandomAccess(source) && portableCollectionsRandomAccess(destination)
	_, sourceExpected, err := source.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	_, destinationExpected, err := destination.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	for index := 0; index < sourceSize; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		if indexed {
			value, err := source.getAt(index)
			if err != nil {
				return Null(), true, err
			}
			if _, err := destination.setAt(index, value); err != nil {
				return Null(), true, err
			}
			continue
		}
		if _, present, err := destination.iteratorValue(index, destinationExpected); err != nil {
			return Null(), true, err
		} else if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		value, present, err := source.iteratorValue(index, sourceExpected)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		if err := destination.iteratorSet(index, destinationExpected, value); err != nil {
			return Null(), true, err
		}
	}
	return Null(), true, nil
}

func portableCollectionsRotate(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, err := list.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if size == 0 {
		return Null(), true, nil
	}
	distance := int(int64(sleepInt32(invocation.Arg(1))) % int64(size))
	if distance < 0 {
		distance += size
	}
	if distance == 0 {
		return Null(), true, nil
	}
	if !portableCollectionsRandomAccess(list) && size >= portableCollectionsRotateThreshold {
		middle := size - distance
		if err := portableCollectionsReverseRange(ctx, list, 0, middle, false); err != nil {
			return Null(), true, err
		}
		if err := portableCollectionsReverseRange(ctx, list, middle, size, false); err != nil {
			return Null(), true, err
		}
		if err := portableCollectionsReverseRange(ctx, list, 0, size, false); err != nil {
			return Null(), true, err
		}
		return Null(), true, nil
	}
	bound := size - distance
	for cycleStart, moved := 0, 0; moved < size; cycleStart++ {
		displaced, err := list.getAt(cycleStart)
		if err != nil {
			return Null(), true, err
		}
		index := cycleStart
		for {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return Null(), true, err
			}
			if index >= bound {
				index -= size
			}
			index += distance
			displaced, err = list.setAt(index, displaced)
			if err != nil {
				return Null(), true, err
			}
			moved++
			if index == cycleStart {
				break
			}
		}
	}
	return Null(), true, nil
}

func portableCollectionsReplaceAll(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	list, ok, err := portableCollectionsListArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, err := list.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	_, expected, err := list.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	indexed := portableCollectionsRandomAccess(list) || size < portableCollectionsReplaceAllThreshold
	replaced := false
	for index := 0; index < size; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		var value Value
		if indexed {
			value, err = list.getAt(index)
		} else {
			var present bool
			value, present, err = list.iteratorValue(index, expected)
			if err == nil && !present {
				err = errors.New("java.util.NoSuchElementException")
			}
		}
		if err != nil {
			return Null(), true, err
		}
		if portableJavaEqual(invocation.Arg(1), value) {
			if indexed {
				if _, err := list.setAt(index, invocation.Arg(2)); err != nil {
					return Null(), true, err
				}
			} else if err := list.iteratorSet(index, expected, invocation.Arg(2)); err != nil {
				return Null(), true, err
			}
			replaced = true
		}
	}
	return portableJavaBooleanValue(replaced), true, nil
}

func portableCollectionsMinMax(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	comparator := Null()
	if len(invocation.Arguments) == 2 {
		comparator = invocation.Arg(1)
		if !portableCollectionsComparatorValue(comparator) {
			return Null(), false, nil
		}
	}
	collection, ok, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, expected, err := collection.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	if size == 0 {
		return Null(), true, errors.New("java.util.NoSuchElementException")
	}
	candidate, present, err := collection.iteratorValue(0, expected)
	if err != nil {
		return Null(), true, err
	}
	if !present {
		return Null(), true, errors.New("java.util.NoSuchElementException")
	}
	for index := 1; index < size; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		value, present, err := collection.iteratorValue(index, expected)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		comparison, handled, err := portableCollectionsCompareValues(ctx, invocation, comparator, value, candidate)
		if !handled {
			return Null(), false, nil
		}
		if err != nil {
			return Null(), true, err
		}
		if invocation.Message == "min" && comparison < 0 || invocation.Message == "max" && comparison > 0 {
			candidate = value
		}
	}
	return candidate, true, nil
}

func portableCollectionsFrequency(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	collection, ok, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	size, expected, err := collection.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	var frequency int32
	for index := 0; index < size; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		value, present, err := collection.iteratorValue(index, expected)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		if portableJavaEqual(invocation.Arg(1), value) {
			frequency++
		}
	}
	return Int(frequency), true, nil
}

func portableCollectionsDisjoint(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	left, leftOK, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	right, rightOK, err := portableCollectionsCollectionArgument(invocation.Runtime, invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !leftOK || !rightOK {
		return portableNoMatchingMethod(invocation, "java.util.Collections"), true, nil
	}
	leftSize, _, err := left.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	rightSize, _, err := right.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	iterate, contains := left, right
	if left.isSet() {
		iterate, contains = right, left
	} else if !right.isSet() {
		if leftSize == 0 || rightSize == 0 {
			return portableJavaBooleanValue(true), true, nil
		}
		if leftSize > rightSize {
			iterate, contains = right, left
		}
	}
	iterateSize, expected, err := iterate.iteratorBounds()
	if err != nil {
		return Null(), true, err
	}
	for index := 0; index < iterateSize; index++ {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return Null(), true, err
		}
		candidate, present, err := iterate.iteratorValue(index, expected)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		values, err := contains.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		for _, value := range values {
			if err := portableCollectionsLoopStep(ctx); err != nil {
				return Null(), true, err
			}
			if portableJavaEqual(candidate, value) {
				return portableJavaBooleanValue(false), true, nil
			}
		}
	}
	return portableJavaBooleanValue(true), true, nil
}

// portableCollectionsCopiesListHash is OpenJDK 8u's O(log n) hash for the
// compact list returned by Collections.nCopies. Operations deliberately use
// int32 overflow so the result is identical to Java's int arithmetic.
func portableCollectionsCopiesListHash(count int, elementHash int32) int32 {
	if count == 0 {
		return 1
	}
	n := uint32(count)
	power, sum := int32(31), int32(1)
	for shift := bits.LeadingZeros32(n) + 1; shift < 32; shift++ {
		sum *= power + 1
		power *= power
		if int32(n<<shift) < 0 {
			power *= 31
			sum = sum*31 + 1
		}
	}
	return power + sum*elementHash
}

func newPortableCollectionsEmptyMap() *portableJavaMap {
	mapping := newPortableJavaMap("Collections$EmptyMap", nil)
	mapping.readOnly = true
	// OpenJDK's EmptyMap uses the same cached EmptySet object for all three
	// views. This identity is observable through Sleep's `is` operator.
	mapping.keySetView = portableCollectionsEmptySet
	mapping.valuesView = portableCollectionsEmptySet
	mapping.entrySetView = portableCollectionsEmptySet
	return mapping
}

func portableJavaBooleanValue(value bool) Value {
	// ObjectUtilities.BuildScalar(true, Boolean) preserves both Java boolean
	// results as integer scalars. Sleep's language-level Bool helper instead
	// uses the empty scalar for false, so Java bridges must not use Bool here.
	if value {
		return Int(1)
	}
	return Int(0)
}

func portableCollectionsListArgument(runtime *Runtime, value Value) (*portableJavaCollection, bool, error) {
	if value.IsNull() {
		return nil, true, errors.New("java.lang.NullPointerException")
	}
	if array, ok := value.Array(); ok && array != nil {
		// Sleep ObjectUtilities converts each native array argument to a fresh,
		// detached java.util.LinkedList. Static mutations therefore do not write
		// back through the original Sleep array.
		values := array.Values()
		if err := reserveCollectionEntries(runtime, len(values)); err != nil {
			return nil, true, err
		}
		return newPortableJavaCollection("LinkedList", values), true, nil
	}
	if object, ok := value.Object(); ok {
		if collection, ok := object.(*portableJavaCollection); ok && collection != nil && collection.isList() {
			return collection, true, nil
		}
	}
	return nil, false, nil
}

func portableCollectionsCollectionArgument(runtime *Runtime, value Value) (*portableJavaCollection, bool, error) {
	if value.IsNull() {
		return nil, true, errors.New("java.lang.NullPointerException")
	}
	if array, ok := value.Array(); ok && array != nil {
		values := array.Values()
		if err := reserveCollectionEntries(runtime, len(values)); err != nil {
			return nil, true, err
		}
		return newPortableJavaCollection("LinkedList", values), true, nil
	}
	if object, ok := value.Object(); ok {
		if collection, ok := object.(*portableJavaCollection); ok && collection != nil {
			return collection, true, nil
		}
	}
	return nil, false, nil
}

func portableCollectionsRandomAccess(list *portableJavaCollection) bool {
	if list == nil {
		return false
	}
	if list.listView != nil && list.listView.root != nil {
		switch list.listView.root.class {
		case "ArrayList", "Collections$EmptyList", "Collections$SingletonList", "Collections$CopiesList":
			return true
		}
		return false
	}
	switch list.class {
	case "ArrayList", "Collections$EmptyList", "Collections$SingletonList", "Collections$CopiesList":
		return true
	default:
		return false
	}
}

func portableCollectionsIteratorGet(list *portableJavaCollection, cursor *int, index int, expected uint64) (Value, error) {
	if *cursor <= index {
		for {
			value, present, err := list.iteratorValue(*cursor, expected)
			if err != nil {
				return Null(), err
			}
			if !present {
				return Null(), errors.New("java.util.NoSuchElementException")
			}
			current := *cursor
			*cursor = *cursor + 1
			if current == index {
				return value, nil
			}
		}
	}
	for {
		previous := *cursor - 1
		value, present, err := list.iteratorValue(previous, expected)
		if err != nil {
			return Null(), err
		}
		if !present {
			return Null(), errors.New("java.util.NoSuchElementException")
		}
		*cursor = previous
		if previous == index {
			return value, nil
		}
	}
}

func portableCollectionsLoopStep(ctx context.Context) error {
	if err := executionContextError(ctx); err != nil {
		return err
	}
	return consumeInstruction(ctx)
}

// portableCollectionsStableComparatorSort shares the TimSort-compatible path
// used by Sleep's stock sort functions. Comparator side effects therefore stay
// in reference order for both small and large lists, and invalid comparison
// contracts retain Java's observable failure.
func portableCollectionsStableComparatorSort(
	ctx context.Context,
	invocation ObjectInvocation,
	values []Value,
	comparator Value,
) ([]Value, error) {
	flow := &sleepSortComparatorFlow{}
	compare := func(left, right Value) (int, error) {
		if err := portableCollectionsLoopStep(ctx); err != nil {
			return 0, err
		}
		return flow.compareProxy(ctx, func() (Value, error) {
			comparison, handled, err := portableCollectionsCompareValues(ctx, invocation, comparator, left, right)
			if !handled {
				return Null(), errors.New("java.lang.ClassCastException")
			}
			return Int(int32(comparison)), err
		})
	}
	result, err := sleepStableTimSort(values, compare)
	if errors.Is(err, errSleepComparatorContract) {
		return nil, errors.New("java.lang.IllegalArgumentException: " + err.Error())
	}
	return result, err
}

func portableCollectionsStableNaturalSort(ctx context.Context, values []Value) ([]Value, error) {
	if len(values) < 2 {
		return append([]Value(nil), values...), nil
	}
	source := append([]Value(nil), values...)
	target := make([]Value, len(values))
	for width := 1; width < len(source); width *= 2 {
		for start := 0; start < len(source); start += 2 * width {
			middle := min(start+width, len(source))
			end := min(start+2*width, len(source))
			left, right := start, middle
			for destination := start; destination < end; destination++ {
				if err := portableCollectionsLoopStep(ctx); err != nil {
					return nil, err
				}
				takeRight := false
				if left >= middle {
					takeRight = true
				} else if right < end {
					comparison, err := portableCollectionsNaturalCompare(source[right], source[left])
					if err != nil {
						return nil, err
					}
					takeRight = comparison < 0
				}
				if takeRight {
					target[destination] = source[right]
					right++
				} else {
					target[destination] = source[left]
					left++
				}
			}
		}
		source, target = target, source
		if width > len(source)/2 {
			break
		}
	}
	return source, nil
}

func portableCollectionsNaturalCompare(left, right Value) (int, error) {
	if left.IsNull() || right.IsNull() {
		return 0, errors.New("java.lang.NullPointerException")
	}
	leftClass, leftValue, leftObject, leftOK := portableCollectionsComparableValue(left)
	rightClass, rightValue, rightObject, rightOK := portableCollectionsComparableValue(right)
	if !leftOK || !rightOK || leftClass != rightClass {
		return 0, errors.New("java.lang.ClassCastException")
	}
	switch leftClass {
	case "java.lang.Byte", "java.lang.Short", "java.lang.Integer":
		return compareOrdered(leftValue.Int32(), rightValue.Int32()), nil
	case "java.lang.Long":
		return compareOrdered(leftValue.Int64(), rightValue.Int64()), nil
	case "java.lang.Float":
		return portableCollectionsFloatCompare(float32(leftValue.Float64()), float32(rightValue.Float64())), nil
	case "java.lang.Double":
		return portableCollectionsDoubleCompare(leftValue.Float64(), rightValue.Float64()), nil
	case "java.lang.Boolean":
		leftBoolean, rightBoolean := leftValue.Int32() != 0, rightValue.Int32() != 0
		if leftBoolean == rightBoolean {
			return 0, nil
		}
		if !leftBoolean {
			return -1, nil
		}
		return 1, nil
	case "java.lang.Character":
		leftUnits, rightUnits := sleepStringUnits(leftValue), sleepStringUnits(rightValue)
		if len(leftUnits) != 1 || len(rightUnits) != 1 {
			return 0, errors.New("java.lang.ClassCastException")
		}
		return compareOrdered(leftUnits[0], rightUnits[0]), nil
	case "java.lang.String":
		return sleepStringCompareValues(leftValue, rightValue), nil
	case "java.io.File":
		return int(portableJavaFileCompareValues(leftObject.(*portableJavaFile).pathValue(), rightObject.(*portableJavaFile).pathValue())), nil
	case "java.util.UUID":
		return int(comparePortableJavaUUID(leftObject.(*portableJavaUUID), rightObject.(*portableJavaUUID))), nil
	default:
		return 0, errors.New("java.lang.ClassCastException")
	}
}

func portableCollectionsComparableValue(value Value) (class string, scalar Value, object any, ok bool) {
	switch value.Kind() {
	case KindInt:
		return "java.lang.Integer", value, nil, true
	case KindLong:
		return "java.lang.Long", value, nil, true
	case KindDouble:
		return "java.lang.Double", value, nil, true
	case KindString:
		return "java.lang.String", value, nil, true
	case KindObject:
		object, ok := value.Object()
		if !ok {
			return "", Null(), nil, false
		}
		switch typed := object.(type) {
		case *portableJavaPrimitive:
			if typed == nil {
				return "", Null(), nil, false
			}
			switch typed.className() {
			case "java.lang.Byte", "java.lang.Short", "java.lang.Integer", "java.lang.Long", "java.lang.Float", "java.lang.Double", "java.lang.Boolean", "java.lang.Character":
				return typed.className(), typed.sleepValue(), typed, true
			}
		case *portableJavaFile:
			return "java.io.File", Null(), typed, typed != nil
		case *portableJavaUUID:
			return "java.util.UUID", Null(), typed, typed != nil
		}
	}
	return "", Null(), nil, false
}

func portableCollectionsDoubleCompare(left, right float64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	leftBits, rightBits := int64(portableJavaDoubleBits(left)), int64(portableJavaDoubleBits(right))
	return compareOrdered(leftBits, rightBits)
}

func portableCollectionsFloatCompare(left, right float32) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	leftBits, rightBits := int32(math.Float32bits(left)), int32(math.Float32bits(right))
	if math.IsNaN(float64(left)) {
		leftBits = int32(0x7fc00000)
	}
	if math.IsNaN(float64(right)) {
		rightBits = int32(0x7fc00000)
	}
	return compareOrdered(leftBits, rightBits)
}

func compareOrdered[T ~int32 | ~int64 | ~uint16](left, right T) int {
	if left == right {
		return 0
	}
	if left < right {
		return -1
	}
	return 1
}
