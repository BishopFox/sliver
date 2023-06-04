package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func newOsFile(openPath string, openFlag int, openPerm fs.FileMode, f *os.File) File {
	return newDefaultOsFile(openPath, openFlag, openPerm, f)
}

func openFile(path string, flag int, perm fs.FileMode) (*os.File, syscall.Errno) {
	flag &= ^(O_DIRECTORY | O_NOFOLLOW) // erase placeholders
	f, err := os.OpenFile(path, flag, perm)
	return f, platform.UnwrapOSError(err)
}
