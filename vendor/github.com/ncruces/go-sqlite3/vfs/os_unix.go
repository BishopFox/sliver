//go:build unix && !sqlite3_nosys

package vfs

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

const _O_NOFOLLOW = unix.O_NOFOLLOW

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

func osTestLock(file *os.File, start, len int64) (int16, _ErrorCode) {
	lock := unix.Flock_t{
		Type:  unix.F_WRLCK,
		Start: start,
		Len:   len,
	}
	if unix.FcntlFlock(file.Fd(), unix.F_GETLK, &lock) != nil {
		return 0, _IOERR_CHECKRESERVEDLOCK
	}
	return lock.Type, _OK
}

func osLockErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
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
			return _PERM
		}
		// notest // usually EWOULDBLOCK == EAGAIN
		if errno == unix.EWOULDBLOCK && unix.EWOULDBLOCK != unix.EAGAIN {
			return _BUSY
		}
	}
	return def
}
