//go:build !(sqlite3_flock || sqlite3_nosys)

package vfs

import (
	"math/rand"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func osSync(file *os.File, _ /*fullsync*/, _ /*dataonly*/ bool) error {
	// SQLite trusts Linux's fdatasync for all fsync's.
	return unix.Fdatasync(int(file.Fd()))
}

func osAllocate(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	return unix.Fallocate(int(file.Fd()), 0, 0, size)
}

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
	switch {
	case timeout == 0:
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
	case timeout < 0:
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLKW, &lock)
	default:
		before := time.Now()
		for {
			err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
			if errno, _ := err.(unix.Errno); errno != unix.EAGAIN {
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

func osReadLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return osLock(file, unix.F_RDLCK, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len int64, timeout time.Duration) _ErrorCode {
	return osLock(file, unix.F_WRLCK, start, len, timeout, _IOERR_LOCK)
}
