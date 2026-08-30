package opfor

import (
	"context"
	"errors"
	"sync"
)

var errReadOnlyIterator = errors.New("opfor: iterator is read-only")

// collectionWrapperSource is the small java.util.Collection surface needed by
// Sleep's CollectionWrapper. Snapshot is used only for operations whose
// upstream implementation calls Collection.toArray or iterates into a new
// list. IteratorNext retains the backing collection's fail-fast revision.
type collectionWrapperSource interface {
	wrapperSize() int
	wrapperSnapshot() []Value
	wrapperSnapshotReserved(func(int) error) ([]Value, error)
	wrapperRevision() uint64
	wrapperIteratorNext(index int, expected uint64) (Value, bool, error)
}

type collectionWrapperArrayBackend struct {
	source collectionWrapperSource
	// account owns every persistent entry materialized by this wrapper. Credit
	// is the live size already admitted when the wrapper was created; a later
	// first indexed access only owes growth beyond that amount.
	account        *runtimeResourceAccount
	admittedCredit int

	indexedMu  sync.Mutex
	indexed    []Value
	indexedSet bool
	taintMu    sync.RWMutex
	tainted    bool
}

func (backend *collectionWrapperArrayBackend) detachedValue(value Value) Value {
	if backend == nil {
		return value
	}
	backend.taintMu.RLock()
	tainted := backend.tainted
	backend.taintMu.RUnlock()
	if !tainted {
		return value
	}
	return taintAllValue(value, make(map[any]struct{}))
}

func (backend *collectionWrapperArrayBackend) taintAll() {
	if backend == nil {
		return
	}
	backend.taintMu.Lock()
	backend.tainted = true
	backend.taintMu.Unlock()
}

func newCollectionWrapperArray(source collectionWrapperSource) *Array {
	return newAdmittedCollectionWrapperArray(nil, 0, source)
}

func newAccountedCollectionWrapperArray(account *runtimeResourceAccount, source collectionWrapperSource) (*Array, error) {
	if source == nil {
		return NewReadOnlyArray(), nil
	}
	credit := 0
	if account != nil {
		credit = source.wrapperSize()
		if err := account.reserve(resourceCollectionEntries, uint64(credit)); err != nil {
			return nil, err
		}
	}
	return newAdmittedCollectionWrapperArray(account, credit, source), nil
}

func newAdmittedCollectionWrapperArray(account *runtimeResourceAccount, credit int, source collectionWrapperSource) *Array {
	if source == nil {
		return NewReadOnlyArray()
	}
	return &Array{backend: &collectionWrapperArrayBackend{
		source:         source,
		account:        account,
		admittedCredit: credit,
	}}
}

func newRuntimeCollectionWrapperArray(runtime *Runtime, source collectionWrapperSource) (*Array, error) {
	var account *runtimeResourceAccount
	if runtime != nil {
		account = runtime.resources
	}
	return newAccountedCollectionWrapperArray(account, source)
}

func (backend *collectionWrapperArrayBackend) len() int {
	if backend == nil || backend.source == nil {
		return 0
	}
	return backend.source.wrapperSize()
}

func cellsFromDetachedValues(values []Value) []*Cell {
	cells := make([]*Cell, len(values))
	for index, value := range values {
		cells[index] = NewCell(value)
	}
	return cells
}

func (backend *collectionWrapperArrayBackend) snapshotCells() []*Cell {
	if backend == nil || backend.source == nil {
		return nil
	}
	// CollectionWrapper.scalarIterator and sublist operate on the live
	// Collection, independently of the Object[] cached by getAt.
	values := backend.source.wrapperSnapshot()
	for index := range values {
		values[index] = backend.detachedValue(values[index])
	}
	return cellsFromDetachedValues(values)
}

func (backend *collectionWrapperArrayBackend) cell(index int) (*Cell, bool) {
	cell, ok, _ := backend.cellContext(index)
	return cell, ok
}

