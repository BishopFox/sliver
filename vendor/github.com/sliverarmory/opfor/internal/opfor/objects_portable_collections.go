package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

const portableCollectionsMaximumMaterializedElements = 1 << 20

const (
	portableCollectionsEmptyNavigableSetClass        = "Collections$UnmodifiableNavigableSet$EmptyNavigableSet"
	portableCollectionsUnmodifiableSortedSetClass    = "Collections$UnmodifiableSortedSet"
	portableCollectionsUnmodifiableNavigableSetClass = "Collections$UnmodifiableNavigableSet"
	portableCollectionsEmptyNavigableMapClass        = "Collections$UnmodifiableNavigableMap$EmptyNavigableMap"
	portableCollectionsUnmodifiableSortedMapClass    = "Collections$UnmodifiableSortedMap"
	portableCollectionsUnmodifiableNavigableMapClass = "Collections$UnmodifiableNavigableMap"
)

var errPortableCollectionsMaterializationLimit = errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")

// portableJavaCollection models the java.util collections that appear in the
// canonical Sleep suite. Values remain Sleep Values so crossing back through
// ObjectUtilities.BuildScalar preserves their scalar type.
type portableJavaCollection struct {
	mu            sync.RWMutex
	class         string
	values        []Value
	mod           uint64
	readOnly      bool
	copies        bool
	copiesCount   int
	copiesValue   Value
	wrapperSource collectionWrapperSource
	listView      *portableJavaListView
	mapView       *portableJavaMapView
	reverseOrder  bool
}

func newPortableJavaCollection(class string, values []Value) *portableJavaCollection {
	collection := &portableJavaCollection{class: class, values: append([]Value(nil), values...)}
	collection.normalizeLocked()
	return collection
}

func portableJavaCollectionEntryCount(class string, values []Value) int {
	if class != "HashSet" && class != "LinkedHashSet" && class != "TreeSet" {
		return len(values)
	}
	count := 0
	for index, value := range values {
		duplicate := false
		for previous := 0; previous < index; previous++ {
			if portableJavaEqual(values[previous], value) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			count++
		}
	}
	return count
}

func (c *portableJavaCollection) className() string {
	if c == nil || c.class == "" {
		return "LinkedList"
	}
	return c.class
}

func portableCollectionsSortedSetClass(class string) bool {
	switch class {
	case portableCollectionsEmptyNavigableSetClass,
		portableCollectionsUnmodifiableSortedSetClass,
		portableCollectionsUnmodifiableNavigableSetClass:
		return true
	default:
		return false
	}
}

func portableCollectionsNavigableSetClass(class string) bool {
	return class == portableCollectionsEmptyNavigableSetClass || class == portableCollectionsUnmodifiableNavigableSetClass
}

func portableCollectionsUnmodifiableCollectionClass(class string) bool {
	return portableCollectionsSortedSetClass(class) || class == "Collections$UnmodifiableSet" ||
		class == "Collections$UnmodifiableCollection" || class == "Collections$UnmodifiableMap$UnmodifiableEntrySet"
}

func newPortableCollectionsSortedSetView(navigable, reverse bool) *portableJavaCollection {
	class := portableCollectionsUnmodifiableSortedSetClass
	if navigable {
		class = portableCollectionsUnmodifiableNavigableSetClass
	}
	return &portableJavaCollection{class: class, readOnly: true, reverseOrder: reverse}
}

func (c *portableJavaCollection) String() string {
	values, err := c.snapshotChecked()
	if err != nil {
		return err.Error()
	}
	return portableJavaCollectionString(values)
}

func portableJavaCollectionString(values []Value) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = portableJavaValueString(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (c *portableJavaCollection) snapshot() []Value {
	values, _ := c.snapshotChecked()
	return values
}

func (c *portableJavaCollection) snapshotChecked() ([]Value, error) {
	if c == nil {
		return nil, nil
	}
	if c.copies {
		if c.copiesCount > portableCollectionsMaximumMaterializedElements {
			return nil, errPortableCollectionsMaterializationLimit
		}
		values := make([]Value, c.copiesCount)
		for index := range values {
			values[index] = c.copiesValue
		}
		return values, nil
	}
	if c.listView != nil {
		return c.listView.snapshot()
	}
	if c.mapView != nil {
		return c.mapView.snapshot(), nil
	}
	c.mu.RLock()
	values := append([]Value(nil), c.values...)
	c.mu.RUnlock()
	return values, nil
}

func (c *portableJavaCollection) normalizeLocked() {
	if c == nil {
		return
	}
	switch c.class {
	case "HashSet", "LinkedHashSet", "TreeSet":
		unique := make([]Value, 0, len(c.values))
		for _, value := range c.values {
			if !portableJavaContains(unique, value) {
				unique = append(unique, value)
			}
		}
		c.values = unique
	}
	if c.class == "TreeSet" {
		sort.SliceStable(c.values, func(left, right int) bool {
			return portableJavaCompare(c.values[left], c.values[right]) < 0
		})
	}
}

func (c *portableJavaCollection) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(c.isA(invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectGet {
		return Null(), false, nil
	}
	if value, handled, err := c.invokeStack(invocation); handled {
		return value, true, err
	}
	if portableCollectionsUnmodifiableCollectionClass(c.class) {
		expected := -1
		switch invocation.Message {
		case "add", "remove", "addAll", "removeAll", "retainAll":
			expected = 1
		case "clear":
			expected = 0
		}
		if expected >= 0 {
			if len(invocation.Arguments) != expected {
				return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
			}
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
	}
	if value, handled, err := c.invokeCollectionsSortedSet(invocation); handled {
		return value, true, err
	}
	switch invocation.Message {
	case "add", "offer", "offerFirst", "offerLast", "addFirst", "addLast", "push":
		return c.add(invocation)
	case "addAll":
		return c.addAll(invocation)
	case "get", "getFirst", "getLast", "peek", "peekFirst", "peekLast", "element":
		return c.get(invocation)
	case "set":
		if len(invocation.Arguments) != 2 || !c.isList() {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		index := int(invocation.Arg(0).Int32())
		previous, err := c.setAt(index, invocation.Arg(1))
		return previous, true, err
	case "size":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		size, err := c.sizeChecked()
		if err != nil {
			return Null(), true, err
		}
		return Int(int32(size)), true, nil
	case "isEmpty":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		size, err := c.sizeChecked()
		if err != nil {
			return Null(), true, err
		}
		empty := size == 0
		return portableJavaBooleanValue(empty), true, nil
	case "contains":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		if c.copies {
			return portableJavaBooleanValue(c.copiesCount != 0 && portableJavaEqual(c.copiesValue, invocation.Arg(0))), true, nil
		}
		values, err := c.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		return portableJavaBooleanValue(portableJavaContains(values, invocation.Arg(0))), true, nil
	case "containsAll":
		return c.containsAll(invocation)
	case "indexOf", "lastIndexOf":
		if len(invocation.Arguments) != 1 || !c.isList() {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		if c.copies {
			if c.copiesCount == 0 || !portableJavaEqual(c.copiesValue, invocation.Arg(0)) {
				return Int(-1), true, nil
			}
			if invocation.Message == "lastIndexOf" {
				return Int(int32(c.copiesCount - 1)), true, nil
			}
			return Int(0), true, nil
		}
		values, err := c.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		if invocation.Message == "lastIndexOf" {
			for index := len(values) - 1; index >= 0; index-- {
				if portableJavaEqual(values[index], invocation.Arg(0)) {
					return Int(int32(index)), true, nil
				}
			}
		} else {
			for index, value := range values {
				if portableJavaEqual(value, invocation.Arg(0)) {
					return Int(int32(index)), true, nil
				}
			}
		}
		return Int(-1), true, nil
	case "remove", "removeFirst", "removeLast", "poll", "pollFirst", "pollLast", "pop":
		return c.remove(invocation)
	case "removeAll", "retainAll":
		return c.filterAll(invocation)
	case "clear":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		return Null(), true, c.clearValues()
	case "subList":
		return c.subList(invocation)
	case "iterator", "listIterator", "elements":
		// OpenJDK's backed SubList iterator is the same ListIterator wrapper
		// returned by listIterator(), even though List.iterator() declares the
		// narrower Iterator result type.
		isListIterator := invocation.Message == "listIterator" || invocation.Message == "iterator" && c.listView != nil
		isEnumeration := invocation.Message == "elements"
		if invocation.Message == "listIterator" && (!c.isList() || len(invocation.Arguments) > 1) || invocation.Message != "listIterator" && len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		index := 0
		if len(invocation.Arguments) == 1 {
			index = int(invocation.Arg(0).Int32())
		}
		size, expected, err := c.iteratorBounds()
		if err != nil {
			return Null(), true, err
		}
		valid := index >= 0 && index <= size
		if !valid {
			// OpenJDK 8 ArrayList.listIterator(int) has its own historical
			// message, while AbstractList/LinkedList and backed sublists use
			// their shared Index/Size form.
			if invocation.Message == "listIterator" && c.listView == nil && c.class == "ArrayList" {
				return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d", index)
			}
			return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, size)
		}
		return ObjectValue(&portableJavaIterator{
			collection: c, index: index, last: -1, expectedMod: expected,
			listIterator: isListIterator, enumeration: isEnumeration,
		}), true, nil
	case "toArray":
		if len(invocation.Arguments) > 1 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		values, err := c.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		array, err := newRuntimeArray(invocation.Runtime, values...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		equal, err := c.equalValueChecked(invocation.Arg(0))
		return portableJavaBooleanValue(equal), true, err
	case "hashCode":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		hash, err := c.javaHashCode()
		if err != nil {
			return Null(), true, err
		}
		return Int(hash), true, nil
	case "toString":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		values, err := c.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		return String(portableJavaCollectionString(values)), true, nil
	}
	return Null(), false, nil
}

func (c *portableJavaCollection) invokeCollectionsSortedSet(invocation ObjectInvocation) (Value, bool, error) {
	if c == nil || !portableCollectionsSortedSetClass(c.class) {
		return Null(), false, nil
	}
	class := "java.util." + c.className()
	navigable := portableCollectionsNavigableSetClass(c.class)
	switch invocation.Message {
	case "comparator":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		if c.reverseOrder {
			return ObjectValue(portableCollectionsReverseComparator), true, nil
		}
		return Null(), true, nil
	case "first", "last":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, errors.New("java.util.NoSuchElementException")
	case "subSet":
		if len(invocation.Arguments) == 2 {
			if err := portableCollectionsValidateRange(invocation.Arg(0), invocation.Arg(1), c.reverseOrder); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedSetView(false, c.reverseOrder)), true, nil
		}
		if navigable && len(invocation.Arguments) == 4 {
			if err := portableCollectionsValidateRange(invocation.Arg(0), invocation.Arg(2), c.reverseOrder); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedSetView(true, c.reverseOrder)), true, nil
		}
		return portableNoMatchingMethod(invocation, class), true, nil
	case "headSet", "tailSet":
		if len(invocation.Arguments) == 1 {
			if err := portableCollectionsValidateEndpoint(invocation.Arg(0)); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedSetView(false, c.reverseOrder)), true, nil
		}
		if navigable && len(invocation.Arguments) == 2 {
			if err := portableCollectionsValidateEndpoint(invocation.Arg(0)); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedSetView(true, c.reverseOrder)), true, nil
		}
		return portableNoMatchingMethod(invocation, class), true, nil
	case "lower", "floor", "ceiling", "higher":
		if !navigable || len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, nil
	case "pollFirst", "pollLast":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	case "descendingSet":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return ObjectValue(newPortableCollectionsSortedSetView(true, !c.reverseOrder)), true, nil
	case "descendingIterator":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		_, expected, err := c.iteratorBounds()
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(&portableJavaIterator{collection: c, last: -1, expectedMod: expected}), true, nil
	default:
		return Null(), false, nil
	}
}

func portableCollectionsValidateEndpoint(value Value) error {
	_, err := portableCollectionsNaturalCompare(value, value)
	return err
}

func portableCollectionsValidateRange(from, to Value, reverse bool) error {
	comparison, err := portableCollectionsNaturalCompare(from, to)
	if err != nil {
		return err
	}
	if reverse {
		comparison = -comparison
	}
	if comparison > 0 {
		return errors.New("java.lang.IllegalArgumentException: fromKey > toKey")
	}
	return nil
}

type portableJavaReverseComparator struct{}

// portableJavaNaturalComparator is the singleton returned by
// Collections.reverseOrder(Collections.reverseOrder()). OpenJDK exposes the
// package-private enum class through the Comparator result, so keeping a
// distinct object preserves getClass(), identity, and reversed() behavior.
type portableJavaNaturalComparator struct{}

// portableJavaReverseComparator2 models Collections.reverseOrder(comparator).
// The wrapped Value is retained by identity, just as OpenJDK's wrapper retains
// the caller-supplied Comparator reference.
type portableJavaReverseComparator2 struct {
	comparator Value
}

func (*portableJavaReverseComparator) String() string {
	return "java.util.Collections$ReverseComparator"
}

func (c *portableJavaReverseComparator) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		return Bool(class == "Comparator" || class == "Object" || class == "Serializable"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "compare":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator"), true, nil
		}
		comparison, err := portableCollectionsNaturalCompare(invocation.Arg(1), invocation.Arg(0))
		if err != nil {
			return Null(), true, err
		}
		return Int(int32(comparison)), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator"), true, nil
		}
		object, ok := invocation.Arg(0).Object()
		return portableJavaBooleanValue(ok && object == c), true, nil
	case "reversed":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator"), true, nil
		}
		return ObjectValue(portableCollectionsNaturalComparator), true, nil
	}
	return Null(), false, nil
}

