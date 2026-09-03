package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Kind identifies the value category stored in a Sleep scalar.
type Kind uint8

const (
	// KindNull identifies Sleep's $null value.
	KindNull Kind = iota
	// KindInt identifies a signed 32-bit integer.
	KindInt
	// KindLong identifies a signed 64-bit integer.
	KindLong
	// KindDouble identifies a double-precision number.
	KindDouble
	// KindString identifies a Sleep string. Sleep strings are sequences of
	// Java UTF-16 code units and may also carry byte provenance when they were
	// produced by a binary API such as pack or readb.
	KindString
	// KindArray identifies a mutable array reference.
	KindArray
	// KindHash identifies a mutable hash reference.
	KindHash
	// KindFunction identifies a callable reference.
	KindFunction
	// KindObject identifies an opaque host object.
	KindObject
)

// String returns a stable name for the value kind.
func (k Kind) String() string {
	switch k {
	case KindNull:
		return "null"
	case KindInt:
		return "int"
	case KindLong:
		return "long"
	case KindDouble:
		return "double"
	case KindString:
		return "string"
	case KindArray:
		return "array"
	case KindHash:
		return "hash"
	case KindFunction:
		return "function"
	case KindObject:
		return "object"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

// Callable is a function value that can be invoked by OPFOR or by an importer.
// Script closures and Go-native callbacks both implement this interface.
//
// Invoke is synchronous and may be called concurrently by independent script
// executions. Implementations should observe ctx and must not retain it after
// returning; asynchronous work should use a context whose lifetime is owned by
// its caller. The argument slice is borrowed for the duration of Invoke. Values
// may be copied, but arrays, hashes, functions, and objects preserve reference
// identity and therefore remain shared with the caller. A returned error is
// authoritative and is propagated by OPFOR.
//
// FunctionValue does not add a script lifecycle guard to an arbitrary
// Callable. Importers that retain a script callback should obtain it through
// Invocation.Callback or Invocation.RetainCallback so calls after unload are
// rejected with ErrScriptUnloaded.
type Callable interface {
	Invoke(context.Context, ...Value) (Value, error)
}

// CallableFunc adapts a function to Callable.
type CallableFunc func(context.Context, ...Value) (Value, error)

// Invoke calls f with ctx and values. A nil CallableFunc returns
// ErrInvalidCallable.
func (f CallableFunc) Invoke(ctx context.Context, values ...Value) (Value, error) {
	if f == nil {
		return Null(), ErrInvalidCallable
	}
	return f(ctx, values...)
}

// Value is OPFOR's host-facing representation of a Sleep scalar. Its zero value
// is the Sleep $null value. Scalar values copy by value; arrays, hashes,
// functions, and objects preserve reference identity.
type Value struct {
	kind        Kind
	data        any
	tainted     bool
	stringUnits []uint16
	stringRaw   []bool
}

// Null returns Sleep's $null value.
func Null() Value { return Value{} }

// Bool returns Sleep's canonical boolean representation: integer 1 for true
// and $null for false.
func Bool(v bool) Value {
	if v {
		return Int(1)
	}
	return Null()
}

// Int returns a Sleep 32-bit integer value.
func Int(v int32) Value { return Value{kind: KindInt, data: v} }

// Long returns a Sleep 64-bit integer value.
func Long(v int64) Value { return Value{kind: KindLong, data: v} }

// Double returns a Sleep double-precision value.
func Double(v float64) Value { return Value{kind: KindDouble, data: v} }

// String returns a textual Sleep string. Valid UTF-8 is mapped to the same
// UTF-16 code units a Java String would contain. Invalid UTF-8 bytes retain the
// historical OPFOR byte-string behavior; new code with binary data should use
// BinaryString so valid UTF-8 byte sequences remain distinguishable from text.
func String(v string) Value { return newSleepString(v) }

// BinaryString returns a Sleep string whose code units are the unsigned values
// of data's bytes. The byte provenance is retained through concatenation,
// interpolation, slicing, and other string operations.
func BinaryString(data []byte) Value { return newSleepBinaryString(data) }

// ArrayValue wraps an array reference. A nil array becomes $null.
func ArrayValue(v *Array) Value {
	if v == nil {
		return Null()
	}
	return Value{kind: KindArray, data: v}
}

// HashValue wraps a hash reference. A nil hash becomes $null.
func HashValue(v *Hash) Value {
	if v == nil {
		return Null()
	}
	return Value{kind: KindHash, data: v}
}

// FunctionValue wraps a callable reference. A nil callable becomes $null.
func FunctionValue(v Callable) Value {
	if isNilInterface(v) {
		return Null()
	}
	return Value{kind: KindFunction, data: v}
}

// ObjectValue wraps an opaque host object. A nil object becomes $null.
func ObjectValue(v any) Value {
	if v == nil {
		return Null()
	}
	return Value{kind: KindObject, data: v}
}

// Kind returns the value's category.
func (v Value) Kind() Kind { return v.kind }

// IsTainted reports whether this scalar carries Sleep's taint marker. For
// arrays and hashes it also inspects their elements recursively, honoring
// cycles. Taint is metadata: it never changes a value's kind, coercions,
// identity, string representation, or host payload.
func (v Value) IsTainted() bool { return valueIsTainted(v) }

// IsNull reports whether v is Sleep's $null value.
func (v Value) IsNull() bool { return v.kind == KindNull }

// Truth implements Sleep truthiness. $null, an empty string, and any scalar
// whose string representation is exactly "0" are false. Arrays and hashes are
// true, including empty ones.
func (v Value) Truth() bool {
	switch v.kind {
	case KindNull:
		return false
	case KindInt:
		return v.data.(int32) != 0
	case KindLong:
		return v.data.(int64) != 0
	case KindDouble:
		return v.data.(float64) != 0
	case KindString:
		s := v.data.(string)
		return s != "" && s != "0"
	case KindArray, KindHash:
		return true
	default:
		return v.String() != ""
	}
}

// String returns Sleep's normal string coercion. Use Describe when a typed,
// inspectable representation is needed.
func (v Value) String() string {
	switch v.kind {
	case KindNull:
		return ""
	case KindInt:
		return strconv.FormatInt(int64(v.data.(int32)), 10)
	case KindLong:
		return strconv.FormatInt(v.data.(int64), 10)
	case KindDouble:
		return formatDouble(v.data.(float64))
	case KindString:
		return v.data.(string)
	case KindArray, KindHash:
		return v.Describe()
	case KindFunction:
		return fmt.Sprint(v.data)
	case KindObject:
		return fmt.Sprint(v.data)
	default:
		return ""
	}
}

// Bytes returns the Go byte spelling exposed by String and reports whether v
// is a string. For values constructed with BinaryString these are the original
// octets, including when they also form valid UTF-8.
func (v Value) Bytes() ([]byte, bool) {
	if v.kind != KindString {
		return nil, false
	}
	return append([]byte(nil), v.data.(string)...), true
}

// IsBinaryString reports whether any code unit in v retains byte provenance
// from BinaryString or an OPFOR binary-producing API.
func (v Value) IsBinaryString() bool {
	if v.kind != KindString {
		return false
	}
	for _, raw := range v.stringRaw {
		if raw {
			return true
		}
	}
	return false
}

// Describe returns the representation Sleep uses for diagnostics and compound
// values, including cycle references such as @0 and %0.
func (v Value) Describe() string {
	state := describeState{seen: make(map[any]int)}
	return state.value(v)
}

type describeState struct {
	seen map[any]int
	next int
}

func (s *describeState) value(v Value) string {
	switch v.kind {
	case KindNull:
		return "$null"
	case KindInt:
		return strconv.FormatInt(int64(v.data.(int32)), 10)
	case KindLong:
		return strconv.FormatInt(v.data.(int64), 10) + "L"
	case KindDouble:
		return formatDouble(v.data.(float64))
	case KindString:
		return "'" + sleepCanonicalString(v) + "'"
	case KindArray:
		return s.array(v.data.(*Array))
	case KindHash:
		return s.hash(v.data.(*Hash))
	case KindObject:
		if describer, ok := v.data.(interface{ SleepDescribe() string }); ok {
			return describer.SleepDescribe()
		}
		return v.String()
	case KindFunction:
		return v.String()
	default:
		return "$null"
	}
}

func formatDouble(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	}

	// DoubleValue.toString delegates to java.lang.Double.toString. Java uses
	// ordinary notation for magnitudes in [10^-3, 10^7), scientific notation
	// outside that range, and an upper-case E without an explicit plus sign.
	// strconv supplies the shortest round-trippable significand; normalize its
	// presentation here so binary-unpacked and calculated doubles retain the
	// spelling Sleep scripts observe on the JVM.
	magnitude := math.Abs(value)
	if magnitude != 0 && (magnitude < 1e-3 || magnitude >= 1e7) {
		text := strconv.FormatFloat(value, 'e', -1, 64)
		separator := strings.IndexByte(text, 'e')
		mantissa, exponent := text[:separator], text[separator+1:]
		if !strings.Contains(mantissa, ".") {
			mantissa += ".0"
		}
		parsedExponent, err := strconv.Atoi(exponent)
		if err == nil {
			exponent = strconv.Itoa(parsedExponent)
		}
		return mantissa + "E" + exponent
	}

	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(text, ".") {
		text += ".0"
	}
	return text
}

func (s *describeState) array(array *Array) string {
	if index, ok := s.seen[array]; ok {
		return "@" + strconv.Itoa(index)
	}
	s.seen[array] = s.next
	s.next++

	values := array.Values()
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = s.value(value)
	}
	return "@(" + strings.Join(parts, ", ") + ")"
}

