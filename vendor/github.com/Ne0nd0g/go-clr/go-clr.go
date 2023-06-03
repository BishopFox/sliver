//go:build windows
// +build windows

// Package clr is a PoC package that wraps Windows syscalls necessary to load and the CLR into the current process and
// execute a managed DLL from disk or a managed EXE from memory
package clr

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// GetInstalledRuntimes is a wrapper function that returns an array of installed runtimes. Requires an existing ICLRMetaHost
func GetInstalledRuntimes(metahost *ICLRMetaHost) ([]string, error) {
	var runtimes []string
	enumICLRRuntimeInfo, err := metahost.EnumerateInstalledRuntimes()
	if err != nil {
		return runtimes, err
	}

	var hr int
	for hr != S_FALSE {
		var runtimeInfo *ICLRRuntimeInfo
		var fetched = uint32(0)
		hr, err = enumICLRRuntimeInfo.Next(1, unsafe.Pointer(&runtimeInfo), &fetched)
		if err != nil {
			return runtimes, fmt.Errorf("InstalledRuntimes Next Error:\r\n%s\n", err)
		}
		if hr == S_FALSE {
			break
		}
		// Only release if an interface pointer was returned
		runtimeInfo.Release()

		version, err := runtimeInfo.GetVersionString()
		if err != nil {
			return runtimes, err
		}
		runtimes = append(runtimes, version)
	}
	if len(runtimes) == 0 {
		return runtimes, fmt.Errorf("Could not find any installed runtimes")
	}
	return runtimes, err
}

// ExecuteDLLFromDisk is a wrapper function that will automatically load the latest installed CLR into the current process
// and execute a DLL on disk in the default app domain. It takes in the target runtime, DLLPath, TypeName, MethodName
// and Argument to use as strings. It returns the return code from the assembly
func ExecuteDLLFromDisk(targetRuntime, dllpath, typeName, methodName, argument string) (retCode int16, err error) {
	retCode = -1
	if targetRuntime == "" {
		targetRuntime = "v4"
	}
	metahost, err := CLRCreateInstance(CLSID_CLRMetaHost, IID_ICLRMetaHost)
	if err != nil {
		return
	}

	runtimes, err := GetInstalledRuntimes(metahost)
	if err != nil {
		return
	}
	var latestRuntime string
	for _, r := range runtimes {
		if strings.Contains(r, targetRuntime) {
			latestRuntime = r
			break
		} else {
			latestRuntime = r
		}
	}
	runtimeInfo, err := GetRuntimeInfo(metahost, latestRuntime)
	if err != nil {
		return
	}

	isLoadable, err := runtimeInfo.IsLoadable()
	if err != nil {
		return
	}
	if !isLoadable {
		return -1, fmt.Errorf("%s is not loadable for some reason", latestRuntime)
	}
	runtimeHost, err := GetICLRRuntimeHost(runtimeInfo)
	if err != nil {
		return
	}

	pDLLPath, err := syscall.UTF16PtrFromString(dllpath)
	if err != nil {
		return
	}
	pTypeName, err := syscall.UTF16PtrFromString(typeName)
	if err != nil {
		return
	}
	pMethodName, err := syscall.UTF16PtrFromString(methodName)
	if err != nil {
		return
	}
	pArgument, err := syscall.UTF16PtrFromString(argument)
	if err != nil {
		return
	}

	ret, err := runtimeHost.ExecuteInDefaultAppDomain(pDLLPath, pTypeName, pMethodName, pArgument)
	if err != nil {
		return
	}
	if *ret != 0 {
		return int16(*ret), fmt.Errorf("the ICLRRuntimeHost::ExecuteInDefaultAppDomain method returned a non-zero return value: %d", *ret)
	}

	runtimeHost.Release()
	runtimeInfo.Release()
	metahost.Release()
	return 0, nil
}

// ExecuteByteArray is a wrapper function that will automatically loads the supplied target framework into the current
// process using the legacy APIs, then load and execute an executable from memory. If no targetRuntime is specified, it
// will default to latest. It takes in a byte array of the executable to load and run and returns the return code.
// You can supply an array of strings as command line arguments.
func ExecuteByteArray(targetRuntime string, rawBytes []byte, params []string) (retCode int32, err error) {
	retCode = -1
	if targetRuntime == "" {
		targetRuntime = "v4"
	}
	metahost, err := CLRCreateInstance(CLSID_CLRMetaHost, IID_ICLRMetaHost)
	if err != nil {
		return
	}

	runtimes, err := GetInstalledRuntimes(metahost)
	if err != nil {
		return
	}
	var latestRuntime string
	for _, r := range runtimes {
		if strings.Contains(r, targetRuntime) {
			latestRuntime = r
			break
		} else {
			latestRuntime = r
		}
	}
	runtimeInfo, err := GetRuntimeInfo(metahost, latestRuntime)
	if err != nil {
		return
	}

	isLoadable, err := runtimeInfo.IsLoadable()
	if err != nil {
		return
	}
	if !isLoadable {
		return -1, fmt.Errorf("%s is not loadable for some reason", latestRuntime)
	}
	runtimeHost, err := GetICORRuntimeHost(runtimeInfo)
	if err != nil {
		return
	}
	appDomain, err := GetAppDomain(runtimeHost)
	if err != nil {
		return
	}
	safeArrayPtr, err := CreateSafeArray(rawBytes)
	if err != nil {
		return
	}

	assembly, err := appDomain.Load_3(safeArrayPtr)
	if err != nil {
		return
	}

	methodInfo, err := assembly.GetEntryPoint()
	if err != nil {
		return
	}

	var paramSafeArray *SafeArray
	methodSignature, err := methodInfo.GetString()
	if err != nil {
		return
	}

	if expectsParams(methodSignature) {
		if paramSafeArray, err = PrepareParameters(params); err != nil {
			return
		}
	}

	nullVariant := Variant{
		VT:  1,
		Val: uintptr(0),
	}
	err = methodInfo.Invoke_3(nullVariant, paramSafeArray)
	if err != nil {
		return
	}
	appDomain.Release()
	runtimeHost.Release()
	runtimeInfo.Release()
	metahost.Release()
	return 0, nil
}

