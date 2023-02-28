package sysfs

import (
	"io/fs"
	"syscall"
)

// See https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499-
const (
	// ERROR_ACCESS_DENIED is a Windows error returned by syscall.Unlink
	// instead of syscall.EPERM
	ERROR_ACCESS_DENIED = syscall.Errno(5)

	// ERROR_INVALID_HANDLE is a Windows error returned by syscall.Write
	// instead of syscall.EBADF
	ERROR_INVALID_HANDLE = syscall.Errno(6)

	// ERROR_NEGATIVE_SEEK is a Windows error returned by os.Truncate
	// instead of syscall.EINVAL
	ERROR_NEGATIVE_SEEK = syscall.Errno(131)

	// ERROR_DIR_NOT_EMPTY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTEMPTY
	ERROR_DIR_NOT_EMPTY = syscall.Errno(145)

	// ERROR_ALREADY_EXISTS is a Windows error returned by os.Mkdir
	// instead of syscall.EEXIST
	ERROR_ALREADY_EXISTS = syscall.Errno(183)

	// ERROR_DIRECTORY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTDIR
	ERROR_DIRECTORY = syscall.Errno(267)

	// ERROR_PRIVILEGE_NOT_HELD is a Windows error returned by os.Symlink
	// instead of syscall.EPERM.
	//
	// Note: This can happen when trying to create symlinks w/o admin perms.
	ERROR_PRIVILEGE_NOT_HELD = syscall.Errno(1314)
)

func adjustErrno(err syscall.Errno) error {
	switch err {
	case ERROR_ALREADY_EXISTS:
		return syscall.EEXIST
	case ERROR_DIR_NOT_EMPTY:
		return syscall.ENOTEMPTY
	case ERROR_INVALID_HANDLE:
		return syscall.EBADF
	case ERROR_ACCESS_DENIED, ERROR_PRIVILEGE_NOT_HELD:
		return syscall.EPERM
	}
	return err
}

func adjustRmdirError(err error) error {
	switch err {
	case ERROR_DIRECTORY:
		return syscall.ENOTDIR
	}
	return err
}

func adjustTruncateError(err error) error {
	if err == ERROR_NEGATIVE_SEEK {
		return syscall.EINVAL
	}
	return err
}

// maybeWrapFile deals with errno portability issues in Windows. This code is
// likely to change as we complete syscall support needed for WASI and GOOS=js.
//
// If we don't map to syscall.Errno, wasm will crash in odd way attempting the
// same. This approach is an alternative to making our own fs.File public type.
// We aren't doing that yet, as mapping problems are generally contained to
// Windows. Hence, file is intentionally not exported.
func maybeWrapFile(f file, fs FS, path string, flag int, perm fs.FileMode) file {
	return &windowsWrappedFile{f, fs, path, flag, perm, false}
}

type windowsWrappedFile struct {
	file
	fs                 FS
	path               string
	flag               int
	perm               fs.FileMode
	readDirInitialized bool
}

// ReadDir implements fs.ReadDirFile.
func (w *windowsWrappedFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !w.readDirInitialized {
		// On Windows, once the directory is opened, changes to the directory
		// is not visible on ReadDir on that already-opened file handle.
		//
		// In order to provide consistent behavior with other platforms, we re-open it.
		if err := w.Close(); err != nil {
			return nil, err
		}
		newW, err := w.fs.OpenFile(w.path, w.flag, w.perm)
		if err != nil {
			return nil, err
		}
		*w = *newW.(*windowsWrappedFile)
		w.readDirInitialized = true
	}
	return w.file.ReadDir(n)
}

// Write implements io.Writer
func (w *windowsWrappedFile) Write(p []byte) (n int, err error) {
	n, err = w.file.Write(p)
	if err == nil {
		return
	}

	// os.File.Wrap wraps the syscall error in a path error
	if pe, ok := err.(*fs.PathError); ok {
		if pe.Err = UnwrapOSError(pe.Err); pe.Err == syscall.EPERM {
			// go1.20 returns access denied, not invalid handle, writing to a directory.
			if stat, statErr := StatPath(w.fs, w.path); statErr == nil && stat.IsDir() {
				pe.Err = syscall.EBADF
			}
		}
	}
	return
}
