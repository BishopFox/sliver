//go:build !(sqlite3_dotlk || sqlite3_nosys)

package vfs

import (
	"math/rand"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func osGetSharedLock(file *os.File) _ErrorCode {
	// Acquire the PENDING lock temporarily before acquiring a new SHARED lock.
	rc := osReadLock(file, _PENDING_BYTE, 1, 0)
	if rc == _OK {
		// Acquire the SHARED lock.
		rc = osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)

		// Release the PENDING lock.
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return rc
}

func osGetReservedLock(file *os.File) _ErrorCode {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, 0)
}

func osGetExclusiveLock(file *os.File, state *LockLevel) _ErrorCode {
	// A PENDING lock is needed before releasing the SHARED lock.
	if *state < LOCK_PENDING {
		// If we were RESERVED, we can block indefinitely.
		var timeout time.Duration
		if *state == LOCK_RESERVED {
			timeout = -1
		}
		if rc := osWriteLock(file, _PENDING_BYTE, 1, timeout); rc != _OK {
			return rc
		}
		*state = LOCK_PENDING
	}

	// Release the SHARED lock.
	osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, time.Millisecond)

	if rc != _OK {
		// Reacquire the SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// notest // this should never happen
			return _IOERR_RDLOCK
		}
	}
	return rc
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	if state >= LOCK_EXCLUSIVE {
		// Release the EXCLUSIVE lock while holding the PENDING lock.
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// notest // this should never happen
			return _IOERR_RDLOCK
		}
	}

	// Release the PENDING and RESERVED locks.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osReleaseLock(file *os.File, state LockLevel) _ErrorCode {
	// Release all locks, PENDING must be last.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_SHARED {
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	// Test the RESERVED lock.
	rc := osLock(file, 0, _RESERVED_BYTE, 1, 0, _IOERR_CHECKRESERVEDLOCK)
	if rc == _BUSY {
		return true, _OK
	}
	if rc == _OK {
		// Release the RESERVED lock.
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	return false, rc
}

func osUnlock(file *os.File, start, len uint32) _ErrorCode {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return _OK
	}
	if err != nil {
		return _IOERR_UNLOCK
	}
	return _OK
}

func osLock(file *os.File, flags, start, len uint32, timeout time.Duration, def _ErrorCode) _ErrorCode {
	var err error
	switch {
	case timeout == 0:
		err = osLockEx(file, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, start, len)
	case timeout < 0:
		err = osLockEx(file, flags, start, len)
	default:
		before := time.Now()
		for {
			err = osLockEx(file, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, start, len)
			if errno, _ := err.(windows.Errno); errno != windows.ERROR_LOCK_VIOLATION {
				break
			}
			if time.Since(before) > timeout {
				break
			}
			const sleepIncrement = 1024*1024 - 1 // power of two, ~1ms
			time.Sleep(time.Duration(rand.Int63() & sleepIncrement))
		}
	}
	return osLockErrorCode(err, def)
}

func osLockEx(file *os.File, flags, start, len uint32) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), flags,
		0, len, 0, &windows.Overlapped{Offset: start})
}

func osReadLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file, 0, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file, windows.LOCKFILE_EXCLUSIVE_LOCK, start, len, timeout, _IOERR_LOCK)
}

func osLockErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(windows.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_IO_PENDING,
			windows.ERROR_OPERATION_ABORTED:
			return _BUSY
		}
	}
	return def
}
