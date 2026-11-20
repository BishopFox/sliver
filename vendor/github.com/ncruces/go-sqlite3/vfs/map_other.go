//go:build !(linux || darwin || freebsd)

package vfs

import "os"

// NewMemoryMapper returns a memory mapper for the given file.
// It will return nil if file mapping is not supported,
// or not appropriate for the given flags.
// Only databases use memory mapping.
func NewMemoryMapper(file *os.File, flags OpenFlag) MemoryMapper {
	return nil
}
