package limits

import (
	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
	"os"
	"syscall"

	// {{if .LimitDomainJoined}}
	"unsafe"
	// {{else}}{{end}}
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
	kernel32 := syscall.MustLoadDLL("kernel32.dll")
	isDebuggerPresent := kernel32.MustFindProc("IsDebuggerPresent")
	var nargs uintptr = 0
	ret, _, _ := isDebuggerPresent.Call(nargs)
	// {{if .Debug}}
	log.Printf("IsDebuggerPresent = %#v\n", int32(ret))
	// {{end}}
	if int32(ret) != 0 {
		os.Exit(1)
	}
}