func (s *describeState) hash(hash *Hash) string {
	if index, ok := s.seen[hash]; ok {
		return "%" + strconv.Itoa(index)
	}
	s.seen[hash] = s.next
	s.next++

	keys := hash.KeyValues()
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := hash.GetValue(key)
		if !ok || value.IsNull() {
			continue
		}
		parts = append(parts, key.String()+" => "+s.value(value))
	}
	return "%(" + strings.Join(parts, ", ") + ")"
}

// Int32 returns v coerced to a 32-bit integer.
func (v Value) Int32() int32 {
	switch v.kind {
	case KindInt:
		return v.data.(int32)
	case KindLong:
		return int32(v.data.(int64))
	case KindDouble:
		return int32(v.data.(float64))
	case KindString:
		if parsed, err := strconv.ParseInt(v.data.(string), 0, 32); err == nil {
			return int32(parsed)
		}
		if parsed, err := strconv.ParseFloat(v.data.(string), 64); err == nil {
			return int32(parsed)
		}
	}
	return 0
}

// Int64 returns v coerced to a 64-bit integer.
func (v Value) Int64() int64 {
	switch v.kind {
	case KindInt:
		return int64(v.data.(int32))
	case KindLong:
		return v.data.(int64)
	case KindDouble:
		return int64(v.data.(float64))
	case KindString:
		if parsed, err := strconv.ParseInt(v.data.(string), 0, 64); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(v.data.(string), 64); err == nil {
			return int64(parsed)
		}
	}
	return 0
}