func (backend *collectionWrapperArrayBackend) cellContext(index int) (*Cell, bool, error) {
	if backend == nil || backend.source == nil {
		return nil, false, nil
	}
	if index < 0 {
		var ok bool
		index, ok = normalizeArrayIndex(index, backend.source.wrapperSize())
		if !ok {
			return nil, false, nil
		}
	}

	backend.indexedMu.Lock()
	if !backend.indexedSet {
		// This is the one persistent snapshot in CollectionWrapper: getAt lazily
		// calls values.toArray() once and retains the resulting Java objects. The
		// source may have grown since wrapper construction, so reserve that exact
		// snapshot's excess over the wrapper's admitted credit before publishing
		// it. A failed reservation leaves the cache unset and retryable.
		indexed, err := backend.source.wrapperSnapshotReserved(func(snapshotLen int) error {
			growth := snapshotLen - backend.admittedCredit
			if growth <= 0 || backend.account == nil {
				return nil
			}
			return backend.account.reserve(resourceCollectionEntries, uint64(growth))
		})
		if err != nil {
			backend.indexedMu.Unlock()
			return nil, false, err
		}
		backend.indexed = indexed
		backend.indexedSet = true
	}
	if index < 0 || index >= len(backend.indexed) {
		backend.indexedMu.Unlock()
		return nil, false, nil
	}
	value := backend.indexed[index]
	backend.indexedMu.Unlock()

	// ObjectUtilities.BuildScalar constructs a new Scalar for every getAt;
	// retaining the Object[] does not retain a mutable Scalar cell.
	return NewCell(backend.detachedValue(value)), true, nil
}

func (backend *collectionWrapperArrayBackend) sublist(start, end int) (*Array, error) {
	return backend.sublistForRuntime(nil, start, end)
}

func (backend *collectionWrapperArrayBackend) sublistForRuntime(runtime *Runtime, start, end int) (*Array, error) {
	if backend == nil || backend.source == nil || start < 0 || end < start {
		return nil, ErrIndexOutOfRange
	}
	// CollectionWrapper.sublist walks values.iterator() until end; it does not
	// use the lazy Object[] held by getAt. Preserve the iterator's fail-fast
	// behavior while copying only the requested live objects. Reserve each
	// retained destination entry immediately before its append so concurrent
	// shrink is charged exactly and no destination capacity is allocated before
	// admission; a later fail-fast error conservatively leaves prior entries
	// charged.
	expected := backend.source.wrapperRevision()
	var selected []Value
	for index := 0; index < end; index++ {
		value, present, err := backend.source.wrapperIteratorNext(index, expected)
		if err != nil {
			return nil, err
		}
		if !present {
			break
		}
		if index >= start {
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				return nil, err
			}
			selected = append(selected, value)
		}
	}
	for index := range selected {
		selected[index] = backend.detachedValue(selected[index])
	}
	// Upstream copies the selected objects into a private LinkedList and wraps
	// that list, so the resulting sublist is detached but remains read-only.
	var account *runtimeResourceAccount
	if runtime != nil {
		account = runtime.resources
	}
	return newAdmittedCollectionWrapperArray(
		account,
		len(selected),
		newPortableJavaCollection("LinkedList", selected),
	), nil
}

func (backend *collectionWrapperArrayBackend) iterator(source Value) valueIterator {
	if backend == nil || backend.source == nil {
		return &sliceIterator{source: source}
	}
	return &collectionWrapperIterator{
		source:   source,
		backing:  backend.source,
		backend:  backend,
		expected: backend.source.wrapperRevision(),
	}
}

type collectionWrapperIterator struct {
	source   Value
	backing  collectionWrapperSource
	backend  *collectionWrapperArrayBackend
	expected uint64
	index    int
	count    int
	stopped  bool
}

