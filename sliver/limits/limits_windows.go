package limits

import (
	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
	"os"
	"syscall"
	"unsafe"
)

// {{if .LimitDomainJoined}}

func isDomainJoined() (bool, error) {
	var domain *uint16
	var status uint32
	err := syscall.NetGetJoinInformation(nil, &domain, &status)
	if err != nil {
		return false, err
	}
	syscall.NetApiBufferFree((*byte)(unsafe.Pointer(domain)))
	return status == syscall.NetSetupDomainName, nil
}

// {{end}}

func PlatformLimits() {
	kernel32, _ = syscall.LoadLibrary("kernel32.dll")
	defer syscall.FreeLibrary(kernel32)
	isDebuggerPresent, _ = syscall.GetProcAddress(kernel32, "IsDebuggerPresent")
	ret, _, err := syscall.Syscall(uintptr(isDebuggerPresent))
	// {{if .Debug}}
	log.Printf("IsDebuggerPresent = %#v", ret)
	// {{end}}
	// {{if .PlatformLimits}}
	if err == nil && ret {
		os.Exit(1)
	}
	// {{end}}
}
