// Code generated by 'go generate'; DO NOT EDIT.

package winutil

import (
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modadvapi32 = windows.NewLazySystemDLL("advapi32.dll")
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procQueryServiceConfig2W       = modadvapi32.NewProc("QueryServiceConfig2W")
	procRegisterApplicationRestart = modkernel32.NewProc("RegisterApplicationRestart")
)

func queryServiceConfig2(hService windows.Handle, infoLevel uint32, buf *byte, bufLen uint32, bytesNeeded *uint32) (err error) {
	r1, _, e1 := syscall.Syscall6(procQueryServiceConfig2W.Addr(), 5, uintptr(hService), uintptr(infoLevel), uintptr(unsafe.Pointer(buf)), uintptr(bufLen), uintptr(unsafe.Pointer(bytesNeeded)), 0)
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return
}

func registerApplicationRestart(cmdLineExclExeName *uint16, flags uint32) (ret wingoes.HRESULT) {
	r0, _, _ := syscall.Syscall(procRegisterApplicationRestart.Addr(), 2, uintptr(unsafe.Pointer(cmdLineExclExeName)), uintptr(flags), 0)
	ret = wingoes.HRESULT(r0)
	return
}
