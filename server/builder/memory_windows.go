package builder

import (
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	Length            uint32
	MemoryLoad        uint32
	TotalPhysical     uint64
	AvailablePhysical uint64
	TotalPageFile     uint64
	AvailablePageFile uint64
	TotalVirtual      uint64
	AvailableVirtual  uint64
	AvailableExtended uint64
}

var globalMemoryStatusEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func availableMemoryBytes() uint64 {
	status := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	ok, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return 0
	}
	return status.AvailablePhysical
}
