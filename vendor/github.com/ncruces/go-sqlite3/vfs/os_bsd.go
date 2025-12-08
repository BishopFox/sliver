//go:build ((freebsd || openbsd || netbsd || dragonfly || illumos) && !sqlite3_dotlk) || sqlite3_flock

package vfs

import (
	"os"

	"golang.org/x/sys/unix"
)

func osGetSharedLock(file *os.File) error {
	return osFlock(file, unix.LOCK_SH|unix.LOCK_NB, _IOERR_RDLOCK)
}

func osGetReservedLock(file *os.File) error {
	err := osFlock(file, unix.LOCK_EX|unix.LOCK_NB, _IOERR_LOCK)
	if err == _BUSY {
		// The documentation states that a lock is upgraded by
		// releasing the previous lock, then acquiring the new lock.
		// Going over the source code of various BSDs, though,
		// with LOCK_NB, the lock is not released,
		// and EAGAIN is returned holding the shared lock.
		// Still, if we're already in a transaction, we want to abort it,
		// so return BUSY_SNAPSHOT here. If there's no transaction active,
		// SQLite will change this back to SQLITE_BUSY,
		// and invoke the busy handler if appropriate.
		return _BUSY_SNAPSHOT
	}
	return err
}

func osGetExclusiveLock(file *os.File, state *LockLevel) error {
	if *state >= LOCK_RESERVED {
		return nil
	}
	return osGetReservedLock(file)
}

func osDowngradeLock(file *os.File, _ LockLevel) error {
	err := osFlock(file, unix.LOCK_SH|unix.LOCK_NB, _IOERR_RDLOCK)
	if err == _BUSY {
		// The documentation states that a lock is downgraded by
		// releasing the previous lock then acquiring the new lock.
		// Going over the source code of various BSDs, though,
		// with LOCK_SH|LOCK_NB this should never happen.
		// Return IOERR_RDLOCK, as BUSY would cause an assert to fail.
		return _IOERR_RDLOCK
	}
	return err
}

func osReleaseLock(file *os.File, _ LockLevel) error {
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_UN)
		if err == nil {
			return nil
		}
		if err != unix.EINTR {
			return sysError{err, _IOERR_UNLOCK}
		}
	}
}

func osCheckReservedLock(file *os.File) (bool, error) {
	// Test the RESERVED lock with fcntl(F_GETLK).
	// This only works on systems where fcntl and flock are compatible.
	// However, SQLite only calls this while holding a shared lock,
	// so the difference is immaterial.
	lock, err := osTestLock(file, _RESERVED_BYTE, 1, _IOERR_CHECKRESERVEDLOCK)
	return lock == unix.F_WRLCK, err
}

func osFlock(file *os.File, how int, def _ErrorCode) error {
	err := unix.Flock(int(file.Fd()), how)
	return osLockErrorCode(err, def)
}

func osReadLock(file *os.File, start, len int64) error {
	return osLock(file, unix.F_RDLCK, start, len, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len int64) error {
	return osLock(file, unix.F_WRLCK, start, len, _IOERR_LOCK)
}

func osLock(file *os.File, typ int16, start, len int64, def _ErrorCode) error {
	err := unix.FcntlFlock(file.Fd(), unix.F_SETLK, &unix.Flock_t{
		Type:  typ,
		Start: start,
		Len:   len,
	})
	return osLockErrorCode(err, def)
}

func osUnlock(file *os.File, start, len int64) error {
	lock := unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	}
	for {
		err := unix.FcntlFlock(file.Fd(), unix.F_SETLK, &lock)
		if err == nil {
			return nil
		}
		if err != unix.EINTR {
			return sysError{err, _IOERR_UNLOCK}
		}
	}
}
