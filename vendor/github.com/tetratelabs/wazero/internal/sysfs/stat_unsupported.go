//go:build (!((amd64 || arm64 || riscv64) && linux) && !((amd64 || arm64) && (darwin || freebsd)) && !((amd64 || arm64) && windows)) || js

package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	t, err := os.Lstat(path)
	if errno := platform.UnwrapOSError(err); errno == 0 {
		return statFromFileInfo(t), 0
	} else {
		return fsapi.Stat_t{}, errno
	}
}

func stat(path string) (fsapi.Stat_t, syscall.Errno) {
	t, err := os.Stat(path)
	if errno := platform.UnwrapOSError(err); errno == 0 {
		return statFromFileInfo(t), 0
	} else {
		return fsapi.Stat_t{}, errno
	}
}

func statFile(f *os.File) (fsapi.Stat_t, syscall.Errno) {
	return defaultStatFile(f)
}

func inoFromFileInfo(_ string, t fs.FileInfo) (ino uint64, err syscall.Errno) {
	return
}

func statFromFileInfo(t fs.FileInfo) fsapi.Stat_t {
	return StatFromDefaultFileInfo(t)
}
