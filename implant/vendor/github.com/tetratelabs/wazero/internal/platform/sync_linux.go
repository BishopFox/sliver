//go:build linux

package platform

import "syscall"

func fdatasync(fd uintptr) error {
	return syscall.Fdatasync(int(fd))
}
