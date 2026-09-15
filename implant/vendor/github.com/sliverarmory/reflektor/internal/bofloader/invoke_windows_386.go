//go:build windows && 386

package bofloader

import (
	"runtime"
	"unsafe"

	_ "github.com/ebitengine/purego"
)

type windows386InvokeFrame struct {
	entry   uintptr
	address uintptr
	length  int32
}

var windows386InvokeABI0 uintptr

// runtimeCGOCall is declared by invoke_linux_386.go on linux/386 and here on
// windows/386. runtime.cgocall is available without CGO and provides the
// scheduler-stack transition required when the BOF calls back into Go.
//
//go:linkname runtimeCGOCall runtime.cgocall
func runtimeCGOCall(function uintptr, argument unsafe.Pointer) int32

//go:uintptrescapes
func invokeEntry(entry, argumentAddress uintptr, argumentLength int32) {
	frame := &windows386InvokeFrame{entry: entry, address: argumentAddress, length: argumentLength}
	runtimeCGOCall(windows386InvokeABI0, unsafe.Pointer(frame))
	runtime.KeepAlive(frame)
}
