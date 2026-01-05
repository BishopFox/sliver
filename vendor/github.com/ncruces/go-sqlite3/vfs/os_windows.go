//go:build !sqlite3_dotlk

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func osReadAt(file *os.File, p []byte, off int64) (int, error) {
	return file.ReadAt(p, off)
}

func osWriteAt(file *os.File, p []byte, off int64) (int, error) {
	n, err := file.WriteAt(p, off)
	if errno, ok := err.(windows.Errno); ok {
		switch errno {
		case
			windows.ERROR_HANDLE_DISK_FULL,
			windows.ERROR_DISK_FULL:
			return n, sysError{err, _FULL}
		}
	}
	return n, err
}

func osGetSharedLock(file *os.File) error {
	// Acquire the PENDING lock temporarily before acquiring a new SHARED lock.
	err := osReadLock(file, _PENDING_BYTE, 1, 0)
	if err == nil {
		// Acquire the SHARED lock.
		err = osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)

		// Release the PENDING lock.
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return err
}

func osGetReservedLock(file *os.File) error {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, 0)
}

func osGetExclusiveLock(file *os.File, state *LockLevel) error {
	// A PENDING lock is needed before releasing the SHARED lock.
	if *state < LOCK_PENDING {
		// If we were RESERVED, we can block indefinitely.
		var timeout time.Duration
		if *state == LOCK_RESERVED {
			timeout = -1
		}
		if err := osWriteLock(file, _PENDING_BYTE, 1, timeout); err != nil {
			return err
		}
		*state = LOCK_PENDING
	}

	// Release the SHARED lock.
	osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	// Can't wait here, because the file is not OVERLAPPED.
	err := osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)

	if err != nil {
		// Reacquire the SHARED lock.
		if err := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); err != nil {
			// notest // this should never happen
			return _IOERR_RDLOCK
		}
	}
	return err
}

func osDowngradeLock(file *os.File, state LockLevel) error {
	if state >= LOCK_EXCLUSIVE {
		// Release the EXCLUSIVE lock while holding the PENDING lock.
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if err := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); err != nil {
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
	return nil
}

func osReleaseLock(file *os.File, state LockLevel) error {
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
	return nil
}

func osCheckReservedLock(file *os.File) (bool, error) {
	// Test the RESERVED lock.
	err := osLock(file, 0, _RESERVED_BYTE, 1, 0, _IOERR_CHECKRESERVEDLOCK)
	if err == _BUSY {
		return true, nil
	}
	if err == nil {
		// Release the RESERVED lock.
		osUnlock(file, _RESERVED_BYTE, 1)
		return false, nil
	}
	return false, err
}

func osReadLock(file *os.File, start, len uint32, timeout time.Duration) error {
	return osLock(file, 0, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len uint32, timeout time.Duration) error {
	return osLock(file, windows.LOCKFILE_EXCLUSIVE_LOCK, start, len, timeout, _IOERR_LOCK)
}

func osLock(file *os.File, flags, start, len uint32, timeout time.Duration, def _ErrorCode) error {
	var err error
	switch {
	default:
		err = osLockEx(file, flags|windows.LOCKFILE_FAIL_IMMEDIATELY, start, len)
	case timeout < 0:
		err = osLockEx(file, flags, start, len)
	}
	return osLockErrorCode(err, def)
}

func osUnlock(file *os.File, start, len uint32) error {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return nil
	}
	if err != nil {
		return sysError{err, _IOERR_UNLOCK}
	}
	return nil
}

func osLockEx(file *os.File, flags, start, len uint32) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), flags,
		0, len, 0, &windows.Overlapped{Offset: start})
}

func osLockErrorCode(err error, def _ErrorCode) error {
	if err == nil {
		return nil
	}
	if errno, ok := err.(windows.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_OPERATION_ABORTED,
			windows.ERROR_IO_PENDING,
			windows.WAIT_TIMEOUT:
			return _BUSY
		}
	}
	return sysError{err, def}
}
