//go:build !windows && !js && !illumos && !solaris

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

// OpenFile is like os.OpenFile except it returns syscall.Errno. A zero
// syscall.Errno is success.
func OpenFile(path string, flag int, perm fs.FileMode) (File, syscall.Errno) {
	f, err := os.OpenFile(path, flag, perm)
	return f, UnwrapOSError(err)
}
