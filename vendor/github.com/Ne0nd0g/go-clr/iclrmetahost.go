// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Couldnt have done any of this without this SO answer I stumbled on:
// https://stackoverflow.com/questions/37781676/how-to-use-com-component-object-model-in-golang

//ICLRMetaHost Interface from metahost.h
type ICLRMetaHost struct {
	vtbl *ICLRMetaHostVtbl
}

// ICLRMetaHostVtbl provides methods that return a specific version of the common language runtime (CLR)
// based on its version number, list all installed CLRs, list all runtimes that are loaded in a specified
// process, discover the CLR version used to compile an assembly, exit a process with a clean runtime
// shutdown, and query legacy API binding.
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrmetahost-interface
type ICLRMetaHostVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// GetRuntime gets the ICLRRuntimeInfo interface that corresponds to a particular CLR version.
	// This method supersedes the CorBindToRuntimeEx function used with the STARTUP_LOADER_SAFEMODE flag.
	GetRuntime uintptr
	// GetVersionFromFile gets the assembly's original .NET Framework compilation version (stored in the metadata),
	// given its file path. This method supersedes GetFileVersion.
	GetVersionFromFile uintptr
	// EnumerateInstalledRuntimes returns an enumeration that contains a valid ICLRRuntimeInfo interface
	// pointer for each CLR version that is installed on a computer.
	EnumerateInstalledRuntimes uintptr
	// EnumerateLoadedRuntimes returns an enumeration that contains a valid ICLRRuntimeInfo interface
	// pointer for each CLR that is loaded in a given process. This method supersedes GetVersionFromProcess.
	EnumerateLoadedRuntimes uintptr
	// RequestRuntimeLoadedNotification guarantees a callback to the specified function pointer when a
	// CLR version is first loaded, but not yet started. This method supersedes LockClrVersion
	RequestRuntimeLoadedNotification uintptr
	// QueryLegacyV2RuntimeBinding returns an interface that represents a runtime to which legacy activation policy
	// has been bound, for example by using the useLegacyV2RuntimeActivationPolicy attribute on the <startup> Element
	// configuration file entry, by direct use of the legacy activation APIs, or by calling the
	// ICLRRuntimeInfo::BindAsLegacyV2Runtime method.
	QueryLegacyV2RuntimeBinding uintptr
	// ExitProcess attempts to shut down all loaded runtimes gracefully and then terminates the process.
	ExitProcess uintptr
}

// CLRCreateInstance provides one of three interfaces: ICLRMetaHost, ICLRMetaHostPolicy, or ICLRDebugging.
// HRESULT CLRCreateInstance(
//   [in]  REFCLSID  clsid,
//   [in]  REFIID     riid,
//   [out] LPVOID  * ppInterface
// );
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/clrcreateinstance-function
func CLRCreateInstance(clsid, riid windows.GUID) (ppInterface *ICLRMetaHost, err error) {
	debugPrint("Entering into iclrmetahost.CLRCreateInstance()...")

	if clsid != CLSID_CLRMetaHost {
		err = fmt.Errorf("the input Class ID (CLSID) is not supported: %s", clsid)
		return
	}

	modMSCoree := syscall.MustLoadDLL("mscoree.dll")
	procCLRCreateInstance := modMSCoree.MustFindProc("CLRCreateInstance")

	// For some reason this procedure call returns "The specified procedure could not be found." even though it works
	hr, _, err := procCLRCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)),
		uintptr(unsafe.Pointer(&riid)),
		uintptr(unsafe.Pointer(&ppInterface)),
	)

	if err != nil {
		// TODO Figure out why "The specified procedure could not be found." is returned even though everything works fine?
		debugPrint(fmt.Sprintf("the mscoree!CLRCreateInstance function returned an error:\r\n%s", err))
	}
	if hr != S_OK {
		err = fmt.Errorf("the mscoree!CLRCreateInstance function returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}

func (obj *ICLRMetaHost) QueryInterface(riid windows.GUID, ppvObject unsafe.Pointer) error {
	debugPrint("Entering into icorruntimehost.QueryInterface()...")
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

func (obj *ICLRMetaHost) AddRef() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

func (obj *ICLRMetaHost) Release() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

// EnumerateInstalledRuntimes returns an enumeration that contains a valid ICLRRuntimeInfo interface for each
// version of the common language runtime (CLR) that is installed on a computer.
// HRESULT EnumerateInstalledRuntimes (
//   [out, retval] IEnumUnknown **ppEnumerator);
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrmetahost-enumerateinstalledruntimes-method
func (obj *ICLRMetaHost) EnumerateInstalledRuntimes() (ppEnumerator *IEnumUnknown, err error) {
	debugPrint("Entering into iclrmetahost.EnumerateInstalledRuntimes()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.EnumerateInstalledRuntimes,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&ppEnumerator)),
		0,
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("there was an error calling the ICLRMetaHost::EnumerateInstalledRuntimes method:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRMetaHost::EnumerateInstalledRuntimes method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}

// GetRuntime gets the ICLRRuntimeInfo interface that corresponds to a particular version of the common language runtime (CLR).
// This method supersedes the CorBindToRuntimeEx function used with the STARTUP_LOADER_SAFEMODE flag.
// HRESULT GetRuntime (
//   [in] LPCWSTR pwzVersion,
//   [in] REFIID riid,
//   [out,iid_is(riid), retval] LPVOID *ppRuntime
// );
// https://docs.microsoft.com/en-us/dotnet/framework/unmanaged-api/hosting/iclrmetahost-getruntime-method
func (obj *ICLRMetaHost) GetRuntime(pwzVersion *uint16, riid windows.GUID) (ppRuntime *ICLRRuntimeInfo, err error) {
	debugPrint("Entering into iclrmetahost.GetRuntime()...")

	hr, _, err := syscall.Syscall6(
		obj.vtbl.GetRuntime,
		4,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pwzVersion)),
		uintptr(unsafe.Pointer(&IID_ICLRRuntimeInfo)),
		uintptr(unsafe.Pointer(&ppRuntime)),
		0,
		0,
	)

	if err != syscall.Errno(0) {
		err = fmt.Errorf("there was an error calling the ICLRMetaHost::GetRuntime method:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the ICLRMetaHost::GetRuntime method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}