// Float64 returns v coerced to a double-precision number.
func (v Value) Float64() float64 {
	switch v.kind {
	case KindInt:
		return float64(v.data.(int32))
	case KindLong:
		return float64(v.data.(int64))
	case KindDouble:
		return v.data.(float64)
	case KindString:
		parsed, _ := strconv.ParseFloat(v.data.(string), 64)
		return parsed
	default:
		return 0
	}
}

// Array returns the referenced array and whether v contains one.
func (v Value) Array() (*Array, bool) {
	if v.kind != KindArray {
		return nil, false
	}
	return v.data.(*Array), true
}

// Hash returns the referenced hash and whether v contains one.
func (v Value) Hash() (*Hash, bool) {
	if v.kind != KindHash {
		return nil, false
	}
	return v.data.(*Hash), true
}

// Function returns the referenced callable and whether v contains one.
func (v Value) Function() (Callable, bool) {
	if v.kind != KindFunction {
		return nil, false
	}
	callable, ok := v.data.(Callable)
	if !ok || isNilInterface(callable) {
		return nil, false
	}
	return callable, true
}

// Object returns the wrapped host object and whether v contains one.
func (v Value) Object() (any, bool) {
	if v.kind != KindObject {
		return nil, false
	}
	return v.data, true
}

// IdentityEqual implements Sleep reference identity for compound values and
// type-and-value identity for scalar values.
func (v Value) IdentityEqual(other Value) bool {
	if v.kind != other.kind {
		return false
	}
	switch v.kind {
	case KindNull:
		return true
	case KindInt:
		return v.data.(int32) == other.data.(int32)
	case KindLong:
		return v.data.(int64) == other.data.(int64)
	case KindDouble:
		return v.data.(float64) == other.data.(float64)
	case KindString:
		return sleepStringValuesEqual(v, other)
	case KindArray:
		return v.data.(*Array) == other.data.(*Array)
	case KindHash:
		return v.data.(*Hash) == other.data.(*Hash)
	case KindFunction:
		return comparableEqual(v.data, other.data)
	case KindObject:
		return comparableEqual(v.data, other.data)
	default:
		return false
	}
}

func comparableEqual(left, right any) (equal bool) {
	defer func() {
		if recover() != nil {
			equal = false
		}
	}()
	return left == right
}

// Cell is a mutable scalar slot. It is exported for host adapters that need to
// honor Sleep pass-by-name arguments while keeping mutation synchronized.
type Cell struct {
	mu      sync.RWMutex
	value   Value
	watcher func(Value, Span)
}

// NewCell returns a mutable cell initialized to value.
func NewCell(value Value) *Cell { return &Cell{value: value} }

// Get returns the cell's current value. A nil cell reads as $null.
func (c *Cell) Get() Value {
	if c == nil {
		return Null()
	}
	c.mu.RLock()
	value := c.value
	c.mu.RUnlock()
	return value
}

// Set replaces the cell's value. Setting a nil cell has no effect.
func (c *Cell) Set(value Value) {
	c.SetAt(value, Span{})
}

// SetAt replaces the cell's value and supplies the source location to an
// installed Sleep watch(). Ordinary host-side writes use Set and therefore
// retain an empty source span.
func (c *Cell) SetAt(value Value, span Span) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.value = value
	watcher := c.watcher
	c.mu.Unlock()
	if watcher != nil {
		watcher(value, span)
	}
}

// setTaintValue changes only taint metadata without notifying watch(). Sleep's
// WatchScalar deliberately ignores the TaintedValue wrapper operation itself;
// an ordinary assignment of a tainted result still goes through SetAt.
func (c *Cell) setTaintValue(value Value) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.value = value
	c.mu.Unlock()
}

func (c *Cell) setWatcher(watcher func(Value, Span)) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.watcher = watcher
	c.mu.Unlock()
}

// ErrIndexOutOfRange reports an invalid array index or range.
var ErrIndexOutOfRange = errors.New("opfor: index out of range")

const unsafeArrayViewWarning = "unsafe data modification: parent @array changed after &sublist creation"

