//go:build (linux && !android && (amd64 || arm64 || ppc64le || riscv64)) || (freebsd && (amd64 || arm64))

// SPDX-License-Identifier: MIT
package linuxmem

import "github.com/ebitengine/purego"

//go:uintptrescapes
func callExportFunction(fn uintptr, args ...uintptr) uintptr {
	if len(args) > maxExportArguments {
		panic("validated ELF export argument count is out of range")
	}
	result, _, _ := purego.SyscallN(fn, args...)
	return result
}
