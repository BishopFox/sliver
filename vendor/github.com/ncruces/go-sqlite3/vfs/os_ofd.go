//go:build linux || illumos

package vfs

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func osUnlock(file *os.File, start, len int64) _ErrorCode {
	err := unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return _IOERR_UNLOCK
	}
	return _OK
}

func osLock(file *os.File, typ int16, start, len int64, timeout time.Duration, def _ErrorCode) _ErrorCode {
	lock := unix.Flock_t{
		Type:  typ,
		Start: start,
		Len:   len,
	}
	var err error
	for {
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
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
	return osLock(file, unix.F_RDLCK, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return osLock(file, unix.F_WRLCK, start, len, timeout, _IOERR_LOCK)
}

func osCheckLock(file *os.File, start, len int64) (bool, _ErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), unix.F_OFD_GETLK, &lock) != nil {
		return false, _IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}
