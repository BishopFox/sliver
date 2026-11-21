//go:build windows

package matcher

import (
	"path/filepath"
	"strings"
)

// Match applies Windows case-insensitive file matching rules.
func Match(pattern, path string) (bool, error) {
	return filepath.Match(strings.ToLower(pattern), strings.ToLower(path))
}
