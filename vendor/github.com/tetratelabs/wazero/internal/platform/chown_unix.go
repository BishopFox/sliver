//go:build !windows

package platform

import "syscall"

func fchown(fd uintptr, uid, gid int) syscall.Errno {
	return UnwrapOSError(syscall.Fchown(int(fd), uid, gid))
}
