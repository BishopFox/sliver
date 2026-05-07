//go:build (linux || darwin) && !(sqlite3_flock || sqlite3_dotlk)

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func osGetSharedLock(file *os.File) error {
	// Test the PENDING lock before acquiring a new SHARED lock.
	if lock, _ := osTestLock(file, _PENDING_BYTE, 1, _IOERR); lock == unix.F_WRLCK {
		return _BUSY
	}
	// Acquire the SHARED lock.
	return osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)
}

func osGetReservedLock(file *os.File) error {
	// Acquire the RESERVED lock.
	return osWriteLock(file, _RESERVED_BYTE, 1, 0)
}

func osGetExclusiveLock(file *os.File, state *LockLevel) error {
	if *state == LOCK_RESERVED {
		// A PENDING lock is needed before acquiring an EXCLUSIVE lock.
		if err := osWriteLock(file, _PENDING_BYTE, 1, -1); err != nil {
			return err
		}
		*state = LOCK_PENDING
	}
	// Acquire the EXCLUSIVE lock.
	return osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, time.Millisecond)
}

func osDowngradeLock(file *os.File, state LockLevel) error {
	if state >= LOCK_EXCLUSIVE {
		// Downgrade to a SHARED lock.
		if err := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); err != nil {
			// notest // this should never happen
			return _IOERR_RDLOCK
		}
	}
	// Release the PENDING and RESERVED locks.
	return osUnlock(file, _PENDING_BYTE, 2)
}

func osReleaseLock(file *os.File, _ LockLevel) error {
	// Release all locks.
	return osUnlock(file, 0, 0)
}

func osCheckReservedLock(file *os.File) (bool, error) {
	// Test the RESERVED lock.
	lock, err := osTestLock(file, _RESERVED_BYTE, 1, _IOERR_CHECKRESERVEDLOCK)
	return lock == unix.F_WRLCK, err
}
