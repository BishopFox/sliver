//go:build android || ios || (darwin && !ios && !(amd64 || arm64)) || (linux && !android && !(386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) || (freebsd && !(amd64 || arm64)) || (!windows && !darwin && !linux && !freebsd)

package memmod

import (
	"fmt"
	"runtime"
)

type Module struct{}

func LoadLibrary(data []byte) (*Module, error) {
	_ = data
	return nil, unsupportedPlatformError()
}

func LoadLibraryRecursive(data []byte, origin string, reader DependencyReader) (*Module, error) {
	_, _, _ = data, origin, reader
	return nil, unsupportedPlatformError()
}

func (module *Module) Free() {}

func (module *Module) CallExport(name string) error {
	_ = name
	return unsupportedPlatformError()
}

// CallExportWithArgs reports that argument-bearing export calls are unsupported
// on this platform.
//
//go:uintptrescapes
func (module *Module) CallExportWithArgs(name string, args ...uintptr) (uintptr, error) {
	_, _ = name, args
	return 0, unsupportedPlatformError()
}

func (module *Module) ProcAddressByName(name string) (uintptr, error) {
	_ = name
	return 0, unsupportedPlatformError()
}

func (module *Module) ProcAddressByOrdinal(ordinal uint16) (uintptr, error) {
	_ = ordinal
	return 0, unsupportedPlatformError()
}

func unsupportedPlatformError() error {
	return fmt.Errorf("memmod shared-library loader is unsupported on %s/%s", runtime.GOOS, runtime.GOARCH)
}
