//go:build ((freebsd || openbsd || netbsd || dragonfly || illumos) && (386 || arm || amd64 || arm64 || riscv64 || ppc64le) && !sqlite3_dotlk) || sqlite3_flock

package vfs

import (
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"

	"github.com/ncruces/go-sqlite3/internal/util"
)

type vfsShmParent struct {
	*os.File
	info os.FileInfo

	refs int // +checklocks:vfsShmListMtx

	lock [_SHM_NLOCK]int8 // +checklocks:Mutex
	sync.Mutex
}

var (
	// +checklocks:vfsShmListMtx
	vfsShmList    []*vfsShmParent
	vfsShmListMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmParent
	path    string
	lock    [_SHM_NLOCK]bool
	regions []*util.MappedRegion
}

func (s *vfsShm) Close() error {
	if s.vfsShmParent == nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	// Decrease reference count.
	if s.vfsShmParent.refs > 0 {
		s.vfsShmParent.refs--
		s.vfsShmParent = nil
		return nil
	}

	err := s.File.Close()
	for i, g := range vfsShmList {
		if g == s.vfsShmParent {
			vfsShmList[i] = nil
			s.vfsShmParent = nil
			return err
		}
	}
	panic(util.AssertErr())
}

func (s *vfsShm) shmOpen() (err error) {
	if s.vfsShmParent != nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Stat file without opening it.
	// Closing it would release all POSIX locks on it.
	fi, err := os.Stat(s.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return sysError{err, _IOERR_FSTAT}
	}

	// Find a shared file, increase the reference count.
	for _, g := range vfsShmList {
		if g != nil && os.SameFile(fi, g.info) {
			s.vfsShmParent = g
			g.refs++
			return nil
		}
	}

	// Always open file read-write, as it will be shared.
	f, err := os.OpenFile(s.path,
		os.O_RDWR|os.O_CREATE|_O_NOFOLLOW, 0666)
	if err != nil {
		return sysError{err, _CANTOPEN}
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()

	// Dead man's switch.
	if lock, err := osTestLock(f, _SHM_DMS, 1, _IOERR_LOCK); err != nil {
		return err
	} else if lock == unix.F_WRLCK {
		return _BUSY
	} else if lock == unix.F_UNLCK {
		if err := osWriteLock(f, _SHM_DMS, 1); err != nil {
			return err
		}
		if err := f.Truncate(0); err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	if err := osReadLock(f, _SHM_DMS, 1); err != nil {
		return err
	}

	fi, err = f.Stat()
	if err != nil {
		return sysError{err, _IOERR_FSTAT}
	}

	// Add the new shared file.
	s.vfsShmParent = &vfsShmParent{
		File: f,
		info: fi,
	}
	for i, g := range vfsShmList {
		if g == nil {
			vfsShmList[i] = s.vfsShmParent
			return nil
		}
	}
	vfsShmList = append(vfsShmList, s.vfsShmParent)
	return nil
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
		if err := osAllocate(s.File, n); err != nil {
			return 0, sysError{err, _IOERR_SHMSIZE}
		}
	}

	r, err := util.MapRegion(ctx, mod, s.File, int64(id)*int64(size), size, false)
	if err != nil {
		return 0, err
	}
	s.regions = append(s.regions, r)
	return r.Ptr, nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) error {
	if s.vfsShmParent == nil {
		return _IOERR_SHMLOCK
	}

	s.Lock()
	defer s.Unlock()

	// Check if we can obtain/release locks locally.
	err := s.shmMemLock(offset, n, flags)
	if err != nil {
		return err
	}

	// Obtain/release the appropriate file locks.
	switch {
	case flags&_SHM_UNLOCK != 0:
		// Relasing a shared lock decrements the counter,
		// but may leave parts of the range still locked.
		begin, end := offset, offset+n
		for i := begin; i < end; i++ {
			if s.vfsShmParent.lock[i] != 0 {
				if i > begin {
					err = cmp.Or(err,
						osUnlock(s.File, _SHM_BASE+int64(begin), int64(i-begin)))
				}
				begin = i + 1
			}
		}
		if end > begin {
			err = cmp.Or(err,
				osUnlock(s.File, _SHM_BASE+int64(begin), int64(end-begin)))
		}
		return err
	case flags&_SHM_SHARED != 0:
		// Acquiring a new shared lock on the file is only necessary
		// if there was a new shared lock in the range.
		for i := offset; i < offset+n; i++ {
			if s.vfsShmParent.lock[i] == 1 {
				err = osReadLock(s.File, _SHM_BASE+int64(offset), int64(n))
				break
			}
		}
	case flags&_SHM_EXCLUSIVE != 0:
		// Acquiring an exclusive lock on the file is always necessary.
		err = osWriteLock(s.File, _SHM_BASE+int64(offset), int64(n))
	default:
		panic(util.AssertErr())
	}

	if err != nil {
		// Release the local locks we had acquired.
		s.shmMemLock(offset, n, flags^(_SHM_UNLOCK|_SHM_LOCK))
	}
	return err
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmParent == nil {
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
}

func (s *vfsShm) shmBarrier() {
	s.Lock()
	//lint:ignore SA2001 memory barrier.
	s.Unlock()
}