// ErrUnsafeArrayView reports an operation on a sublist after its backing array
// was structurally changed through the root or another view. Sleep surfaces
// the same condition as "parent @array changed after &sublist creation".
var ErrUnsafeArrayView = errors.New("opfor: " + unsafeArrayViewWarning)

// ErrArrayChangedDuringIteration reports a structural array mutation made
// outside the active foreach iterator.
var ErrArrayChangedDuringIteration = errors.New("opfor: @array changed during iteration")

const readOnlyArrayWarning = "array is read-only"

// ErrReadOnlyArray reports an attempted mutation of a read-only array wrapper.
// Sleep uses these wrappers for Java collections returned by functions such as
// ls() and SleepUtils.getArrayWrapper().
var ErrReadOnlyArray = errors.New("opfor: " + readOnlyArrayWarning)

type arrayStorage struct {
	mu       sync.RWMutex
	items    []*Cell
	root     *arrayWindow
	views    map[*arrayWindow]struct{}
	modCount uint64
	readOnly bool
}

type arrayWindow struct {
	start  int
	end    int
	valid  bool
	cached []*Cell
}

// readOnlyArrayBackend supplies a non-owning ScalarArray view. The ordinary
// Array storage above remains the mutable Sleep ListContainer implementation;
// adapters such as CollectionWrapper use this boundary so their live backing
// semantics are not approximated by repeatedly replacing storage snapshots.
type readOnlyArrayBackend interface {
	len() int
	snapshotCells() []*Cell
	cell(int) (*Cell, bool)
	cellContext(int) (*Cell, bool, error)
	sublist(int, int) (*Array, error)
	iterator(Value) valueIterator
	taintAll()
}

// Array is Sleep's mutable, reference-identity array container.
type Array struct {
	once    sync.Once
	storage *arrayStorage
	window  *arrayWindow
	backend readOnlyArrayBackend
}

// NewArray returns a mutable array containing values in argument order.
func NewArray(values ...Value) *Array {
	cells := make([]*Cell, len(values))
	for index, value := range values {
		cells[index] = NewCell(value)
	}
	return newArrayFromCells(cells)
}

// newArrayFromCells preserves scalar identity between positional parameters
// and @_. Sleep stores the exact argument Scalars in both locations.
func newArrayFromCells(cells []*Cell) *Array {
	array := &Array{}
	array.initializeArray(cells)
	return array
}

// NewReadOnlyArray returns an array whose elements can be inspected and
// iterated but whose structure and indexed values cannot be changed.
func NewReadOnlyArray(values ...Value) *Array {
	array := NewArray(values...)
	storage, _ := array.arrayStorage()
	storage.mu.Lock()
	storage.readOnly = true
	storage.mu.Unlock()
	return array
}

func (a *Array) initializeArray(cells []*Cell) {
	if a == nil || a.backend != nil {
		return
	}
	a.once.Do(func() {
		cells = append([]*Cell(nil), cells...)
		storage := &arrayStorage{
			items: cells,
			views: make(map[*arrayWindow]struct{}),
		}
		window := &arrayWindow{end: len(cells), valid: true, cached: append([]*Cell(nil), cells...)}
		storage.root = window
		storage.views[window] = struct{}{}
		a.storage = storage
		a.window = window
	})
}

func (a *Array) arrayStorage() (*arrayStorage, *arrayWindow) {
	if a == nil || a.backend != nil {
		return nil, nil
	}
	a.initializeArray(nil)
	return a.storage, a.window
}

func unsafeArrayViewError() error { return ErrUnsafeArrayView }

func (a *Array) snapshotCells() ([]*Cell, error) {
	if a != nil && a.backend != nil {
		return a.backend.snapshotCells(), nil
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return nil, nil
	}
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	if !window.valid {
		return nil, unsafeArrayViewError()
	}
	return append([]*Cell(nil), storage.items[window.start:window.end]...), nil
}

func (a *Array) cachedCells() []*Cell {
	if a != nil && a.backend != nil {
		return a.backend.snapshotCells()
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return nil
	}
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	return append([]*Cell(nil), window.cached...)
}

func (a *Array) viewError() error {
	if a != nil && a.backend != nil {
		return nil
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return nil
	}
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	if !window.valid {
		return unsafeArrayViewError()
	}
	return nil
}

func (a *Array) isReadOnly() bool {
	if a != nil && a.backend != nil {
		return true
	}
	storage, _ := a.arrayStorage()
	if storage == nil {
		return false
	}
	storage.mu.RLock()
	readOnly := storage.readOnly
	storage.mu.RUnlock()
	return readOnly
}

func (storage *arrayStorage) syncCachesLocked() {
	for window := range storage.views {
		if !window.valid {
			continue
		}
		window.cached = append(window.cached[:0], storage.items[window.start:window.end]...)
		clear(window.cached[len(window.cached):cap(window.cached)])
	}
}

