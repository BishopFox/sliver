//go:build freebsd || openbsd || netbsd || dragonfly || (darwin && sqlite3_bsd)

package sqlite3

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) unlock(file *os.File, start, len int64) xErrorCode {
	if start == 0 && len == 0 {
		err := unix.Flock(int(file.Fd()), unix.LOCK_UN)
		if err != nil {
			return IOERR_UNLOCK
		}
	}
	return _OK
}

func (vfsOSMethods) lock(file *os.File, how int, timeout time.Duration, def xErrorCode) xErrorCode {
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
	return vfsOS.lockErrorCode(err, def)
}

func (vfsOSMethods) readLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.LOCK_SH|unix.LOCK_NB, timeout, IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.LOCK_EX|unix.LOCK_NB, timeout, IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len int64) (bool, xErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), unix.F_GETLK, &lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}
