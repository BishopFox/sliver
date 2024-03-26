//go:build linux || darwin || windows

package sysfs

import (
	"syscall"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
)

// syscallConnControl extracts a syscall.RawConn from the given syscall.Conn and applies
// the given fn to a file descriptor, returning an integer or a nonzero syscall.Errno on failure.
//
// syscallConnControl streamlines the pattern of extracting the syscall.Rawconn,
// invoking its syscall.RawConn.Control method, then handling properly the errors that may occur
// within fn or returned by syscall.RawConn.Control itself.
func syscallConnControl(conn syscall.Conn, fn func(fd uintptr) (int, experimentalsys.Errno)) (n int, errno experimentalsys.Errno) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return 0, experimentalsys.UnwrapOSError(err)
	}
	// Prioritize the inner errno over Control
	if controlErr := syscallConn.Control(func(fd uintptr) {
		n, errno = fn(fd)
	}); errno == 0 {
		errno = experimentalsys.UnwrapOSError(controlErr)
	}
	return
}
