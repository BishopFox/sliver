// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type IErrorInfo struct {
	vtbl *IErrorInfoVtbl
}

// IErrorInfoVtbl returns information about an error in addition to the return code.
// It returns the error message, name of the component and GUID of the interface in
// which the error occurred, and the name and topic of the Help file that applies to the error.
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/ms723041(v=vs.85)
type IErrorInfoVtbl struct {
	// QueryInterface Retrieves pointers to the supported interfaces on an object.
	QueryInterface uintptr
	// AddRef Increments the reference count for an interface pointer to a COM object.
	// You should call this method whenever you make a copy of an interface pointer.
	AddRef uintptr
	// Release Decrements the reference count for an interface on a COM object.
	Release uintptr
	// GetDescription Returns a text description of the error
	GetDescription uintptr
	// GetGUID Returns the GUID of the interface that defined the error.
	GetGUID uintptr
	// GetHelpContext Returns the Help context ID for the error.
	GetHelpContext uintptr
	// GetHelpFile Returns the path of the Help file that describes the error.
	GetHelpFile uintptr
	// GetSource Returns the name of the component that generated the error, such as "ODBC driver-name".
	GetSource uintptr
}

// GetDescription Returns a text description of the error.
// HRESULT GetDescription (
//   BSTR *pbstrDescription);
// https://docs.microsoft.com/en-us/previous-versions/windows/desktop/ms714318(v=vs.85)
func (obj *IErrorInfo) GetDescription() (pbstrDescription *string, err error) {
	debugPrint("Entering into ierrorinfo.GetDescription()...")

	hr, _, err := syscall.Syscall(
		obj.vtbl.GetDescription,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pbstrDescription)),
		0,
	)

	if err != syscall.Errno(0) {
		err = fmt.Errorf("the IErrorInfo::GetDescription method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the IErrorInfo::GetDescription method method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}

// GetGUID Returns the globally unique identifier (GUID) of the interface that defined the error.
// HRESULT GetGUID(
//   GUID *pGUID
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oaidl/nf-oaidl-ierrorinfo-getguid
func (obj *IErrorInfo) GetGUID() (pGUID *windows.GUID, err error) {
	debugPrint("Entering into ierrorinfo.GetGUID()...")

	hr, _, err := syscall.Syscall(
		obj.vtbl.GetGUID,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pGUID)),
		0,
	)

	if err != syscall.Errno(0) {
		err = fmt.Errorf("the IErrorInfo::GetGUID method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the IErrorInfo::GetGUID method method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}

// GetErrorInfo Obtains the error information pointer set by the previous call to SetErrorInfo in the current logical thread.
// HRESULT GetErrorInfo(
//   ULONG      dwReserved,
//   IErrorInfo **pperrinfo
// );
// https://docs.microsoft.com/en-us/windows/win32/api/oleauto/nf-oleauto-geterrorinfo
func GetErrorInfo() (pperrinfo *IErrorInfo, err error) {
	debugPrint("Entering into ierrorinfo.GetErrorInfo()...")
	modOleAut32 := syscall.MustLoadDLL("OleAut32.dll")
	procGetErrorInfo := modOleAut32.MustFindProc("GetErrorInfo")
	hr, _, err := procGetErrorInfo.Call(0, uintptr(unsafe.Pointer(&pperrinfo)))
	if err != syscall.Errno(0) {
		err = fmt.Errorf("the OleAu32.GetErrorInfo procedure call returned an error:\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the OleAu32.GetErrorInfo procedure call returned a non-zero HRESULT code: 0x%x", hr)
		return
	}
	err = nil
	return
}
