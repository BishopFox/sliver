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
	resolved, err := module.resolveExportAddress(name)
	if err != nil {
		return err
	}

	_ = invokeWindowsExport(resolved.address)
	return nil
}

// CallExportWithArgs resolves and calls an exported function with up to
// MaxExportArguments machine-word arguments and returns its machine-word result.
//
//go:uintptrescapes
func (module *Module) CallExportWithArgs(name string, args ...uintptr) (uintptr, error) {
	if module.goRuntime {
		return 0, ErrGoExportArgumentsUnsupported
	}
	if err := validateExportArguments(args); err != nil {
		return 0, err
	}

	resolved, err := module.resolveExportAddress(name)
	if err != nil {
		return 0, err
	}
	if resolved.owner != nil && resolved.owner.goRuntime {
		return 0, ErrGoExportArgumentsUnsupported
	}
	return invokeWindowsExport(resolved.address, args...), nil
}

func (module *Module) resolveExportAddress(name string) (resolvedWindowsExport, error) {
	name, candidates, err := windowsExportCandidates(name)
	if err != nil {
		return resolvedWindowsExport{}, err
	}

	var resolved resolvedWindowsExport
	for _, candidate := range candidates {
		if module.recursive != nil {
			resolved, err = module.recursiveProcAddressByName(candidate, make(map[string]struct{}))
		} else {
			resolved.address, err = module.ProcAddressByName(candidate)
			resolved.owner = module
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		return resolvedWindowsExport{}, fmt.Errorf("resolve export %q: %w", name, err)
	}
	return resolved, nil
}

func windowsExportCandidates(name string) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, errors.New("export name cannot be empty")
	}

	candidates := []string{name}
	if strings.HasPrefix(name, "_") {
		candidates = append(candidates, strings.TrimPrefix(name, "_"))
	} else {
		candidates = append(candidates, "_"+name)
	}
	return name, candidates, nil
}

func invokeWindowsExport(addr uintptr, args ...uintptr) uintptr {
	// SyscallN uses Go's Windows ABI trampoline. Its 386 implementation restores
	// SP after the call, so both ordinary cdecl exports and stdcall exports are
	// safe; amd64 and arm64 use their platform's unified register/stack ABI.
	result, _, _ := syscall.SyscallN(addr, args...)
	return result
}