func sameCellSequence(left, right []*Cell) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// mutateCells replaces this array or view's structural contents atomically.
// A structural mutation keeps the root and the writing view live and
// invalidates every other sublist, matching MyLinkedList's modCount contract.
func (a *Array) mutateCells(structural bool, mutate func([]*Cell) ([]*Cell, error)) error {
	if a != nil && a.backend != nil {
		return ErrReadOnlyArray
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return ErrIndexOutOfRange
	}
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return mutateArrayCellsLocked(storage, window, structural, mutate)
}

// mutateArrayCellsLocked applies one array mutation while storage.mu is held.
// Keeping the actual commit separate from lock acquisition lets evaluator
// mutations coordinate the container lock with Script unload without changing
// the direct host-facing Array methods.
func mutateArrayCellsLocked(
	storage *arrayStorage,
	window *arrayWindow,
	structural bool,
	mutate func([]*Cell) ([]*Cell, error),
) error {
	if !window.valid {
		return unsafeArrayViewError()
	}
	if storage.readOnly {
		return ErrReadOnlyArray
	}

	current := append([]*Cell(nil), storage.items[window.start:window.end]...)
	replacement, err := mutate(current)
	if err != nil {
		return err
	}
	if !structural && len(replacement) != len(current) {
		return errors.New("opfor: non-structural array mutation changed its size")
	}
	structural = structural && !sameCellSequence(current, replacement)

	root := make([]*Cell, 0, len(storage.items)-len(current)+len(replacement))
	root = append(root, storage.items[:window.start]...)
	root = append(root, replacement...)
	root = append(root, storage.items[window.end:]...)
	storage.items = root
	if structural {
		storage.modCount++
		for candidate := range storage.views {
			if candidate != storage.root && candidate != window {
				candidate.valid = false
			}
		}
	}
	window.end = window.start + len(replacement)
	storage.root.start = 0
	storage.root.end = len(root)
	storage.root.valid = true
	storage.syncCachesLocked()
	return nil
}

func (a *Array) sublist(start, end int) (*Array, error) {
	if a != nil && a.backend != nil {
		return a.backend.sublist(start, end)
	}
	storage, parent := a.arrayStorage()
	if storage == nil || parent == nil {
		return nil, ErrIndexOutOfRange
	}
	storage.mu.Lock()
	defer storage.mu.Unlock()
	return sublistArrayLocked(storage, parent, start, end)
}

func sublistArrayLocked(storage *arrayStorage, parent *arrayWindow, start, end int) (*Array, error) {
	if !parent.valid {
		return nil, unsafeArrayViewError()
	}
	length := parent.end - parent.start
	if start < 0 || end < start || end > length {
		return nil, ErrIndexOutOfRange
	}

	view := &Array{}
	view.once.Do(func() {
		window := &arrayWindow{
			start:  parent.start + start,
			end:    parent.start + end,
			valid:  true,
			cached: append([]*Cell(nil), storage.items[parent.start+start:parent.start+end]...),
		}
		view.storage = storage
		view.window = window
		storage.views[window] = struct{}{}
		goruntime.SetFinalizer(view, func(*Array) {
			storage.mu.Lock()
			delete(storage.views, window)
			storage.mu.Unlock()
		})
	})
	return view, nil
}

func valuesFromCells(cells []*Cell) []Value {
	values := make([]Value, len(cells))
	for index, cell := range cells {
		values[index] = cell.Get()
	}
	return values
}

// Len returns the number of elements. A nil array has length zero. For an
// invalidated sublist view, Len reports the size of its last valid snapshot.
func (a *Array) Len() int {
	if a == nil {
		return 0
	}
	if a.backend != nil {
		return a.backend.len()
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return 0
	}
	storage.mu.RLock()
	if !window.valid {
		length := len(window.cached)
		storage.mu.RUnlock()
		return length
	}
	length := window.end - window.start
	storage.mu.RUnlock()
	return length
}

func normalizeArrayIndex(index, length int) (int, bool) {
	if index < 0 {
		if length == 0 {
			return 0, false
		}
		index %= length
		if index < 0 {
			index += length
		}
	}
	return index, index >= 0 && index < length
}

// Cell returns the mutable cell at index. Negative indices address from the
// end. It reports false for a nil array, an invalidated view, or a missing
// index.
func (a *Array) Cell(index int) (*Cell, bool) {
	if a == nil {
		return nil, false
	}
	if a.backend != nil {
		return a.backend.cell(index)
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return nil, false
	}
	storage.mu.RLock()
	if !window.valid {
		storage.mu.RUnlock()
		return nil, false
	}
	index, ok := normalizeArrayIndex(index, window.end-window.start)
	if !ok {
		storage.mu.RUnlock()
		return nil, false
	}
	cell := storage.items[window.start+index]
	storage.mu.RUnlock()
	return cell, true
}

// Get returns the value at index using Cell's negative-index behavior.
func (a *Array) Get(index int) (Value, bool) {
	if a == nil {
		return Null(), false
	}
	if a.backend != nil {
		cell, ok := a.backend.cell(index)
		if !ok {
			return Null(), false
		}
		return cell.Get(), true
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return Null(), false
	}
	storage.mu.RLock()
	if !window.valid {
		storage.mu.RUnlock()
		return Null(), false
	}
	index, ok := normalizeArrayIndex(index, window.end-window.start)
	if !ok {
		storage.mu.RUnlock()
		return Null(), false
	}
	cell := storage.items[window.start+index]
	storage.mu.RUnlock()
	return cell.Get(), true
}

