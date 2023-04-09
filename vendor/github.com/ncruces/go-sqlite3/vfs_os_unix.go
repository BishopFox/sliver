//go:build unix

package sqlite3

import (
	"io/fs"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (vfsOSMethods) Access(path string, flags _AccessFlag) error {
	var access uint32 // unix.F_OK
	switch flags {
	case _ACCESS_READWRITE:
		access = unix.R_OK | unix.W_OK
	case _ACCESS_READ:
		access = unix.R_OK
	}
	return unix.Access(path, access)
}

func (vfsOSMethods) GetSharedLock(file *os.File, timeout time.Duration) xErrorCode {
	// Test the PENDING lock before acquiring a new SHARED lock.
	if pending, _ := vfsOS.checkLock(file, _PENDING_BYTE, 1); pending {
		return xErrorCode(BUSY)
	}
	// Acquire the SHARED lock.
	return vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)
}

func (vfsOSMethods) GetExclusiveLock(file *os.File, timeout time.Duration) xErrorCode {
	if timeout == 0 {
		timeout = time.Millisecond
	}

	// Acquire the EXCLUSIVE lock.
	return vfsOS.writeLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)
}

func (vfsOSMethods) DowngradeLock(file *os.File, state vfsLockState) xErrorCode {
	if state >= _EXCLUSIVE_LOCK {
		// Downgrade to a SHARED lock.
		if rc := vfsOS.readLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// In theory, the downgrade to a SHARED cannot fail because another
			// process is holding an incompatible lock. If it does, this
			// indicates that the other process is not following the locking
			// protocol. If this happens, return IOERR_RDLOCK. Returning
			// BUSY would confuse the upper layer.
			return IOERR_RDLOCK
		}
	}
	// Release the PENDING and RESERVED locks.
	return vfsOS.unlock(file, _PENDING_BYTE, 2)
}

func (vfsOSMethods) ReleaseLock(file *os.File, _ vfsLockState) xErrorCode {
	// Release all locks.
	return vfsOS.unlock(file, 0, 0)
}

func (vfsOSMethods) lockErrorCode(err error, def xErrorCode) xErrorCode {
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
			return xErrorCode(BUSY)
		case unix.EPERM:
			return xErrorCode(PERM)
		}
	}
	return def
}
