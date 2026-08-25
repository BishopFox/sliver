//go:build windows && arm64

package bofloader

import (
	"fmt"
	"strings"
	"sync"
	"syscall"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

const windowsCRTStandardSNPrintfBehavior = uintptr(0x2)

var windowsARM64Vsnprintf struct {
	sync.Once
	target   uintptr
	callback uintptr
	err      error
}

func resolveWindowsCRTCompatibility(libraryName, functionName string) (uintptr, bool, error) {
	if !windowsMSVCRTLibrary(libraryName) || functionName != "vsnprintf" {
		return 0, false, nil
	}
	windowsARM64Vsnprintf.Do(initializeWindowsARM64Vsnprintf)
	return windowsARM64Vsnprintf.callback, true, windowsARM64Vsnprintf.err
}

func windowsMSVCRTLibrary(name string) bool {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(strings.ToLower(name), ".dll") {
		name = name[:len(name)-len(".dll")]
	}
	return strings.EqualFold(name, "msvcrt")
}

func initializeWindowsARM64Vsnprintf() {
	defer func() {
		if recovered := recover(); recovered != nil {
			windowsARM64Vsnprintf.err = fmt.Errorf("register UCRT vsnprintf callback: %v", recovered)
		}
	}()

	handle, err := loadWindowsSystemLibrary("ucrtbase.dll")
	if err != nil {
		windowsARM64Vsnprintf.err = err
		return
	}
	target, err := windows.GetProcAddress(handle, "__stdio_common_vsprintf")
	if err != nil {
		windowsARM64Vsnprintf.err = fmt.Errorf("resolve ucrtbase!__stdio_common_vsprintf: %w", err)
		return
	}
	if target == 0 {
		windowsARM64Vsnprintf.err = fmt.Errorf("resolve ucrtbase!__stdio_common_vsprintf: address is zero")
		return
	}
	windowsARM64Vsnprintf.target = target
	windowsARM64Vsnprintf.callback = purego.NewCallback(callWindowsARM64Vsnprintf)
	if windowsARM64Vsnprintf.callback == 0 {
		windowsARM64Vsnprintf.err = fmt.Errorf("register UCRT vsnprintf callback: address is zero")
	}
}

// callWindowsARM64Vsnprintf preserves C99 vsnprintf behavior. In particular,
// buffer == NULL and count == 0 must return the required output length. The
// legacy _vsnprintf export has different truncation and return semantics, so
// it is not an ABI-compatible fallback for BOFs importing MSVCRT$vsnprintf.
func callWindowsARM64Vsnprintf(_ purego.CDecl, buffer, count, format, argumentList uintptr) uintptr {
	result, _, _ := syscall.SyscallN(
		windowsARM64Vsnprintf.target,
		windowsCRTStandardSNPrintfBehavior,
		buffer,
		count,
		format,
		0, // Use the current thread locale.
		argumentList,
	)
	return normalizeWindowsVsnprintfResult(result)
}

func normalizeWindowsVsnprintfResult(result uintptr) uintptr {
	// The public UCRT vsnprintf wrapper converts every negative internal result
	// to C -1. BOFs rely on that exact sentinel rather than arbitrary negative
	// __stdio_common_vsprintf processor results.
	value := int32(result)
	if value < 0 {
		return uintptr(^uint32(0))
	}
	return uintptr(uint32(value))
}