func (*portableJavaNaturalComparator) String() string {
	return "java.util.Comparators$NaturalOrderComparator"
}

func (c *portableJavaNaturalComparator) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		return Bool(class == "Comparator" || class == "Object" || class == "Serializable" || class == "Enum" || class == "Comparable"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "compare":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Comparators$NaturalOrderComparator"), true, nil
		}
		comparison, err := portableCollectionsNaturalCompare(invocation.Arg(0), invocation.Arg(1))
		if err != nil {
			return Null(), true, err
		}
		return Int(int32(comparison)), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Comparators$NaturalOrderComparator"), true, nil
		}
		object, ok := invocation.Arg(0).Object()
		return portableJavaBooleanValue(ok && object == c), true, nil
	case "reversed":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Comparators$NaturalOrderComparator"), true, nil
		}
		return ObjectValue(portableCollectionsReverseComparator), true, nil
	}
	return Null(), false, nil
}

func (*portableJavaReverseComparator2) String() string {
	return "java.util.Collections$ReverseComparator2"
}

func (c *portableJavaReverseComparator2) invokeContext(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		return Bool(class == "Comparator" || class == "Object" || class == "Serializable"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "compare":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator2"), true, nil
		}
		comparison, handled, err := portableCollectionsCompareValues(ctx, invocation, c.comparator, invocation.Arg(1), invocation.Arg(0))
		if !handled {
			return Null(), false, nil
		}
		if err != nil {
			return Null(), true, err
		}
		return Int(int32(comparison)), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator2"), true, nil
		}
		object, ok := invocation.Arg(0).Object()
		other, ok := object.(*portableJavaReverseComparator2)
		if !ok || other == nil {
			return portableJavaBooleanValue(false), true, nil
		}
		if other == c {
			return portableJavaBooleanValue(true), true, nil
		}
		equal, err := portableCollectionsComparatorEquals(ctx, c.comparator, other.comparator)
		return portableJavaBooleanValue(equal), true, err
	case "hashCode":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator2"), true, nil
		}
		hash, err := portableCollectionsComparatorHashCode(ctx, c.comparator)
		return Int(hash ^ int32(-1<<31)), true, err
	case "reversed":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Collections$ReverseComparator2"), true, nil
		}
		return c.comparator, true, nil
	}
	return Null(), false, nil
}

func (c *portableJavaReverseComparator2) invoke(invocation ObjectInvocation) (Value, bool, error) {
	return c.invokeContext(context.Background(), invocation)
}

func portableCollectionsComparatorEquals(ctx context.Context, comparator, other Value) (bool, error) {
	if object, ok := comparator.Object(); ok {
		if proxy, ok := object.(*portableJavaProxy); ok && proxy != nil {
			value, err := proxy.call(ctx, "equals", []Argument{{Value: other}}, true)
			if err != nil {
				return false, portableCollectionsProxyCallbackFailure(
					proxy, err,
					"public boolean java.lang.Object.equals(java.lang.Object)",
				)
			}
			return sleepInt32(value) != 0, nil
		}
	}
	return portableJavaEqual(comparator, other), nil
}

func portableCollectionsComparatorHashCode(ctx context.Context, comparator Value) (int32, error) {
	if object, ok := comparator.Object(); ok {
		if proxy, ok := object.(*portableJavaProxy); ok && proxy != nil {
			value, err := proxy.call(ctx, "hashCode", nil, true)
			if err != nil {
				return 0, portableCollectionsProxyCallbackFailure(
					proxy, err,
					"public native int java.lang.Object.hashCode()",
				)
			}
			return sleepInt32(value), nil
		}
	}
	return portableJavaValueHash(comparator), nil
}

func (c *portableJavaCollection) isList() bool {
	if c == nil || c.mapView != nil {
		return false
	}
	if c.listView != nil {
		return true
	}
	switch c.class {
	case "LinkedList", "ArrayList", "Stack", "Collections$EmptyList", "Collections$SingletonList", "Collections$CopiesList":
		return true
	default:
		return false
	}
}

func (c *portableJavaCollection) isSet() bool {
	if c == nil {
		return false
	}
	if c.mapView != nil {
		return c.mapView.kind == portableJavaMapKeySet || c.mapView.kind == portableJavaMapEntrySet
	}
	if portableCollectionsSortedSetClass(c.class) || c.class == "Collections$UnmodifiableSet" || c.class == "Collections$UnmodifiableMap$UnmodifiableEntrySet" {
		return true
	}
	switch c.class {
	case "HashSet", "LinkedHashSet", "TreeSet", "Collections$EmptySet", "Collections$SingletonSet":
		return true
	default:
		return false
	}
}

func (c *portableJavaCollection) isReadOnly() bool {
	if c == nil {
		return false
	}
	if c.listView != nil && c.listView.root != nil {
		return c.listView.root.readOnly
	}
	if c.mapView != nil && c.mapView.mapping != nil {
		return c.mapView.mapping.readOnly
	}
	return c.readOnly
}

func (c *portableJavaCollection) isA(class string) bool {
	class = portableJavaClassName(class)
	if class == c.className() || class == "Collection" || class == "Iterable" || class == "Object" {
		return true
	}
	if class == "Serializable" && portableCollectionsUnmodifiableCollectionClass(c.class) {
		return true
	}
	if c.isList() {
		matched := class == "List" || class == "AbstractList" || c.listView == nil && c.class == "LinkedList" && class == "Deque" || c.listView == nil && c.class == "LinkedList" && class == "Queue"
		if c.listView == nil && c.class == "Stack" {
			matched = matched || class == "Vector" || class == "RandomAccess" || class == "Cloneable" || class == "Serializable"
		}
		return matched
	}
	if c.isSet() {
		matched := class == "Set" || class == "AbstractSet" || c.class == "TreeSet" && (class == "SortedSet" || class == "NavigableSet")
		if portableCollectionsSortedSetClass(c.class) {
			matched = matched || class == "SortedSet"
		}
		if portableCollectionsNavigableSetClass(c.class) {
			matched = matched || class == "NavigableSet"
		}
		return matched
	}
	return false
}

// invokeStack models Stack's synchronized LIFO methods separately from the
// Deque-shaped LinkedList operations. OpenJDK Stack stores its top at the end
// of the inherited Vector and push returns the pushed item, not a boolean.
func (c *portableJavaCollection) invokeStack(invocation ObjectInvocation) (Value, bool, error) {
	if c == nil || c.class != "Stack" || c.listView != nil || c.mapView != nil {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "push":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Stack"), true, nil
		}
		value := invocation.Arg(0)
		c.mu.Lock()
		if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
			c.mu.Unlock()
			return Null(), true, err
		}
		c.values = append(c.values, value)
		c.mod++
		c.mu.Unlock()
		return value, true, nil
	case "peek", "pop":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Stack"), true, nil
		}
		c.mu.Lock()
		if len(c.values) == 0 {
			c.mu.Unlock()
			return Null(), true, errors.New("java.util.EmptyStackException")
		}
		last := len(c.values) - 1
		value := c.values[last]
		if invocation.Message == "pop" {
			c.values[last] = Null()
			c.values = c.values[:last]
			c.mod++
		}
		c.mu.Unlock()
		return value, true, nil
	case "empty":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Stack"), true, nil
		}
		c.mu.RLock()
		empty := len(c.values) == 0
		c.mu.RUnlock()
		return portableJavaBooleanValue(empty), true, nil
	case "search":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Stack"), true, nil
		}
		needle := invocation.Arg(0)
		c.mu.RLock()
		values := append([]Value(nil), c.values...)
		c.mu.RUnlock()
		for index := len(values) - 1; index >= 0; index-- {
			if portableJavaEqual(values[index], needle) {
				distance := len(values) - index
				return Int(int32(distance)), true, nil
			}
		}
		return Int(-1), true, nil
	default:
		return Null(), false, nil
	}
}

