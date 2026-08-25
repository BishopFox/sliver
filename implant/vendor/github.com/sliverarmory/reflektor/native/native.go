// Package native loads native C and Rust shared-library images without linking
// Reflektor's Go c-shared runtime support.
package native

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/sliverarmory/reflektor/native/internal/rejection"
)

// ErrLibraryClosed reports an operation on a closed Library.
var ErrLibraryClosed = errors.New("reflektor/native: library is closed")

// ErrGoSharedLibraryUnsupported reports that LoadLibrary was given a Go
// c-shared image. Go c-shared images require the root reflektor package, whose
// runtime-aware loader intentionally has a larger process footprint.
var ErrGoSharedLibraryUnsupported = rejection.ErrGoSharedLibraryUnsupported

// MaxExportArguments is the maximum number of machine-word arguments accepted
// by CallExportWithArgs on every supported platform.
const MaxExportArguments = 3

type module interface {
	CallExport(name string) error
	CallExportWithArgs(name string, args ...uintptr) (uintptr, error)
	Free()
}

// Library is an in-memory native shared library.
type Library struct {
	mu     sync.RWMutex
	module module
	closed bool
}

// LoadLibrary loads a native C or Rust shared-library image from memory.
// Images containing a Go runtime are rejected before platform loading begins.
func LoadLibrary(data []byte) (*Library, error) {
	if len(data) == 0 {
		return nil, errors.New("reflektor/native: empty library image")
	}
	if imageHasGoBuildInfo(data) {
		return nil, ErrGoSharedLibraryUnsupported
	}

	loaded, err := loadPlatformLibrary(data)
	if err != nil {
		return nil, fmt.Errorf("reflektor/native: load library: %w", err)
	}
	return &Library{module: loaded}, nil
}

// CallExport resolves and calls a zero-argument exported function.
func (library *Library) CallExport(name string) error {
	library.mu.RLock()
	defer library.mu.RUnlock()

	if library.closed || library.module == nil {
		return ErrLibraryClosed
	}
	if err := library.module.CallExport(name); err != nil {
		return fmt.Errorf("reflektor/native: call export %q: %w", name, err)
	}
	return nil
}

// CallExportWithArgs resolves an export, calls it with up to three machine-word
// arguments, and returns the value from the platform's primary return register.
//
//go:uintptrescapes
func (library *Library) CallExportWithArgs(name string, args ...uintptr) (uintptr, error) {
	library.mu.RLock()
	defer library.mu.RUnlock()

	if library.closed || library.module == nil {
		return 0, ErrLibraryClosed
	}
	if len(args) > MaxExportArguments {
		return 0, fmt.Errorf("reflektor/native: export call has %d arguments; maximum is %d", len(args), MaxExportArguments)
	}
	result, err := library.module.CallExportWithArgs(name, args...)
	runtime.KeepAlive(args)
	if err != nil {
		return 0, fmt.Errorf("reflektor/native: call export %q with arguments: %w", name, err)
	}
	return result, nil
}

// Close releases library resources. It is safe to call Close more than once.
func (library *Library) Close() error {
	library.mu.Lock()
	if library.closed {
		library.mu.Unlock()
		return nil
	}
	library.closed = true
	loaded := library.module
	library.module = nil
	library.mu.Unlock()

	// Destructors are arbitrary native code and may re-enter Library methods.
	// Mark the handle closed before running them, and do not retain the lock.
	if loaded != nil {
		loaded.Free()
	}
	return nil
}
