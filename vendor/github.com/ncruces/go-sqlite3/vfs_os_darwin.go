//go:build !sqlite3_bsd

package sqlite3

import (
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// https://github.com/apple/darwin-xnu/blob/main/bsd/sys/fcntl.h
	_F_OFD_SETLK         = 90
	_F_OFD_SETLKW        = 91
	_F_OFD_GETLK         = 92
	_F_OFD_SETLKWTIMEOUT = 93
)

type flocktimeout_t struct {
	fl      unix.Flock_t
	timeout unix.Timespec
}

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if fullsync {
		return file.Sync()
	}
	return unix.Fsync(int(file.Fd()))
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if size <= off {
		return nil
	}

	// https://stackoverflow.com/a/11497568/867786
	store := unix.Fstore_t{
		Flags:   unix.F_ALLOCATECONTIG,
		Posmode: unix.F_PEOFPOSMODE,
		Offset:  0,
		Length:  size,
	}

	// Try to get a continous chunk of disk space.
	err = unix.FcntlFstore(file.Fd(), unix.F_PREALLOCATE, &store)
	if err != nil {
		// OK, perhaps we are too fragmented, allocate non-continuous.
		store.Flags = unix.F_ALLOCATEALL
		unix.FcntlFstore(file.Fd(), unix.F_PREALLOCATE, &store)
	}
	return file.Truncate(size)
}

func (vfsOSMethods) unlock(file *os.File, start, len int64) xErrorCode {
	err := unix.FcntlFlock(file.Fd(), _F_OFD_SETLK, &unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	})
	if err != nil {
		return IOERR_UNLOCK
	}
	return _OK
}

func (vfsOSMethods) lock(file *os.File, typ int16, start, len int64, timeout time.Duration, def xErrorCode) xErrorCode {
	lock := flocktimeout_t{fl: unix.Flock_t{
		Type:  typ,
		Start: start,
		Len:   len,
	}}
	var err error
	if timeout == 0 {
		err = unix.FcntlFlock(file.Fd(), _F_OFD_SETLK, &lock.fl)
	} else {
		lock.timeout = unix.NsecToTimespec(int64(timeout / time.Nanosecond))
		err = unix.FcntlFlock(file.Fd(), _F_OFD_SETLKWTIMEOUT, &lock.fl)
	}
	return vfsOS.lockErrorCode(err, def)
}

func (vfsOSMethods) readLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.F_RDLCK, start, len, timeout, IOERR_RDLOCK)
}

func (vfsOSMethods) writeLock(file *os.File, start, len int64, timeout time.Duration) xErrorCode {
	return vfsOS.lock(file, unix.F_WRLCK, start, len, timeout, IOERR_LOCK)
}

func (vfsOSMethods) checkLock(file *os.File, start, len int64) (bool, xErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_RDLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), _F_OFD_GETLK, &lock) != nil {
		return false, IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type != unix.F_UNLCK, _OK
}
