package util

// TODO rename package update/optimize functions

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// FindReverse traverses the filetree upwards to find given file/directory.
func FindReverse(path string, name string) (target string, err error) {
	var absPath string
	if absPath, err = filepath.Abs(path); err == nil {
		target = absPath + "/" + name
		if _, err = os.Stat(target); err != nil {
			parent := filepath.Dir(absPath)
			if parent != path {
				return FindReverse(parent, name)
			} else {
				err = errors.New("could not find: " + name)
			}
		}
	}
	return
}

// HasPathPrefix checks if given string has a path prefix.
func HasPathPrefix(s string) bool {
	return strings.HasPrefix(s, ".") ||
		strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "~") ||
		HasVolumePrefix(s)
}

// HasVolumePrefix checks if given path has a volume prefix (only for GOOS=windows).
func HasVolumePrefix(s string) bool {
	switch {
	case runtime.GOOS != "windows":
		return false
	case len(s) < 2:
		return false
	case unicode.IsLetter(rune(s[0])) && s[1] == ':':
		return true
	default:
		return false
	}
}
