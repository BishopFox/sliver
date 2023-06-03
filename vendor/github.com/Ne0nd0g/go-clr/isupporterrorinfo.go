// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ISupportErrorInfo Ensures that error information can be propagated up the call chain correctly.
// Automation objects that use the error handling interfaces must implement ISupportErrorInfo
// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/nn-oaidl-isupporterrorinfo
type ISupportErrorInfo struct {
	vtbl *ISupportErrorInfoVtbl
}

type ISupportErrorInfoVtbl struct {
	// QueryInterface Retrieves pointers to the supported interfaces on an object.
	QueryInterface uintptr
	// AddRef Increments the reference count for an interface pointer to a COM object.
	// You should call this method whenever you make a copy of an interface pointer.
	AddRef uintptr
	// Release Decrements the reference count for an interface on a COM object.
	Release uintptr
	// InterfaceSupportsErrorInfo Indicates whether an interface supports the IErrorInfo interface.
	// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/nf-oaidl-isupporterrorinfo-interfacesupportserrorinfo
	InterfaceSupportsErrorInfo uintptr
}

// QueryInterface queries a COM object for a pointer to one of its interface;
// identifying the interface by a reference to its interface identifier (IID).
// If the COM object implements the interface, then it returns a pointer to that interface after calling IUnknown::AddRef on it.
// HRESULT QueryInterface(
//   REFIID riid,
//   void   **ppvObject
// );
// https://docs.microsoft.com/en-us/windows/win32/api/unknwn/nf-unknwn-iunknown-queryinterface(refiid_void)
func (obj *ISupportErrorInfo) QueryInterface(riid windows.GUID, ppvObject unsafe.Pointer) error {
	debugPrint("Entering into iunknown.QueryInterface()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.QueryInterface,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&riid)), // A reference to the interface identifier (IID) of the interface being queried for.
		uintptr(ppvObject),
	)
	if err != syscall.Errno(0) {
		return fmt.Errorf("the IUknown::QueryInterface method returned an error:\r\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the IUknown::QueryInterface method method returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}

// AddRef Increments the reference count for an interface pointer to a COM object.
// You should call this method whenever you make a copy of an interface pointer
// ULONG AddRef();
// https://docs.microsoft.com/en-us/windows/win32/api/unknwn/nf-unknwn-iunknown-addref
func (obj *ISupportErrorInfo) AddRef() (count uint32, err error) {
	debugPrint("Entering into iunknown.AddRef()...")
	ret, _, err := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		return 0, fmt.Errorf("the IUnknown::AddRef method returned an error:\r\n%s", err)
	}
	err = nil
	// Unable to avoid misuse of unsafe.Pointer because the Windows API call returns the safeArray pointer in the "ret" value. This is a go vet false positive
	count = *(*uint32)(unsafe.Pointer(ret))
	return
}

// Release Decrements the reference count for an interface on a COM object.
// ULONG Release();
// https://docs.microsoft.com/en-us/windows/win32/api/unknwn/nf-unknwn-iunknown-release
func (obj *ISupportErrorInfo) Release() (count uint32, err error) {
	debugPrint("Entering into iunknown.Release()...")
	ret, _, err := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		return 0, fmt.Errorf("the IUnknown::Release method returned an error:\r\n%s", err)
	}
	err = nil
	// Unable to avoid misuse of unsafe.Pointer because the Windows API call returns the safeArray pointer in the "ret" value. This is a go vet false positive
	count = *(*uint32)(unsafe.Pointer(ret))
	return
}

// InterfaceSupportsErrorInfo
// HRESULT InterfaceSupportsErrorInfo(
//   REFIID riid
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/nf-oaidl-isupporterrorinfo-interfacesupportserrorinfo
func (obj *ISupportErrorInfo) InterfaceSupportsErrorInfo(riid windows.GUID) error {
	debugPrint("Entering into isupporterrorinfo.InterfaceSupportsErrorInfo()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.InterfaceSupportsErrorInfo,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&riid)),
		0,
	)
	if err != syscall.Errno(0) {
		return fmt.Errorf("the ISupportErrorInfo::InterfaceSupportsErrorInfo method returned an error:\r\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the ISupportErrorInfo::InterfaceSupportsErrorInfo method method returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}
