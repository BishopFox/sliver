//go:build !windows

package matcher

import "path/filepath"

// Match applies the host's file matching rules to an upload path.
func Match(pattern, path string) (bool, error) {
	return filepath.Match(pattern, path)
}
