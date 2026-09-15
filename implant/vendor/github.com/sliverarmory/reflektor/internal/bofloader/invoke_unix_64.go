//go:build (darwin && !ios && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && !android && (amd64 || arm64 || ppc64le || riscv64))

package bofloader

import "github.com/ebitengine/purego"

//go:uintptrescapes
func invokeEntry(entry, argumentAddress uintptr, argumentLength int32) {
	purego.SyscallN(entry, argumentAddress, uintptr(uint32(argumentLength)))
}
