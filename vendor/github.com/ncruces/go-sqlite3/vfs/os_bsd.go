//go:build (freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock) && !sqlite3_nosys

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func osUnlock(file *os.File, start, len int64) _ErrorCode {
	if start == 0 && len == 0 {
		err := unix.Flock(int(file.Fd()), unix.LOCK_UN)
		if err != nil {
			return _IOERR_UNLOCK
		}
	}
	return _OK
}

func osLock(file *os.File, how int, def _ErrorCode) _ErrorCode {
	err := unix.Flock(int(file.Fd()), how)
	return osLockErrorCode(err, def)
}

func osReadLock(file *os.File, _ /*start*/, _ /*len*/ int64, _ /*timeout*/ time.Duration) _ErrorCode {
	return osLock(file, unix.LOCK_SH|unix.LOCK_NB, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, _ /*start*/, _ /*len*/ int64, _ /*timeout*/ time.Duration) _ErrorCode {
	rc := osLock(file, unix.LOCK_EX|unix.LOCK_NB, _IOERR_LOCK)
	if rc == _BUSY {
		// The documentation states the lock is upgraded by releasing the previous lock,
		// then acquiring the new lock.
		// This is a race, so return BUSY_SNAPSHOT to ensure the transaction is aborted.
		return _BUSY_SNAPSHOT
	}
	return rc
}
