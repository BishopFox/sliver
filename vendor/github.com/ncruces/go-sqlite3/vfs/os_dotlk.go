//go:build sqlite3_dotlk

package vfs

import (
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/dotlk"
)

var (
	// +checklocks:vfsDotLocksMtx
	vfsDotLocks    = map[string]*vfsDotLocker{}
	vfsDotLocksMtx sync.Mutex
)

type vfsDotLocker struct {
	shared   int      // +checklocks:vfsDotLocksMtx
	pending  *os.File // +checklocks:vfsDotLocksMtx
	reserved *os.File // +checklocks:vfsDotLocksMtx
}

func osGetSharedLock(file *os.File) error {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		if err := dotlk.TryLock(name + ".lock"); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return _BUSY // Another process has the lock.
			}
			return sysError{err, _IOERR_LOCK}
		}
		locker = &vfsDotLocker{}
		vfsDotLocks[name] = locker
	}

	if locker.pending != nil {
		return _BUSY
	}
	locker.shared++
	return nil
}

func osGetReservedLock(file *os.File) error {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_LOCK
	}

	if locker.reserved != nil && locker.reserved != file {
		return _BUSY
	}
	locker.reserved = file
	return nil
}

func osGetExclusiveLock(file *os.File, _ *LockLevel) error {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_LOCK
	}

	if locker.pending != nil && locker.pending != file {
		return _BUSY
	}
	locker.pending = file
	if locker.shared > 1 {
		return _BUSY
	}
	return nil
}

func osDowngradeLock(file *os.File, _ LockLevel) error {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_UNLOCK
	}

	if locker.reserved == file {
		locker.reserved = nil
	}
	if locker.pending == file {
		locker.pending = nil
	}
	return nil
}

func osReleaseLock(file *os.File, state LockLevel) error {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_UNLOCK
	}

	if locker.shared == 1 {
		if err := dotlk.Unlock(name + ".lock"); err != nil {
			return sysError{err, _IOERR_UNLOCK}
		}
		delete(vfsDotLocks, name)
	}

	if locker.reserved == file {
		locker.reserved = nil
	}
	if locker.pending == file {
		locker.pending = nil
	}
	locker.shared--
	return nil
}

func osCheckReservedLock(file *os.File) (bool, error) {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	return locker != nil && locker.reserved != nil, nil
}
