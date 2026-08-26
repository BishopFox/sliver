//go:build (darwin && !ios && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64))

package bofloader

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

var unixLibraryHandles sync.Map

func resolveSymbol(symbol string) (uintptr, error) {
	if address, ok, err := resolveBeaconCallback(symbol); err != nil {
		return 0, err
	} else if ok {
		return address, nil
	}

	name := normalizeImportedSymbol(symbol)
	if separator := strings.IndexByte(name, '$'); separator > 0 && separator < len(name)-1 {
		library := name[:separator]
		name = trimStdcallSuffix(name[separator+1:])
		var resolutionErrors []error
		for _, candidate := range unixQualifiedLibraryCandidates(library) {
			handle, err := openUnixSystemLibrary(candidate)
			if err != nil {
				resolutionErrors = append(resolutionErrors, err)
				continue
			}
			address, err := purego.Dlsym(handle, name)
			if err == nil && address != 0 {
				return address, nil
			}
			if err != nil {
				resolutionErrors = append(resolutionErrors, err)
			}
		}
		return 0, fmt.Errorf("system symbol %q was not found in library %q: %w", name, library, errors.Join(resolutionErrors...))
	}

	candidates := []string{name}
	if strings.HasPrefix(name, "_") {
		candidates = append(candidates, name[1:])
	}
	var resolutionErrors []error
	for _, candidate := range candidates {
		address, err := purego.Dlsym(purego.RTLD_DEFAULT, candidate)
		if err == nil && address != 0 {
			return address, nil
		}
		if err != nil {
			resolutionErrors = append(resolutionErrors, err)
		}
	}
	return 0, fmt.Errorf("system symbol %q was not found: %w", name, errors.Join(resolutionErrors...))
}

func unixQualifiedLibraryCandidates(name string) []string {
	candidates := []string{name}
	// Mach-O adds one leading underscore to C symbols, including Reflektor's
	// library$symbol spelling. Preserve an intentionally underscore-prefixed
	// library as the first lookup and try the Mach-O spelling as a fallback.
	if strings.HasPrefix(name, "_") && len(name) > 1 {
		candidates = append(candidates, name[1:])
	}
	return candidates
}

func openUnixSystemLibrary(name string) (uintptr, error) {
	if strings.ContainsAny(name, `/\\:`) || name == "." || name == ".." {
		return 0, fmt.Errorf("invalid system library name %q", name)
	}
	if existing, ok := unixLibraryHandles.Load(name); ok {
		return existing.(uintptr), nil
	}
	candidates := unixSystemLibraryCandidates(runtime.GOOS, name)
	var openErrors []error
	for _, candidate := range candidates {
		handle, err := purego.Dlopen(candidate, purego.RTLD_NOW|purego.RTLD_LOCAL)
		if err == nil && handle != 0 {
			actual, loaded := unixLibraryHandles.LoadOrStore(name, handle)
			if loaded {
				_ = purego.Dlclose(handle)
			}
			return actual.(uintptr), nil
		}
		if err != nil {
			openErrors = append(openErrors, err)
		}
	}
	return 0, fmt.Errorf("load system library %q: %w", name, errors.Join(openErrors...))
}

func unixSystemLibraryCandidates(goos, name string) []string {
	candidates := []string{name}
	switch goos {
	case "freebsd":
		if strings.Contains(name, ".so") {
			return candidates
		}
		stem := name
		if !strings.HasPrefix(stem, "lib") {
			stem = "lib" + stem
		}
		switch stem {
		case "libc":
			return append(candidates, "libc.so.7", "libc.so")
		case "libm":
			return append(candidates, "libm.so.5", "libm.so")
		case "libpthread", "libthr":
			return append(candidates, "libthr.so.3", "libthr.so")
		default:
			return append(candidates, stem+".so")
		}
	case "linux":
		if strings.Contains(name, ".so") {
			return candidates
		}
		stem := name
		if !strings.HasPrefix(stem, "lib") {
			stem = "lib" + stem
		}
		return append(candidates, stem+".so.6", stem+".so")
	case "darwin":
		if strings.HasSuffix(name, ".dylib") {
			return candidates
		}
		if name == "c" || name == "libc" || name == "System" || name == "libSystem" {
			return append(candidates, "/usr/lib/libSystem.B.dylib")
		}
		stem := name
		if !strings.HasPrefix(stem, "lib") {
			stem = "lib" + stem
		}
		return append(candidates, "/usr/lib/"+stem+".dylib")
	default:
		return candidates
	}
}
