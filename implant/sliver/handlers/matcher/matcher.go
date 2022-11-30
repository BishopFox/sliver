//go:build !windows

package matcher

import "path/filepath"

// PathMatch - Match a path against a pattern, generic implementation
func Match(pattern string, s string) (bool, error) {
	return filepath.Match(pattern, s)
}
