//go:build !unix && !linux && !darwin

package sysfs

import "syscall"

const NonBlockingFileIoSupported = false

// readFd returns ENOSYS on unsupported platforms.
func readFd(fd uintptr, buf []byte) (int, syscall.Errno) {
	return -1, syscall.ENOSYS
}
