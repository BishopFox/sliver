//go:build linux && !android && 386

// SPDX-License-Identifier: MIT
// Adapted from Reflektor's memmod Linux call bridge; see
// ../../../memmod/COPYING.
package linuxmem

import (
	"runtime"
	"unsafe"

	_ "github.com/ebitengine/purego"
)

type linux386CallFrame struct {
	fn     uintptr
	a0     uintptr
	a1     uintptr
	a2     uintptr
	result uintptr
}

var linux386CallABI0 uintptr

//go:linkname runtimeCGOCall runtime.cgocall
func runtimeCGOCall(fn uintptr, arg unsafe.Pointer) int32

// Use a dedicated integer-only dispatcher on 386 because purego v0.10.1's
// generic syscall trampoline unconditionally pops x87 ST0 after every call,
// including integer-returning functions that leave the x87 stack empty.
//
//go:uintptrescapes
func callExportFunction(fn uintptr, args ...uintptr) uintptr {
	if len(args) > maxExportArguments {
		panic("validated Linux export argument count is out of range")
	}
	frame := &linux386CallFrame{fn: fn}
	if len(args) > 0 {
		frame.a0 = args[0]
	}
	if len(args) > 1 {
		frame.a1 = args[1]
	}
	if len(args) > 2 {
		frame.a2 = args[2]
	}

	runtimeCGOCall(linux386CallABI0, unsafe.Pointer(frame))
	result := frame.result
	runtime.KeepAlive(frame)
	return result
}
