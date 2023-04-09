//go:build windows

package platform

import (
	"os"
	"syscall"
)

func Unlink(name string) syscall.Errno {
	err := syscall.Unlink(name)
	if err == nil {
		return 0
	}
	errno := UnwrapOSError(err)
	if errno == syscall.EPERM {
		lstat, errLstat := os.Lstat(name)
		if errLstat == nil && lstat.Mode()&os.ModeSymlink != 0 {
			errno = UnwrapOSError(os.Remove(name))
		} else {
			errno = syscall.EISDIR
		}
	}
	return errno
}
