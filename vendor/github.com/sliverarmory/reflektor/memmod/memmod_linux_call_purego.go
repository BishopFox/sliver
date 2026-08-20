//go:build linux && !cgo && (amd64 || arm64)

package memmod

import "github.com/ebitengine/purego"

//go:uintptrescapes
func callExportFunction(fn uintptr, args ...uintptr) uintptr {
	if len(args) > MaxExportArguments {
		panic("validated Linux export argument count is out of range")
	}
	result, _, _ := purego.SyscallN(fn, args...)
	return result
}
