//go:build !windows && !js && !illumos && !solaris

package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func newOsFile(openPath string, openFlag int, openPerm fs.FileMode, f *os.File) fsapi.File {
	return newDefaultOsFile(openPath, openFlag, openPerm, f)
}

// OpenFile is like os.OpenFile except it returns syscall.Errno. A zero
// syscall.Errno is success.
func openFile(path string, flag int, perm fs.FileMode) (*os.File, syscall.Errno) {
	f, err := os.OpenFile(path, flag, perm)
	// Note: This does not return a fsapi.File because fsapi.FS that returns
	// one may want to hide the real OS path. For example, this is needed for
	// pre-opens.
	return f, platform.UnwrapOSError(err)
}
