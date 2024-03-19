package wasm

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/internalapi"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/wasmruntime"
)

const (
	// MemoryPageSize is the unit of memory length in WebAssembly,
	// and is defined as 2^16 = 65536.
	// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#memory-instances%E2%91%A0
	MemoryPageSize = uint32(65536)
	// MemoryLimitPages is maximum number of pages defined (2^16).
	// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#grow-mem
	MemoryLimitPages = uint32(65536)
	// MemoryPageSizeInBits satisfies the relation: "1 << MemoryPageSizeInBits == MemoryPageSize".
	MemoryPageSizeInBits = 16
)

// compile-time check to ensure MemoryInstance implements api.Memory
var _ api.Memory = &MemoryInstance{}

type waiters struct {
	mux sync.Mutex
	l   *list.List
}

// MemoryInstance represents a memory instance in a store, and implements api.Memory.
//
// Note: In WebAssembly 1.0 (20191205), there may be up to one Memory per store, which means the precise memory is always
// wasm.Store Memories index zero: `store.Memories[0]`
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#memory-instances%E2%91%A0.
type MemoryInstance struct {
	internalapi.WazeroOnlyType

	Buffer        []byte
	Min, Cap, Max uint32
	Shared        bool
	// definition is known at compile time.
	definition api.MemoryDefinition

	// Mux is used in interpreter mode to prevent overlapping calls to atomic instructions,
	// introduced with WebAssembly threads proposal.
	Mux sync.Mutex

	// waiters implements atomic wait and notify. It is implemented similarly to golang.org/x/sync/semaphore,
	// with a fixed weight of 1 and no spurious notifications.
	waiters sync.Map

	mmappedBuffer []byte
	closed        bool
}

// NewMemoryInstance creates a new instance based on the parameters in the SectionIDMemory.
func NewMemoryInstance(memSec *Memory) *MemoryInstance {
	min := MemoryPagesToBytesNum(memSec.Min)
	capacity := MemoryPagesToBytesNum(memSec.Cap)

	var buffer []byte
	var cap uint32
	var mmappedBuffer []byte
	if memSec.IsShared {
		// Memory accesses can happen at the same time that memory is resized, meaning
		// we cannot have the memory base move during operation. mmap allows allocating memory virtually so
		// we can grow without changing the base. The spec requires max for shared memory currently because
		// all threads implementations are effectively expected to use mmap for shared memory.
		max := MemoryPagesToBytesNum(memSec.Max)
		var b []byte
		if platform.MmapSupported && max > 0 {
			var err error
			b, err = platform.MmapMemory(int(max))
			if err != nil {
				panic(fmt.Errorf("unable to mmap memory: %w", err))
			}
			mmappedBuffer = b
		} else {
			// mmap not supported so we just preallocate a normal buffer. This will often be large, i.e. ~4GB,
			// and likely isn't practical, but interpreter usage should be rare and the Wasm binary can be
			// edited to reduce max memory size if support for non-mmap platforms is required.
			b = make([]byte, max)
		}
		buffer = b[:MemoryPagesToBytesNum(memSec.Min)]
		cap = memSec.Max
	} else {
		buffer = make([]byte, min, capacity)
		cap = memSec.Cap
	}

	return &MemoryInstance{
		Buffer:        buffer,
		Min:           memSec.Min,
		Cap:           cap,
		Max:           memSec.Max,
		Shared:        memSec.IsShared,
		mmappedBuffer: mmappedBuffer,
	}
}

func (m *MemoryInstance) Close() error {
	if m.mmappedBuffer == nil {
		// No need to release anything for non-mmapped memory.
		return nil
	}

	m.Mux.Lock()
	defer m.Mux.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true
	return platform.MunmapCodeSegment(m.mmappedBuffer)
}

// Definition implements the same method as documented on api.Memory.
func (m *MemoryInstance) Definition() api.MemoryDefinition {
	return m.definition
}

// Size implements the same method as documented on api.Memory.
func (m *MemoryInstance) Size() uint32 {
	return m.size()
}

// ReadByte implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadByte(offset uint32) (byte, bool) {
	if offset >= m.size() {
		return 0, false
	}
	return m.Buffer[offset], true
}

// ReadUint16Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadUint16Le(offset uint32) (uint16, bool) {
	if !m.hasSize(offset, 2) {
		return 0, false
	}
	return binary.LittleEndian.Uint16(m.Buffer[offset : offset+2]), true
}

// ReadUint32Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadUint32Le(offset uint32) (uint32, bool) {
	return m.readUint32Le(offset)
}

// ReadFloat32Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadFloat32Le(offset uint32) (float32, bool) {
	v, ok := m.readUint32Le(offset)
	if !ok {
		return 0, false
	}
	return math.Float32frombits(v), true
}

// ReadUint64Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadUint64Le(offset uint32) (uint64, bool) {
	return m.readUint64Le(offset)
}

