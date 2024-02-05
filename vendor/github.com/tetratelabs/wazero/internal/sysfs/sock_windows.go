//go:build windows

package sysfs

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
	socketapi "github.com/tetratelabs/wazero/internal/sock"
)

const (
	// MSG_PEEK is the flag PEEK for syscall.Recvfrom on Windows.
	// This constant is not exported on this platform.
	MSG_PEEK = 0x2
	// _FIONBIO is the flag to set the O_NONBLOCK flag on socket handles using ioctlsocket.
	_FIONBIO = 0x8004667e
)

var (
	// modws2_32 is WinSock.
	modws2_32 = syscall.NewLazyDLL("ws2_32.dll")
	// procrecvfrom exposes recvfrom from WinSock.
	procrecvfrom = modws2_32.NewProc("recvfrom")
	// procioctlsocket exposes ioctlsocket from WinSock.
	procioctlsocket = modws2_32.NewProc("ioctlsocket")
)

// recvfrom exposes the underlying syscall in Windows.
//
// Note: since we are only using this to expose MSG_PEEK,
// we do not need really need all the parameters that are actually
// allowed in WinSock.
// We ignore `from *sockaddr` and `fromlen *int`.
func recvfrom(s syscall.Handle, buf []byte, flags int32) (n int, errno sys.Errno) {
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
	return int(r0), sys.UnwrapOSError(e1)
}

func setNonblockSocket(fd syscall.Handle, enabled bool) sys.Errno {
	opt := uint64(0)
	if enabled {
		opt = 1
	}
	// ioctlsocket(fd, FIONBIO, &opt)
	_, _, errno := syscall.SyscallN(
		procioctlsocket.Addr(),
		uintptr(fd),
		uintptr(_FIONBIO),
		uintptr(unsafe.Pointer(&opt)))
	return sys.UnwrapOSError(errno)
}

// syscallConnControl extracts a syscall.RawConn from the given syscall.Conn and applies
// the given fn to a file descriptor, returning an integer or a nonzero syscall.Errno on failure.
//
// syscallConnControl streamlines the pattern of extracting the syscall.Rawconn,
// invoking its syscall.RawConn.Control method, then handling properly the errors that may occur
// within fn or returned by syscall.RawConn.Control itself.
func syscallConnControl(conn syscall.Conn, fn func(fd uintptr) (int, sys.Errno)) (n int, errno sys.Errno) {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		return 0, sys.UnwrapOSError(err)
	}
	// Prioritize the inner errno over Control
	if controlErr := syscallConn.Control(func(fd uintptr) {
		n, errno = fn(fd)
	}); errno == 0 {
		errno = sys.UnwrapOSError(controlErr)
	}
	return
}

func _pollSock(conn syscall.Conn, flag fsapi.Pflag, timeoutMillis int32) (bool, sys.Errno) {
	if flag != fsapi.POLLIN {
		return false, sys.ENOTSUP
	}
	n, errno := syscallConnControl(conn, func(fd uintptr) (int, sys.Errno) {
		return _poll([]pollFd{newPollFd(fd, _POLLIN, 0)}, timeoutMillis)
	})
	return n > 0, errno
}

// newTCPListenerFile is a constructor for a socketapi.TCPSock.
//
// Note: currently the Windows implementation of socketapi.TCPSock
// returns a winTcpListenerFile, which is a specialized TCPSock
// that delegates to a .net.TCPListener.
// The current strategy is to delegate most behavior to the Go
// standard library, instead of invoke syscalls/Win32 APIs
// because they are sensibly different from Unix's.
func newTCPListenerFile(tl *net.TCPListener) socketapi.TCPSock {
	return &winTcpListenerFile{tl: tl}
}

var _ socketapi.TCPSock = (*winTcpListenerFile)(nil)

type winTcpListenerFile struct {
	baseSockFile

	tl       *net.TCPListener
	closed   bool
	nonblock bool
}

// Accept implements the same method as documented on socketapi.TCPSock
func (f *winTcpListenerFile) Accept() (socketapi.TCPConn, sys.Errno) {
	// Ensure we have an incoming connection using winsock_select, otherwise return immediately.
	if f.nonblock {
		if ready, errno := _pollSock(f.tl, fsapi.POLLIN, 0); !ready || errno != 0 {
			return nil, sys.EAGAIN
		}
	}

	// Accept normally blocks goroutines, but we
	// made sure that we have an incoming connection,
	// so we should be safe.
	if conn, err := f.tl.Accept(); err != nil {
		return nil, sys.UnwrapOSError(err)
	} else {
		return newTcpConn(conn.(*net.TCPConn)), 0
	}
}

