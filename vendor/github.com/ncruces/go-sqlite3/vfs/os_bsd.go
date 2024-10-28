//go:build (freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock) && !sqlite3_nosys

package vfs

import (
	"os"

	"golang.org/x/sys/unix"
)

func osGetSharedLock(file *os.File) _ErrorCode {
	return osLock(file, unix.LOCK_SH|unix.LOCK_NB, _IOERR_RDLOCK)
}

func osGetReservedLock(file *os.File) _ErrorCode {
	rc := osLock(file, unix.LOCK_EX|unix.LOCK_NB, _IOERR_LOCK)
	if rc == _BUSY {
		// The documentation states the lock is upgraded by releasing the previous lock,
		// then acquiring the new lock.
		// This is a race, so return BUSY_SNAPSHOT to ensure the transaction is aborted.
		return _BUSY_SNAPSHOT
	}
	return rc
}

func osGetExclusiveLock(file *os.File, state *LockLevel) _ErrorCode {
	if *state >= LOCK_RESERVED {
		return _OK
	}
	return osGetReservedLock(file)
}

func osDowngradeLock(file *os.File, _ LockLevel) _ErrorCode {
	rc := osLock(file, unix.LOCK_SH|unix.LOCK_NB, _IOERR_RDLOCK)
	if rc == _BUSY {
		// The documentation states the lock is upgraded by releasing the previous lock,
		// then acquiring the new lock.
		// This is a race, so return IOERR_RDLOCK to ensure the transaction is aborted.
		return _IOERR_RDLOCK
	}
	return _OK
}

func osReleaseLock(file *os.File, _ LockLevel) _ErrorCode {
	err := unix.Flock(int(file.Fd()), unix.LOCK_UN)
	if err != nil {
		return _IOERR_UNLOCK
	}
	return _OK
}

func osCheckReservedLock(file *os.File) (bool, _ErrorCode) {
	// Test the RESERVED lock.
	lock, rc := osTestLock(file, _RESERVED_BYTE, 1)
	return lock == unix.F_WRLCK, rc
}

func osLock(file *os.File, how int, def _ErrorCode) _ErrorCode {
	err := unix.Flock(int(file.Fd()), how)
	return osLockErrorCode(err, def)
}
