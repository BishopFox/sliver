// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ICLRRuntimeInfo struct {
	vtbl *ICLRRuntimeInfoVtbl
}

// ICLRRuntimeInfoVtbl Provides methods that return information about a specific common language runtime (CLR),
// including version, directory, and load status. This interface also provides runtime-specific functionality
// without initializing the runtime. It includes the runtime-relative LoadLibrary method, the runtime
// module-specific GetProcAddress method, and runtime-provided interfaces through the GetInterface method.
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimeinfo-interface
type ICLRRuntimeInfoVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// GetVersionString Gets common language runtime (CLR) version information associated with a given
	// ICLRRuntimeInfo interface. This method supersedes the GetRequestedRuntimeInfo and GetRequestedRuntimeVersion methods.
	GetVersionString uintptr
	// GetRuntimeDirectory Gets the installation directory of the CLR associated with this interface.
	// This method supersedes the GetCORSystemDirectory method.
	GetRuntimeDirectory uintptr
	// IsLoaded Indicates whether the CLR associated with the ICLRRuntimeInfo interface is loaded into a process.
	IsLoaded uintptr
	// LoadErrorString Translates an HRESULT value into an appropriate error message for the specified culture.
	// This method supersedes the LoadStringRC and LoadStringRCEx methods.
	LoadErrorString uintptr
	// LoadLibrary Loads a library from the framework directory of the CLR represented by an ICLRRuntimeInfo interface.
	// This method supersedes the LoadLibraryShim method.
	LoadLibrary uintptr
	// GetProcAddress Gets the address of a specified function that was exported from the CLR associated with
	// this interface. This method supersedes the GetRealProcAddress method.
	GetProcAddress uintptr
	// GetInterface Loads the CLR into the current process and returns runtime interface pointers,
	// such as ICLRRuntimeHost, ICLRStrongName and IMetaDataDispenser. This method supersedes all the CorBindTo* functions.
	GetInterface uintptr
	// IsLoadable Indicates whether the runtime associated with this interface can be loaded into the current
	// process, taking into account other runtimes that might already be loaded into the process.
	IsLoadable uintptr
	// SetDefaultStartupFlags Sets the CLR startup flags and host configuration file.
	SetDefaultStartupFlags uintptr
	// GetDefaultStartupFlags Gets the CLR startup flags and host configuration file.
	GetDefaultStartupFlags uintptr
	// BindAsLegacyV2Runtime Binds this runtime for all legacy CLR version 2 activation policy decisions.
	BindAsLegacyV2Runtime uintptr
	// IsStarted Indicates whether the CLR that is associated with the ICLRRuntimeInfo interface has been started.
	IsStarted uintptr
}

// GetRuntimeInfo is a wrapper function to return an ICLRRuntimeInfo from a standard version string
func GetRuntimeInfo(metahost *ICLRMetaHost, version string) (*ICLRRuntimeInfo, error) {
	pwzVersion, err := syscall.UTF16PtrFromString(version)
	if err != nil {
		return nil, err
	}
	return metahost.GetRuntime(pwzVersion, IID_ICLRRuntimeInfo)
}

func (obj *ICLRRuntimeInfo) AddRef() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

