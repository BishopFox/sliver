//go:build windows

package builder

import (
	"math/bits"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getProcessAffinityMask = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessAffinityMask")

// A process affinity mask narrows the CPUs available to a Windows builder.
// If the OS does not return one, runtime.NumCPU remains the safe fallback.
func osCPULimits() (affinity, quota int, quotaKnown bool) {
	handle, err := windows.GetCurrentProcess()
	if err != nil {
		return 0, 0, false
	}
	var processMask, systemMask uintptr
	ok, _, _ := getProcessAffinityMask.Call(uintptr(handle), uintptr(unsafe.Pointer(&processMask)), uintptr(unsafe.Pointer(&systemMask)))
	if ok == 0 || processMask == 0 {
		return 0, 0, false
	}
	return bits.OnesCount64(uint64(processMask)), 0, false
}
