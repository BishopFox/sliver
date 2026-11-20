//go:build !sqlite3_dotlk

package vfs

import (
	"io"
	"os"
	"sync/atomic"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

const _WALINDEX_PGSZ = 32768

type vfsShm struct {
	*os.File
	path     string
	regions  []*sqlite3_wrap.MappedRegion
	fileLock bool
}

func (s *vfsShm) shmOpen() error {
	if s.fileLock {
		return nil
	}
	if s.File == nil {
		f, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return sysError{err, _CANTOPEN}
		}
		s.fileLock = false
		s.File = f
	}

	// Dead man's switch.
	if osWriteLock(s.File, _SHM_DMS, 1, 0) == nil {
		err := s.Truncate(0)
		osUnlock(s.File, _SHM_DMS, 1)
		if err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	err := osReadLock(s.File, _SHM_DMS, 1, 0)
	s.fileLock = err == nil
	return err
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (_ ptr_t, err error) {
	// Ensure pages are the expected size, and that we can map files.
	if size != _WALINDEX_PGSZ || !wrp.CanMapFiles() {
		return 0, _IOERR_SHMMAP
	}

	if err := s.shmOpen(); err != nil {
		return 0, err
	}

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, sysError{err, _IOERR_SHMSIZE}
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, nil
		}
		if err := osAllocate(s.File, n); err != nil {
			return 0, sysError{err, _IOERR_SHMSIZE}
		}
	}

	r, err := wrp.MapRegion(s.File, int64(id)*int64(size), size, false)
	if err != nil {
		return 0, err
	}
	if r == nil {
		return 0, _IOERR_NOMEM
	}
	s.regions = append(s.regions, r)
	return r.Ptr, nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (err error) {
	if s.File == nil {
		return _IOERR_SHMLOCK
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+uint32(offset), uint32(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	default:
		panic(errutil.AssertErr())
	}
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.File == nil {
		return
	}

	// Unmap regions.
	for _, r := range s.regions {
		r.Unmap()
	}
	s.regions = nil

	// Close the file.
	s.Close()
	s.File = nil
	s.fileLock = false
	if delete {
		os.Remove(s.path)
	}
}

func (s *vfsShm) shmBarrier() {
	var b atomic.Bool
	b.Swap(true)
}
