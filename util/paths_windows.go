//go:build windows

package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ResolvePath - Resolve a path from an assumed root path, whilst dealing with
// Windows' stupid file system paths.
func ResolvePath(in string) string {
	if strings.Contains(in, ":\\") {
		parts := strings.Split(in, ":\\")
		in = parts[len(parts)-1]
	}
	out := filepath.Clean(fmt.Sprintf("x:\\%s", in))
	return strings.TrimPrefix(out, "x:")
}
