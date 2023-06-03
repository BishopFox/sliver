package platform

import (
	"io"
	"io/fs"
	"os"
	"syscall"
)

// UnwrapOSError returns a syscall.Errno or zero if the input is nil.
func UnwrapOSError(err error) syscall.Errno {
	if err == nil {
		return 0
	}
	err = underlyingError(err)
	if se, ok := err.(syscall.Errno); ok {
		return adjustErrno(se)
	}
	// Below are all the fs.ErrXXX in fs.go.
	//
	// Note: Once we have our own file type, we should never see these.
	switch err {
	case nil, io.EOF:
		return 0 // EOF is not a syscall.Errno
	case fs.ErrInvalid:
		return syscall.EINVAL
	case fs.ErrPermission:
		return syscall.EPERM
	case fs.ErrExist:
		return syscall.EEXIST
	case fs.ErrNotExist:
		return syscall.ENOENT
	case fs.ErrClosed:
		return syscall.EBADF
	}
	return syscall.EIO
}

// underlyingError returns the underlying error if a well-known OS error type.
//
// This impl is basically the same as os.underlyingError in os/error.go
func underlyingError(err error) error {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err
	case *os.LinkError:
		return err.Err
	case *os.SyscallError:
		return err.Err
	}
	return err
}
