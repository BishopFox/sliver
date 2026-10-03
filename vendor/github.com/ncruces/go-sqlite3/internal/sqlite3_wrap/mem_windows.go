package sqlite3_wrap

import (
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Memory struct {
	Buf     []byte
	Max     int64
	pieces  []uintptr
	regions []*MappedRegion
	ptr     uintptr
}

func (m *Memory) Slice() *[]byte {
	return &m.Buf
}

func (m *Memory) Grow(delta, max int64) int64 {
	if m.Buf == nil {
		m.allocate(uint64(m.Max) << 16)
	}

	len := int64(len(m.Buf))
	old := len >> 16
	if delta == 0 {
		return old
	}
	new := old + delta
	max = min(max, m.Max, int64(math.MaxInt)>>16)
	if new > max || new < old {
		return -1
	}
	m.commit(uint64(new) << 16)
	return old
}

func (m *Memory) allocate(max uint64) {
	if !placeholdersSupported() {
		return
	}

	if max > math.MaxInt {
		// This ensures uintptr(max) overflows to a large value,
		// and VirtualAlloc2 returns an error.
		max = math.MaxUint64
	}

	// Reserve max bytes of address space, to ensure we won't need to move it.
	// Use virtual memory placeholders so we can later map files.
	// https://devblogs.microsoft.com/oldnewthing/?p=109346
	r, err := virtualAlloc2(0, uintptr(max),
		windows.MEM_RESERVE|_MEM_RESERVE_PLACEHOLDER, windows.PAGE_NOACCESS)
	if err != nil {
		panic(err)
	}
	m.pieces = append(m.pieces, 0)
	m.ptr = r

	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&m.ptr))
	m.Buf = unsafe.Slice((*byte)(ptr), max)[:0]
}

func (m *Memory) commit(size uint64) {
	if m.ptr == 0 {
		m.Buf = append(m.Buf, make([]byte, size-uint64(len(m.Buf)))...)
		return
	}

	com := uint64(len(m.Buf))
	res := uint64(cap(m.Buf))
	if com < size && size <= res {
		// Split the trailing placeholder.
		if size < res {
			err := windows.VirtualFree(m.ptr+uintptr(com), uintptr(size-com),
				windows.MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
			if err != nil {
				panic(err)
			}
			m.pieces = append(m.pieces, uintptr(size))
		}
		// Replace the placeholder with committed memory.
		_, err := virtualAlloc2(m.ptr+uintptr(com), uintptr(size-com),
			windows.MEM_COMMIT|windows.MEM_RESERVE|_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READWRITE)
		if err != nil {
			panic(err)
		}
	}
	m.Buf = m.Buf[:size]
}

func (m *Memory) reserve(size int64) int64 {
	if m.ptr == 0 || size <= 0 {
		return 0
	}

	com := int64(len(m.Buf))
	res := int64(cap(m.Buf))
	new := com + size

	if new > res || new < com {
		return 0
	}

	// Split the trailing placeholder.
	if new < res {
		err := windows.VirtualFree(m.ptr+uintptr(com), uintptr(new-com),
			windows.MEM_RELEASE|_MEM_PRESERVE_PLACEHOLDER)
		if err != nil {
			panic(err)
		}
		m.pieces = append(m.pieces, uintptr(new))
	}
	m.Buf = m.Buf[:new]
	return com
}

func (m *Memory) Close() (err error) {
	if m.ptr == 0 {
		m.Buf = nil
		return nil
	}

	for _, r := range m.regions {
		e := r.Close()
		if err == nil {
			err = e
		}
	}

	for _, off := range m.pieces {
		e := windows.VirtualFree(m.ptr+off, 0, windows.MEM_RELEASE)
		if err == nil {
			err = e
		}
	}

	m.Buf = nil
	m.pieces = nil
	m.regions = nil
	m.ptr = 0
	return err
}

// CanMapFiles reports whether file views can be mapped into this memory.
func (m *Memory) CanMapFiles() bool { return m.ptr != 0 }