func (obj *ICLRRuntimeInfo) Release() uintptr {
	debugPrint("Entering into iclrruntimeinfo.Release()...")
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

// GetVersionString gets common language runtime (CLR) version information associated with a given ICLRRuntimeInfo interface.
// HRESULT GetVersionString(
//   [out, size_is(*pcchBuffer)] LPWSTR pwzBuffer,
//   [in, out]  DWORD *pcchBuffer);
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimeinfo-getversionstring-method
func (obj *ICLRRuntimeInfo) GetVersionString() (version string, err error) {
	debugPrint("Entering into iclrruntimeinfo.GetVersion()...")
	// [in, out] Specifies the size of pwzBuffer to avoid buffer overruns. If pwzBuffer is null, pchBuffer returns the required size of pwzBuffer to allow preallocation.
	var pchBuffer uint32
	hr, _, err := syscall.Syscall(
		obj.vtbl.GetVersionString,
		3,
		uintptr(unsafe.Pointer(obj)),
		0,
		uintptr(unsafe.Pointer(&pchBuffer)),
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("there was an error calling the ICLRRuntimeInfo::GetVersionString method during preallocation:\r\n%s", err)
		return
	}
	// 0x8007007a = The data area passed to a system call is too small, expected when passing a nil buffer for preallocation
	if hr != S_OK && hr != 0x8007007a {
		err = fmt.Errorf("the ICLRRuntimeInfo::GetVersionString method (preallocation) returned a non-zero HRESULT: 0x%x", hr)
		return
	}

	pwzBuffer := make([]uint16, 20)

	hr, _, err = syscall.Syscall(
		obj.vtbl.GetVersionString,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pwzBuffer[0])),
		uintptr(unsafe.Pointer(&pchBuffer)),
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("there was an error calling the ICLRRuntimeInfo::GetVersionString method:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRRuntimeInfo::GetVersionString method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	version = syscall.UTF16ToString(pwzBuffer)
	return
}

// GetInterface loads the CLR into the current process and returns runtime interface pointers,
// such as ICLRRuntimeHost, ICLRStrongName, and IMetaDataDispenserEx.
// HRESULT GetInterface(
//   [in]  REFCLSID rclsid,
//   [in]  REFIID   riid,
//   [out, iid_is(riid), retval] LPVOID *ppUnk); unsafe pointer of a pointer to an object pointer
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimeinfo-getinterface-method
func (obj *ICLRRuntimeInfo) GetInterface(rclsid windows.GUID, riid windows.GUID, ppUnk unsafe.Pointer) error {
	debugPrint("Entering into iclrruntimeinfo.GetInterface()...")
	hr, _, err := syscall.Syscall6(
		obj.vtbl.GetInterface,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&rclsid)),
		uintptr(unsafe.Pointer(&riid)),
		uintptr(ppUnk),
		0,
		0,
	)
	// The syscall returns "The requested lookup key was not found in any active activation context." in the error position
	// TODO Why is this error message returned?
	if err != syscall.Errno(0) && err.Error() != "The requested lookup key was not found in any active activation context." {
		return fmt.Errorf("the ICLRRuntimeInfo::GetInterface method returned an error:\r\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the ICLRRuntimeInfo::GetInterface method returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}

// BindAsLegacyV2Runtime binds the current runtime for all legacy common language runtime (CLR) version 2 activation policy decisions.
// HRESULT BindAsLegacyV2Runtime ();
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimeinfo-bindaslegacyv2runtime-method
func (obj *ICLRRuntimeInfo) BindAsLegacyV2Runtime() error {
	debugPrint("Entering into iclrruntimeinfo.BindAsLegacyV2Runtime()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.BindAsLegacyV2Runtime,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	if err != syscall.Errno(0) {
		return fmt.Errorf("the ICLRRuntimeInfo::BindAsLegacyV2Runtime method returned an error:\r\n%s", err)
	}
	if hr != S_OK {
		return fmt.Errorf("the ICLRRuntimeInfo::BindAsLegacyV2Runtime method returned a non-zero HRESULT: 0x%x", hr)
	}
	return nil
}

// IsLoadable indicates whether the runtime associated with this interface can be loaded into the current process,
// taking into account other runtimes that might already be loaded into the process.
// HRESULT IsLoadable(
//   [out, retval] BOOL *pbLoadable);
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrruntimeinfo-isloadable-method
func (obj *ICLRRuntimeInfo) IsLoadable() (pbLoadable bool, err error) {
	debugPrint("Entering into iclrruntimeinfo.IsLoadable()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.IsLoadable,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pbLoadable)),
		0)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("the ICLRRuntimeInfo::IsLoadable method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRRuntimeInfo::IsLoadable method  returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}
