//go:build sqlite3_dotlk

package vfs

import (
	"errors"
	"io/fs"
	"os"
	"sync"
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

func osGetSharedLock(file *os.File) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		f, err := os.OpenFile(name+".lock", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		f.Close()
		if errors.Is(err, fs.ErrExist) {
			return _BUSY // Another process has the lock.
		}
		if err != nil {
			return _IOERR_LOCK
		}
		locker = &vfsDotLocker{}
		vfsDotLocks[name] = locker
	}

	if locker.pending != nil {
		return _BUSY
	}
	locker.shared++
	return _OK
}

func osGetReservedLock(file *os.File) _ErrorCode {
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
	return _OK
}

func osGetExclusiveLock(file *os.File, _ *LockLevel) _ErrorCode {
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
	return _OK
}

func osDowngradeLock(file *os.File, _ LockLevel) _ErrorCode {
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
	return _OK
}

func osReleaseLock(file *os.File, state LockLevel) _ErrorCode {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return _IOERR_UNLOCK
	}

	if locker.shared == 1 {
		err := os.Remove(name + ".lock")
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return _IOERR_UNLOCK
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
	return _OK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	vfsDotLocksMtx.Lock()
	defer vfsDotLocksMtx.Unlock()

	name := file.Name()
	locker := vfsDotLocks[name]
	if locker == nil {
		return false, _OK
	}
	return locker.reserved != nil, _OK
}