// Set replaces an existing array element. Negative indices address from the
// end. Use Append or Ensure when growth is intended.
func (a *Array) Set(index int, value Value) error {
	if a != nil && a.backend != nil {
		cell, ok := a.backend.cell(index)
		if !ok {
			return ErrIndexOutOfRange
		}
		// CollectionWrapper.getAt constructs a fresh Scalar around the cached
		// Java object on every access. Assignment therefore succeeds against a
		// detached cell and cannot mutate either the cache or the collection.
		cell.Set(value)
		return nil
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return ErrIndexOutOfRange
	}
	storage.mu.RLock()
	if storage.readOnly {
		storage.mu.RUnlock()
		return ErrReadOnlyArray
	}
	if !window.valid {
		storage.mu.RUnlock()
		return ErrIndexOutOfRange
	}
	index, ok := normalizeArrayIndex(index, window.end-window.start)
	if !ok {
		storage.mu.RUnlock()
		return ErrIndexOutOfRange
	}
	cell := storage.items[window.start+index]
	storage.mu.RUnlock()
	cell.Set(value)
	return nil
}

// Ensure returns the cell at index, extending the array with $null cells for a
// non-negative out-of-range index.
func (a *Array) Ensure(index int) (*Cell, error) {
	if a == nil || index < 0 {
		return nil, ErrIndexOutOfRange
	}
	if a.backend != nil {
		if cell, ok := a.backend.cell(index); ok {
			return cell, nil
		}
		return nil, ErrIndexOutOfRange
	}
	var ensured *Cell
	err := a.mutateCells(true, func(cells []*Cell) ([]*Cell, error) {
		for len(cells) <= index {
			cells = append(cells, NewCell(Null()))
		}
		ensured = cells[index]
		return cells, nil
	})
	return ensured, err
}

// Append adds values to the end of the array. It has no effect on a nil array
// or an invalidated sublist view.
func (a *Array) Append(values ...Value) {
	if a == nil {
		return
	}
	_ = a.appendValues(values...)
}

func (a *Array) appendValues(values ...Value) error {
	if a == nil {
		return ErrIndexOutOfRange
	}
	if a.backend != nil {
		return ErrReadOnlyArray
	}
	if len(values) == 0 {
		return nil
	}
	storage, window := a.arrayStorage()
	if storage == nil || window == nil {
		return ErrIndexOutOfRange
	}
	storage.mu.Lock()
	if canAppendRootArrayLocked(storage, window) {
		err := appendRootArrayValuesLocked(storage, values)
		storage.mu.Unlock()
		return err
	}
	storage.mu.Unlock()
	return a.mutateCells(true, func(cells []*Cell) ([]*Cell, error) {
		for _, value := range values {
			cells = append(cells, NewCell(value))
		}
		return cells, nil
	})
}

// canAppendRootArrayLocked reports whether an append can grow storage in
// place. Once a sublist exists, appends must use mutateArrayCellsLocked so the
// MyLinkedList-compatible view invalidation rules remain centralized there.
// The caller must hold storage.mu.
func canAppendRootArrayLocked(storage *arrayStorage, window *arrayWindow) bool {
	return storage != nil && window != nil && window == storage.root &&
		window.valid && len(storage.views) == 1
}

// appendRootArrayValuesLocked is the allocation-linear root-only append path.
// Root cached cells can share the storage slice because the root is never
// invalidated. A later general mutation refreshes and clears that cache through
// syncCachesLocked, retaining the existing slice-retention guarantees.
func appendRootArrayValuesLocked(storage *arrayStorage, values []Value) error {
	if storage.readOnly {
		return ErrReadOnlyArray
	}
	for _, value := range values {
		storage.items = append(storage.items, NewCell(value))
	}
	storage.modCount++
	storage.root.start = 0
	storage.root.end = len(storage.items)
	storage.root.valid = true
	storage.root.cached = storage.items
	return nil
}

// Values returns a detached value snapshot. For an invalidated sublist view,
// it returns the view's last valid snapshot.
func (a *Array) Values() []Value {
	if a == nil {
		return nil
	}
	cells, err := a.snapshotCells()
	if err != nil {
		cells = a.cachedCells()
	}
	return valuesFromCells(cells)
}

const readOnlyHashWarning = "hash is read-only"

// ErrReadOnlyHash reports a structural mutation of a read-only hash wrapper.
// Sleep uses these wrappers for maps returned by functions such as
// systemProperties(). Indexed writes target detached scalars and are silent
// no-ops; remove operations surface this error.
var ErrReadOnlyHash = errors.New("opfor: " + readOnlyHashWarning)

type hashBackendEntry struct {
	key   Value
	value Value
}

