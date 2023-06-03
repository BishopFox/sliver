//go:build (amd64 || arm64 || riscv64) && linux

// Note: This expression is not the same as compiler support, even if it looks
// similar. Platform functions here are used in interpreter mode as well.

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

func lstat(path string) (Stat_t, syscall.Errno) {
	if t, err := os.Lstat(path); err != nil {
		return Stat_t{}, UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func stat(path string) (Stat_t, syscall.Errno) {
	if t, err := os.Stat(path); err != nil {
		return Stat_t{}, UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func statFile(f fs.File) (Stat_t, syscall.Errno) {
	return defaultStatFile(f)
}

func inoFromFileInfo(_ readdirFile, t fs.FileInfo) (ino uint64, err syscall.Errno) {
	if d, ok := t.Sys().(*syscall.Stat_t); ok {
		ino = d.Ino
	}
	return
}

func statFromFileInfo(t fs.FileInfo) Stat_t {
	if d, ok := t.Sys().(*syscall.Stat_t); ok {
		st := Stat_t{}
		st.Dev = uint64(d.Dev)
		st.Ino = uint64(d.Ino)
		st.Uid = d.Uid
		st.Gid = d.Gid
		st.Mode = t.Mode()
		st.Nlink = uint64(d.Nlink)
		st.Size = d.Size
		atime := d.Atim
		st.Atim = atime.Sec*1e9 + atime.Nsec
		mtime := d.Mtim
		st.Mtim = mtime.Sec*1e9 + mtime.Nsec
		ctime := d.Ctim
		st.Ctim = ctime.Sec*1e9 + ctime.Nsec
		return st
	}
	return statFromDefaultFileInfo(t)
}
