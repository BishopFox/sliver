//go:build windows

package memmod

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// CallExport resolves and calls an exported zero-argument function.
func (module *Module) CallExport(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("export name cannot be empty")
	}

	candidates := []string{name}
	if strings.HasPrefix(name, "_") {
		candidates = append(candidates, strings.TrimPrefix(name, "_"))
	} else {
		candidates = append(candidates, "_"+name)
	}

	var (
		addr uintptr
		err  error
	)
	for _, candidate := range candidates {
		addr, err = module.ProcAddressByName(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("resolve export %q: %w", name, err)
	}

	_, _, _ = syscall.SyscallN(addr)
	return nil
}
