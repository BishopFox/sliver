//go:build !windows

package sys

import "syscall"

func errorToErrno(err error) Errno {
	switch err := err.(type) {
	case Errno:
		return err
	case syscall.Errno:
		return syscallToErrno(err)
	default:
		return EIO
	}
}
