package platform

import (
	"syscall"
	"unsafe"
)

var procGetConsoleMode = kernel32.NewProc("GetConsoleMode")

func isTerminal(fd uintptr) bool {
	handle := mapToWindowsHandle(fd)

	var st uint32
	r, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, handle, uintptr(unsafe.Pointer(&st)), 0)
	return r != 0 && e == 0
}

// mapToWindowsHandle maps file descriptors 0..2 to a valid Windows handle
func mapToWindowsHandle(fd uintptr) uintptr {
	var handle uintptr
	switch fd {
	case 0:
		handle = uintptr(syscall.Stdin)
	case 1:
		handle = uintptr(syscall.Stdout)
	case 2:
		handle = uintptr(syscall.Stderr)
	default:
		handle = fd
	}
	return handle
}
