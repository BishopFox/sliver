//go:build windows && (386 || amd64 || arm64)

package bofloader

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

var windowsSystemLibraries sync.Map

func resolveSymbol(symbol string) (uintptr, error) {
	if address, ok, err := resolveBeaconCallback(symbol); err != nil {
		return 0, err
	} else if ok {
		return address, nil
	}

	name := normalizeWindowsExternal(symbol)
	separator := strings.IndexByte(name, '$')
	if separator <= 0 || separator == len(name)-1 {
		return resolveUnqualifiedWindowsSymbol(name)
	}
	libraryName := name[:separator]
	functionName := trimStdcallSuffix(name[separator+1:])
	handle, err := loadWindowsSystemLibrary(libraryName)
	if err != nil {
		return 0, err
	}
	for _, candidate := range windowsFunctionCandidates(functionName) {
		address, lookupErr := windows.GetProcAddress(handle, candidate)
		if lookupErr == nil && address != 0 {
			return address, nil
		}
	}
	compatibilityAddress, compatibilityHandled, compatibilityErr := resolveWindowsCRTCompatibility(libraryName, functionName)
	if compatibilityHandled && compatibilityErr == nil {
		return compatibilityAddress, nil
	}
	// A few widely distributed BOF headers contain incorrect DLL qualifiers
	// (for example KERNEL32$OpenProcessToken). Keep fallback resolution inside
	// trusted System32 modules so those objects remain loadable without allowing
	// an arbitrary DLL search path.
	address, fallbackErr := resolveUnqualifiedWindowsSymbol(functionName)
	if fallbackErr == nil {
		return address, nil
	}
	if compatibilityErr != nil {
		fallbackErr = errors.Join(compatibilityErr, fallbackErr)
	}
	return 0, fmt.Errorf("resolve %s!%s: %w", libraryName, functionName, fallbackErr)
}

func normalizeWindowsExternal(symbol string) string {
	name := strings.TrimSpace(symbol)
	if strings.HasPrefix(name, "__imp_") {
		name = strings.TrimPrefix(name, "__imp_")
		if strings.HasPrefix(name, "_") {
			name = name[1:]
		}
	} else if strings.HasPrefix(name, "_") && strings.Contains(name, "$") {
		name = name[1:]
	}
	return trimStdcallSuffix(name)
}

func loadWindowsSystemLibrary(name string) (windows.Handle, error) {
	if strings.ContainsAny(name, `/\\:`) || name == "." || name == ".." {
		return 0, fmt.Errorf("invalid system library name %q", name)
	}
	if !strings.HasSuffix(strings.ToLower(name), ".dll") {
		name += ".dll"
	}
	key := strings.ToLower(name)
	if existing, ok := windowsSystemLibraries.Load(key); ok {
		return existing.(windows.Handle), nil
	}
	handle, err := windows.LoadLibraryEx(name, 0, windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
	if err != nil {
		return 0, fmt.Errorf("load system DLL %q: %w", name, err)
	}
	actual, loaded := windowsSystemLibraries.LoadOrStore(key, handle)
	if loaded {
		_ = windows.FreeLibrary(handle)
	}
	return actual.(windows.Handle), nil
}

func resolveUnqualifiedWindowsSymbol(name string) (uintptr, error) {
	if strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "__") {
		name = name[1:]
	}
	libraries := []string{
		"kernel32.dll", "kernelbase.dll", "ntdll.dll", "advapi32.dll",
		"bcrypt.dll", "crypt32.dll", "dnsapi.dll", "imagehlp.dll",
		"iphlpapi.dll", "mpr.dll", "msvcrt.dll", "ucrtbase.dll",
		"netapi32.dll", "ole32.dll", "oleaut32.dll", "psapi.dll",
		"rpcrt4.dll", "secur32.dll", "shlwapi.dll", "user32.dll",
		"version.dll", "wldap32.dll", "ws2_32.dll", "wtsapi32.dll",
	}
	var resolutionErrors []error
	for _, library := range libraries {
		handle, err := loadWindowsSystemLibrary(library)
		if err != nil {
			resolutionErrors = append(resolutionErrors, err)
			continue
		}
		for _, candidate := range windowsFunctionCandidates(name) {
			address, lookupErr := windows.GetProcAddress(handle, candidate)
			if lookupErr == nil && address != 0 {
				return address, nil
			}
			if lookupErr != nil {
				resolutionErrors = append(resolutionErrors, lookupErr)
			}
		}
	}
	return 0, fmt.Errorf("unqualified system symbol %q was not found: %w", name, errors.Join(resolutionErrors...))
}

func windowsFunctionCandidates(name string) []string {
	candidates := []string{name}
	switch name {
	case "GetEnvironmentStrings":
		candidates = append(candidates, "GetEnvironmentStringsA")
	}
	return candidates
}
