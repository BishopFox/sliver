//go:build unix || darwin || linux

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

const NonBlockingFileIoSupported = true

// readFd exposes syscall.Read.
func readFd(fd uintptr, buf []byte) (int, syscall.Errno) {
	if len(buf) == 0 {
		return 0, 0 // Short-circuit 0-len reads.
	}
	n, err := syscall.Read(int(fd), buf)
	errno := platform.UnwrapOSError(err)
	return n, errno
}
