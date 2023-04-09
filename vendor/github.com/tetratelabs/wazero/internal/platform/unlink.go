//go:build !windows

package platform

import "syscall"

func Unlink(name string) (errno syscall.Errno) {
	err := syscall.Unlink(name)
	if errno = UnwrapOSError(err); errno == syscall.EPERM {
		errno = syscall.EISDIR
	}
	return errno
}
