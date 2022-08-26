// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"
)

type ICLRRuntimeHost struct {
	vtbl *ICLRRuntimeHostVtbl
}

// ICLRRuntimeHostVtbl provides functionality similar to that of the ICorRuntimeHost interface
// provided in the .NET Framework version 1, with the following changes
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimehost-interface
type ICLRRuntimeHostVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// Start Initializes the CLR into a process.
	Start uintptr
	// Stop Stops the execution of code by the runtime.
	Stop uintptr
	// SetHostControl sets the host control interface. You must call SetHostControl before calling Start.
	SetHostControl uintptr
	// GetCLRControl gets an interface pointer of type ICLRControl that hosts can use to customize
	// aspects of the common language runtime (CLR).
	GetCLRControl uintptr
	// UnloadAppDomain Unloads the AppDomain that corresponds to the specified numeric identifier.
	UnloadAppDomain uintptr
	// ExecuteInAppDomain Specifies the AppDomain in which to execute the specified managed code.
	ExecuteInAppDomain uintptr
	// GetCurrentAppDomainID gets the numeric identifier of the AppDomain that is currently executing.
	GetCurrentAppDomainId uintptr
	// ExecuteApplication used in manifest-based ClickOnce deployment scenarios to specify the application
	// to be activated in a new domain.
	ExecuteApplication uintptr
	// ExecuteInDefaultAppDomain Invokes the specified method of the specified type in the specified assembly.
	ExecuteInDefaultAppDomain uintptr
}

// GetICLRRuntimeHost is a wrapper function that takes an ICLRRuntimeInfo object and
// returns an ICLRRuntimeHost and loads it into the current process
func GetICLRRuntimeHost(runtimeInfo *ICLRRuntimeInfo) (*ICLRRuntimeHost, error) {
	debugPrint("Entering into iclrruntimehost.GetICLRRuntimeHost()...")
	var runtimeHost *ICLRRuntimeHost
	err := runtimeInfo.GetInterface(CLSID_CLRRuntimeHost, IID_ICLRRuntimeHost, unsafe.Pointer(&runtimeHost))
	if err != nil {
		return nil, err
	}

	err = runtimeHost.Start()
	return runtimeHost, err
}

func (obj *ICLRRuntimeHost) AddRef() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

func (obj *ICLRRuntimeHost) Release() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

// Start Initializes the common language runtime (CLR) into a process.
// HRESULT Start();
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimehost-start-method
func (obj *ICLRRuntimeHost) Start() error {
	debugPrint("Entering into iclrruntimehost.Start()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.Start,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		return fmt.Errorf("the ICLRRuntimeHost::Start method returned an error:\r\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the ICLRRuntimeHost::Start method method returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}

// ExecuteInDefaultAppDomain Calls the specified method of the specified type in the specified managed assembly.
// HRESULT ExecuteInDefaultAppDomain (
//   [in] LPCWSTR pwzAssemblyPath,
//   [in] LPCWSTR pwzTypeName,
//   [in] LPCWSTR pwzMethodName,
//   [in] LPCWSTR pwzArgument,
// [out] DWORD *pReturnValue
// );
// An LPCWSTR is a 32-bit pointer to a constant string of 16-bit Unicode characters, which MAY be null-terminated.
// Use syscall.UTF16PtrFromString to turn a string into a LPCWSTR
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimehost-executeindefaultappdomain-method
func (obj *ICLRRuntimeHost) ExecuteInDefaultAppDomain(pwzAssemblyPath, pwzTypeName, pwzMethodName, pwzArgument *uint16) (pReturnValue *uint32, err error) {
	hr, _, err := syscall.Syscall9(
		obj.vtbl.ExecuteInDefaultAppDomain,
		6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pwzAssemblyPath)),
		uintptr(unsafe.Pointer(pwzTypeName)),
		uintptr(unsafe.Pointer(pwzMethodName)),
		uintptr(unsafe.Pointer(pwzArgument)),
		uintptr(unsafe.Pointer(pReturnValue)),
		0,
		0,
		0)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("the ICLRRuntimeHost::ExecuteInDefaultAppDomain method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRRuntimeHost::ExecuteInDefaultAppDomain method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}

// GetCurrentAppDomainID Gets the numeric identifier of the AppDomain that is currently executing.
// HRESULT GetCurrentAppDomainId(
//   [out] DWORD* pdwAppDomainId
// );
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimehost-getcurrentappdomainid-method
func (obj *ICLRRuntimeHost) GetCurrentAppDomainID() (pdwAppDomainId uint32, err error) {
	hr, _, err := syscall.Syscall(
		obj.vtbl.GetCurrentAppDomainId,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pdwAppDomainId)),
		0,
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("the ICLRRuntimeHost::GetCurrentAppDomainID method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRRuntimeHost::GetCurrentAppDomainID method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}
