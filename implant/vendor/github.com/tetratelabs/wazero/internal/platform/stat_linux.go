//go:build (amd64 || arm64 || riscv64) && linux

// Note: This expression is not the same as compiler support, even if it looks
// similar. Platform functions here are used in interpreter mode as well.

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

func statTimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	d := t.Sys().(*syscall.Stat_t)
	atime := d.Atim
	mtime := d.Mtim
	ctime := d.Ctim
	return atime.Sec*1e9 + atime.Nsec, mtime.Sec*1e9 + mtime.Nsec, ctime.Sec*1e9 + ctime.Nsec
}

func stat(_ fs.File, t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64, nlink, dev, inode uint64, err error) {
	d := t.Sys().(*syscall.Stat_t)
	atime := d.Atim
	mtime := d.Mtim
	ctime := d.Ctim
	return atime.Sec*1e9 + atime.Nsec, mtime.Sec*1e9 + mtime.Nsec, ctime.Sec*1e9 + ctime.Nsec, uint64(d.Nlink), uint64(d.Dev), uint64(d.Ino), nil
}