// readOnlyHashBackend supplies a live, non-owning ScalarHash view. dataSnapshot
// is intentionally explicit: MapWrapper.getData is the source-defined point
// where Sleep materializes a detached HashMap for foreach/value traversal.
type readOnlyHashBackend interface {
	len() int
	cell(Value) (*Cell, bool)
	get(Value) (Value, bool)
	ensure(Value) *Cell
	keyValues() []Value
	keysArray(*runtimeResourceAccount) (*Array, error)
	dataSnapshot() *Hash
	dataSnapshotReserved(func(int) error) (*Hash, error)
	taintAll()
}

// Hash is Sleep's string-keyed dictionary container. Ordinary hashes reproduce
// Sleep 2.1's deterministic Java 7 HashMap traversal; ordered hashes preserve
// insertion order. NewReadOnlyHash models Sleep's MapWrapper boundary.
type Hash struct {
	mu            sync.RWMutex
	items         map[string]*Cell
	keyValues     map[string]Value
	order         []string
	modCount      uint64
	ordered       bool
	accessOrdered bool
	readOnly      bool
	shouldClean   bool
	missPolicy    Callable
	removalPolicy Callable
	backend       readOnlyHashBackend
}

// NewHash creates an ordinary hash with Sleep-compatible key traversal order.
func NewHash() *Hash {
	return &Hash{items: make(map[string]*Cell), keyValues: make(map[string]Value)}
}

// NewOrderedHash creates Sleep's insertion-ordered hash variant (ohash).
func NewOrderedHash() *Hash {
	return &Hash{items: make(map[string]*Cell), keyValues: make(map[string]Value), ordered: true}
}

// NewAccessOrderedHash creates Sleep's access-ordered hash variant (ohasha).
// Reading or replacing an existing entry moves it to the end of iteration
// order. Script-level miss and removal policies are installed separately.
func NewAccessOrderedHash() *Hash {
	return &Hash{
		items:         make(map[string]*Cell),
		keyValues:     make(map[string]Value),
		ordered:       true,
		accessOrdered: true,
	}
}

// NewReadOnlyHash returns a deterministic detached map wrapper. Keys and
// values can be inspected and iterated, while indexed writes affect only a
// detached scalar and structural removals are rejected. Input values are
// copied into the new hash in lexical key order.
func NewReadOnlyHash(values map[string]Value) *Hash {
	hash := NewHash()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		hash.Set(key, values[key])
	}
	hash.readOnly = true
	return hash
}

func (h *Hash) isReadOnly() bool {
	if h == nil {
		return false
	}
	if h.backend != nil {
		return true
	}
	h.mu.RLock()
	readOnly := h.readOnly
	h.mu.RUnlock()
	return readOnly
}

// Len returns the number of keys. A nil hash has length zero.
func (h *Hash) Len() int {
	if h == nil {
		return 0
	}
	if h.backend != nil {
		return h.backend.len()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.items)
}

// Cell performs a non-autovivifying lookup of key. Read-only wrappers return a
// detached cell so callers cannot mutate their backing values.
func (h *Hash) Cell(key string) (*Cell, bool) {
	return h.CellValue(String(key))
}

// CellValue performs a non-autovivifying lookup using the exact Java UTF-16
// identity of key. Read-only wrappers return a detached cell.
func (h *Hash) CellValue(key Value) (*Cell, bool) {
	if h == nil {
		return nil, false
	}
	if h.backend != nil {
		return h.backend.cell(key)
	}
	keyText := sleepCanonicalString(key)
	h.mu.RLock()
	defer h.mu.RUnlock()
	cell, ok := h.items[keyText]
	if ok && h.readOnly {
		cell = NewCell(cell.Get())
	}
	return cell, ok
}

// Get performs a non-autovivifying value lookup of key.
func (h *Hash) Get(key string) (Value, bool) {
	return h.GetValue(String(key))
}

// GetValue performs a non-autovivifying lookup using the exact Java UTF-16
// identity of key.
func (h *Hash) GetValue(key Value) (Value, bool) {
	if h == nil {
		return Null(), false
	}
	if h.backend != nil {
		return h.backend.get(key)
	}
	keyText := sleepCanonicalString(key)
	h.mu.Lock()
	cell, ok := h.items[keyText]
	if ok && h.accessOrdered && h.moveToEndLocked(keyText) {
		h.modCount++
	}
	h.mu.Unlock()
	if !ok {
		return Null(), false
	}
	return cell.Get(), true
}

// Ensure returns the key's cell, inserting a $null cell when absent. This is
// the autovivifying lookup used by ordinary Sleep hash indexing. A read-only
// wrapper instead returns a detached copy (or detached $null for a miss), so
// indexed assignment cannot change the wrapper or add a key.
func (h *Hash) Ensure(key string) *Cell {
	return h.EnsureValue(String(key))
}

