//go:build unix

package vfs

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	isUnix      = true
	_O_NOFOLLOW = unix.O_NOFOLLOW
)

func osAccess(path string, flags AccessFlag) error {
	var access uint32 // unix.F_OK
	switch flags {
	case ACCESS_READWRITE:
		access = unix.R_OK | unix.W_OK
	case ACCESS_READ:
		access = unix.R_OK
	}
	return unix.Access(path, access)
}

func osReadAt(file *os.File, p []byte, off int64) (int, error) {
	n, err := file.ReadAt(p, off)
	if errno, ok := err.(unix.Errno); ok {
		switch errno {
		case
			unix.ERANGE,
			unix.EIO,
			unix.ENXIO:
			return n, sysError{err, _IOERR_CORRUPTFS}
		}
	}
	return n, err
}

func osWriteAt(file *os.File, p []byte, off int64) (int, error) {
	n, err := file.WriteAt(p, off)
	if errno, ok := err.(unix.Errno); ok && errno == unix.ENOSPC {
		return n, sysError{err, _FULL}
	}
	return n, err
}

func osSetMode(file *os.File, modeof string) error {
	fi, err := os.Stat(modeof)
	if err != nil {
		return err
	}
	file.Chmod(fi.Mode())
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		file.Chown(int(sys.Uid), int(sys.Gid))
	}
	return nil
}

func osTestLock(file *os.File, start, len int64, def _ErrorCode) (int16, error) {
	lock := unix.Flock_t{
		Type:  unix.F_WRLCK,
		Start: start,
		Len:   len,
	}
	for {
		err := unix.FcntlFlock(file.Fd(), unix.F_GETLK, &lock)
		if err == nil {
			return lock.Type, nil
		}
		if err != unix.EINTR {
			return 0, sysError{err, def}
		}
	}
}

func osLockErrorCode(err error, def _ErrorCode) error {
	if err == nil {
		return nil
	}
	if errno, ok := err.(unix.Errno); ok {
		switch errno {
		case
			unix.EACCES,
			unix.EAGAIN,
			unix.EBUSY,
			unix.EINTR,
			unix.ENOLCK,
			unix.EDEADLK,
			unix.ETIMEDOUT:
			return _BUSY
		case unix.EPERM:
			return sysError{err, _PERM}
		}
		// notest // usually EWOULDBLOCK == EAGAIN
		if errno == unix.EWOULDBLOCK && unix.EWOULDBLOCK != unix.EAGAIN {
			return _BUSY
		}
	}
	return sysError{err, def}
}
