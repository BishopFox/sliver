//go:build (linux || darwin || freebsd || openbsd || netbsd || dragonfly || illumos) && !sqlite3_nosys

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func osGetSharedLock(file *os.File) _ErrorCode {
	// Test the PENDING lock before acquiring a new SHARED lock.
	if pending, _ := osCheckLock(file, _PENDING_BYTE, 1); pending {
		return _BUSY
	}
	// Acquire the SHARED lock.
	return osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)
}

func osGetReservedLock(file *os.File) _ErrorCode {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, 0)
}

func osGetPendingLock(file *os.File) _ErrorCode {
	// Acquire the PENDING lock.
	return osWriteLock(file, _PENDING_BYTE, 1, 0)
}

func osGetExclusiveLock(file *os.File) _ErrorCode {
	// Acquire the EXCLUSIVE lock.
	return osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, time.Millisecond)
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	if state >= LOCK_EXCLUSIVE {
		// Downgrade to a SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// In theory, the downgrade to a SHARED cannot fail because another
			// process is holding an incompatible lock. If it does, this
			// indicates that the other process is not following the locking
			// protocol. If this happens, return _IOERR_RDLOCK. Returning
			// BUSY would confuse the upper layer.
			return _IOERR_RDLOCK
		}
	}
	// Release the PENDING and RESERVED locks.
	return osUnlock(file, _PENDING_BYTE, 2)
}

func osReleaseLock(file *os.File, _ LockLevel) _ErrorCode {
	// Release all locks.
	return osUnlock(file, 0, 0)
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	// Test the RESERVED lock.
	return osCheckLock(file, _RESERVED_BYTE, 1)
}

func osLockErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(unix.Errno); ok {
		switch errno {
		case
			unix.EACCES,
			unix.EAGAIN,
			unix.EBUSY,
			unix.EINTR,
			unix.ENOLCK,
			unix.EDEADLK,
			unix.ETIMEDOUT:
			return _BUSY
		case unix.EPERM:
			return _PERM
		}
	}
	return def
}