func (c *portableJavaCollection) add(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	if c.isReadOnly() {
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	}
	value := invocation.Arg(0)
	positionAtEnd := true
	size, err := c.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	position := size
	returnsValue := true
	if len(invocation.Arguments) == 2 {
		if !c.isList() || invocation.Message != "add" {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		position = int(invocation.Arg(0).Int32())
		value = invocation.Arg(1)
		returnsValue = false
		positionAtEnd = false
	} else {
		switch invocation.Message {
		case "addFirst", "offerFirst", "push":
			if c.class != "LinkedList" || c.listView != nil {
				return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
			}
			position = 0
			positionAtEnd = false
		case "addLast", "offerLast", "offer":
			if c.class != "LinkedList" || c.listView != nil {
				return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
			}
		}
	}
	if position < 0 || position > size {
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", position, size)
	}
	if c.mapView != nil {
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		if err := c.listView.insert(invocation.Runtime, position, []Value{value}); err != nil {
			return Null(), true, err
		}
		if !returnsValue {
			return Null(), true, nil
		}
		return portableJavaBooleanValue(true), true, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	size = len(c.values)
	if positionAtEnd {
		position = size
	}
	if position < 0 || position > size {
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", position, size)
	}
	if !c.isList() && portableJavaContains(c.values, value) {
		return portableJavaBooleanValue(false), true, nil
	}
	if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
		return Null(), true, err
	}
	c.values = append(c.values, Null())
	copy(c.values[position+1:], c.values[position:])
	c.values[position] = value
	c.normalizeLocked()
	c.mod++
	if !returnsValue {
		return Null(), true, nil
	}
	return portableJavaBooleanValue(true), true, nil
}

func (c *portableJavaCollection) get(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) > 1 || !c.isList() {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	size, err := c.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if size == 0 && invocation.Message != "get" {
		if strings.HasPrefix(invocation.Message, "peek") {
			return Null(), true, nil
		}
		return Null(), true, errors.New("java.util.NoSuchElementException")
	}
	index := 0
	if invocation.Message == "get" {
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		index = int(invocation.Arg(0).Int32())
	} else if invocation.Message == "getLast" || invocation.Message == "peekLast" {
		index = size - 1
	} else if len(invocation.Arguments) != 0 {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	value, err := c.getAt(index)
	return value, true, err
}

func (c *portableJavaCollection) remove(invocation ObjectInvocation) (Value, bool, error) {
	if !c.isList() && invocation.Message != "remove" {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	if invocation.Message == "remove" && len(invocation.Arguments) == 1 && c.isList() {
		argument := invocation.Arg(0)
		if (argument.Kind() == KindInt || argument.Kind() == KindLong) && c.isReadOnly() {
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
	}
	size, err := c.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if size == 0 && invocation.Message != "remove" {
		if strings.HasPrefix(invocation.Message, "poll") {
			return Null(), true, nil
		}
		return Null(), true, errors.New("java.util.NoSuchElementException")
	}
	index := -1
	returnBoolean := false
	if invocation.Message == "remove" {
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		argument := invocation.Arg(0)
		if c.isList() && (argument.Kind() == KindInt || argument.Kind() == KindLong) {
			index = int(argument.Int32())
		} else {
			returnBoolean = true
			removed, removeErr := c.removeValue(argument)
			return portableJavaBooleanValue(removed), true, removeErr
		}
	} else {
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		index = 0
		if invocation.Message == "removeLast" || invocation.Message == "pollLast" {
			index = size - 1
		}
	}
	if index < 0 || index >= size {
		if returnBoolean {
			return portableJavaBooleanValue(false), true, nil
		}
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, size)
	}
	removed, err := c.removeAt(index)
	if err != nil {
		return Null(), true, err
	}
	if returnBoolean {
		return portableJavaBooleanValue(true), true, nil
	}
	return removed, true, nil
}

func (c *portableJavaCollection) sizeChecked() (int, error) {
	if c == nil {
		return 0, nil
	}
	if c.copies {
		return c.copiesCount, nil
	}
	if c.listView != nil {
		return c.listView.length()
	}
	if c.mapView != nil {
		return c.mapView.size(), nil
	}
	c.mu.RLock()
	size := len(c.values)
	c.mu.RUnlock()
	return size, nil
}

func (c *portableJavaCollection) getAt(index int) (Value, error) {
	if c.copies {
		if index < 0 || index >= c.copiesCount {
			return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, c.copiesCount)
		}
		return c.copiesValue, nil
	}
	if c.listView != nil {
		return c.listView.get(index)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if index < 0 || index >= len(c.values) {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, len(c.values))
	}
	return c.values[index], nil
}

func (c *portableJavaCollection) setAt(index int, value Value) (Value, error) {
	if c.isReadOnly() {
		return Null(), errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.set(index, value)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if index < 0 || index >= len(c.values) {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, len(c.values))
	}
	previous := c.values[index]
	c.values[index] = value
	return previous, nil
}

func (c *portableJavaCollection) removeAt(index int) (Value, error) {
	if c.isReadOnly() {
		return Null(), errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.remove(index)
	}
	if c.mapView != nil {
		return c.mapView.removeAt(index, 0, false)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if index < 0 || index >= len(c.values) {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, len(c.values))
	}
	removed := c.values[index]
	copy(c.values[index:], c.values[index+1:])
	c.values[len(c.values)-1] = Null()
	c.values = c.values[:len(c.values)-1]
	c.mod++
	return removed, nil
}

func (c *portableJavaCollection) removeValue(value Value) (bool, error) {
	if c.mapView != nil {
		return c.mapView.removeValue(value)
	}
	values, err := c.snapshotChecked()
	if err != nil {
		return false, err
	}
	for index, candidate := range values {
		if portableJavaEqual(candidate, value) {
			_, err := c.removeAt(index)
			return err == nil, err
		}
	}
	return false, nil
}

func (c *portableJavaCollection) clearValues() error {
	size, err := c.sizeChecked()
	if err != nil {
		return err
	}
	if c.isReadOnly() && size != 0 {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.clear()
	}
	if c.mapView != nil {
		return c.mapView.clear()
	}
	c.mu.Lock()
	if len(c.values) != 0 {
		for index := range c.values {
			c.values[index] = Null()
		}
		c.values = nil
		c.mod++
	}
	c.mu.Unlock()
	return nil
}

func (c *portableJavaCollection) addAll(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	argumentIndex := 0
	positionAtEnd := true
	position, err := c.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if len(invocation.Arguments) == 2 {
		if !c.isList() || c.mapView != nil {
			return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
		}
		position = int(invocation.Arg(0).Int32())
		argumentIndex = 1
		positionAtEnd = false
	}
	values, ok, err := portableCollectionValuesChecked(invocation.Arg(argumentIndex))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	size, err := c.sizeChecked()
	if err != nil {
		return Null(), true, err
	}
	if position < 0 || position > size {
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", position, size)
	}
	if len(values) == 0 {
		return portableJavaBooleanValue(false), true, nil
	}
	if c.isReadOnly() {
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.mapView != nil {
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		if err := c.listView.insert(invocation.Runtime, position, values); err != nil {
			return Null(), true, err
		}
		return portableJavaBooleanValue(true), true, nil
	}
	c.mu.Lock()
	changed := false
	if c.isList() {
		size = len(c.values)
		if positionAtEnd {
			position = size
		}
		if position < 0 || position > size {
			c.mu.Unlock()
			return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", position, size)
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(values)); err != nil {
			c.mu.Unlock()
			return Null(), true, err
		}
		inserted := append([]Value(nil), values...)
		c.values = append(c.values, make([]Value, len(inserted))...)
		copy(c.values[position+len(inserted):], c.values[position:len(c.values)-len(inserted)])
		copy(c.values[position:], inserted)
		changed = true
	} else {
		inserted := make([]Value, 0, len(values))
		for _, value := range values {
			if !portableJavaContains(c.values, value) && !portableJavaContains(inserted, value) {
				inserted = append(inserted, value)
			}
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(inserted)); err != nil {
			c.mu.Unlock()
			return Null(), true, err
		}
		c.values = append(c.values, inserted...)
		changed = len(inserted) != 0
		c.normalizeLocked()
	}
	if changed {
		c.mod++
	}
	c.mu.Unlock()
	return portableJavaBooleanValue(changed), true, nil
}

func (c *portableJavaCollection) containsAll(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	needles, ok, err := portableCollectionValuesChecked(invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	values, err := c.snapshotChecked()
	if err != nil {
		return Null(), true, err
	}
	for _, needle := range needles {
		if !portableJavaContains(values, needle) {
			return portableJavaBooleanValue(false), true, nil
		}
	}
	return portableJavaBooleanValue(true), true, nil
}

func (c *portableJavaCollection) filterAll(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	filter, ok, err := portableCollectionValuesChecked(invocation.Arg(0))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	values, err := c.snapshotChecked()
	if err != nil {
		return Null(), true, err
	}
	removeContained := invocation.Message == "removeAll"
	indices := make([]int, 0)
	for index, value := range values {
		contained := portableJavaContains(filter, value)
		if contained == removeContained {
			indices = append(indices, index)
		}
	}
	for index := len(indices) - 1; index >= 0; index-- {
		if _, err := c.removeAt(indices[index]); err != nil {
			return Null(), true, err
		}
	}
	return portableJavaBooleanValue(len(indices) != 0), true, nil
}

func (c *portableJavaCollection) subList(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 || !c.isList() {
		return portableNoMatchingMethod(invocation, "java.util."+c.className()), true, nil
	}
	start, end := int(invocation.Arg(0).Int32()), int(invocation.Arg(1).Int32())
	if c.copies {
		if start < 0 {
			return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: fromIndex = %d", start)
		}
		if end > c.copiesCount {
			return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: toIndex = %d", end)
		}
		if start > end {
			return Null(), true, fmt.Errorf("java.lang.IllegalArgumentException: fromIndex(%d) > toIndex(%d)", start, end)
		}
		return ObjectValue(&portableJavaCollection{
			class: "Collections$CopiesList", readOnly: true, copies: true,
			copiesCount: end - start, copiesValue: c.copiesValue,
		}), true, nil
	}
	root, offset, expected := c, start, uint64(0)
	size := 0
	if c.listView != nil {
		view := c.listView
		root = view.root
		root.mu.RLock()
		if err := view.checkLocked(); err != nil {
			root.mu.RUnlock()
			return Null(), true, err
		}
		size = view.size
		offset += view.offset
		expected = root.mod
		root.mu.RUnlock()
	} else {
		c.mu.RLock()
		size = len(c.values)
		expected = c.mod
		c.mu.RUnlock()
	}
	if start < 0 {
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: fromIndex = %d", start)
	}
	if end > size {
		return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: toIndex = %d", end)
	}
	if start > end {
		return Null(), true, fmt.Errorf("java.lang.IllegalArgumentException: fromIndex(%d) > toIndex(%d)", start, end)
	}
	viewCollection := &portableJavaCollection{class: "AbstractList$SubList"}
	viewCollection.listView = &portableJavaListView{
		root: root, parent: c, owner: viewCollection, offset: offset, size: end - start, expectedMod: expected,
	}
	return ObjectValue(viewCollection), true, nil
}

func (c *portableJavaCollection) iteratorBounds() (int, uint64, error) {
	if c.copies {
		return c.copiesCount, 0, nil
	}
	if c.listView != nil {
		return c.listView.iteratorBounds()
	}
	if c.mapView != nil {
		return c.mapView.iteratorBounds()
	}
	c.mu.RLock()
	size, revision := len(c.values), c.mod
	c.mu.RUnlock()
	return size, revision, nil
}

func (c *portableJavaCollection) iteratorValue(index int, expected uint64) (Value, bool, error) {
	if c.copies {
		if expected != 0 {
			return Null(), false, errors.New("java.util.ConcurrentModificationException")
		}
		if index < 0 || index >= c.copiesCount {
			return Null(), false, nil
		}
		return c.copiesValue, true, nil
	}
	if c.listView != nil {
		return c.listView.iteratorValue(index, expected)
	}
	if c.mapView != nil {
		return c.mapView.iteratorValue(index, expected)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if expected != c.mod {
		return Null(), false, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= len(c.values) {
		return Null(), false, nil
	}
	return c.values[index], true, nil
}

func (c *portableJavaCollection) iteratorRemove(index int, expected uint64) (uint64, error) {
	if c.copies {
		if expected != 0 {
			return expected, errors.New("java.util.ConcurrentModificationException")
		}
		if index < 0 || index >= c.copiesCount {
			return expected, errors.New("java.lang.IllegalStateException")
		}
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.iteratorRemove(index, expected)
	}
	if c.mapView != nil {
		_, err := c.mapView.removeAt(index, expected, true)
		if err != nil {
			return expected, err
		}
		return c.mapView.revision(), nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if expected != c.mod {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= len(c.values) {
		return expected, errors.New("java.lang.IllegalStateException")
	}
	if c.readOnly {
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	copy(c.values[index:], c.values[index+1:])
	c.values[len(c.values)-1] = Null()
	c.values = c.values[:len(c.values)-1]
	c.mod++
	return c.mod, nil
}

func (c *portableJavaCollection) iteratorSet(index int, expected uint64, value Value) error {
	if c == nil || !c.isList() {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	if c.copies {
		if expected != 0 {
			return errors.New("java.util.ConcurrentModificationException")
		}
		if index < 0 || index >= c.copiesCount {
			return errors.New("java.lang.IllegalStateException")
		}
		return errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.iteratorSet(index, expected, value)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if expected != c.mod {
		return errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= len(c.values) {
		return errors.New("java.lang.IllegalStateException")
	}
	if c.readOnly {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	c.values[index] = value
	return nil
}

func (c *portableJavaCollection) iteratorAdd(runtime *Runtime, index int, expected uint64, value Value) (uint64, error) {
	if c == nil || !c.isList() {
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.copies {
		if expected != 0 {
			return expected, errors.New("java.util.ConcurrentModificationException")
		}
		if index < 0 || index > c.copiesCount {
			return expected, errors.New("java.util.ConcurrentModificationException")
		}
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	if c.listView != nil {
		return c.listView.iteratorAdd(runtime, index, expected, value)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if expected != c.mod {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index > len(c.values) {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if c.readOnly {
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	if err := reserveCollectionEntries(runtime, 1); err != nil {
		return expected, err
	}
	c.values = append(c.values, Null())
	copy(c.values[index+1:], c.values[index:])
	c.values[index] = value
	c.mod++
	return c.mod, nil
}

// iteratorSizeUnchecked mirrors the JDK iterator cursor predicates: hasNext,
// hasPrevious, nextIndex, and previousIndex do not perform the fail-fast
// modCount check. Operations that dereference or mutate the cursor do.
func (c *portableJavaCollection) iteratorSizeUnchecked() int {
	if c == nil {
		return 0
	}
	if c.copies {
		return c.copiesCount
	}
	if c.listView != nil {
		view := c.listView
		view.root.mu.RLock()
		size := view.size
		view.root.mu.RUnlock()
		return size
	}
	if c.mapView != nil {
		return c.mapView.size()
	}
	c.mu.RLock()
	size := len(c.values)
	c.mu.RUnlock()
	return size
}

func (c *portableJavaCollection) equalValue(value Value) bool {
	equal, _ := c.equalValueChecked(value)
	return equal
}

func (c *portableJavaCollection) equalValueChecked(value Value) (bool, error) {
	object, ok := value.Object()
	if !ok {
		return false, nil
	}
	other, ok := object.(*portableJavaCollection)
	if !ok || other == nil {
		return false, nil
	}
	if c == other {
		return true, nil
	}
	if c.copies && other.copies {
		if c.copiesCount != other.copiesCount {
			return false, nil
		}
		return c.copiesCount == 0 || portableJavaEqual(c.copiesValue, other.copiesValue), nil
	}
	if c.isList() != other.isList() || c.isSet() != other.isSet() || !c.isList() && !c.isSet() {
		return false, nil
	}
	if c.isList() {
		leftSize, leftExpected, err := c.iteratorBounds()
		if err != nil {
			return false, err
		}
		rightSize, rightExpected, err := other.iteratorBounds()
		if err != nil {
			return false, err
		}
		if leftSize != rightSize {
			return false, nil
		}
		if (c.copies || other.copies) && leftSize > portableCollectionsMaximumMaterializedElements {
			return false, errPortableCollectionsMaterializationLimit
		}
		for index := 0; index < leftSize; index++ {
			left, present, err := c.iteratorValue(index, leftExpected)
			if err != nil {
				return false, err
			}
			if !present {
				return false, errors.New("java.util.NoSuchElementException")
			}
			right, present, err := other.iteratorValue(index, rightExpected)
			if err != nil {
				return false, err
			}
			if !present {
				return false, errors.New("java.util.NoSuchElementException")
			}
			if !portableJavaEqual(left, right) {
				return false, nil
			}
		}
		return true, nil
	}
	left, leftErr := c.snapshotChecked()
	right, rightErr := other.snapshotChecked()
	if leftErr != nil {
		return false, leftErr
	}
	if rightErr != nil {
		return false, rightErr
	}
	if len(left) != len(right) {
		return false, nil
	}
	for _, value := range left {
		if !portableJavaContains(right, value) {
			return false, nil
		}
	}
	return true, nil
}

func (c *portableJavaCollection) javaHashCode() (int32, error) {
	if c.isList() {
		if c.copies {
			return portableCollectionsCopiesListHash(c.copiesCount, portableJavaValueHash(c.copiesValue)), nil
		}
		size, expected, err := c.iteratorBounds()
		if err != nil {
			return 0, err
		}
		hash := int32(1)
		for index := 0; index < size; index++ {
			value, present, err := c.iteratorValue(index, expected)
			if err != nil {
				return 0, err
			}
			if !present {
				return 0, errors.New("java.util.NoSuchElementException")
			}
			hash = 31*hash + portableJavaValueHash(value)
		}
		return hash, nil
	}
	if c.isSet() {
		values, err := c.snapshotChecked()
		if err != nil {
			return 0, err
		}
		var hash int32
		for _, value := range values {
			hash += portableJavaValueHash(value)
		}
		return hash, nil
	}
	// AbstractCollection inherits Object.hashCode. Returning zero is a stable
	// identity-hash approximation and preserves the required equals contract.
	return 0, nil
}

type portableJavaListView struct {
	root        *portableJavaCollection
	parent      *portableJavaCollection
	owner       *portableJavaCollection
	offset      int
	size        int
	expectedMod uint64
}

func (view *portableJavaListView) checkLocked() error {
	if view == nil || view.root == nil || view.expectedMod != view.root.mod {
		return errors.New("java.util.ConcurrentModificationException")
	}
	return nil
}

func (view *portableJavaListView) snapshot() ([]Value, error) {
	view.root.mu.RLock()
	defer view.root.mu.RUnlock()
	if err := view.checkLocked(); err != nil {
		return nil, err
	}
	return append([]Value(nil), view.root.values[view.offset:view.offset+view.size]...), nil
}

func (view *portableJavaListView) length() (int, error) {
	view.root.mu.RLock()
	defer view.root.mu.RUnlock()
	if err := view.checkLocked(); err != nil {
		return 0, err
	}
	return view.size, nil
}

func (view *portableJavaListView) get(index int) (Value, error) {
	view.root.mu.RLock()
	defer view.root.mu.RUnlock()
	if err := view.checkLocked(); err != nil {
		return Null(), err
	}
	if index < 0 || index >= view.size {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, view.size)
	}
	return view.root.values[view.offset+index], nil
}

func (view *portableJavaListView) set(index int, value Value) (Value, error) {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return Null(), err
	}
	if index < 0 || index >= view.size {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, view.size)
	}
	if view.root.readOnly {
		return Null(), errors.New("java.lang.UnsupportedOperationException")
	}
	absolute := view.offset + index
	previous := view.root.values[absolute]
	view.root.values[absolute] = value
	return previous, nil
}

func (view *portableJavaListView) insert(runtime *Runtime, index int, values []Value) error {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return err
	}
	if index < 0 || index > view.size {
		return fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, view.size)
	}
	if len(values) == 0 {
		return nil
	}
	if view.root.readOnly {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	if err := reserveCollectionEntries(runtime, len(values)); err != nil {
		return err
	}
	absolute := view.offset + index
	inserted := append([]Value(nil), values...)
	view.root.values = append(view.root.values, make([]Value, len(inserted))...)
	copy(view.root.values[absolute+len(inserted):], view.root.values[absolute:len(view.root.values)-len(inserted)])
	copy(view.root.values[absolute:], inserted)
	view.root.mod++
	view.adjustAncestorsLocked(len(inserted))
	return nil
}

func (view *portableJavaListView) remove(index int) (Value, error) {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return Null(), err
	}
	if index < 0 || index >= view.size {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, view.size)
	}
	if view.root.readOnly {
		return Null(), errors.New("java.lang.UnsupportedOperationException")
	}
	absolute := view.offset + index
	removed := view.root.values[absolute]
	copy(view.root.values[absolute:], view.root.values[absolute+1:])
	view.root.values[len(view.root.values)-1] = Null()
	view.root.values = view.root.values[:len(view.root.values)-1]
	view.root.mod++
	view.adjustAncestorsLocked(-1)
	return removed, nil
}

func (view *portableJavaListView) clear() error {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return err
	}
	if view.size == 0 {
		return nil
	}
	if view.root.readOnly {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	end := view.offset + view.size
	removed := view.size
	copy(view.root.values[view.offset:], view.root.values[end:])
	for index := len(view.root.values) - removed; index < len(view.root.values); index++ {
		view.root.values[index] = Null()
	}
	view.root.values = view.root.values[:len(view.root.values)-removed]
	view.root.mod++
	view.adjustAncestorsLocked(-removed)
	return nil
}

func (view *portableJavaListView) adjustAncestorsLocked(delta int) {
	for collection := view.owner; collection != nil && collection.listView != nil; collection = collection.listView.parent {
		collection.listView.size += delta
		collection.listView.expectedMod = view.root.mod
	}
}

func (view *portableJavaListView) iteratorBounds() (int, uint64, error) {
	view.root.mu.RLock()
	defer view.root.mu.RUnlock()
	if err := view.checkLocked(); err != nil {
		return 0, 0, err
	}
	return view.size, view.root.mod, nil
}

func (view *portableJavaListView) iteratorValue(index int, expected uint64) (Value, bool, error) {
	view.root.mu.RLock()
	defer view.root.mu.RUnlock()
	if err := view.checkLocked(); err != nil {
		return Null(), false, err
	}
	if expected != view.root.mod {
		return Null(), false, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= view.size {
		return Null(), false, nil
	}
	return view.root.values[view.offset+index], true, nil
}

func (view *portableJavaListView) iteratorRemove(index int, expected uint64) (uint64, error) {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return expected, err
	}
	if expected != view.root.mod {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= view.size {
		return expected, errors.New("java.lang.IllegalStateException")
	}
	if view.root.readOnly {
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	absolute := view.offset + index
	copy(view.root.values[absolute:], view.root.values[absolute+1:])
	view.root.values[len(view.root.values)-1] = Null()
	view.root.values = view.root.values[:len(view.root.values)-1]
	view.root.mod++
	view.adjustAncestorsLocked(-1)
	return view.root.mod, nil
}

func (view *portableJavaListView) iteratorSet(index int, expected uint64, value Value) error {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return err
	}
	if expected != view.root.mod {
		return errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index >= view.size {
		return errors.New("java.lang.IllegalStateException")
	}
	if view.root.readOnly {
		return errors.New("java.lang.UnsupportedOperationException")
	}
	view.root.values[view.offset+index] = value
	return nil
}

func (view *portableJavaListView) iteratorAdd(runtime *Runtime, index int, expected uint64, value Value) (uint64, error) {
	view.root.mu.Lock()
	defer view.root.mu.Unlock()
	if err := view.checkLocked(); err != nil {
		return expected, err
	}
	if expected != view.root.mod {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if index < 0 || index > view.size {
		return expected, errors.New("java.util.ConcurrentModificationException")
	}
	if view.root.readOnly {
		return expected, errors.New("java.lang.UnsupportedOperationException")
	}
	if err := reserveCollectionEntries(runtime, 1); err != nil {
		return expected, err
	}
	absolute := view.offset + index
	view.root.values = append(view.root.values, Null())
	copy(view.root.values[absolute+1:], view.root.values[absolute:])
	view.root.values[absolute] = value
	view.root.mod++
	view.adjustAncestorsLocked(1)
	return view.root.mod, nil
}

type portableJavaIterator struct {
	// Cursor/last/expectedMod follow OpenJDK 8u AbstractList.ListItr and
	// ArrayList.ListItr. The mutex makes concurrent host reentry race-safe;
	// Java still makes no semantic thread-safety guarantee for an iterator.
	mu           sync.Mutex
	collection   *portableJavaCollection
	index        int
	last         int
	expectedMod  uint64
	listIterator bool
	enumeration  bool
}

func (i *portableJavaIterator) String() string {
	if i != nil && i.listIterator {
		return "java.util.ListIterator"
	}
	if i != nil && i.enumeration {
		return "java.util.Enumeration"
	}
	return "java.util.Iterator"
}

func (i *portableJavaIterator) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		if i != nil && i.listIterator {
			return Bool(class == "Iterator" || class == "ListIterator" || class == "Object"), true, nil
		}
		if i != nil && i.enumeration {
			return Bool(class == "Enumeration" || class == "Object"), true, nil
		}
		return Bool(class == "Iterator" || class == "Object"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if i == nil || i.collection == nil {
		return Null(), true, errors.New("java.util.NoSuchElementException")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	switch invocation.Message {
	case "hasNext", "hasMoreElements":
		if len(invocation.Arguments) != 0 || invocation.Message == "hasNext" && i.enumeration || invocation.Message == "hasMoreElements" && !i.enumeration {
			return Null(), false, nil
		}
		size := i.collection.iteratorSizeUnchecked()
		return portableJavaBooleanValue(i.index != size), true, nil
	case "next", "nextElement":
		if len(invocation.Arguments) != 0 || invocation.Message == "next" && i.enumeration || invocation.Message == "nextElement" && !i.enumeration {
			return Null(), false, nil
		}
		value, present, err := i.collection.iteratorValue(i.index, i.expectedMod)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		i.last = i.index
		i.index++
		return value, true, nil
	case "hasPrevious":
		if len(invocation.Arguments) != 0 || !i.listIterator {
			return Null(), false, nil
		}
		return portableJavaBooleanValue(i.index > 0), true, nil
	case "previous":
		if len(invocation.Arguments) != 0 || !i.listIterator {
			return Null(), false, nil
		}
		previous := i.index - 1
		value, present, err := i.collection.iteratorValue(previous, i.expectedMod)
		if err != nil {
			return Null(), true, err
		}
		if !present {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		i.index = previous
		i.last = previous
		return value, true, nil
	case "nextIndex":
		if len(invocation.Arguments) != 0 || !i.listIterator {
			return Null(), false, nil
		}
		return Int(int32(i.index)), true, nil
	case "previousIndex":
		if len(invocation.Arguments) != 0 || !i.listIterator {
			return Null(), false, nil
		}
		return Int(int32(i.index - 1)), true, nil
	case "remove":
		if len(invocation.Arguments) != 0 || i.enumeration {
			return Null(), false, nil
		}
		if portableCollectionsUnmodifiableCollectionClass(i.collection.class) {
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		if !i.listIterator && (i.collection.class == "Collections$SingletonList" || i.collection.class == "Collections$SingletonSet") {
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		if i.last < 0 {
			return Null(), true, errors.New("java.lang.IllegalStateException")
		}
		revision, err := i.collection.iteratorRemove(i.last, i.expectedMod)
		if err != nil {
			return Null(), true, err
		}
		i.expectedMod = revision
		i.index = i.last
		i.last = -1
		return Null(), true, nil
	case "set":
		if len(invocation.Arguments) != 1 || !i.listIterator {
			return Null(), false, nil
		}
		if portableCollectionsUnmodifiableCollectionClass(i.collection.class) {
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		if i.last < 0 {
			return Null(), true, errors.New("java.lang.IllegalStateException")
		}
		if err := i.collection.iteratorSet(i.last, i.expectedMod, invocation.Arg(0)); err != nil {
			return Null(), true, err
		}
		return Null(), true, nil
	case "add":
		if len(invocation.Arguments) != 1 || !i.listIterator {
			return Null(), false, nil
		}
		if portableCollectionsUnmodifiableCollectionClass(i.collection.class) {
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		revision, err := i.collection.iteratorAdd(invocation.Runtime, i.index, i.expectedMod, invocation.Arg(0))
		if err != nil {
			return Null(), true, err
		}
		i.index++
		i.last = -1
		i.expectedMod = revision
		return Null(), true, nil
	}
	return Null(), false, nil
}

// portableJavaMap models the map constructors needed by Sleep's conversion
// fixtures. TreeMap sorts by key; linked variants retain insertion order.
type portableJavaMap struct {
	mu        sync.RWMutex
	class     string
	keys      []string
	keyValues map[string]Value
	values    map[string]Value
	// entries retains the stable node identity returned by entrySet iterators.
	// Removal drops the node from this table, so a later canonical-key insert
	// cannot silently reattach an older Map.Entry reference.
	entries             map[string]*portableJavaMapEntry
	mod                 uint64
	readOnly            bool
	keySetView          *portableJavaCollection
	valuesView          *portableJavaCollection
	entrySetView        *portableJavaCollection
	navigableKeySetView *portableJavaCollection
	reverseOrder        bool
	// scriptEnvironment is populated only for java.util.Hashtable instances
	// passed to ScriptLoader. Keeping the state on the table preserves sharing
	// when the same object crosses loader instances without a process-global
	// pointer registry.
	scriptEnvironment *portableScriptSharedEnvironment
}

func portableCollectionsSortedMapClass(class string) bool {
	switch class {
	case portableCollectionsEmptyNavigableMapClass,
		portableCollectionsUnmodifiableSortedMapClass,
		portableCollectionsUnmodifiableNavigableMapClass:
		return true
	default:
		return false
	}
}

func portableCollectionsNavigableMapClass(class string) bool {
	return class == portableCollectionsEmptyNavigableMapClass || class == portableCollectionsUnmodifiableNavigableMapClass
}

func newPortableCollectionsEmptyNavigableMap() *portableJavaMap {
	mapping := newPortableJavaMap(portableCollectionsEmptyNavigableMapClass, nil)
	mapping.readOnly = true
	mapping.navigableKeySetView = portableCollectionsEmptyNavigableSet
	return mapping
}

func newPortableCollectionsSortedMapView(navigable, reverse bool) *portableJavaMap {
	class := portableCollectionsUnmodifiableSortedMapClass
	if navigable {
		class = portableCollectionsUnmodifiableNavigableMapClass
	}
	mapping := newPortableJavaMap(class, nil)
	mapping.readOnly = true
	mapping.reverseOrder = reverse
	return mapping
}

func newPortableJavaMap(class string, hash *Hash) *portableJavaMap {
	var entries []portableJavaMapSnapshotEntry
	if hash != nil {
		entries, _ = portableMapEntries(HashValue(hash))
	}
	return newPortableJavaMapFromEntries(class, entries)
}

func newPortableJavaMapFromEntries(class string, source []portableJavaMapSnapshotEntry) *portableJavaMap {
	mapping := &portableJavaMap{
		class:     class,
		keyValues: make(map[string]Value),
		values:    make(map[string]Value),
		entries:   make(map[string]*portableJavaMapEntry),
	}
	for _, entry := range source {
		mapping.keys = append(mapping.keys, entry.key)
		mapping.keyValues[entry.key] = entry.keyValue
		mapping.values[entry.key] = entry.value
		mapping.entries[entry.key] = &portableJavaMapEntry{
			mapping: mapping, key: entry.key, keyValue: entry.keyValue, value: entry.value,
		}
	}
	if class == "TreeMap" {
		mapping.sortKeysLocked()
	}
	return mapping
}

func (m *portableJavaMap) className() string {
	if m == nil || m.class == "" {
		return "HashMap"
	}
	return m.class
}

func (m *portableJavaMap) sortKeysLocked() {
	sort.SliceStable(m.keys, func(left, right int) bool {
		return sleepStringCompareValues(m.keyValues[m.keys[left]], m.keyValues[m.keys[right]]) < 0
	})
}

func (m *portableJavaMap) wrapperKeysLocked() []string {
	if m == nil {
		return nil
	}
	if m.class != "HashMap" && m.class != "" {
		return append([]string(nil), m.keys...)
	}
	capacity := 16
	for len(m.values) > capacity*3/4 {
		capacity *= 2
	}
	buckets := make([][]string, capacity)
	for _, key := range m.keys {
		if _, exists := m.values[key]; !exists {
			continue
		}
		bucket := int(java7StringHashValue(m.keyValues[key]) & uint32(capacity-1))
		buckets[bucket] = append([]string{key}, buckets[bucket]...)
	}
	keys := make([]string, 0, len(m.values))
	for _, bucket := range buckets {
		keys = append(keys, bucket...)
	}
	return keys
}

func (m *portableJavaMap) String() string {
	if m == nil {
		return "{}"
	}
	m.mu.RLock()
	keys := append([]string(nil), m.keys...)
	values := make(map[string]Value, len(m.values))
	keyValues := make(map[string]Value, len(m.keyValues))
	for key, value := range m.values {
		values[key] = value
		keyValues[key] = m.keyValues[key]
	}
	m.mu.RUnlock()
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, sleepCanonicalString(keyValues[key])+"="+portableJavaValueString(values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (m *portableJavaMap) snapshotHash() *Hash {
	hash := NewHash()
	if m == nil {
		return hash
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, key := range m.keys {
		hash.SetValue(m.keyValues[key], m.values[key])
	}
	return hash
}

func (m *portableJavaMap) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		matched := class == m.className() || class == "Map" || class == "Object"
		if m.class == "TreeMap" {
			matched = matched || class == "SortedMap" || class == "NavigableMap"
		}
		if portableCollectionsSortedMapClass(m.class) {
			matched = matched || class == "SortedMap"
		}
		if portableCollectionsNavigableMapClass(m.class) {
			matched = matched || class == "NavigableMap"
		}
		if portableCollectionsSortedMapClass(m.class) {
			matched = matched || class == "Serializable"
		}
		return Bool(matched), true, nil
	}
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectGet {
		return Null(), false, nil
	}
	if portableCollectionsSortedMapClass(m.class) {
		expected := -1
		switch invocation.Message {
		case "clear":
			expected = 0
		case "put":
			expected = 2
		case "putAll":
			expected = 1
		case "remove":
			if len(invocation.Arguments) == 1 || len(invocation.Arguments) == 2 {
				return Null(), true, errors.New("java.lang.UnsupportedOperationException")
			}
			expected = 1
		case "putIfAbsent":
			expected = 2
		case "replace":
			if len(invocation.Arguments) == 2 || len(invocation.Arguments) == 3 {
				return Null(), true, errors.New("java.lang.UnsupportedOperationException")
			}
			expected = 2
		}
		if expected >= 0 {
			if len(invocation.Arguments) != expected {
				return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
			}
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
	}
	if value, handled, err := m.invokeCollectionsSortedMap(invocation); handled {
		return value, true, err
	}
	switch invocation.Message {
	case "size":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		m.mu.RLock()
		size := len(m.values)
		m.mu.RUnlock()
		return Int(int32(size)), true, nil
	case "isEmpty":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		m.mu.RLock()
		empty := len(m.values) == 0
		m.mu.RUnlock()
		return portableJavaBooleanValue(empty), true, nil
	case "get", "containsKey":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key := sleepCanonicalString(invocation.Arg(0))
		m.mu.RLock()
		value, ok := m.values[key]
		m.mu.RUnlock()
		if invocation.Message == "containsKey" {
			return portableJavaBooleanValue(ok), true, nil
		}
		if !ok {
			return Null(), true, nil
		}
		return value, true, nil
	case "getOrDefault":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key := sleepCanonicalString(invocation.Arg(0))
		m.mu.RLock()
		value, exists := m.values[key]
		m.mu.RUnlock()
		if !exists {
			return invocation.Arg(1), true, nil
		}
		return value, true, nil
	case "remove":
		if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key := sleepCanonicalString(invocation.Arg(0))
		m.mu.Lock()
		if len(invocation.Arguments) == 2 && m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		value, exists := m.values[key]
		if len(invocation.Arguments) == 2 {
			if !exists || !portableJavaEqual(value, invocation.Arg(1)) {
				m.mu.Unlock()
				return portableJavaBooleanValue(false), true, nil
			}
			m.removeKeyLocked(key)
			m.mu.Unlock()
			return portableJavaBooleanValue(true), true, nil
		}
		if exists && m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		if exists {
			value, _ = m.removeKeyLocked(key)
		}
		m.mu.Unlock()
		if !exists {
			return Null(), true, nil
		}
		return value, true, nil
	case "containsValue":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		needle := invocation.Arg(0)
		m.mu.RLock()
		values := make([]Value, 0, len(m.values))
		for _, value := range m.values {
			values = append(values, value)
		}
		m.mu.RUnlock()
		return portableJavaBooleanValue(portableJavaContains(values, needle)), true, nil
	case "put":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key, keyValue := sleepHashKey(invocation.Arg(0))
		m.mu.Lock()
		if m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		previous, existed := m.values[key]
		if !existed {
			if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
				m.mu.Unlock()
				return Null(), true, err
			}
			m.keys = append(m.keys, key)
			m.keyValues[key] = keyValue
			m.entries[key] = &portableJavaMapEntry{
				mapping: m, key: key, keyValue: keyValue, value: invocation.Arg(1),
			}
			m.mod++
		} else if entry := m.entries[key]; entry != nil {
			entry.mu.Lock()
			entry.value = invocation.Arg(1)
			entry.mu.Unlock()
		}
		m.values[key] = invocation.Arg(1)
		if m.class == "TreeMap" {
			m.sortKeysLocked()
		}
		m.mu.Unlock()
		if !existed {
			return Null(), true, nil
		}
		return previous, true, nil
	case "putIfAbsent":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key, keyValue := sleepHashKey(invocation.Arg(0))
		m.mu.Lock()
		if m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		previous, existed := m.values[key]
		if !existed || previous.IsNull() {
			if !existed {
				if err := reserveCollectionEntries(invocation.Runtime, 1); err != nil {
					m.mu.Unlock()
					return Null(), true, err
				}
				m.keys = append(m.keys, key)
				m.keyValues[key] = keyValue
				m.entries[key] = &portableJavaMapEntry{
					mapping: m, key: key, keyValue: keyValue, value: invocation.Arg(1),
				}
				m.mod++
			} else if entry := m.entries[key]; entry != nil {
				entry.mu.Lock()
				entry.value = invocation.Arg(1)
				entry.mu.Unlock()
			}
			m.values[key] = invocation.Arg(1)
			if m.class == "TreeMap" {
				m.sortKeysLocked()
			}
		}
		m.mu.Unlock()
		if !existed {
			return Null(), true, nil
		}
		return previous, true, nil
	case "replace":
		if len(invocation.Arguments) != 2 && len(invocation.Arguments) != 3 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		key := sleepCanonicalString(invocation.Arg(0))
		m.mu.Lock()
		if m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		previous, exists := m.values[key]
		if !exists || len(invocation.Arguments) == 3 && !portableJavaEqual(previous, invocation.Arg(1)) {
			m.mu.Unlock()
			if len(invocation.Arguments) == 3 {
				return portableJavaBooleanValue(false), true, nil
			}
			return Null(), true, nil
		}
		value := invocation.Arg(1)
		if len(invocation.Arguments) == 3 {
			value = invocation.Arg(2)
		}
		m.values[key] = value
		if entry := m.entries[key]; entry != nil {
			entry.mu.Lock()
			entry.value = value
			entry.mu.Unlock()
		}
		m.mu.Unlock()
		if len(invocation.Arguments) == 3 {
			return portableJavaBooleanValue(true), true, nil
		}
		return previous, true, nil
	case "putAll":
		return m.putAll(invocation)
	case "keySet", "values", "entrySet":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		m.mu.Lock()
		if cached := m.cachedViewLocked(invocation.Message); cached != nil {
			m.mu.Unlock()
			return ObjectValue(cached), true, nil
		}
		class := m.className() + "$KeySet"
		kind := portableJavaMapKeySet
		var wrapperSource collectionWrapperSource = &portableJavaMapKeySetSource{mapping: m}
		if invocation.Message == "values" {
			class = m.className() + "$Values"
			kind = portableJavaMapValues
			wrapperSource = &portableJavaMapValuesSource{mapping: m}
		} else if invocation.Message == "entrySet" {
			class = m.className() + "$EntrySet"
			kind = portableJavaMapEntrySet
			wrapperSource = &portableJavaMapEntrySetSource{mapping: m}
		}
		collection := newPortableJavaCollection(class, nil)
		collection.wrapperSource = wrapperSource
		collection.mapView = &portableJavaMapView{mapping: m, kind: kind}
		m.cacheViewLocked(invocation.Message, collection)
		m.mu.Unlock()
		return ObjectValue(collection), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		return portableJavaBooleanValue(m.equalValue(invocation.Arg(0))), true, nil
	case "hashCode":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		return Int(m.javaHashCode()), true, nil
	case "clear":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		m.mu.Lock()
		changed := len(m.values) != 0
		if changed && m.readOnly {
			m.mu.Unlock()
			return Null(), true, errors.New("java.lang.UnsupportedOperationException")
		}
		m.detachEntriesLocked()
		m.keys = nil
		m.keyValues = make(map[string]Value)
		m.values = make(map[string]Value)
		m.entries = make(map[string]*portableJavaMapEntry)
		if changed {
			m.mod++
		}
		m.mu.Unlock()
		return Null(), true, nil
	case "toString":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
		}
		return String(m.String()), true, nil
	}
	return Null(), false, nil
}

func (m *portableJavaMap) invokeCollectionsSortedMap(invocation ObjectInvocation) (Value, bool, error) {
	if m == nil || !portableCollectionsSortedMapClass(m.class) {
		return Null(), false, nil
	}
	class := "java.util." + m.className()
	navigable := portableCollectionsNavigableMapClass(m.class)
	switch invocation.Message {
	case "comparator":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		if m.reverseOrder {
			return ObjectValue(portableCollectionsReverseComparator), true, nil
		}
		return Null(), true, nil
	case "firstKey", "lastKey":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, errors.New("java.util.NoSuchElementException")
	case "subMap":
		if len(invocation.Arguments) == 2 {
			if err := portableCollectionsValidateRange(invocation.Arg(0), invocation.Arg(1), m.reverseOrder); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedMapView(false, m.reverseOrder)), true, nil
		}
		if navigable && len(invocation.Arguments) == 4 {
			if err := portableCollectionsValidateRange(invocation.Arg(0), invocation.Arg(2), m.reverseOrder); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedMapView(true, m.reverseOrder)), true, nil
		}
		return portableNoMatchingMethod(invocation, class), true, nil
	case "headMap", "tailMap":
		if len(invocation.Arguments) == 1 {
			if err := portableCollectionsValidateEndpoint(invocation.Arg(0)); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedMapView(false, m.reverseOrder)), true, nil
		}
		if navigable && len(invocation.Arguments) == 2 {
			if err := portableCollectionsValidateEndpoint(invocation.Arg(0)); err != nil {
				return Null(), true, err
			}
			return ObjectValue(newPortableCollectionsSortedMapView(true, m.reverseOrder)), true, nil
		}
		return portableNoMatchingMethod(invocation, class), true, nil
	case "lowerEntry", "lowerKey", "floorEntry", "floorKey", "ceilingEntry", "ceilingKey", "higherEntry", "higherKey":
		if !navigable || len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, nil
	case "firstEntry", "lastEntry":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, nil
	case "pollFirstEntry", "pollLastEntry":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	case "descendingMap":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return ObjectValue(newPortableCollectionsSortedMapView(true, !m.reverseOrder)), true, nil
	case "navigableKeySet":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		m.mu.Lock()
		if m.navigableKeySetView == nil {
			m.navigableKeySetView = newPortableCollectionsSortedSetView(true, m.reverseOrder)
		}
		view := m.navigableKeySetView
		m.mu.Unlock()
		return ObjectValue(view), true, nil
	case "descendingKeySet":
		if !navigable || len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		return ObjectValue(newPortableCollectionsSortedSetView(true, !m.reverseOrder)), true, nil
	case "keySet", "values", "entrySet":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		m.mu.Lock()
		if cached := m.cachedViewLocked(invocation.Message); cached != nil {
			m.mu.Unlock()
			return ObjectValue(cached), true, nil
		}
		viewClass := "Collections$UnmodifiableSet"
		kind := portableJavaMapKeySet
		var wrapperSource collectionWrapperSource = &portableJavaMapKeySetSource{mapping: m}
		if invocation.Message == "values" {
			viewClass = "Collections$UnmodifiableCollection"
			kind = portableJavaMapValues
			wrapperSource = &portableJavaMapValuesSource{mapping: m}
		} else if invocation.Message == "entrySet" {
			viewClass = "Collections$UnmodifiableMap$UnmodifiableEntrySet"
			kind = portableJavaMapEntrySet
			wrapperSource = &portableJavaMapEntrySetSource{mapping: m}
		}
		view := newPortableJavaCollection(viewClass, nil)
		view.readOnly = true
		view.wrapperSource = wrapperSource
		view.mapView = &portableJavaMapView{mapping: m, kind: kind}
		m.cacheViewLocked(invocation.Message, view)
		m.mu.Unlock()
		return ObjectValue(view), true, nil
	default:
		return Null(), false, nil
	}
}

func (m *portableJavaMap) cachedViewLocked(message string) *portableJavaCollection {
	switch message {
	case "keySet":
		return m.keySetView
	case "values":
		return m.valuesView
	case "entrySet":
		return m.entrySetView
	default:
		return nil
	}
}

func (m *portableJavaMap) cacheViewLocked(message string, collection *portableJavaCollection) {
	switch message {
	case "keySet":
		m.keySetView = collection
	case "values":
		m.valuesView = collection
	case "entrySet":
		m.entrySetView = collection
	}
}

type portableJavaMapSnapshotEntry struct {
	key      string
	keyValue Value
	value    Value
}

func (m *portableJavaMap) snapshotEntries() []portableJavaMapSnapshotEntry {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	entries := make([]portableJavaMapSnapshotEntry, 0, len(m.values))
	for _, key := range m.keys {
		value, exists := m.values[key]
		if !exists {
			continue
		}
		entries = append(entries, portableJavaMapSnapshotEntry{key: key, keyValue: m.keyValues[key], value: value})
	}
	m.mu.RUnlock()
	return entries
}

func (m *portableJavaMap) putAll(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
	}
	entries, ok := portableMapEntries(invocation.Arg(0))
	if !ok {
		return portableNoMatchingMethod(invocation, "java.util."+m.className()), true, nil
	}
	if len(entries) == 0 {
		return Null(), true, nil
	}
	m.mu.Lock()
	if m.readOnly {
		m.mu.Unlock()
		return Null(), true, errors.New("java.lang.UnsupportedOperationException")
	}
	growth := 0
	for _, entry := range entries {
		if _, exists := m.values[entry.key]; !exists {
			growth++
		}
	}
	if err := reserveCollectionEntries(invocation.Runtime, growth); err != nil {
		m.mu.Unlock()
		return Null(), true, err
	}
	for _, entry := range entries {
		if _, exists := m.values[entry.key]; !exists {
			m.keys = append(m.keys, entry.key)
			m.keyValues[entry.key] = entry.keyValue
			m.entries[entry.key] = &portableJavaMapEntry{
				mapping: m, key: entry.key, keyValue: entry.keyValue, value: entry.value,
			}
			m.mod++
		} else if live := m.entries[entry.key]; live != nil {
			live.mu.Lock()
			live.value = entry.value
			live.mu.Unlock()
		}
		m.values[entry.key] = entry.value
	}
	if m.class == "TreeMap" {
		m.sortKeysLocked()
	}
	m.mu.Unlock()
	return Null(), true, nil
}

func (m *portableJavaMap) equalValue(value Value) bool {
	if object, ok := value.Object(); ok && object == m {
		return true
	}
	other, ok := portableMapEntries(value)
	if !ok {
		return false
	}
	entries := m.snapshotEntries()
	if len(entries) != len(other) {
		return false
	}
	lookup := make(map[string]portableJavaMapSnapshotEntry, len(other))
	for _, entry := range other {
		lookup[entry.key] = entry
	}
	for _, entry := range entries {
		candidate, exists := lookup[entry.key]
		if !exists || !portableJavaEqual(entry.value, candidate.value) {
			return false
		}
	}
	return true
}

func (m *portableJavaMap) javaHashCode() int32 {
	var hash int32
	for _, entry := range m.snapshotEntries() {
		hash += portableJavaValueHash(entry.keyValue) ^ portableJavaValueHash(entry.value)
	}
	return hash
}

func (m *portableJavaMap) removeKeyLocked(key string) (Value, bool) {
	value, exists := m.values[key]
	if !exists {
		return Null(), false
	}
	if entry := m.entries[key]; entry != nil {
		entry.mu.Lock()
		entry.value = value
		entry.mu.Unlock()
	}
	delete(m.values, key)
	delete(m.keyValues, key)
	delete(m.entries, key)
	for index, existing := range m.keys {
		if existing == key {
			copy(m.keys[index:], m.keys[index+1:])
			m.keys[len(m.keys)-1] = ""
			m.keys = m.keys[:len(m.keys)-1]
			break
		}
	}
	m.mod++
	return value, true
}

func (m *portableJavaMap) detachEntriesLocked() {
	for key, entry := range m.entries {
		if entry == nil {
			continue
		}
		if value, exists := m.values[key]; exists {
			entry.mu.Lock()
			entry.value = value
			entry.mu.Unlock()
		}
	}
}

type portableJavaMapViewKind uint8

const (
	portableJavaMapKeySet portableJavaMapViewKind = iota
	portableJavaMapValues
	portableJavaMapEntrySet
)

type portableJavaMapView struct {
	mapping *portableJavaMap
	kind    portableJavaMapViewKind
}

func (view *portableJavaMapView) size() int {
	if view == nil || view.mapping == nil {
		return 0
	}
	view.mapping.mu.RLock()
	size := len(view.mapping.values)
	view.mapping.mu.RUnlock()
	return size
}

func (view *portableJavaMapView) revision() uint64 {
	if view == nil || view.mapping == nil {
		return 0
	}
	view.mapping.mu.RLock()
	revision := view.mapping.mod
	view.mapping.mu.RUnlock()
	return revision
}

func (view *portableJavaMapView) snapshot() []Value {
	if view == nil || view.mapping == nil {
		return nil
	}
	view.mapping.mu.RLock()
	keys := view.mapping.wrapperKeysLocked()
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, view.valueLocked(key))
	}
	view.mapping.mu.RUnlock()
	return values
}

func (view *portableJavaMapView) valueLocked(key string) Value {
	switch view.kind {
	case portableJavaMapKeySet:
		return view.mapping.keyValues[key]
	case portableJavaMapValues:
		return view.mapping.values[key]
	default:
		return ObjectValue(newPortableJavaMapEntryLocked(view.mapping, key))
	}
}

func (view *portableJavaMapView) removeValue(value Value) (bool, error) {
	if view == nil || view.mapping == nil {
		return false, nil
	}
	key := ""
	found := false
	switch view.kind {
	case portableJavaMapKeySet:
		key = sleepCanonicalString(value)
		view.mapping.mu.RLock()
		_, found = view.mapping.values[key]
		view.mapping.mu.RUnlock()
	case portableJavaMapValues:
		for _, candidate := range view.mapping.snapshotEntries() {
			if portableJavaEqual(candidate.value, value) {
				key, found = candidate.key, true
				break
			}
		}
	case portableJavaMapEntrySet:
		if entry, ok := portableJavaMapEntryValue(value); ok {
			key = sleepCanonicalString(entry.keyValue)
			entryValue := entry.currentValue()
			view.mapping.mu.RLock()
			current, exists := view.mapping.values[key]
			view.mapping.mu.RUnlock()
			found = exists && portableJavaEqual(current, entryValue)
		}
	}
	if !found {
		return false, nil
	}
	view.mapping.mu.Lock()
	defer view.mapping.mu.Unlock()
	if view.mapping.readOnly {
		return false, errors.New("java.lang.UnsupportedOperationException")
	}
	_, removed := view.mapping.removeKeyLocked(key)
	return removed, nil
}

func (view *portableJavaMapView) clear() error {
	if view == nil || view.mapping == nil {
		return nil
	}
	view.mapping.mu.Lock()
	if len(view.mapping.values) != 0 {
		if view.mapping.readOnly {
			view.mapping.mu.Unlock()
			return errors.New("java.lang.UnsupportedOperationException")
		}
		view.mapping.detachEntriesLocked()
		view.mapping.keys = nil
		view.mapping.keyValues = make(map[string]Value)
		view.mapping.values = make(map[string]Value)
		view.mapping.entries = make(map[string]*portableJavaMapEntry)
		view.mapping.mod++
	}
	view.mapping.mu.Unlock()
	return nil
}

func (view *portableJavaMapView) iteratorBounds() (int, uint64, error) {
	if view == nil || view.mapping == nil {
		return 0, 0, nil
	}
	view.mapping.mu.RLock()
	size, revision := len(view.mapping.values), view.mapping.mod
	view.mapping.mu.RUnlock()
	return size, revision, nil
}

func (view *portableJavaMapView) iteratorValue(index int, expected uint64) (Value, bool, error) {
	if view == nil || view.mapping == nil {
		return Null(), false, nil
	}
	view.mapping.mu.RLock()
	defer view.mapping.mu.RUnlock()
	if expected != view.mapping.mod {
		return Null(), false, errors.New("java.util.ConcurrentModificationException")
	}
	keys := view.mapping.wrapperKeysLocked()
	if index < 0 || index >= len(keys) {
		return Null(), false, nil
	}
	return view.valueLocked(keys[index]), true, nil
}

func (view *portableJavaMapView) removeAt(index int, expected uint64, checkRevision bool) (Value, error) {
	if view == nil || view.mapping == nil {
		return Null(), errors.New("java.lang.IllegalStateException")
	}
	view.mapping.mu.Lock()
	defer view.mapping.mu.Unlock()
	if checkRevision && expected != view.mapping.mod {
		return Null(), errors.New("java.util.ConcurrentModificationException")
	}
	keys := view.mapping.wrapperKeysLocked()
	if index < 0 || index >= len(keys) {
		return Null(), fmt.Errorf("java.lang.IndexOutOfBoundsException: Index: %d, Size: %d", index, len(keys))
	}
	if view.mapping.readOnly {
		return Null(), errors.New("java.lang.UnsupportedOperationException")
	}
	key := keys[index]
	value := view.valueLocked(key)
	view.mapping.removeKeyLocked(key)
	return value, nil
}

type portableJavaMapEntry struct {
	mu       sync.RWMutex
	mapping  *portableJavaMap
	key      string
	keyValue Value
	value    Value
}

func newPortableJavaMapEntryLocked(mapping *portableJavaMap, key string) *portableJavaMapEntry {
	if entry := mapping.entries[key]; entry != nil {
		return entry
	}
	return &portableJavaMapEntry{mapping: mapping, key: key, keyValue: mapping.keyValues[key], value: mapping.values[key]}
}

func (entry *portableJavaMapEntry) String() string {
	if entry == nil {
		return "null=null"
	}
	return portableJavaValueString(entry.keyValue) + "=" + portableJavaValueString(entry.currentValue())
}

func (entry *portableJavaMapEntry) currentValue() Value {
	if entry == nil {
		return Null()
	}
	if entry.mapping != nil {
		entry.mapping.mu.RLock()
		value, exists := entry.mapping.values[entry.key]
		attached := exists && entry.mapping.entries[entry.key] == entry
		entry.mapping.mu.RUnlock()
		if attached {
			entry.mu.Lock()
			entry.value = value
			entry.mu.Unlock()
			return value
		}
	}
	entry.mu.RLock()
	value := entry.value
	entry.mu.RUnlock()
	return value
}

func (entry *portableJavaMapEntry) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := portableJavaClassName(invocation.Class)
		return Bool(class == "Map$Entry" || class == "Object"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getKey":
		if len(invocation.Arguments) == 0 {
			return entry.keyValue, true, nil
		}
	case "getValue":
		if len(invocation.Arguments) == 0 {
			return entry.currentValue(), true, nil
		}
	case "setValue":
		if len(invocation.Arguments) == 1 {
			value := invocation.Arg(0)
			if entry.mapping != nil {
				entry.mapping.mu.Lock()
				if entry.mapping.entries[entry.key] == entry {
					if entry.mapping.readOnly {
						entry.mapping.mu.Unlock()
						return Null(), true, errors.New("java.lang.UnsupportedOperationException")
					}
					entry.mu.Lock()
					previous := entry.value
					entry.value = value
					entry.mu.Unlock()
					entry.mapping.values[entry.key] = value
					entry.mapping.mu.Unlock()
					return previous, true, nil
				}
				entry.mapping.mu.Unlock()
			}
			entry.mu.Lock()
			previous := entry.value
			entry.value = value
			entry.mu.Unlock()
			return previous, true, nil
		}
	case "equals":
		if len(invocation.Arguments) == 1 {
			other, ok := portableJavaMapEntryValue(invocation.Arg(0))
			return portableJavaBooleanValue(ok && portableJavaEqual(entry.keyValue, other.keyValue) && portableJavaEqual(entry.currentValue(), other.currentValue())), true, nil
		}
	case "hashCode":
		if len(invocation.Arguments) == 0 {
			return Int(portableJavaValueHash(entry.keyValue) ^ portableJavaValueHash(entry.currentValue())), true, nil
		}
	case "toString":
		if len(invocation.Arguments) == 0 {
			return String(entry.String()), true, nil
		}
	}
	return Null(), false, nil
}

func portableJavaMapEntryValue(value Value) (*portableJavaMapEntry, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	entry, ok := object.(*portableJavaMapEntry)
	return entry, ok && entry != nil
}

func portableCollectionTarget(ctx context.Context, target any, invocation ObjectInvocation) (Value, bool, error) {
	switch object := target.(type) {
	case *portableJavaCollection:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaMap:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaIterator:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaMapEntry:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaReverseComparator:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaNaturalComparator:
		if object == nil {
			return Null(), false, nil
		}
		return object.invoke(invocation)
	case *portableJavaReverseComparator2:
		if object == nil {
			return Null(), false, nil
		}
		return object.invokeContext(ctx, invocation)
	}
	return Null(), false, nil
}

func newPortableCollectionObject(class string, invocation ObjectInvocation) (Value, bool, error) {
	if strings.HasSuffix(class, "Map") || class == "Hashtable" {
		if len(invocation.Arguments) > 1 {
			return portableNoMatchingMethod(invocation, "java.util."+class), true, nil
		}
		var entries []portableJavaMapSnapshotEntry
		if len(invocation.Arguments) == 1 {
			var ok bool
			entries, ok = portableMapEntries(invocation.Arg(0))
			if !ok {
				return portableNoMatchingMethod(invocation, "java.util."+class), true, nil
			}
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(entries)); err != nil {
			return Null(), true, err
		}
		return ObjectValue(newPortableJavaMapFromEntries(class, entries)), true, nil
	}
	if len(invocation.Arguments) > 1 {
		return portableNoMatchingMethod(invocation, "java.util."+class), true, nil
	}
	var values []Value
	if len(invocation.Arguments) == 1 {
		var ok bool
		values, ok = portableCollectionValues(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.util."+class), true, nil
		}
	}
	if err := reserveCollectionEntries(invocation.Runtime, portableJavaCollectionEntryCount(class, values)); err != nil {
		return Null(), true, err
	}
	return ObjectValue(newPortableJavaCollection(class, values)), true, nil
}

func portableCollections(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "shuffle":
		return portableCollectionsShuffle(ctx, invocation)
	case "addAll":
		return portableCollectionsAddAll(ctx, invocation)
	case "indexOfSubList", "lastIndexOfSubList":
		return portableCollectionsIndexOfSubList(ctx, invocation)
	case "binarySearch":
		return portableCollectionsBinarySearch(ctx, invocation)
	case "sort":
		return portableCollectionsSort(ctx, invocation)
	case "reverseOrder":
		return portableCollectionsReverseOrder(invocation)
	case "reverse":
		return portableCollectionsReverse(ctx, invocation)
	case "swap":
		return portableCollectionsSwap(ctx, invocation)
	case "fill":
		return portableCollectionsFill(ctx, invocation)
	case "copy":
		return portableCollectionsCopy(ctx, invocation)
	case "rotate":
		return portableCollectionsRotate(ctx, invocation)
	case "replaceAll":
		return portableCollectionsReplaceAll(ctx, invocation)
	case "min", "max":
		return portableCollectionsMinMax(ctx, invocation)
	case "frequency":
		return portableCollectionsFrequency(ctx, invocation)
	case "disjoint":
		return portableCollectionsDisjoint(ctx, invocation)
	case "enumeration":
		return portableCollectionsEnumeration(invocation)
	case "list":
		return portableCollectionsList(ctx, invocation)
	case "nCopies", "singleton", "singletonList", "singletonMap", "emptyList", "emptySet", "emptyMap",
		"emptySortedSet", "emptyNavigableSet", "emptySortedMap", "emptyNavigableMap":
		return portableCollectionsFactory(invocation)
	}
	return Null(), false, nil
}

func portableEnumerationClosureCall(ctx context.Context, value Value, callable Callable, message string) (Value, error) {
	caller := currentFiber(ctx)
	span := Span{Source: "<Java>", Start: Position{Line: -1}}
	var traceFrame *callTraceFrame
	if caller != nil {
		traceFrame = caller.beginCallTrace(formatClosureCall(value, message, nil), span)
	}

	var result Value
	var err error
	if closure, ok := callable.(*scriptClosure); ok && closure != nil {
		result, err = closure.invoke(ctx, []Argument{{Name: "$0", Value: String(message)}})
	} else if target, ok := callable.(interface {
		invokeArguments(context.Context, []Argument) (Value, error)
	}); ok {
		result, err = target.invokeArguments(ctx, []Argument{{Name: "$0", Value: String(message)}})
	} else {
		result, err = callable.Invoke(ctx, String(message))
	}
	if traceFrame != nil {
		caller.finishCallTrace(traceFrame, result, err)
	}

	var thrown *scriptThrow
	if errors.As(err, &thrown) {
		signature := "public abstract boolean java.util.Enumeration.hasMoreElements()"
		if message == "nextElement" {
			signature = "public abstract java.lang.Object java.util.Enumeration.nextElement()"
		}
		thrown.addFrame(fmt.Sprintf("   <Java>:-1 %s as %s", describeTraceValue(value), signature))
	}
	return result, err
}

func portableCollectionsListError(err error) error {
	exception := newPortableJavaException(err)
	if exception != nil {
		exception.frame = "public static java.util.ArrayList java.util.Collections.list(java.util.Enumeration)"
	}
	return exception
}

func portableCollectionValues(value Value) ([]Value, bool) {
	values, ok, _ := portableCollectionValuesChecked(value)
	return values, ok
}

func portableCollectionValuesChecked(value Value) ([]Value, bool, error) {
	if array, ok := value.Array(); ok && array != nil {
		return array.Values(), true, nil
	}
	if object, ok := value.Object(); ok {
		if collection, ok := object.(*portableJavaCollection); ok && collection != nil {
			values, err := collection.snapshotChecked()
			return values, true, err
		}
	}
	return nil, false, nil
}

func portableMapValues(value Value) (*Hash, bool) {
	if hash, ok := value.Hash(); ok && hash != nil {
		return hash, true
	}
	if object, ok := value.Object(); ok {
		if mapping, ok := object.(*portableJavaMap); ok && mapping != nil {
			return mapping.snapshotHash(), true
		}
	}
	return nil, false
}

func portableMapEntries(value Value) ([]portableJavaMapSnapshotEntry, bool) {
	if hash, ok := value.Hash(); ok && hash != nil {
		if hash.backend == nil {
			hash.mu.RLock()
			keys := hash.compatibleKeysLocked()
			entries := make([]portableJavaMapSnapshotEntry, 0, len(keys))
			for _, key := range keys {
				cell := hash.items[key]
				if cell == nil {
					continue
				}
				keyValue := hash.keyValueLocked(key)
				canonical, exactKey := sleepHashKey(keyValue)
				entries = append(entries, portableJavaMapSnapshotEntry{
					key: canonical, keyValue: exactKey, value: cell.Get(),
				})
			}
			hash.mu.RUnlock()
			return entries, true
		}
		entries := make([]portableJavaMapSnapshotEntry, 0, hash.Len())
		for _, keyValue := range hash.KeyValues() {
			entryValue, exists := hash.GetValue(keyValue)
			if !exists {
				continue
			}
			key, exactKey := sleepHashKey(keyValue)
			entries = append(entries, portableJavaMapSnapshotEntry{key: key, keyValue: exactKey, value: entryValue})
		}
		return entries, true
	}
	if object, ok := value.Object(); ok {
		if mapping, ok := object.(*portableJavaMap); ok && mapping != nil {
			return mapping.snapshotEntries(), true
		}
	}
	return nil, false
}

func portableMapEntriesReserved(
	value Value,
	reserve func(int) error,
) ([]portableJavaMapSnapshotEntry, bool, error) {
	entries, ok := portableMapEntries(value)
	if !ok {
		return nil, false, nil
	}
	if reserve != nil {
		if err := reserve(len(entries)); err != nil {
			return nil, true, err
		}
	}
	return entries, true, nil
}

func portableJavaClassName(class string) string {
	if index := strings.LastIndex(class, "."); index >= 0 {
		return class[index+1:]
	}
	return class
}

func portableJavaValueString(value Value) string {
	if value.IsNull() {
		return "null"
	}
	return value.String()
}

func portableJavaContains(values []Value, needle Value) bool {
	for _, value := range values {
		if portableJavaEqual(value, needle) {
			return true
		}
	}
	return false
}

func portableJavaEqual(left, right Value) bool {
	if left.Kind() != right.Kind() {
		return false
	}
	switch left.Kind() {
	case KindNull:
		return true
	case KindInt:
		return left.Int32() == right.Int32()
	case KindLong:
		return left.Int64() == right.Int64()
	case KindDouble:
		return portableJavaDoubleBits(left.Float64()) == portableJavaDoubleBits(right.Float64())
	case KindString:
		return sleepStringValuesEqual(left, right)
	case KindArray, KindHash, KindFunction:
		return left.IdentityEqual(right)
	case KindObject:
		if left.IdentityEqual(right) {
			return true
		}
		leftObject, leftOK := left.Object()
		rightObject, rightOK := right.Object()
		if !leftOK || !rightOK {
			return false
		}
		switch object := leftObject.(type) {
		case *portableJavaCollection:
			return object != nil && object.equalValue(right)
		case *portableJavaMap:
			return object != nil && object.equalValue(right)
		case *portableJavaMapEntry:
			other, ok := rightObject.(*portableJavaMapEntry)
			return object != nil && ok && other != nil && portableJavaEqual(object.keyValue, other.keyValue) && portableJavaEqual(object.currentValue(), other.currentValue())
		case *portableJavaFile:
			other, ok := rightObject.(*portableJavaFile)
			return object != nil && ok && other != nil && portableJavaFileCompareValues(object.pathValue(), other.pathValue()) == 0
		default:
			return false
		}
	default:
		return false
	}
}

func portableJavaValueHash(value Value) int32 {
	switch value.Kind() {
	case KindNull:
		return 0
	case KindInt:
		return value.Int32()
	case KindLong:
		bits := uint64(value.Int64())
		return int32(bits ^ bits>>32)
	case KindDouble:
		bits := portableJavaDoubleBits(value.Float64())
		return int32(bits ^ bits>>32)
	case KindString:
		var hash int32
		for _, unit := range sleepStringUnits(value) {
			hash = 31*hash + int32(unit)
		}
		return hash
	case KindObject:
		object, ok := value.Object()
		if !ok {
			return 0
		}
		switch object := object.(type) {
		case *portableJavaCollection:
			hash, _ := object.javaHashCode()
			return hash
		case *portableJavaMap:
			return object.javaHashCode()
		case *portableJavaMapEntry:
			return portableJavaValueHash(object.keyValue) ^ portableJavaValueHash(object.currentValue())
		case *portableJavaFile:
			return portableJavaFileHashValue(object.pathValue())
		}
	}
	// Arrays and unmodelled opaque objects use a stable collision. Java permits
	// unequal objects to share a hash; equal references therefore still honor
	// the Object.hashCode contract without leaking Go addresses.
	return 0
}

func portableJavaDoubleBits(value float64) uint64 {
	if math.IsNaN(value) {
		return 0x7ff8000000000000
	}
	return math.Float64bits(value)
}

func portableJavaCompare(left, right Value) int {
	if left.Kind() == right.Kind() {
		switch left.Kind() {
		case KindInt, KindLong:
			if left.Int64() < right.Int64() {
				return -1
			}
			if left.Int64() > right.Int64() {
				return 1
			}
			return 0
		case KindDouble:
			if left.Float64() < right.Float64() {
				return -1
			}
			if left.Float64() > right.Float64() {
				return 1
			}
			return 0
		}
	}
	return sleepStringCompareValues(left, right)
}
