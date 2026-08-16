//go:build !windows && !darwin && !linux

package memmod

import "errors"

type Module struct{}

func LoadLibrary(data []byte) (*Module, error) {
	_ = data
	return nil, errors.New("memmod is only supported on windows, darwin, and linux")
}

func (module *Module) Free() {}

func (module *Module) CallExport(name string) error {
	_ = name
	return errors.New("memmod is only supported on windows, darwin, and linux")
}

func (module *Module) ProcAddressByName(name string) (uintptr, error) {
	_ = name
	return 0, errors.New("memmod is only supported on windows, darwin, and linux")
}

func (module *Module) ProcAddressByOrdinal(ordinal uint16) (uintptr, error) {
	_ = ordinal
	return 0, errors.New("memmod is only supported on windows, darwin, and linux")
}
