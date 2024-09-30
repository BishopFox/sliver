//go:build (darwin || linux) && (amd64 || arm64 || riscv64 || ppc64le) && !(sqlite3_flock || sqlite3_noshm || sqlite3_nosys)

package vfs

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

// SupportsSharedMemory is false on platforms that do not support shared memory.
// To use [WAL without shared-memory], you need to set [EXCLUSIVE locking mode].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = true

const (
	_SHM_NLOCK = 8
	_SHM_BASE  = 120
	_SHM_DMS   = _SHM_BASE + _SHM_NLOCK
)

func (f *vfsFile) SharedMemory() SharedMemory { return f.shm }

// NewSharedMemory returns a shared-memory WAL-index
// backed by a file with the given path.
// It will return nil if shared-memory is not supported,
// or not appropriate for the given flags.
// Only [OPEN_MAIN_DB] databases may need a WAL-index.
// You must ensure all concurrent accesses to a database
// use shared-memory instances created with the same path.
func NewSharedMemory(path string, flags OpenFlag) SharedMemory {
	if flags&OPEN_MAIN_DB == 0 || flags&(OPEN_DELETEONCLOSE|OPEN_MEMORY) != 0 {
		return nil
	}
	return &vfsShm{
		path:     path,
		readOnly: flags&OPEN_READONLY != 0,
	}
}

type vfsShm struct {
	*os.File
	path     string
	regions  []*util.MappedRegion
	readOnly bool
	blocking bool
}

func (s *vfsShm) shmOpen() _ErrorCode {
	if s.File == nil {
		var flag int
		if s.readOnly {
			flag = unix.O_RDONLY
		} else {
			flag = unix.O_RDWR
		}
		f, err := os.OpenFile(s.path,
			flag|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
		if err != nil {
			return _CANTOPEN
		}
		s.File = f
	}

	// Dead man's switch.
	if lock, rc := osTestLock(s.File, _SHM_DMS, 1); rc != _OK {
		return _IOERR_LOCK
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
		if rc := osWriteLock(s.File, _SHM_DMS, 1, 0); rc != _OK {
			return rc
		}
		if err := s.Truncate(0); err != nil {
			return _IOERR_SHMOPEN
		}
	}
	if rc := osReadLock(s.File, _SHM_DMS, 1, time.Millisecond); rc != _OK {
		return rc
	}
	return _OK
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, _ErrorCode) {
	// Ensure size is a multiple of the OS page size.
	if int(size)&(unix.Getpagesize()-1) != 0 {
		return 0, _IOERR_SHMMAP
	}

	if rc := s.shmOpen(); rc != _OK {
		return 0, rc
	}

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, _OK
		}
		err := osAllocate(s.File, n)
		if err != nil {
			return 0, _IOERR_SHMSIZE
		}
	}

	var prot int
	if s.readOnly {
		prot = unix.PROT_READ
	} else {
		prot = unix.PROT_READ | unix.PROT_WRITE
	}
	r, err := util.MapRegion(ctx, mod, s.File, int64(id)*int64(size), size, prot)
	if err != nil {
		return 0, _IOERR_SHMMAP
	}
	s.regions = append(s.regions, r)
	if s.readOnly {
		return r.Ptr, _READONLY
	}
	return r.Ptr, _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) _ErrorCode {
	// Argument check.
	if n <= 0 || offset < 0 || offset+n > _SHM_NLOCK {
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
	if n != 1 && flags&_SHM_EXCLUSIVE == 0 {
		panic(util.AssertErr())
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
	clear(s.regions)
	s.regions = s.regions[:0]

	// Close the file.
	if delete {
		os.Remove(s.path)
	}
	s.Close()
	s.File = nil
}

func (s *vfsShm) shmEnableBlocking(block bool) {
	s.blocking = block
}