// ReadFloat64Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) ReadFloat64Le(offset uint32) (float64, bool) {
	v, ok := m.readUint64Le(offset)
	if !ok {
		return 0, false
	}
	return math.Float64frombits(v), true
}

// Read implements the same method as documented on api.Memory.
func (m *MemoryInstance) Read(offset, byteCount uint32) ([]byte, bool) {
	if !m.hasSize(offset, uint64(byteCount)) {
		return nil, false
	}
	return m.Buffer[offset : offset+byteCount : offset+byteCount], true
}

// WriteByte implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteByte(offset uint32, v byte) bool {
	if offset >= m.size() {
		return false
	}
	m.Buffer[offset] = v
	return true
}

// WriteUint16Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteUint16Le(offset uint32, v uint16) bool {
	if !m.hasSize(offset, 2) {
		return false
	}
	binary.LittleEndian.PutUint16(m.Buffer[offset:], v)
	return true
}

// WriteUint32Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteUint32Le(offset, v uint32) bool {
	return m.writeUint32Le(offset, v)
}

// WriteFloat32Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteFloat32Le(offset uint32, v float32) bool {
	return m.writeUint32Le(offset, math.Float32bits(v))
}

// WriteUint64Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteUint64Le(offset uint32, v uint64) bool {
	return m.writeUint64Le(offset, v)
}

// WriteFloat64Le implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteFloat64Le(offset uint32, v float64) bool {
	return m.writeUint64Le(offset, math.Float64bits(v))
}

// Write implements the same method as documented on api.Memory.
func (m *MemoryInstance) Write(offset uint32, val []byte) bool {
	if !m.hasSize(offset, uint64(len(val))) {
		return false
	}
	copy(m.Buffer[offset:], val)
	return true
}

// WriteString implements the same method as documented on api.Memory.
func (m *MemoryInstance) WriteString(offset uint32, val string) bool {
	if !m.hasSize(offset, uint64(len(val))) {
		return false
	}
	copy(m.Buffer[offset:], val)
	return true
}

// MemoryPagesToBytesNum converts the given pages into the number of bytes contained in these pages.
func MemoryPagesToBytesNum(pages uint32) (bytesNum uint64) {
	return uint64(pages) << MemoryPageSizeInBits
}

// Grow implements the same method as documented on api.Memory.
func (m *MemoryInstance) Grow(delta uint32) (result uint32, ok bool) {
	currentPages := memoryBytesNumToPages(uint64(len(m.Buffer)))
	if delta == 0 {
		return currentPages, true
	}

	// If exceeds the max of memory size, we push -1 according to the spec.
	newPages := currentPages + delta
	if newPages > m.Max || int32(delta) < 0 {
		return 0, false
	} else if newPages > m.Cap { // grow the memory.
		if m.Shared {
			panic("shared memory cannot be grown, this is a bug in wazero")
		}
		m.Buffer = append(m.Buffer, make([]byte, MemoryPagesToBytesNum(delta))...)
		m.Cap = newPages
		return currentPages, true
	} else { // We already have the capacity we need.
		sp := (*reflect.SliceHeader)(unsafe.Pointer(&m.Buffer))
		if m.Shared {
			// Use atomic write to ensure new length is visible across threads.
			atomic.StoreUintptr((*uintptr)(unsafe.Pointer(&sp.Len)), uintptr(MemoryPagesToBytesNum(newPages)))
		} else {
			sp.Len = int(MemoryPagesToBytesNum(newPages))
		}
		return currentPages, true
	}
}

// PageSize returns the current memory buffer size in pages.
func (m *MemoryInstance) PageSize() (result uint32) {
	return memoryBytesNumToPages(uint64(len(m.Buffer)))
}

// PagesToUnitOfBytes converts the pages to a human-readable form similar to what's specified. e.g. 1 -> "64Ki"
//
// See https://www.w3.org/TR/2019/REC-wasm-core-1-20191205/#memory-instances%E2%91%A0
func PagesToUnitOfBytes(pages uint32) string {
	k := pages * 64
	if k < 1024 {
		return fmt.Sprintf("%d Ki", k)
	}
	m := k / 1024
	if m < 1024 {
		return fmt.Sprintf("%d Mi", m)
	}
	g := m / 1024
	if g < 1024 {
		return fmt.Sprintf("%d Gi", g)
	}
	return fmt.Sprintf("%d Ti", g/1024)
}

// Below are raw functions used to implement the api.Memory API:

// memoryBytesNumToPages converts the given number of bytes into the number of pages.
func memoryBytesNumToPages(bytesNum uint64) (pages uint32) {
	return uint32(bytesNum >> MemoryPageSizeInBits)
}

// size returns the size in bytes of the buffer.
func (m *MemoryInstance) size() uint32 {
	return uint32(len(m.Buffer)) // We don't lock here because size can't become smaller.
}

