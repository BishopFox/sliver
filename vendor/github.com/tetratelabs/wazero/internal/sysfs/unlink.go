//go:build !windows

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func Unlink(name string) (errno syscall.Errno) {
	err := syscall.Unlink(name)
	if errno = platform.UnwrapOSError(err); errno == syscall.EPERM {
		errno = syscall.EISDIR
	}
	return errno
}
