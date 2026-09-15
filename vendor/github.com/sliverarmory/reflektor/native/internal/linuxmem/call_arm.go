//go:build linux && !android && arm && arm.7

// SPDX-License-Identifier: MIT
package linuxmem

import "github.com/ebitengine/purego"

//go:uintptrescapes
func callExportFunction(fn uintptr, args ...uintptr) uintptr {
	if len(args) > maxExportArguments {
		panic("validated Linux export argument count is out of range")
	}
	result, _, _ := purego.SyscallN(fn, args...)
	return result
}
