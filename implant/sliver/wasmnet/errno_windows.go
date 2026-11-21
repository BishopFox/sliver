//go:build windows && (amd64 || arm64 || 386)

package wasmnet

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

//nolint:gocyclo // The host-to-WASI errno mapping is intentionally exhaustive.
func wasmNetworkSystemErrno(err error) (uint32, bool) {
	switch {
	case errors.Is(err, syscall.EACCES), errors.Is(err, windows.WSAEACCES):
		return wasiErrnoAccess, true
	case errors.Is(err, syscall.EADDRINUSE), errors.Is(err, windows.WSAEADDRINUSE):
		return wasiErrnoAddrInUse, true
	case errors.Is(err, syscall.EADDRNOTAVAIL), errors.Is(err, windows.WSAEADDRNOTAVAIL):
		return wasiErrnoAddrNotAvailable, true
	case errors.Is(err, syscall.EAFNOSUPPORT), errors.Is(err, windows.WSAEAFNOSUPPORT):
		return wasiErrnoAddressFamily, true
	case errors.Is(err, syscall.EAGAIN), errors.Is(err, windows.WSAEWOULDBLOCK):
		return wasiErrnoAgain, true
	case errors.Is(err, syscall.EBADF), errors.Is(err, windows.WSAEBADF):
		return wasiErrnoBadFile, true
	case errors.Is(err, syscall.ECONNABORTED), errors.Is(err, windows.WSAECONNABORTED):
		return wasiErrnoConnAborted, true
	case errors.Is(err, syscall.ECONNREFUSED), errors.Is(err, windows.WSAECONNREFUSED):
		return wasiErrnoConnRefused, true
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, windows.WSAECONNRESET):
		return wasiErrnoConnReset, true
	case errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, windows.WSAEHOSTUNREACH):
		return wasiErrnoHostUnreachable, true
	case errors.Is(err, syscall.EINTR), errors.Is(err, windows.WSAEINTR):
		return wasiErrnoInterrupted, true
	case errors.Is(err, syscall.EINVAL), errors.Is(err, windows.WSAEINVAL):
		return wasiErrnoInvalid, true
	case errors.Is(err, syscall.EIO):
		return wasiErrnoIO, true
	case errors.Is(err, syscall.EISCONN), errors.Is(err, windows.WSAEISCONN):
		return wasiErrnoIsConnected, true
	case errors.Is(err, syscall.EMSGSIZE), errors.Is(err, windows.WSAEMSGSIZE):
		return wasiErrnoMessageSize, true
	case errors.Is(err, syscall.ENETDOWN), errors.Is(err, windows.WSAENETDOWN):
		return wasiErrnoNetworkDown, true
	case errors.Is(err, syscall.ENETRESET), errors.Is(err, windows.WSAENETRESET):
		return wasiErrnoNetworkReset, true
	case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, windows.WSAENETUNREACH):
		return wasiErrnoNetworkUnreachable, true
	case errors.Is(err, syscall.ENOBUFS), errors.Is(err, windows.WSAENOBUFS):
		return wasiErrnoNoBuffer, true
	case errors.Is(err, syscall.ENOENT):
		return wasiErrnoNoEntry, true
	case errors.Is(err, syscall.ENOPROTOOPT), errors.Is(err, windows.WSAENOPROTOOPT):
		return wasiErrnoNoProtocolOption, true
	case errors.Is(err, syscall.ENOSYS):
		return wasiErrnoNotImplemented, true
	case errors.Is(err, syscall.ENOTCONN), errors.Is(err, windows.WSAENOTCONN):
		return wasiErrnoNotConnected, true
	case errors.Is(err, syscall.ENOTSOCK), errors.Is(err, windows.WSAENOTSOCK):
		return wasiErrnoNotSocket, true
	case errors.Is(err, syscall.ENOTSUP), errors.Is(err, windows.WSAEOPNOTSUPP):
		return wasiErrnoNotSupported, true
	case errors.Is(err, syscall.EPIPE), errors.Is(err, windows.WSAESHUTDOWN):
		return wasiErrnoPipe, true
	case errors.Is(err, syscall.EPROTONOSUPPORT), errors.Is(err, windows.WSAEPROTONOSUPPORT):
		return wasiErrnoProtocolNotSupported, true
	case errors.Is(err, syscall.ERANGE):
		return wasiErrnoRange, true
	case errors.Is(err, syscall.ETIMEDOUT), errors.Is(err, windows.WSAETIMEDOUT):
		return wasiErrnoTimedOut, true
	default:
		return 0, false
	}
}
