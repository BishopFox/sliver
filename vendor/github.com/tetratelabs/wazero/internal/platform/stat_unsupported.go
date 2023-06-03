//go:build (!((amd64 || arm64 || riscv64) && linux) && !((amd64 || arm64) && (darwin || freebsd)) && !((amd64 || arm64) && windows)) || js

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

func lstat(path string) (Stat_t, syscall.Errno) {
	t, err := os.Lstat(path)
	if errno := UnwrapOSError(err); errno == 0 {
		return statFromFileInfo(t), 0
	} else {
		return Stat_t{}, errno
	}
}

func stat(path string) (Stat_t, syscall.Errno) {
	t, err := os.Stat(path)
	if errno := UnwrapOSError(err); errno == 0 {
		return statFromFileInfo(t), 0
	} else {
		return Stat_t{}, errno
	}
}

func statFile(f fs.File) (Stat_t, syscall.Errno) {
	return defaultStatFile(f)
}

func inoFromFileInfo(_ readdirFile, t fs.FileInfo) (ino uint64, err syscall.Errno) {
	return
}

func statFromFileInfo(t fs.FileInfo) Stat_t {
	return statFromDefaultFileInfo(t)
}
