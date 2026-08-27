//go:build linux && !android && 386

package bofloader

import (
	"runtime"
	"unsafe"

	_ "github.com/ebitengine/purego"
)

type linux386InvokeFrame struct {
	entry   uintptr
	address uintptr
	length  int32
}

var linux386InvokeABI0 uintptr

//go:linkname runtimeCGOCall runtime.cgocall
func runtimeCGOCall(function uintptr, argument unsafe.Pointer) int32

//go:uintptrescapes
func invokeEntry(entry, argumentAddress uintptr, argumentLength int32) {
	frame := &linux386InvokeFrame{entry: entry, address: argumentAddress, length: argumentLength}
	runtimeCGOCall(linux386InvokeABI0, unsafe.Pointer(frame))
	runtime.KeepAlive(frame)
}
