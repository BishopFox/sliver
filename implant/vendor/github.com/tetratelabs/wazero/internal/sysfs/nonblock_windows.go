//go:build windows

package sysfs

import "syscall"

func setNonblock(fd uintptr, enable bool) error {
	return syscall.SetNonblock(syscall.Handle(fd), enable)
}
