//go:build windows

package sysfs

import (
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func Unlink(name string) syscall.Errno {
	err := syscall.Unlink(name)
	if err == nil {
		return 0
	}
	errno := platform.UnwrapOSError(err)
	if errno == syscall.EBADF {
		lstat, errLstat := os.Lstat(name)
		if errLstat == nil && lstat.Mode()&os.ModeSymlink != 0 {
			errno = platform.UnwrapOSError(os.Remove(name))
		} else {
			errno = syscall.EISDIR
		}
	}
	return errno
}