func (iterator *collectionWrapperIterator) next(context.Context) (iteratorItem, bool, error) {
	if iterator == nil || iterator.backing == nil || iterator.stopped {
		return iteratorItem{}, false, nil
	}
	value, ok, err := iterator.backing.wrapperIteratorNext(iterator.index, iterator.expected)
	if err != nil {
		iterator.stopped = true
		return iteratorItem{}, false, err
	}
	if !ok {
		return iteratorItem{}, false, nil
	}
	value = iterator.backend.detachedValue(value)
	iterator.index++
	cell := NewCell(value)
	item := iteratorItem{key: Int(int32(iterator.count)), value: value, cell: cell}
	iterator.count++
	return item, true, nil
}

func (*collectionWrapperIterator) remove(context.Context) error {
	return errReadOnlyIterator
}

func (iterator *collectionWrapperIterator) sourceValue() Value {
	if iterator == nil {
		return Null()
	}
	return iterator.source
}

func (collection *portableJavaCollection) wrapperSize() int {
	if collection == nil {
		return 0
	}
	size, _ := collection.sizeChecked()
	return size
}

func (collection *portableJavaCollection) wrapperSnapshot() []Value {
	return collection.snapshot()
}

func (collection *portableJavaCollection) wrapperSnapshotReserved(reserve func(int) error) ([]Value, error) {
	if collection == nil {
		if reserve != nil {
			if err := reserve(0); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if collection.wrapperSource != nil && collection.wrapperSource != collection {
		return collection.wrapperSource.wrapperSnapshotReserved(reserve)
	}
	if collection.copies {
		if collection.copiesCount > portableCollectionsMaximumMaterializedElements {
			// wrapperSnapshot historically treats an unmaterializable copies
			// collection as empty. Preserve that behavior without charging entries
			// which cannot become part of the persistent indexed cache.
			return nil, nil
		}
		if reserve != nil {
			if err := reserve(collection.copiesCount); err != nil {
				return nil, err
			}
		}
		values := make([]Value, collection.copiesCount)
		for index := range values {
			values[index] = collection.copiesValue
		}
		return values, nil
	}
	if collection.listView != nil {
		view := collection.listView
		view.root.mu.RLock()
		defer view.root.mu.RUnlock()
		if err := view.checkLocked(); err != nil {
			// wrapperSnapshot drops invalid-view errors and exposes an empty
			// snapshot; retain that compatibility here.
			return nil, nil
		}
		if reserve != nil {
			if err := reserve(view.size); err != nil {
				return nil, err
			}
		}
		values := make([]Value, view.size)
		copy(values, view.root.values[view.offset:view.offset+view.size])
		return values, nil
	}
	if collection.mapView != nil {
		return collection.mapView.wrapperSnapshotReserved(reserve)
	}
	collection.mu.RLock()
	defer collection.mu.RUnlock()
	if reserve != nil {
		if err := reserve(len(collection.values)); err != nil {
			return nil, err
		}
	}
	values := make([]Value, len(collection.values))
	copy(values, collection.values)
	return values, nil
}

func (collection *portableJavaCollection) wrapperRevision() uint64 {
	if collection == nil {
		return 0
	}
	_, revision, _ := collection.iteratorBounds()
	return revision
}

func (collection *portableJavaCollection) wrapperIteratorNext(index int, expected uint64) (Value, bool, error) {
	if collection == nil {
		return Null(), false, nil
	}
	value, present, err := collection.iteratorValue(index, expected)
	if err != nil && err.Error() == "java.util.ConcurrentModificationException" {
		return Null(), false, ErrArrayChangedDuringIteration
	}
	return value, present, err
}

type portableJavaMapKeySetSource struct {
	mapping *portableJavaMap
}

type portableJavaMapValuesSource struct {
	mapping *portableJavaMap
}

type portableJavaMapEntrySetSource struct {
	mapping *portableJavaMap
}

func (source *portableJavaMapValuesSource) wrapperSize() int {
	if source == nil || source.mapping == nil {
		return 0
	}
	source.mapping.mu.RLock()
	size := len(source.mapping.values)
	source.mapping.mu.RUnlock()
	return size
}

func (source *portableJavaMapValuesSource) wrapperSnapshot() []Value {
	if source == nil || source.mapping == nil {
		return nil
	}
	source.mapping.mu.RLock()
	keys := source.mapping.wrapperKeysLocked()
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		values = append(values, source.mapping.values[key])
	}
	source.mapping.mu.RUnlock()
	return values
}

func (source *portableJavaMapValuesSource) wrapperSnapshotReserved(reserve func(int) error) ([]Value, error) {
	if source == nil {
		return nil, nil
	}
	return portableJavaMapWrapperSnapshotReserved(source.mapping, portableJavaMapValues, reserve)
}

func (source *portableJavaMapValuesSource) wrapperRevision() uint64 {
	if source == nil || source.mapping == nil {
		return 0
	}
	source.mapping.mu.RLock()
	revision := source.mapping.mod
	source.mapping.mu.RUnlock()
	return revision
}

func (source *portableJavaMapValuesSource) wrapperIteratorNext(index int, expected uint64) (Value, bool, error) {
	if source == nil || source.mapping == nil {
		return Null(), false, nil
	}
	source.mapping.mu.RLock()
	defer source.mapping.mu.RUnlock()
	keys := source.mapping.wrapperKeysLocked()
	if index == len(keys) {
		return Null(), false, nil
	}
	if expected != source.mapping.mod {
		return Null(), false, ErrArrayChangedDuringIteration
	}
	if index < 0 || index >= len(keys) {
		return Null(), false, nil
	}
	return source.mapping.values[keys[index]], true, nil
}

func (source *portableJavaMapKeySetSource) wrapperSize() int {
	if source == nil || source.mapping == nil {
		return 0
	}
	source.mapping.mu.RLock()
	size := len(source.mapping.values)
	source.mapping.mu.RUnlock()
	return size
}

func (source *portableJavaMapKeySetSource) wrapperSnapshot() []Value {
	if source == nil || source.mapping == nil {
		return nil
	}
	source.mapping.mu.RLock()
	orderedKeys := source.mapping.wrapperKeysLocked()
	keys := make([]Value, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		keys = append(keys, source.mapping.keyValues[key])
	}
	source.mapping.mu.RUnlock()
	return keys
}

func (source *portableJavaMapKeySetSource) wrapperSnapshotReserved(reserve func(int) error) ([]Value, error) {
	if source == nil {
		return nil, nil
	}
	return portableJavaMapWrapperSnapshotReserved(source.mapping, portableJavaMapKeySet, reserve)
}

func (source *portableJavaMapKeySetSource) wrapperRevision() uint64 {
	if source == nil || source.mapping == nil {
		return 0
	}
	source.mapping.mu.RLock()
	revision := source.mapping.mod
	source.mapping.mu.RUnlock()
	return revision
}

func (source *portableJavaMapKeySetSource) wrapperIteratorNext(index int, expected uint64) (Value, bool, error) {
	if source == nil || source.mapping == nil {
		return Null(), false, nil
	}
	source.mapping.mu.RLock()
	defer source.mapping.mu.RUnlock()
	keys := source.mapping.wrapperKeysLocked()
	if index == len(keys) {
		return Null(), false, nil
	}
	if expected != source.mapping.mod {
		return Null(), false, ErrArrayChangedDuringIteration
	}
	if index < 0 || index >= len(keys) {
		return Null(), false, nil
	}
	return source.mapping.keyValues[keys[index]], true, nil
}

func (source *portableJavaMapEntrySetSource) wrapperSize() int {
	if source == nil || source.mapping == nil {
		return 0
	}
	return (&portableJavaMapView{mapping: source.mapping, kind: portableJavaMapEntrySet}).size()
}

func (source *portableJavaMapEntrySetSource) wrapperSnapshot() []Value {
	if source == nil || source.mapping == nil {
		return nil
	}
	return (&portableJavaMapView{mapping: source.mapping, kind: portableJavaMapEntrySet}).snapshot()
}

func (source *portableJavaMapEntrySetSource) wrapperSnapshotReserved(reserve func(int) error) ([]Value, error) {
	if source == nil {
		return nil, nil
	}
	return portableJavaMapWrapperSnapshotReserved(source.mapping, portableJavaMapEntrySet, reserve)
}

func (view *portableJavaMapView) wrapperSnapshotReserved(reserve func(int) error) ([]Value, error) {
	if view == nil {
		return portableJavaMapWrapperSnapshotReserved(nil, portableJavaMapKeySet, reserve)
	}
	return portableJavaMapWrapperSnapshotReserved(view.mapping, view.kind, reserve)
}

func portableJavaMapWrapperSnapshotReserved(
	mapping *portableJavaMap,
	kind portableJavaMapViewKind,
	reserve func(int) error,
) ([]Value, error) {
	if mapping == nil {
		if reserve != nil {
			if err := reserve(0); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	mapping.mu.RLock()
	defer mapping.mu.RUnlock()
	snapshotLen := len(mapping.values)
	if reserve != nil {
		if err := reserve(snapshotLen); err != nil {
			return nil, err
		}
	}
	// wrapperKeysLocked allocates HashMap traversal buckets, so it must remain
	// after the reservation preflight along with the result slice and entry
	// wrappers.
	keys := mapping.wrapperKeysLocked()
	values := make([]Value, 0, snapshotLen)
	for _, key := range keys {
		switch kind {
		case portableJavaMapKeySet:
			values = append(values, mapping.keyValues[key])
		case portableJavaMapValues:
			values = append(values, mapping.values[key])
		default:
			values = append(values, ObjectValue(newPortableJavaMapEntryLocked(mapping, key)))
		}
	}
	return values, nil
}

func (source *portableJavaMapEntrySetSource) wrapperRevision() uint64 {
	if source == nil || source.mapping == nil {
		return 0
	}
	return (&portableJavaMapView{mapping: source.mapping, kind: portableJavaMapEntrySet}).revision()
}

func (source *portableJavaMapEntrySetSource) wrapperIteratorNext(index int, expected uint64) (Value, bool, error) {
	if source == nil || source.mapping == nil {
		return Null(), false, nil
	}
	value, present, err := (&portableJavaMapView{mapping: source.mapping, kind: portableJavaMapEntrySet}).iteratorValue(index, expected)
	if err != nil && err.Error() == "java.util.ConcurrentModificationException" {
		return Null(), false, ErrArrayChangedDuringIteration
	}
	return value, present, err
}

type mapWrapperHashBackend struct {
	mapping *portableJavaMap
	account *runtimeResourceAccount
	taintMu sync.RWMutex
	tainted bool
}

func newMapWrapperHash(mapping *portableJavaMap) *Hash {
	return newAccountedMapWrapperHash(nil, mapping)
}

func newRuntimeMapWrapperHash(runtime *Runtime, mapping *portableJavaMap) *Hash {
	var account *runtimeResourceAccount
	if runtime != nil {
		account = runtime.resources
	}
	return newAccountedMapWrapperHash(account, mapping)
}

func newAccountedMapWrapperHash(account *runtimeResourceAccount, mapping *portableJavaMap) *Hash {
	hash := NewHash()
	hash.readOnly = true
	hash.backend = &mapWrapperHashBackend{mapping: mapping, account: account}
	return hash
}

func (backend *mapWrapperHashBackend) len() int {
	if backend == nil || backend.mapping == nil {
		return 0
	}
	backend.mapping.mu.RLock()
	size := len(backend.mapping.values)
	backend.mapping.mu.RUnlock()
	return size
}

func (backend *mapWrapperHashBackend) get(key Value) (Value, bool) {
	if backend == nil || backend.mapping == nil {
		return Null(), false
	}
	keyText := sleepCanonicalString(sleepStringCoercion(key))
	backend.mapping.mu.RLock()
	value, exists := backend.mapping.values[keyText]
	backend.mapping.mu.RUnlock()
	if !exists {
		return Null(), false
	}
	return backend.detachedValue(value), true
}

func (backend *mapWrapperHashBackend) detachedValue(value Value) Value {
	if backend == nil {
		return value
	}
	backend.taintMu.RLock()
	tainted := backend.tainted
	backend.taintMu.RUnlock()
	if !tainted {
		return value
	}
	return taintAllValue(value, make(map[any]struct{}))
}

func (backend *mapWrapperHashBackend) taintAll() {
	if backend == nil {
		return
	}
	backend.taintMu.Lock()
	backend.tainted = true
	backend.taintMu.Unlock()
}

func (backend *mapWrapperHashBackend) cell(key Value) (*Cell, bool) {
	value, exists := backend.get(key)
	if !exists {
		return nil, false
	}
	return NewCell(value), true
}

func (backend *mapWrapperHashBackend) ensure(key Value) *Cell {
	value, _ := backend.get(key)
	return NewCell(value)
}

func (backend *mapWrapperHashBackend) keyValues() []Value {
	return (&portableJavaMapKeySetSource{mapping: backend.mapping}).wrapperSnapshot()
}

func (backend *mapWrapperHashBackend) keysArray(fallback *runtimeResourceAccount) (*Array, error) {
	account := backend.account
	if account == nil {
		account = fallback
	}
	return newAccountedCollectionWrapperArray(account, &portableJavaMapKeySetSource{mapping: backend.mapping})
}

func (backend *mapWrapperHashBackend) dataSnapshot() *Hash {
	snapshot, _ := backend.dataSnapshotReserved(nil)
	if snapshot == nil {
		return NewHash()
	}
	return snapshot
}

func (backend *mapWrapperHashBackend) dataSnapshotReserved(reserve func(int) error) (*Hash, error) {
	if backend == nil || backend.mapping == nil {
		return NewHash(), nil
	}
	backend.mapping.mu.RLock()
	// Count without calling wrapperKeysLocked: its Java HashMap traversal
	// allocates a capacity-sized bucket table. Keep the map read lock from this
	// allocation-free preflight through capture so a concurrent mutation cannot
	// grow the materialized snapshot beyond the successful reservation.
	count := 0
	for _, key := range backend.mapping.keys {
		keyValue := backend.mapping.keyValues[key]
		value, exists := backend.mapping.values[key]
		if exists && !keyValue.IsNull() && !value.IsNull() {
			count++
		}
	}
	if reserve != nil {
		if err := reserve(count); err != nil {
			backend.mapping.mu.RUnlock()
			return nil, err
		}
	}
	// MapWrapper.getData walks the wrapped map's entrySet and inserts those
	// entries into a fresh HashMap. In particular, a HashMap source must first
	// expose its own bucket traversal order; inserting that order into the new
	// hash can reverse colliding entries a second time.
	keys := backend.mapping.wrapperKeysLocked()
	entries := make([]hashBackendEntry, 0, len(keys))
	for _, key := range keys {
		keyValue := backend.mapping.keyValues[key]
		value, exists := backend.mapping.values[key]
		// MapWrapper.getData omits Java null keys and values. Portable map keys
		// are normalized strings, while a Sleep null value models Java null.
		if !exists || keyValue.IsNull() || value.IsNull() {
			continue
		}
		entries = append(entries, hashBackendEntry{key: keyValue, value: backend.detachedValue(value)})
	}
	backend.mapping.mu.RUnlock()
	snapshot := NewHash()
	for _, entry := range entries {
		snapshot.SetValue(entry.key, entry.value)
	}
	return snapshot, nil
}
