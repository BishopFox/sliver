// +build windows

package clr

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// from mscorlib.tlh

type Assembly struct {
	vtbl *AssemblyVtbl
}

// AssemblyVtbl is a COM virtual table of functions for the Assembly Class
// https://docs.microsoft.com/en-us/dotnet/api/system.reflection.assembly?view=netframework-4.8
type AssemblyVtbl struct {
	QueryInterface              uintptr
	AddRef                      uintptr
	Release                     uintptr
	GetTypeInfoCount            uintptr
	GetTypeInfo                 uintptr
	GetIDsOfNames               uintptr
	Invoke                      uintptr
	get_ToString                uintptr
	Equals                      uintptr
	GetHashCode                 uintptr
	GetType                     uintptr
	get_CodeBase                uintptr
	get_EscapedCodeBase         uintptr
	GetName                     uintptr
	GetName_2                   uintptr
	get_FullName                uintptr
	get_EntryPoint              uintptr
	GetType_2                   uintptr
	GetType_3                   uintptr
	GetExportedTypes            uintptr
	GetTypes                    uintptr
	GetManifestResourceStream   uintptr
	GetManifestResourceStream_2 uintptr
	GetFile                     uintptr
	GetFiles                    uintptr
	GetFiles_2                  uintptr
	GetManifestResourceNames    uintptr
	GetManifestResourceInfo     uintptr
	get_Location                uintptr
	get_Evidence                uintptr
	GetCustomAttributes         uintptr
	GetCustomAttributes_2       uintptr
	IsDefined                   uintptr
	GetObjectData               uintptr
	add_ModuleResolve           uintptr
	remove_ModuleResolve        uintptr
	GetType_4                   uintptr
	GetSatelliteAssembly        uintptr
	GetSatelliteAssembly_2      uintptr
	LoadModule                  uintptr
	LoadModule_2                uintptr
	CreateInstance              uintptr
	CreateInstance_2            uintptr
	CreateInstance_3            uintptr
	GetLoadedModules            uintptr
	GetLoadedModules_2          uintptr
	GetModules                  uintptr
	GetModules_2                uintptr
	GetModule                   uintptr
	GetReferencedAssemblies     uintptr
	get_GlobalAssemblyCache     uintptr
}

func (obj *Assembly) QueryInterface(riid *windows.GUID, ppvObject *uintptr) uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.QueryInterface,
		3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(ppvObject)))
	return ret
}

func (obj *Assembly) AddRef() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.AddRef,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

func (obj *Assembly) Release() uintptr {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return ret
}

// GetEntryPoint returns the assembly's MethodInfo
//      virtual HRESULT __stdcall get_EntryPoint (
//     /*[out,retval]*/ struct _MethodInfo * * pRetVal ) = 0;
// https://docs.microsoft.com/en-us/dotnet/api/system.reflection.assembly.entrypoint?view=netframework-4.8#System_Reflection_Assembly_EntryPoint
// https://docs.microsoft.com/en-us/dotnet/api/system.reflection.methodinfo?view=netframework-4.8
func (obj *Assembly) GetEntryPoint() (pRetVal *MethodInfo, err error) {
	debugPrint("Entering into assembly.GetEntryPoint()...")
	hr, _, err := syscall.Syscall(
		obj.vtbl.get_EntryPoint,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pRetVal)),
		0,
	)
	if err != syscall.Errno(0) {
		err = fmt.Errorf("the Assembly::GetEntryPoint method returned an error:\r\n%s", err)
		return
	}
	if hr != S_OK {
		err = fmt.Errorf("the Assembly::GetEntryPoint method returned a non-zero HRESULT: 0x%x", hr)
		return
	}
	err = nil
	return
}
