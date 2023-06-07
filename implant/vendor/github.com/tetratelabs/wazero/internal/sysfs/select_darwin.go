package sysfs

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/tetratelabs/wazero/internal/platform"
)

// syscall_select invokes select on Darwin, with the given timeout Duration.
// We implement our own version instead of relying on syscall.Select because the latter
// only returns the error and discards the result.
func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	var t *syscall.Timeval
	if timeout != nil {
		tv := syscall.NsecToTimeval(timeout.Nanoseconds())
		t = &tv
	}
	result, _, errno := syscall_syscall6(
		libc_select_trampoline_addr,
		uintptr(n),
		uintptr(unsafe.Pointer(r)),
		uintptr(unsafe.Pointer(w)),
		uintptr(unsafe.Pointer(e)),
		uintptr(unsafe.Pointer(t)),
		0)
	res := int(result)
	if errno == 0 {
		return res, nil
	}
	return res, errno
}

// libc_select_trampoline_addr is the address of the
// `libc_select_trampoline` symbol, defined in `select_darwin.s`.
//
// We use this to invoke the syscall through syscall_syscall6 imported below.
var libc_select_trampoline_addr uintptr

// Imports the select symbol from libc as `libc_select`.
//
// Note: CGO mechanisms are used in darwin regardless of the CGO_ENABLED value
// or the "cgo" build flag. See /RATIONALE.md for why.
//go:cgo_import_dynamic libc_select select "/usr/lib/libSystem.B.dylib"
