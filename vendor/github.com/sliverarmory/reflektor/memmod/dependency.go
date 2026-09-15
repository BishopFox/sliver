package memmod

import "errors"

// ErrDependencyNotFound is returned by a DependencyReader when an imported
// library cannot be found in any of its configured search locations.
var ErrDependencyNotFound = errors.New("dependency library not found")

// DependencyRequest describes one shared-library import discovered while a
// recursive load graph is being built.
type DependencyRequest struct {
	Name         string
	ImporterPath string
	SearchPaths  []string
}

// Dependency contains the bytes and canonical path of a resolved import.
type Dependency struct {
	Data []byte
	Path string
}

// DependencyReader resolves and reads an imported application library.
// Implementations must return the path whose bytes they read so nested relative
// imports can be resolved from the dependency's own directory. Platform
// backends may bypass the reader for known native runtime libraries. Returning
// an error that wraps ErrDependencyNotFound allows a backend to use its native
// system-library fallback; other errors are terminal.
type DependencyReader func(DependencyRequest) (Dependency, error)
