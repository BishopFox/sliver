//go:build unix

package sqlite3_wrap

import (
	"math"

	"golang.org/x/sys/unix"
)

type Memory struct {
	Buf       []byte
	Max       int64
	regions   []*MappedRegion
	committed int
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
	// Round up to the page size.
	rnd := uint64(unix.Getpagesize() - 1)
	res := (max + rnd) &^ rnd

	if res > math.MaxInt {
		// This ensures int(res) overflows to a negative value,
		// and unix.Mmap returns EINVAL.
		res = math.MaxUint64
	}

	// Reserve res bytes of address space, to ensure we won't need to move it.
	// A protected, private, anonymous mapping should not commit memory.
	b, err := unix.Mmap(-1, 0, int(res), unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		panic(err)
	}
	m.Buf = b[:0]
}

func (m *Memory) commit(size uint64) {
	com := uint64(m.committed)
	res := uint64(cap(m.Buf))
	if com < size && size <= res {
		// Grow geometrically, round up to the page size.
		rnd := uint64(unix.Getpagesize() - 1)
		new := com + com>>3
		new = min(max(size, new), res)
		new = (new + rnd) &^ rnd

		// Commit additional memory up to new bytes.
		err := unix.Mprotect(m.Buf[m.committed:new], unix.PROT_READ|unix.PROT_WRITE)
		if err != nil {
			panic(err)
		}
		m.committed = int(new)
	}
	m.Buf = m.Buf[:size]
}

func (m *Memory) Close() error {
	err := unix.Munmap(m.Buf[:cap(m.Buf)])
	m.Buf = nil
	m.regions = nil
	m.committed = 0
	return err
}