// LoadCLR loads the target runtime into the current process and returns the runtimehost
// The intended purpose is for the runtimehost to be reused for subsequent operations
// throughout the duration of the program. Commonly used with C2 frameworks
func LoadCLR(targetRuntime string) (runtimeHost *ICORRuntimeHost, err error) {
	if targetRuntime == "" {
		targetRuntime = "v4"
	}

	metahost, err := CLRCreateInstance(CLSID_CLRMetaHost, IID_ICLRMetaHost)
	if err != nil {
		return runtimeHost, fmt.Errorf("there was an error enumerating the installed CLR runtimes:\n%s", err)
	}

	runtimes, err := GetInstalledRuntimes(metahost)
	debugPrint(fmt.Sprintf("Installed Runtimes: %v", runtimes))
	if err != nil {
		return
	}
	var latestRuntime string
	for _, r := range runtimes {
		if strings.Contains(r, targetRuntime) {
			latestRuntime = r
			break
		} else {
			latestRuntime = r
		}
	}
	runtimeInfo, err := GetRuntimeInfo(metahost, latestRuntime)
	if err != nil {
		return
	}

	isLoadable, err := runtimeInfo.IsLoadable()
	if err != nil {
		return
	}
	if !isLoadable {
		err = fmt.Errorf("%s is not loadable for some reason", latestRuntime)
	}

	return GetICORRuntimeHost(runtimeInfo)
}

// ExecuteByteArrayDefaultDomain uses a previously instantiated runtimehost, gets the default AppDomain,
// loads the assembly into, executes the assembly, and then releases AppDomain
// Intended to be used by C2 frameworks to quickly execute an assembly one time
func ExecuteByteArrayDefaultDomain(runtimeHost *ICORRuntimeHost, rawBytes []byte, params []string) (stdout string, stderr string) {
	appDomain, err := GetAppDomain(runtimeHost)
	if err != nil {
		stderr = err.Error()
		return
	}
	safeArrayPtr, err := CreateSafeArray(rawBytes)
	if err != nil {
		stderr = err.Error()
		return
	}

	assembly, err := appDomain.Load_3(safeArrayPtr)
	if err != nil {
		stderr = err.Error()
		return
	}

	methodInfo, err := assembly.GetEntryPoint()
	if err != nil {
		stderr = err.Error()
		return
	}

	var paramSafeArray *SafeArray
	methodSignature, err := methodInfo.GetString()
	if err != nil {
		stderr = err.Error()
		return
	}

	if expectsParams(methodSignature) {
		if paramSafeArray, err = PrepareParameters(params); err != nil {
			stderr = err.Error()
			return
		}
	}

	nullVariant := Variant{
		VT:  1,
		Val: uintptr(0),
	}

	err = methodInfo.Invoke_3(nullVariant, paramSafeArray)
	if err != nil {
		stderr = err.Error()
		return
	}

	assembly.Release()
	appDomain.Release()
	return
}

// LoadAssembly uses a previously instantiated runtimehost and loads an assembly into the default AppDomain
// and returns the assembly's methodInfo structure. The intended purpose is for the assembly to be loaded
// once but executed many times throughout the duration of the program. Commonly used with C2 frameworks
func LoadAssembly(runtimeHost *ICORRuntimeHost, rawBytes []byte) (methodInfo *MethodInfo, err error) {
	appDomain, err := GetAppDomain(runtimeHost)
	if err != nil {
		return
	}
	safeArrayPtr, err := CreateSafeArray(rawBytes)
	if err != nil {
		return
	}

	assembly, err := appDomain.Load_3(safeArrayPtr)
	if err != nil {
		return
	}
	return assembly.GetEntryPoint()
}

// InvokeAssembly uses the MethodInfo structure of a previously loaded assembly and executes it.
// The intended purpose is for the assembly to be executed many times throughout the duration of the
// program. Commonly used with C2 frameworks
func InvokeAssembly(methodInfo *MethodInfo, params []string) (stdout string, stderr string) {
	var paramSafeArray *SafeArray
	methodSignature, err := methodInfo.GetString()
	if err != nil {
		stderr = err.Error()
		return
	}

	if expectsParams(methodSignature) {
		if paramSafeArray, err = PrepareParameters(params); err != nil {
			stderr = err.Error()
			return
		}
	}

	nullVariant := Variant{
		VT:  1,
		Val: uintptr(0),
	}

	defer SafeArrayDestroy(paramSafeArray)

	// Ensure exclusive access to read/write STDOUT/STDERR
	mutex.Lock()
	defer mutex.Unlock()

	err = methodInfo.Invoke_3(nullVariant, paramSafeArray)
	if err != nil {
		stderr = err.Error()
		// Don't return because there could be data on STDOUT/STDERR
	}

	// Read data from previously redirected STDOUT/STDERR
	if wSTDOUT != nil {
		var e string
		stdout, e, err = ReadStdoutStderr()
		stderr += e
		if err != nil {
			stderr += err.Error()
		}
	}

	return
}
