//go:build (darwin || linux || illumos) && (amd64 || arm64 || riscv64) && !sqlite3_flock && !sqlite3_noshm && !sqlite3_nosys

package vfs

import (
	"context"
	"io"
	"os"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = true

type vfsShm struct {
	*os.File
	regions []*util.MappedRegion
}

const (
	_SHM_NLOCK = 8
	_SHM_BASE  = 120
	_SHM_DMS   = _SHM_BASE + _SHM_NLOCK
)

func (f *vfsFile) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, error) {
	// Ensure size is a multiple of the OS page size.
	if int(size)&(unix.Getpagesize()-1) != 0 {
		return 0, _IOERR_SHMMAP
	}

	if f.shm.File == nil {
		var flag int
		if f.readOnly {
			flag = unix.O_RDONLY
		} else {
			flag = unix.O_RDWR
		}
		s, err := os.OpenFile(f.Name()+"-shm",
			flag|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
		if err != nil {
			return 0, _CANTOPEN
		}
		f.shm.File = s
	}

	// Dead man's switch.
	if lock, rc := osGetLock(f.shm.File, _SHM_DMS, 1); rc != _OK {
		return 0, _IOERR_LOCK
	} else if lock == unix.F_WRLCK {
		return 0, _BUSY
	} else if lock == unix.F_UNLCK {
		if f.readOnly {
			return 0, _READONLY_CANTINIT
		}
		if rc := osWriteLock(f.shm.File, _SHM_DMS, 1, 0); rc != _OK {
			return 0, rc
		}
		if err := f.shm.Truncate(0); err != nil {
			return 0, _IOERR_SHMOPEN
		}
	}
	if rc := osReadLock(f.shm.File, _SHM_DMS, 1, 0); rc != _OK {
		return 0, rc
	}

	// Check if file is big enough.
	s, err := f.shm.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > s {
		if !extend {
			return 0, nil
		}
		err := osAllocate(f.shm.File, n)
		if err != nil {
			return 0, _IOERR_SHMSIZE
		}
	}

	var prot int
	if f.readOnly {
		prot = unix.PROT_READ
	} else {
		prot = unix.PROT_READ | unix.PROT_WRITE
	}
	r, err := util.MapRegion(ctx, mod, f.shm.File, int64(id)*int64(size), size, prot)
	if err != nil {
		return 0, err
	}
	f.shm.regions = append(f.shm.regions, r)
	return r.Ptr, nil
}

func (f *vfsFile) shmLock(offset, n int32, flags _ShmFlag) error {
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

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(f.shm.File, _SHM_BASE+int64(offset), int64(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(f.shm.File, _SHM_BASE+int64(offset), int64(n), 0)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(f.shm.File, _SHM_BASE+int64(offset), int64(n), 0)
	default:
		panic(util.AssertErr())
	}
}

func (f *vfsFile) shmUnmap(delete bool) {
	// Unmap regions.
	for _, r := range f.shm.regions {
		r.Unmap()
	}
	clear(f.shm.regions)
	f.shm.regions = f.shm.regions[:0]

	// Close the file.
	if delete && f.shm.File != nil {
		os.Remove(f.shm.Name())
	}
	f.shm.Close()
	f.shm.File = nil
}
