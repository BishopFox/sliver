//go:build (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

package wasmnet

import (
	"errors"
	"syscall"
)

func wasmNetworkSystemErrno(err error) (uint32, bool) {
	switch {
	case errors.Is(err, syscall.EACCES):
		return wasiErrnoAccess, true
	case errors.Is(err, syscall.EADDRINUSE):
		return wasiErrnoAddrInUse, true
	case errors.Is(err, syscall.EADDRNOTAVAIL):
		return wasiErrnoAddrNotAvailable, true
	case errors.Is(err, syscall.EAFNOSUPPORT):
		return wasiErrnoAddressFamily, true
	case errors.Is(err, syscall.EAGAIN):
		return wasiErrnoAgain, true
	case errors.Is(err, syscall.EBADF):
		return wasiErrnoBadFile, true
	case errors.Is(err, syscall.ECONNABORTED):
		return wasiErrnoConnAborted, true
	case errors.Is(err, syscall.ECONNREFUSED):
		return wasiErrnoConnRefused, true
	case errors.Is(err, syscall.ECONNRESET):
		return wasiErrnoConnReset, true
	case errors.Is(err, syscall.EHOSTUNREACH):
		return wasiErrnoHostUnreachable, true
	case errors.Is(err, syscall.EINTR):
		return wasiErrnoInterrupted, true
	case errors.Is(err, syscall.EINVAL):
		return wasiErrnoInvalid, true
	case errors.Is(err, syscall.EIO):
		return wasiErrnoIO, true
	case errors.Is(err, syscall.EISCONN):
		return wasiErrnoIsConnected, true
	case errors.Is(err, syscall.EMSGSIZE):
		return wasiErrnoMessageSize, true
	case errors.Is(err, syscall.ENETDOWN):
		return wasiErrnoNetworkDown, true
	case errors.Is(err, syscall.ENETRESET):
		return wasiErrnoNetworkReset, true
	case errors.Is(err, syscall.ENETUNREACH):
		return wasiErrnoNetworkUnreachable, true
	case errors.Is(err, syscall.ENOBUFS):
		return wasiErrnoNoBuffer, true
	case errors.Is(err, syscall.ENOENT):
		return wasiErrnoNoEntry, true
	case errors.Is(err, syscall.ENOPROTOOPT):
		return wasiErrnoNoProtocolOption, true
	case errors.Is(err, syscall.ENOSYS):
		return wasiErrnoNotImplemented, true
	case errors.Is(err, syscall.ENOTCONN):
		return wasiErrnoNotConnected, true
	case errors.Is(err, syscall.ENOTSOCK):
		return wasiErrnoNotSocket, true
	case errors.Is(err, syscall.ENOTSUP):
		return wasiErrnoNotSupported, true
	case errors.Is(err, syscall.EPIPE):
		return wasiErrnoPipe, true
	case errors.Is(err, syscall.EPROTONOSUPPORT):
		return wasiErrnoProtocolNotSupported, true
	case errors.Is(err, syscall.ERANGE):
		return wasiErrnoRange, true
	case errors.Is(err, syscall.ETIMEDOUT):
		return wasiErrnoTimedOut, true
	default:
		return 0, false
	}
}
