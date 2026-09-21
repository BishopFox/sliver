//go:build !sqlite3_flock

package vfs

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

func osCreateTemp(flags OpenFlag) (*os.File, error) {
	dir := os.Getenv("SQLITE_TMPDIR")
	if dir == "" {
		dir = os.TempDir()
	}

	for {
		fd, err := unix.Open(dir, unix.O_RDWR|unix.O_EXCL|unix.O_TMPFILE|unix.O_CLOEXEC, 0600)
		if err == nil {
			path := filepath.Join(dir, "tmp.db")
			return os.NewFile(uintptr(fd), path), nil
		}
		if err == unix.EISDIR || err == unix.EOPNOTSUPP {
			break
		}
		if err != unix.EINTR {
			return nil, sysError{err, _IOERR_GETTEMPPATH}
		}
	}

	f, err := os.CreateTemp(dir, "*.db")
	if err != nil {
		return nil, sysError{err, _IOERR_GETTEMPPATH}
	}
	if flags&OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}
	return f, nil
}

func osSync(file *os.File, _ OpenFlag, _ SyncFlag) error {
	// SQLite trusts Linux's fdatasync for all fsync's.
	for {
		err := unix.Fdatasync(int(file.Fd()))
		if err != unix.EINTR {
			return err
		}
	}
}

func osAllocate(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	end, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if end >= size {
		return nil
	}

	for {
		err := unix.Fallocate(int(file.Fd()), 0, end, size-end)
		if err == unix.EOPNOTSUPP {
			break
		}
		if err != unix.EINTR {
			return err
		}
	}
	return file.Truncate(size)
}

func osReadLock(file *os.File, start, len int64, timeout time.Duration) error {
	return osLock(file, unix.F_RDLCK, start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len int64, timeout time.Duration) error {
	return osLock(file, unix.F_WRLCK, start, len, timeout, _IOERR_LOCK)
}

func osLock(file *os.File, typ int16, start, len int64, timeout time.Duration, def _ErrorCode) error {
	lock := unix.Flock_t{
		Type:  typ,
		Start: start,
		Len:   len,
	}
	var err error
	switch {
	default:
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
	case timeout < 0:
		err = unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLKW, &lock)
	}
	return osLockErrorCode(err, def)
}

func osUnlock(file *os.File, start, len int64) error {
	lock := unix.Flock_t{
		Type:  unix.F_UNLCK,
		Start: start,
		Len:   len,
	}
	for {
		err := unix.FcntlFlock(file.Fd(), unix.F_OFD_SETLK, &lock)
		if err == nil {
			return nil
		}
		if err != unix.EINTR {
			return sysError{err, _IOERR_UNLOCK}
		}
	}
}
