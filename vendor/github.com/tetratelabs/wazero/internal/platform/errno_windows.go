package platform

import "syscall"

// See https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499-
const (
	// ERROR_ACCESS_DENIED is a Windows error returned by syscall.Unlink
	// instead of syscall.EACCES
	ERROR_ACCESS_DENIED = syscall.Errno(5)

	// ERROR_INVALID_HANDLE is a Windows error returned by syscall.Write
	// instead of syscall.EBADF
	ERROR_INVALID_HANDLE = syscall.Errno(6)

	// ERROR_FILE_EXISTS is a Windows error returned by os.OpenFile
	// instead of syscall.EEXIST
	ERROR_FILE_EXISTS = syscall.Errno(0x50)

	// ERROR_INVALID_NAME is a Windows error returned by open when a file
	// path has a trailing slash
	ERROR_INVALID_NAME = syscall.Errno(0x7B)

	// ERROR_NEGATIVE_SEEK is a Windows error returned by os.Truncate
	// instead of syscall.EINVAL
	ERROR_NEGATIVE_SEEK = syscall.Errno(0x83)

	// ERROR_DIR_NOT_EMPTY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTEMPTY
	ERROR_DIR_NOT_EMPTY = syscall.Errno(0x91)

	// ERROR_ALREADY_EXISTS is a Windows error returned by os.Mkdir
	// instead of syscall.EEXIST
	ERROR_ALREADY_EXISTS = syscall.Errno(0xB7)

	// ERROR_DIRECTORY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTDIR
	ERROR_DIRECTORY = syscall.Errno(0x10B)
)

// See https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--1300-1699-
const (
	// ERROR_PRIVILEGE_NOT_HELD is a Windows error returned by os.Symlink
	// instead of syscall.EPERM.
	//
	// Note: This can happen when trying to create symlinks w/o admin perms.
	ERROR_PRIVILEGE_NOT_HELD = syscall.Errno(0x522)
)

func adjustErrno(err syscall.Errno) syscall.Errno {
	// Note: In windows, ERROR_PATH_NOT_FOUND(0x3) maps to syscall.ENOTDIR
	switch err {
	case ERROR_ALREADY_EXISTS:
		return syscall.EEXIST
	case ERROR_DIRECTORY:
		return syscall.ENOTDIR
	case ERROR_DIR_NOT_EMPTY:
		return syscall.ENOTEMPTY
	case ERROR_FILE_EXISTS:
		return syscall.EEXIST
	case ERROR_INVALID_HANDLE:
		return syscall.EBADF
	case ERROR_ACCESS_DENIED:
		// POSIX read and write functions expect EBADF, not EACCES when not
		// open for reading or writing.
		return syscall.EBADF
	case ERROR_PRIVILEGE_NOT_HELD:
		return syscall.EPERM
	case ERROR_NEGATIVE_SEEK, ERROR_INVALID_NAME:
		return syscall.EINVAL
	}
	return err
}
