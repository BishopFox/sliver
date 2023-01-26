package syscallfs

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

const (
	// ERROR_ACCESS_DENIED is a Windows error returned by syscall.Unlink
	// instead of syscall.EPERM
	ERROR_ACCESS_DENIED = syscall.Errno(5)

	// ERROR_ALREADY_EXISTS is a Windows error returned by os.Mkdir
	// instead of syscall.EEXIST
	ERROR_ALREADY_EXISTS = syscall.Errno(183)

	// ERROR_DIRECTORY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTDIR
	ERROR_DIRECTORY = syscall.Errno(267)

	// ERROR_DIR_NOT_EMPTY is a Windows error returned by syscall.Rmdir
	// instead of syscall.ENOTEMPTY
	ERROR_DIR_NOT_EMPTY = syscall.Errno(145)
)

func adjustMkdirError(err error) error {
	// os.Mkdir wraps the syscall error in a path error
	if pe, ok := err.(*fs.PathError); ok && pe.Err == ERROR_ALREADY_EXISTS {
		pe.Err = syscall.EEXIST // adjust it
	}
	return err
}

func adjustRmdirError(err error) error {
	switch err {
	case ERROR_DIRECTORY:
		return syscall.ENOTDIR
	case ERROR_DIR_NOT_EMPTY:
		return syscall.ENOTEMPTY
	}
	return err
}

func adjustUnlinkError(err error) error {
	if err == ERROR_ACCESS_DENIED {
		return syscall.EISDIR
	}
	return err
}

// rename uses os.Rename as `windows.Rename` is internal in Go's source tree.
func rename(old, new string) (err error) {
	if err = os.Rename(old, new); err == nil {
		return
	}
	err = errors.Unwrap(err) // unwrap the link error
	if err == ERROR_ACCESS_DENIED {
		var newIsDir bool
		if stat, statErr := os.Stat(new); statErr == nil && stat.IsDir() {
			newIsDir = true
		}

		var oldIsDir bool
		if stat, statErr := os.Stat(old); statErr == nil && stat.IsDir() {
			oldIsDir = true
		}

		if oldIsDir && newIsDir {
			// Windows doesn't let you overwrite a directory
			return syscall.EINVAL
		} else if newIsDir {
			err = syscall.EISDIR
		} else { // use a mappable code
			err = syscall.EPERM
		}
	}
	return
}