// Close implements the same method as documented on sys.File
func (f *winTcpListenerFile) Close() sys.Errno {
	if !f.closed {
		return sys.UnwrapOSError(f.tl.Close())
	}
	return 0
}

// Addr is exposed for testing.
func (f *winTcpListenerFile) Addr() *net.TCPAddr {
	return f.tl.Addr().(*net.TCPAddr)
}

// IsNonblock implements the same method as documented on fsapi.File
func (f *winTcpListenerFile) IsNonblock() bool {
	return f.nonblock
}

// SetNonblock implements the same method as documented on fsapi.File
func (f *winTcpListenerFile) SetNonblock(enabled bool) sys.Errno {
	f.nonblock = enabled
	_, errno := syscallConnControl(f.tl, func(fd uintptr) (int, sys.Errno) {
		return 0, setNonblockSocket(syscall.Handle(fd), enabled)
	})
	return errno
}

// Poll implements the same method as documented on fsapi.File
func (f *winTcpListenerFile) Poll(fsapi.Pflag, int32) (ready bool, errno sys.Errno) {
	return false, sys.ENOSYS
}

var _ socketapi.TCPConn = (*winTcpConnFile)(nil)

// winTcpConnFile is a blocking connection.
//
// It is a wrapper for an underlying net.TCPConn.
type winTcpConnFile struct {
	baseSockFile

	tc *net.TCPConn

	// nonblock is true when the underlying connection is flagged as non-blocking.
	// This ensures that reads and writes return sys.EAGAIN without blocking the caller.
	nonblock bool
	// closed is true when closed was called. This ensures proper sys.EBADF
	closed bool
}

func newTcpConn(tc *net.TCPConn) socketapi.TCPConn {
	return &winTcpConnFile{tc: tc}
}

// Read implements the same method as documented on sys.File
func (f *winTcpConnFile) Read(buf []byte) (n int, errno sys.Errno) {
	if len(buf) == 0 {
		return 0, 0 // Short-circuit 0-len reads.
	}
	if nonBlockingFileReadSupported && f.IsNonblock() {
		n, errno = syscallConnControl(f.tc, func(fd uintptr) (int, sys.Errno) {
			return readSocket(syscall.Handle(fd), buf)
		})
	} else {
		n, errno = read(f.tc, buf)
	}
	if errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Write implements the same method as documented on sys.File
func (f *winTcpConnFile) Write(buf []byte) (n int, errno sys.Errno) {
	if nonBlockingFileWriteSupported && f.IsNonblock() {
		return syscallConnControl(f.tc, func(fd uintptr) (int, sys.Errno) {
			return writeSocket(fd, buf)
		})
	} else {
		n, errno = write(f.tc, buf)
	}
	if errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Recvfrom implements the same method as documented on socketapi.TCPConn
func (f *winTcpConnFile) Recvfrom(p []byte, flags int) (n int, errno sys.Errno) {
	if flags != MSG_PEEK {
		errno = sys.EINVAL
		return
	}
	return syscallConnControl(f.tc, func(fd uintptr) (int, sys.Errno) {
		return recvfrom(syscall.Handle(fd), p, MSG_PEEK)
	})
}

// Shutdown implements the same method as documented on sys.Conn
func (f *winTcpConnFile) Shutdown(how int) sys.Errno {
	// FIXME: can userland shutdown listeners?
	var err error
	switch how {
	case syscall.SHUT_RD:
		err = f.tc.CloseRead()
	case syscall.SHUT_WR:
		err = f.tc.CloseWrite()
	case syscall.SHUT_RDWR:
		return f.close()
	default:
		return sys.EINVAL
	}
	return sys.UnwrapOSError(err)
}

// Close implements the same method as documented on sys.File
func (f *winTcpConnFile) Close() sys.Errno {
	return f.close()
}

func (f *winTcpConnFile) close() sys.Errno {
	if f.closed {
		return 0
	}
	f.closed = true
	return f.Shutdown(syscall.SHUT_RDWR)
}

// IsNonblock implements the same method as documented on fsapi.File
func (f *winTcpConnFile) IsNonblock() bool {
	return f.nonblock
}

// SetNonblock implements the same method as documented on fsapi.File
func (f *winTcpConnFile) SetNonblock(enabled bool) (errno sys.Errno) {
	f.nonblock = true
	_, errno = syscallConnControl(f.tc, func(fd uintptr) (int, sys.Errno) {
		return 0, sys.UnwrapOSError(setNonblockSocket(syscall.Handle(fd), enabled))
	})
	return
}

// Poll implements the same method as documented on fsapi.File
func (f *winTcpConnFile) Poll(fsapi.Pflag, int32) (ready bool, errno sys.Errno) {
	return false, sys.ENOSYS
}
