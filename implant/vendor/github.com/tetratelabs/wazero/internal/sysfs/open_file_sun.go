//go:build illumos || solaris

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

func openFile(path string, flag int, perm fs.FileMode) (*os.File, syscall.Errno) {
	f, err := os.OpenFile(path, flag, perm)
	return f, platform.UnwrapOSError(err)
}
