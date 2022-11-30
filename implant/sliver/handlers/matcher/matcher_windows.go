package matcher

import (
	"path/filepath"
	"strings"
)

// Match - Windows specific implementation: disregard the case
func Match(pattern string, s string) (bool, error) {
	return filepath.Match(strings.ToLower(pattern), strings.ToLower(s))
}
