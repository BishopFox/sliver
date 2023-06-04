//go:build (amd64 || arm64) && (darwin || freebsd)

package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	if t, err := os.Lstat(path); err != nil {
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func stat(path string) (fsapi.Stat_t, syscall.Errno) {
	if t, err := os.Stat(path); err != nil {
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func statFile(f *os.File) (fsapi.Stat_t, syscall.Errno) {
	return defaultStatFile(f)
}

func inoFromFileInfo(_ string, t fs.FileInfo) (ino uint64, err syscall.Errno) {
	if d, ok := t.Sys().(*syscall.Stat_t); ok {
		ino = d.Ino
	}
	return
}

func statFromFileInfo(t fs.FileInfo) fsapi.Stat_t {
	if d, ok := t.Sys().(*syscall.Stat_t); ok {
		st := fsapi.Stat_t{}
		st.Dev = uint64(d.Dev)
		st.Ino = d.Ino
		st.Uid = d.Uid
		st.Gid = d.Gid
		st.Mode = t.Mode()
		st.Nlink = uint64(d.Nlink)
		st.Size = d.Size
		atime := d.Atimespec
		st.Atim = atime.Sec*1e9 + atime.Nsec
		mtime := d.Mtimespec
		st.Mtim = mtime.Sec*1e9 + mtime.Nsec
		ctime := d.Ctimespec
		st.Ctim = ctime.Sec*1e9 + ctime.Nsec
		return st
	}
	return StatFromDefaultFileInfo(t)
}
