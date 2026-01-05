//go:build (linux || darwin) && (386 || arm || amd64 || arm64 || riscv64 || ppc64le || loong64) && !(sqlite3_flock || sqlite3_dotlk)

package vfs

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/tetratelabs/wazero/api"

	"github.com/ncruces/go-sqlite3/internal/util"
)

type vfsShm struct {
	*os.File
	path     string
	regions  []*util.MappedRegion
	readOnly bool
	fileLock bool
	blocking bool
	sync.Mutex
}

var _ blockingSharedMemory = &vfsShm{}

func (s *vfsShm) shmOpen() error {
	if s.fileLock {
		return nil
	}
	if s.File == nil {
		f, err := os.OpenFile(s.path,
			os.O_RDWR|os.O_CREATE|_O_NOFOLLOW, 0666)
		if err != nil {
			f, err = os.OpenFile(s.path,
				os.O_RDONLY|os.O_CREATE|_O_NOFOLLOW, 0666)
			s.readOnly = true
		}
		if err != nil {
			return sysError{err, _CANTOPEN}
		}
		s.fileLock = false
		s.File = f
	}

	// Dead man's switch.
	if lock, err := osTestLock(s.File, _SHM_DMS, 1, _IOERR_LOCK); err != nil {
		return err
	} else if lock == unix.F_WRLCK {
		return _BUSY
	} else if lock == unix.F_UNLCK {
		if s.readOnly {
			return _READONLY_CANTINIT
		}
		// Do not use a blocking lock here.
		// If the lock cannot be obtained immediately,
		// it means some other connection is truncating the file.
		// And after it has done so, it will not release its lock,
		// but only downgrade it to a shared lock.
		// So no point in blocking here.
		// The call below to obtain the shared DMS lock may use a blocking lock.
		if err := osWriteLock(s.File, _SHM_DMS, 1, 0); err != nil {
			return err
		}
		if err := s.Truncate(0); err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	err := osReadLock(s.File, _SHM_DMS, 1, time.Millisecond)
	s.fileLock = err == nil
	return err
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (ptr_t, error) {
	// Ensure size is a multiple of the OS page size.
	if int(size)&(unix.Getpagesize()-1) != 0 {
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
		if s.readOnly {
			return 0, _IOERR_SHMSIZE
		}
		if err := osAllocate(s.File, n); err != nil {
			return 0, sysError{err, _IOERR_SHMSIZE}
		}
	}

	r, err := util.MapRegion(ctx, mod, s.File, int64(id)*int64(size), size, s.readOnly)
	if err != nil {
		return 0, err
	}
	s.regions = append(s.regions, r)
	if s.readOnly {
		return r.Ptr, _READONLY
	}
	return r.Ptr, nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) error {
	// Argument check.
	switch {
	case n <= 0:
		panic(util.AssertErr())
	case offset < 0 || offset+n > _SHM_NLOCK:
		panic(util.AssertErr())
	case n != 1 && flags&_SHM_EXCLUSIVE == 0:
		panic(util.AssertErr())
	}
	switch flags {
	case
		_SHM_LOCK | _SHM_SHARED,
		_SHM_LOCK | _SHM_EXCLUSIVE,
		_SHM_UNLOCK | _SHM_SHARED,
		_SHM_UNLOCK | _SHM_EXCLUSIVE:
		//
	default:
		panic(util.AssertErr())
	}

	if s.File == nil {
		return _IOERR_SHMLOCK
	}

	var timeout time.Duration
	if s.blocking {
		timeout = time.Millisecond
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+int64(offset), int64(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+int64(offset), int64(n), timeout)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+int64(offset), int64(n), timeout)
	default:
		panic(util.AssertErr())
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
	if delete {
		os.Remove(s.path)
	}
	s.Close()
	s.File = nil
	s.fileLock = false
}

func (s *vfsShm) shmBarrier() {
	s.Lock()
	//lint:ignore SA2001 memory barrier.
	s.Unlock()
}

func (s *vfsShm) shmEnableBlocking(block bool) {
	s.blocking = block
}
