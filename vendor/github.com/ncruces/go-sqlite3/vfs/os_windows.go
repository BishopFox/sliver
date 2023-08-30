package vfs

import (
	"io/fs"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// osOpenFile is a simplified copy of [os.openFileNolog]
// that uses syscall.FILE_SHARE_DELETE.
// https://go.dev/src/os/file_windows.go
//
// See: https://go.dev/issue/32088
func osOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	r, e := syscallOpen(name, flag, uint32(perm.Perm()))
	if e != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: e}
	}
	return os.NewFile(uintptr(r), name), nil
}

func osAccess(path string, flags AccessFlag) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if flags == ACCESS_EXISTS {
		return nil
	}

	var want fs.FileMode = windows.S_IRUSR
	if flags == ACCESS_READWRITE {
		want |= windows.S_IWUSR
	}
	if fi.IsDir() {
		want |= windows.S_IXUSR
	}
	if fi.Mode()&want != want {
		return fs.ErrPermission
	}
	return nil
}

func osSetMode(file *os.File, modeof string) error {
	fi, err := os.Stat(modeof)
	if err != nil {
		return err
	}
	file.Chmod(fi.Mode())
	return nil
}

func osGetSharedLock(file *os.File, timeout time.Duration) _ErrorCode {
	// Acquire the PENDING lock temporarily before acquiring a new SHARED lock.
	rc := osReadLock(file, _PENDING_BYTE, 1, timeout)

	if rc == _OK {
		// Acquire the SHARED lock.
		rc = osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)

		// Release the PENDING lock.
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return rc
}

func osGetExclusiveLock(file *os.File, timeout time.Duration) _ErrorCode {
	if timeout == 0 {
		timeout = time.Millisecond
	}

	// Release the SHARED lock.
	osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

	// Acquire the EXCLUSIVE lock.
	rc := osWriteLock(file, _SHARED_FIRST, _SHARED_SIZE, timeout)

	if rc != _OK {
		// Reacquire the SHARED lock.
		osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0)
	}
	return rc
}

func osDowngradeLock(file *os.File, state LockLevel) _ErrorCode {
	if state >= LOCK_EXCLUSIVE {
		// Release the SHARED lock.
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)

		// Reacquire the SHARED lock.
		if rc := osReadLock(file, _SHARED_FIRST, _SHARED_SIZE, 0); rc != _OK {
			// This should never happen.
			// We should always be able to reacquire the read lock.
			return _IOERR_RDLOCK
		}
	}

	// Release the PENDING and RESERVED locks.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osReleaseLock(file *os.File, state LockLevel) _ErrorCode {
	// Release all locks.
	if state >= LOCK_RESERVED {
		osUnlock(file, _RESERVED_BYTE, 1)
	}
	if state >= LOCK_SHARED {
		osUnlock(file, _SHARED_FIRST, _SHARED_SIZE)
	}
	if state >= LOCK_PENDING {
		osUnlock(file, _PENDING_BYTE, 1)
	}
	return _OK
}

func osUnlock(file *os.File, start, len uint32) _ErrorCode {
	err := windows.UnlockFileEx(windows.Handle(file.Fd()),
		0, len, 0, &windows.Overlapped{Offset: start})
	if err == windows.ERROR_NOT_LOCKED {
		return _OK
	}
	if err != nil {
		return _IOERR_UNLOCK
	}
	return _OK
}

func osLock(file *os.File, flags, start, len uint32, timeout time.Duration, def _ErrorCode) _ErrorCode {
	var err error
	for {
		err = windows.LockFileEx(windows.Handle(file.Fd()), flags,
			0, len, 0, &windows.Overlapped{Offset: start})
		if errno, _ := err.(windows.Errno); errno != windows.ERROR_LOCK_VIOLATION {
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

func osReadLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file,
		windows.LOCKFILE_FAIL_IMMEDIATELY,
		start, len, timeout, _IOERR_RDLOCK)
}

func osWriteLock(file *os.File, start, len uint32, timeout time.Duration) _ErrorCode {
	return osLock(file,
		windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK,
		start, len, timeout, _IOERR_LOCK)
}

func osCheckLock(file *os.File, start, len uint32) (bool, _ErrorCode) {
	rc := osLock(file,
		windows.LOCKFILE_FAIL_IMMEDIATELY,
		start, len, 0, _IOERR_CHECKRESERVEDLOCK)
	if rc == _BUSY {
		return true, _OK
	}
	if rc == _OK {
		osUnlock(file, start, len)
	}
	return false, rc
}

func osLockErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	if errno, ok := err.(windows.Errno); ok {
		// https://devblogs.microsoft.com/oldnewthing/20140905-00/?p=63
		switch errno {
		case
			windows.ERROR_LOCK_VIOLATION,
			windows.ERROR_IO_PENDING,
			windows.ERROR_OPERATION_ABORTED:
			return _BUSY
		}
	}
	return def
}

// syscallOpen is a simplified copy of [syscall.Open]
// that uses syscall.FILE_SHARE_DELETE.
// https://go.dev/src/syscall/syscall_windows.go
func syscallOpen(path string, mode int, perm uint32) (fd syscall.Handle, err error) {
	if len(path) == 0 {
		return syscall.InvalidHandle, syscall.ERROR_FILE_NOT_FOUND
	}
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	var access uint32
	switch mode & (syscall.O_RDONLY | syscall.O_WRONLY | syscall.O_RDWR) {
	case syscall.O_RDONLY:
		access = syscall.GENERIC_READ
	case syscall.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case syscall.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}
	if mode&syscall.O_CREAT != 0 {
		access |= syscall.GENERIC_WRITE
	}
	if mode&syscall.O_APPEND != 0 {
		access &^= syscall.GENERIC_WRITE
		access |= syscall.FILE_APPEND_DATA
	}
	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	var createmode uint32
	switch {
	case mode&(syscall.O_CREAT|syscall.O_EXCL) == (syscall.O_CREAT | syscall.O_EXCL):
		createmode = syscall.CREATE_NEW
	case mode&(syscall.O_CREAT|syscall.O_TRUNC) == (syscall.O_CREAT | syscall.O_TRUNC):
		createmode = syscall.CREATE_ALWAYS
	case mode&syscall.O_CREAT == syscall.O_CREAT:
		createmode = syscall.OPEN_ALWAYS
	case mode&syscall.O_TRUNC == syscall.O_TRUNC:
		createmode = syscall.TRUNCATE_EXISTING
	default:
		createmode = syscall.OPEN_EXISTING
	}
	var attrs uint32 = syscall.FILE_ATTRIBUTE_NORMAL
	if perm&syscall.S_IWRITE == 0 {
		attrs = syscall.FILE_ATTRIBUTE_READONLY
	}
	if createmode == syscall.OPEN_EXISTING && access == syscall.GENERIC_READ {
		// Necessary for opening directory handles.
		attrs |= syscall.FILE_FLAG_BACKUP_SEMANTICS
	}
	return syscall.CreateFile(pathp, access, sharemode, nil, createmode, attrs, 0)
}