// EnsureValue returns the key's cell, inserting a $null cell when absent. The
// first spelling and byte-provenance metadata for a Java-equal key are retained
// for enumeration, policy callbacks, and serialization.
func (h *Hash) EnsureValue(key Value) *Cell {
	if h == nil {
		return nil
	}
	if h.backend != nil {
		return h.backend.ensure(key)
	}
	keyText, keyValue := sleepHashKey(key)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.readOnly {
		if cell, ok := h.items[keyText]; ok {
			return NewCell(cell.Get())
		}
		return NewCell(Null())
	}
	if cell, ok := h.items[keyText]; ok {
		if h.accessOrdered && h.moveToEndLocked(keyText) {
			h.modCount++
		}
		return cell
	}
	cell := NewCell(Null())
	h.items[keyText] = cell
	h.rememberKeyLocked(keyText, keyValue)
	h.order = append(h.order, keyText)
	h.modCount++
	return cell
}

// Set inserts or replaces key. Setting a nil or read-only hash has no effect.
func (h *Hash) Set(key string, value Value) { h.Ensure(key).Set(value) }

// SetValue inserts or replaces a key using its exact Java UTF-16 identity and
// retains its spelling/provenance metadata when it is first inserted.
func (h *Hash) SetValue(key, value Value) { h.EnsureValue(key).Set(value) }

// Delete removes key and reports whether it was present.
func (h *Hash) Delete(key string) bool {
	return h.DeleteValue(String(key))
}

// DeleteValue removes a key using its exact Java UTF-16 identity and reports
// whether it was present.
func (h *Hash) DeleteValue(key Value) bool {
	if h == nil {
		return false
	}
	if h.backend != nil {
		return false
	}
	keyText := sleepCanonicalString(key)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.readOnly {
		return false
	}
	if _, ok := h.items[keyText]; !ok {
		return false
	}
	delete(h.items, keyText)
	delete(h.keyValues, keyText)
	for index, candidate := range h.order {
		if candidate == keyText {
			h.order = append(h.order[:index], h.order[index+1:]...)
			break
		}
	}
	h.modCount++
	return true
}

func (h *Hash) moveToEndLocked(key string) bool {
	if len(h.order) < 2 || h.order[len(h.order)-1] == key {
		return false
	}
	for index, candidate := range h.order {
		if candidate == key {
			copy(h.order[index:], h.order[index+1:])
			h.order[len(h.order)-1] = key
			return true
		}
	}
	return false
}

// Keys returns a detached snapshot in the hash's Sleep-compatible traversal
// order.
func (h *Hash) Keys() []string {
	if h == nil {
		return nil
	}
	if h.backend != nil {
		keys := h.backend.keyValues()
		result := make([]string, len(keys))
		for index, key := range keys {
			result[index] = sleepCanonicalString(key)
		}
		return result
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.compatibleKeysLocked()
}

// KeyValues returns the exact stored key values in Sleep-compatible traversal
// order. Unlike Keys, it preserves unpaired UTF-16 units and OPFOR's per-unit
// byte provenance, so script-facing enumeration and interop should use it.
func (h *Hash) KeyValues() []Value {
	if h == nil {
		return nil
	}
	if h.backend != nil {
		return h.backend.keyValues()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	keys := h.compatibleKeysLocked()
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, h.keyValueLocked(key))
	}
	return values
}

// compatibleKeysLocked reproduces the deterministic bucket traversal used by
// the Java 7 HashMap backing Sleep 2.1 hashes. Ordered hashes retain insertion
// order. The canonical Sleep golden corpus was produced with this iteration
// behavior, so using Go's randomized map order would not be compatible.
func (h *Hash) compatibleKeysLocked() []string {
	if h.ordered {
		return append([]string(nil), h.order...)
	}
	capacity := 16
	for len(h.items) > capacity*3/4 {
		capacity *= 2
	}
	buckets := make([][]string, capacity)
	for _, key := range h.order {
		if _, exists := h.items[key]; !exists {
			continue
		}
		bucket := int(java7StringHashValue(h.keyValueLocked(key)) & uint32(capacity-1))
		buckets[bucket] = append([]string{key}, buckets[bucket]...)
	}
	keys := make([]string, 0, len(h.items))
	for _, bucket := range buckets {
		keys = append(keys, bucket...)
	}
	return keys
}

func java7StringHash(value string) uint32 {
	return java7StringHashValue(String(value))
}

func java7StringHashValue(value Value) uint32 {
	var hash uint32
	for _, unit := range sleepStringUnits(value) {
		hash = 31*hash + uint32(unit)
	}
	hash ^= (hash >> 20) ^ (hash >> 12)
	return hash ^ (hash >> 7) ^ (hash >> 4)
}

func sleepHashKey(value Value) (string, Value) {
	value = sleepStringCoercion(value)
	keyValue := sleepStringValueFromUnits(sleepStringUnits(value), sleepStringRawMask(value))
	return sleepCanonicalString(keyValue), keyValue
}

func (h *Hash) rememberKeyLocked(key string, value Value) {
	if h.keyValues == nil {
		h.keyValues = make(map[string]Value)
	}
	if _, exists := h.keyValues[key]; !exists {
		h.keyValues[key] = value
	}
}

func (h *Hash) keyValueLocked(key string) Value {
	if value, exists := h.keyValues[key]; exists {
		return value
	}
	// Hashes created before exact key metadata was introduced can only carry
	// the canonical host spelling. Preserve the historical fallback for those
	// values; all public constructors record exact metadata on insertion.
	return String(key)
}