// hasSize returns true if Len is sufficient for byteCount at the given offset.
//
// Note: This is always fine, because memory can grow, but never shrink.
func (m *MemoryInstance) hasSize(offset uint32, byteCount uint64) bool {
	return uint64(offset)+byteCount <= uint64(len(m.Buffer)) // uint64 prevents overflow on add
}

// readUint32Le implements ReadUint32Le without using a context. This is extracted as both ints and floats are stored in
// memory as uint32le.
func (m *MemoryInstance) readUint32Le(offset uint32) (uint32, bool) {
	if !m.hasSize(offset, 4) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(m.Buffer[offset : offset+4]), true
}

// readUint64Le implements ReadUint64Le without using a context. This is extracted as both ints and floats are stored in
// memory as uint64le.
func (m *MemoryInstance) readUint64Le(offset uint32) (uint64, bool) {
	if !m.hasSize(offset, 8) {
		return 0, false
	}
	return binary.LittleEndian.Uint64(m.Buffer[offset : offset+8]), true
}

// writeUint32Le implements WriteUint32Le without using a context. This is extracted as both ints and floats are stored
// in memory as uint32le.
func (m *MemoryInstance) writeUint32Le(offset uint32, v uint32) bool {
	if !m.hasSize(offset, 4) {
		return false
	}
	binary.LittleEndian.PutUint32(m.Buffer[offset:], v)
	return true
}

// writeUint64Le implements WriteUint64Le without using a context. This is extracted as both ints and floats are stored
// in memory as uint64le.
func (m *MemoryInstance) writeUint64Le(offset uint32, v uint64) bool {
	if !m.hasSize(offset, 8) {
		return false
	}
	binary.LittleEndian.PutUint64(m.Buffer[offset:], v)
	return true
}

// Wait32 suspends the caller until the offset is notified by a different agent.
func (m *MemoryInstance) Wait32(offset uint32, exp uint32, timeout int64, reader func(mem *MemoryInstance, offset uint32) uint32) uint64 {
	w := m.getWaiters(offset)
	w.mux.Lock()

	cur := reader(m, offset)
	if cur != exp {
		w.mux.Unlock()
		return 1
	}

	return m.wait(w, timeout)
}

// Wait64 suspends the caller until the offset is notified by a different agent.
func (m *MemoryInstance) Wait64(offset uint32, exp uint64, timeout int64, reader func(mem *MemoryInstance, offset uint32) uint64) uint64 {
	w := m.getWaiters(offset)
	w.mux.Lock()

	cur := reader(m, offset)
	if cur != exp {
		w.mux.Unlock()
		return 1
	}

	return m.wait(w, timeout)
}

func (m *MemoryInstance) wait(w *waiters, timeout int64) uint64 {
	if w.l == nil {
		w.l = list.New()
	}

	// The specification requires a trap if the number of existing waiters + 1 == 2^32, so we add a check here.
	// In practice, it is unlikely the application would ever accumulate such a large number of waiters as it
	// indicates several GB of RAM used just for the list of waiters.
	// https://github.com/WebAssembly/threads/blob/main/proposals/threads/Overview.md#wait
	if uint64(w.l.Len()+1) == 1<<32 {
		w.mux.Unlock()
		panic(wasmruntime.ErrRuntimeTooManyWaiters)
	}

	ready := make(chan struct{})
	elem := w.l.PushBack(ready)
	w.mux.Unlock()

	if timeout < 0 {
		<-ready
		return 0
	} else {
		select {
		case <-ready:
			return 0
		case <-time.After(time.Duration(timeout)):
			// While we could see if the channel completed by now and ignore the timeout, similar to x/sync/semaphore,
			// the Wasm spec doesn't specify this behavior, so we keep things simple by prioritizing the timeout.
			w.mux.Lock()
			w.l.Remove(elem)
			w.mux.Unlock()
			return 2
		}
	}
}

func (m *MemoryInstance) getWaiters(offset uint32) *waiters {
	wAny, ok := m.waiters.Load(offset)
	if !ok {
		// The first time an address is waited on, simultaneous waits will cause extra allocations.
		// Further operations will be loaded above, which is also the general pattern of usage with
		// mutexes.
		wAny, _ = m.waiters.LoadOrStore(offset, &waiters{})
	}

	return wAny.(*waiters)
}

// Notify wakes up at most count waiters at the given offset.
func (m *MemoryInstance) Notify(offset uint32, count uint32) uint32 {
	wAny, ok := m.waiters.Load(offset)
	if !ok {
		return 0
	}
	w := wAny.(*waiters)

	w.mux.Lock()
	defer w.mux.Unlock()
	if w.l == nil {
		return 0
	}

	res := uint32(0)
	for num := w.l.Len(); num > 0 && res < count; num = w.l.Len() {
		w := w.l.Remove(w.l.Front()).(chan struct{})
		close(w)
		res++
	}

	return res
}
