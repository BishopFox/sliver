//go:build linux || darwin

package sysfs

import (
	"net"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
	socketapi "github.com/tetratelabs/wazero/internal/sock"
)

// MSG_PEEK is the constant syscall.MSG_PEEK
const MSG_PEEK = syscall.MSG_PEEK

// newTCPListenerFile is a constructor for a socketapi.TCPSock.
//
// Note: the implementation of socketapi.TCPSock goes straight
// to the syscall layer, bypassing most of the Go library.
// For an alternative approach, consider winTcpListenerFile
// where most APIs are implemented with regular Go std-lib calls.
func newTCPListenerFile(tl *net.TCPListener) socketapi.TCPSock {
	conn, err := tl.File()
	if err != nil {
		panic(err)
	}
	fd := conn.Fd()
	// We need to duplicate this file handle, or the lifecycle will be tied
	// to the TCPListener. We rely on the TCPListener only to set up
	// the connection correctly and parse/resolve the TCP Address
	// (notice we actually rely on the listener in the Windows implementation).
	sysfd, err := syscall.Dup(int(fd))
	if err != nil {
		panic(err)
	}
	return &tcpListenerFile{fd: uintptr(sysfd), addr: tl.Addr().(*net.TCPAddr)}
}

var _ socketapi.TCPSock = (*tcpListenerFile)(nil)

type tcpListenerFile struct {
	baseSockFile

	fd   uintptr
	addr *net.TCPAddr
}

// Accept implements the same method as documented on socketapi.TCPSock
func (f *tcpListenerFile) Accept() (socketapi.TCPConn, syscall.Errno) {
	nfd, _, err := syscall.Accept(int(f.fd))
	errno := platform.UnwrapOSError(err)
	if errno != 0 {
		return nil, errno
	}
	return &tcpConnFile{fd: uintptr(nfd)}, 0
}

// SetNonblock implements the same method as documented on fsapi.File
func (f *tcpListenerFile) SetNonblock(enabled bool) syscall.Errno {
	return platform.UnwrapOSError(setNonblock(f.fd, enabled))
}

// Close implements the same method as documented on fsapi.File
func (f *tcpListenerFile) Close() syscall.Errno {
	return platform.UnwrapOSError(syscall.Close(int(f.fd)))
}

// Addr is exposed for testing.
func (f *tcpListenerFile) Addr() *net.TCPAddr {
	return f.addr
}

var _ socketapi.TCPConn = (*tcpConnFile)(nil)

type tcpConnFile struct {
	baseSockFile

	fd uintptr

	// closed is true when closed was called. This ensures proper syscall.EBADF
	closed bool
}

func newTcpConn(tc *net.TCPConn) socketapi.TCPConn {
	f, err := tc.File()
	if err != nil {
		panic(err)
	}
	return &tcpConnFile{fd: f.Fd()}
}

// SetNonblock implements the same method as documented on fsapi.File
func (f *tcpConnFile) SetNonblock(enabled bool) (errno syscall.Errno) {
	return platform.UnwrapOSError(setNonblock(f.fd, enabled))
}

// Read implements the same method as documented on fsapi.File
func (f *tcpConnFile) Read(buf []byte) (n int, errno syscall.Errno) {
	n, err := syscall.Read(int(f.fd), buf)
	if err != nil {
		// Defer validation overhead until we've already had an error.
		errno = platform.UnwrapOSError(err)
		errno = fileError(f, f.closed, errno)
	}
	return n, errno
}

// Write implements the same method as documented on fsapi.File
func (f *tcpConnFile) Write(buf []byte) (n int, errno syscall.Errno) {
	n, err := syscall.Write(int(f.fd), buf)
	if err != nil {
		// Defer validation overhead until we've already had an error.
		errno = platform.UnwrapOSError(err)
		errno = fileError(f, f.closed, errno)
	}
	return n, errno
}

// Recvfrom implements the same method as documented on socketapi.TCPConn
func (f *tcpConnFile) Recvfrom(p []byte, flags int) (n int, errno syscall.Errno) {
	if flags != MSG_PEEK {
		errno = syscall.EINVAL
		return
	}
	n, _, recvfromErr := syscall.Recvfrom(int(f.fd), p, MSG_PEEK)
	errno = platform.UnwrapOSError(recvfromErr)
	return n, errno
}

// Shutdown implements the same method as documented on fsapi.Conn
func (f *tcpConnFile) Shutdown(how int) syscall.Errno {
	var err error
	switch how {
	case syscall.SHUT_RD, syscall.SHUT_WR:
		err = syscall.Shutdown(int(f.fd), how)
	case syscall.SHUT_RDWR:
		return f.close()
	default:
		return syscall.EINVAL
	}
	return platform.UnwrapOSError(err)
}

// Close implements the same method as documented on fsapi.File
func (f *tcpConnFile) Close() syscall.Errno {
	return f.close()
}

func (f *tcpConnFile) close() syscall.Errno {
	if f.closed {
		return 0
	}
	f.closed = true
	return platform.UnwrapOSError(syscall.Shutdown(int(f.fd), syscall.SHUT_RDWR))
}
