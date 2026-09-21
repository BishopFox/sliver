//go:build linux || darwin || freebsd

package vfs

import (
	"os"

	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

func (f *vfsFile) MemoryMapper() MemoryMapper { return f.mmap }

// NewMemoryMapper returns a memory mapper for the given file.
// It will return nil if file mapping is not supported,
// or not appropriate for the given flags.
// Only databases use memory mapping.
func NewMemoryMapper(file *os.File, flags OpenFlag) MemoryMapper {
	if flags&(OPEN_MAIN_DB|OPEN_TEMP_DB|OPEN_TRANSIENT_DB) == 0 || flags&OPEN_MEMORY != 0 {
		return nil
	}
	return &vfsMapper{File: file}
}

type vfsMapper struct {
	*os.File
	mmap *sqlite3_wrap.MappedRegion
	size int32
}

func (m *vfsMapper) mmapSize(wrp *sqlite3_wrap.Wrapper, p ptr_t) {
	size := int64(wrp.Read64(p))
	wrp.Write64(p, uint64(m.size))
	if size >= 0 && m.mmap == nil {
		m.size = int32(min(size, 1024*1024*1024))
	}
}

func (m *vfsMapper) fetch(wrp *sqlite3_wrap.Wrapper, iOfst int64, iAmt int32, pp ptr_t) error {
	var ptr ptr_t
	if iOfst+int64(iAmt)+256 <= int64(m.size) {
		if m.mmap == nil {
			var err error
			m.mmap, err = wrp.MapRegion(m.File, 0, m.size, true)
			if err != nil {
				return err
			}
		}
		if m.mmap != nil {
			ptr = m.mmap.Ptr + ptr_t(iOfst)
		}
	}
	wrp.Write32(pp, uint32(ptr))
	return nil
}

func (m *vfsMapper) Close() error {
	c := m.mmap
	if c == nil {
		return nil
	}
	m.mmap = nil
	return c.Unmap()
}
