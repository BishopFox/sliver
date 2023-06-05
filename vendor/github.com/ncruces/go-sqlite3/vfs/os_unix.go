//go:build unix

package vfs

import (
	"io/fs"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func osOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

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

func osGetSharedLock(file *os.File, timeout time.Duration) _ErrorCode {
	// Test the PENDING lock before acquiring a new SHARED lock.
	if pending, _ := osCheckLock(file, _PENDING_BYTE, 1); pending {
		return _BUSY
	}
	// Acquire the SHARED lock.
	return osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)
}

func osGetExclusiveLock(file *os.File, timeout time.Duration) _ErrorCode {
	if timeout == 0 {
		timeout = time.Millisecond
	}

	// Acquire the EXCLUSIVE lock.
	return osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	if state >= LOCK_EXCLUSIVE {
		// Downgrade to a SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// In theory, the downgrade to a SHARED cannot fail because another
			// process is holding an incompatible lock. If it does, this
			// indicates that the other process is not following the locking
			// protocol. If this happens, return _IOERR_RDLOCK. Returning
			// BUSY would confuse the upper layer.
			return _IOERR_RDLOCK
		}
	}
	// Release the PENDING and RESERVED locks.
	return osUnlock(file, _PENDING_BYTE, 2)
}

func osReleaseLock(file *os.File, _ LockLevel) _ErrorCode {
	// Release all locks.
	return osUnlock(file, 0, 0)
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
	}
	return def
}
