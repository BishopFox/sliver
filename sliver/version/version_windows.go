package version

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

type osVersionInfoEx struct {
	osVersionInfoSize uint32
	major             uint32
	minor             uint32
	build             uint32
	platformID        uint32
	csdVersion        [128]uint16
	servicePackMajor  uint16
	servicePackMinor  uint16
	suiteMask         uint16
	productType       byte
	wReserved         byte
}

func GetVersion() string {
	kernel32 := syscall.MustLoadDLL("ntdll.dll")
	procGetProductInfo := kernel32.MustFindProc("RtlGetVersion")
	osVersion := osVersionInfoEx{}
	osVersion.osVersionInfoSize = uint32(unsafe.Sizeof(osVersion))
	r1, _, err := procGetProductInfo.Call(uintptr(unsafe.Pointer(&osVersion)))
	if r1 != 0 {
		log.Fatal(err)
	}
	return fmt.Sprintf("%d.%d build %d", osVersion.major, osVersion.minor, osVersion.build)
}
