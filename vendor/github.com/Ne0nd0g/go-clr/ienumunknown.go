// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"
)

type IEnumUnknown struct {
	vtbl *IEnumUnknownVtbl
}

// IEnumUnknownVtbl Enumerates objects implementing the root COM interface, IUnknown.
// Commonly implemented by a component containing multiple objects. For more information, see IEnumUnknown.
// https://docs.microsoft.com/en-us/windows/win32/api/objidl/nn-objidl-ienumunknown
type IEnumUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// Next Retrieves the specified number of items in the enumeration sequence.
	Next uintptr
	// Skip Skips over the specified number of items in the enumeration sequence.
	Skip uintptr
	// Reset Resets the enumeration sequence to the beginning.
	Reset uintptr
	// Clone Creates a new enumerator that contains the same enumeration state as the current one.
	Clone uintptr
}

func (obj *IEnumUnknown) AddRef() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

func (obj *IEnumUnknown) Release() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

// Next retrieves the specified number of items in the enumeration sequence.
// HRESULT Next(
//   ULONG    celt,
//   IUnknown **rgelt,
//   ULONG    *pceltFetched
// );
// https://docs.microsoft.com/en-us/windows/win32/api/objidl/nf-objidl-ienumunknown-next
func (obj *IEnumUnknown) Next(celt uint32, pEnumRuntime unsafe.Pointer, pceltFetched *uint32) (hresult int, err error) {
	debugPrint("Entering into ienumunknown.Next()...")
	hr, _, err := syscall.Syscall6(
		obj.vtbl.Next,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(celt),
		uintptr(pEnumRuntime),
		uintptr(unsafe.Pointer(pceltFetched)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("there was an error calling the IEnumUnknown::Next method:\r\n%s", err)
		return
	}
	if hr != S_OK && hr != S_FALSE {
		err = fmt.Errorf("the IEnumUnknown::Next method method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	hresult = int(hr)
	return
}
