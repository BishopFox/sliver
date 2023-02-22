//go:build !windows && !js

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

// Simple aliases to constants in the syscall package for portability with
// platforms which do not have them (e.g. windows)
const (
	O_DIRECTORY = syscall.O_DIRECTORY
	O_NOFOLLOW  = syscall.O_NOFOLLOW
)

func OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}
