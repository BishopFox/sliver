//go:build freebsd || openbsd || netbsd || dragonfly || (darwin && sqlite3_bsd)

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

func osLock(file *os.File, how int, timeout time.Duration, def _ErrorCode) _ErrorCode {
	var err error
	for {
		err = unix.Flock(int(file.Fd()), how)
		if errno, _ := err.(unix.Errno); errno != unix.EAGAIN {
			break
		}
		if timeout < time.Millisecond {
			break
		}
		timeout -= time.Millisecond
		time.Sleep(time.Millisecond)
	}
	return osLockErrorCode(err, def)
}

func osReadLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return osLock(file, unix.LOCK_SH|unix.LOCK_NB, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return osLock(file, unix.LOCK_EX|unix.LOCK_NB, timeout, _IOERR_LOCK)
}

func osCheckLock(file *os.File, start, len int64) (bool, _ErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), unix.F_GETLK, &lock) != nil {
		return false, _IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}
