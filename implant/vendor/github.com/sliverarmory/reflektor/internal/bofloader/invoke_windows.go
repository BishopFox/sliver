//go:build windows && (amd64 || arm64)

package bofloader

import "syscall"

//go:uintptrescapes
func invokeEntry(entry, argumentAddress uintptr, argumentLength int32) {
	syscall.SyscallN(entry, argumentAddress, uintptr(uint32(argumentLength)))
}
