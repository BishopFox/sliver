//go:build linux && !android && arm && arm.7

package bofloader

import "github.com/ebitengine/purego"

//go:uintptrescapes
func invokeEntry(entry, argumentAddress uintptr, argumentLength int32) {
	purego.SyscallN(entry, argumentAddress, uintptr(uint32(argumentLength)))
}
