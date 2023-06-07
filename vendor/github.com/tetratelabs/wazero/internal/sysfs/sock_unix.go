//go:build linux || darwin

package sysfs

import (
	"net"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

const MSG_PEEK = syscall.MSG_PEEK

// recvfromPeek exposes syscall.Recvfrom with flag MSG_PEEK on POSIX systems.
func recvfromPeek(conn *net.TCPConn, p []byte) (n int, errno syscall.Errno) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return 0, platform.UnwrapOSError(err)
	}

	// Prioritize the error from Recvfrom over Control
	if controlErr := syscallConn.Control(func(fd uintptr) {
		var recvfromErr error
		n, _, recvfromErr = syscall.Recvfrom(int(fd), p, MSG_PEEK)
		errno = platform.UnwrapOSError(recvfromErr)
	}); errno == 0 {
		errno = platform.UnwrapOSError(controlErr)
	}
	return
}
