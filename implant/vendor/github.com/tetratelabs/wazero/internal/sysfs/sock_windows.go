//go:build windows

package sysfs

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/platform"
)

// MSG_PEEK is the flag PEEK for syscall.Recvfrom on Windows.
// This constant is not exported on this platform.
const MSG_PEEK = 0x2

// recvfromPeek exposes syscall.Recvfrom with flag MSG_PEEK on Windows.
func recvfromPeek(conn *net.TCPConn, p []byte) (n int, errno syscall.Errno) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		errno = platform.UnwrapOSError(err)
		return
	}

	// Prioritize the error from recvfrom over Control
	if controlErr := syscallConn.Control(func(fd uintptr) {
		var recvfromErr error
		n, recvfromErr = recvfrom(syscall.Handle(fd), p, MSG_PEEK)
		errno = platform.UnwrapOSError(recvfromErr)
	}); errno == 0 {
		errno = platform.UnwrapOSError(controlErr)
	}
	return
}

var (
	// modws2_32 is WinSock.
	modws2_32 = syscall.NewLazyDLL("ws2_32.dll")
	// procrecvfrom exposes recvfrom from WinSock.
	procrecvfrom = modws2_32.NewProc("recvfrom")
)

// recvfrom exposes the underlying syscall in Windows.
//
// Note: since we are only using this to expose MSG_PEEK,
// we do not need really need all the parameters that are actually
// allowed in WinSock.
// We ignore `from *sockaddr` and `fromlen *int`.
func recvfrom(s syscall.Handle, buf []byte, flags int32) (n int, errno syscall.Errno) {
	var _p0 *byte
	if len(buf) > 0 {
		_p0 = &buf[0]
	}
	r0, _, e1 := syscall.SyscallN(
		procrecvfrom.Addr(),
		uintptr(s),
		uintptr(unsafe.Pointer(_p0)),
		uintptr(len(buf)),
		uintptr(flags),
		0, // from *sockaddr (optional)
		0) // fromlen *int (optional)
	return int(r0), e1
}
