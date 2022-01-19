//go:build !windows

package util

import "path/filepath"

// ResolvePath - Resolve a path from an assumed root path
func ResolvePath(in string) string {
	return filepath.Clean("/" + in)
}
