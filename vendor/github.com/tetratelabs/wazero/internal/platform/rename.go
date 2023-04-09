//go:build !windows

package platform

import "syscall"

func Rename(from, to string) syscall.Errno {
	if from == to {
		return 0
	}
	return UnwrapOSError(syscall.Rename(from, to))
}
