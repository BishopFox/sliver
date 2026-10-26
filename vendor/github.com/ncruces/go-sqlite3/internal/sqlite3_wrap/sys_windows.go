package sqlite3_wrap

import "golang.org/x/sys/windows"

// https://devblogs.microsoft.com/oldnewthing/?p=42223
const allocationGranularity = 64 * 1024

const (
	_MEM_PRESERVE_PLACEHOLDER = 0x00000002
	_MEM_REPLACE_PLACEHOLDER  = 0x00004000
	_MEM_RESERVE_PLACEHOLDER  = 0x00040000
)

var (
	kernelbase           = windows.NewLazySystemDLL("kernelbase.dll")
	procVirtualAlloc2    = kernelbase.NewProc("VirtualAlloc2")
	procMapViewOfFile3   = kernelbase.NewProc("MapViewOfFile3")
	procUnmapViewOfFile2 = kernelbase.NewProc("UnmapViewOfFile2")
)

// Reports whether the placeholder APIs are available
// (Windows 10 1803+ / Server 2019+).
func placeholdersSupported() bool {
	return procVirtualAlloc2.Find() == nil &&
		procMapViewOfFile3.Find() == nil &&
		procUnmapViewOfFile2.Find() == nil
}

func virtualAlloc2(address, size uintptr, alloctype, protect uint32) (addr uintptr, err error) {
	addr, _, err = procVirtualAlloc2.Call(
		^uintptr(0), // current process pseudo handle
		address, size, uintptr(alloctype), uintptr(protect),
		0, 0) // no extended parameters
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

func mapViewOfFile3(handle windows.Handle, address uintptr, offset uint64, size uintptr, alloctype, protect uint32) (addr uintptr, err error) {
	addr, _, err = procMapViewOfFile3.Call(
		uintptr(handle),
		^uintptr(0), // current process pseudo handle
		address, uintptr(offset), size, uintptr(alloctype), uintptr(protect),
		0, 0) // no extended parameters
	if addr == 0 {
		return 0, err
	}
	return addr, nil
}

func unmapViewOfFile2(addr uintptr, flags uint32) (err error) {
	r, _, err := procUnmapViewOfFile2.Call(
		^uintptr(0), // current process pseudo handle
		addr, uintptr(flags))
	if r == 0 {
		return err
	}
	return nil
}
