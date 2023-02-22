//go:build (amd64 || arm64) && (darwin || freebsd)

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

func statTimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	d := t.Sys().(*syscall.Stat_t)
	atime := d.Atimespec
	mtime := d.Mtimespec
	ctime := d.Ctimespec
	return atime.Sec*1e9 + atime.Nsec, mtime.Sec*1e9 + mtime.Nsec, ctime.Sec*1e9 + ctime.Nsec
}

func stat(_ fs.File, t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64, nlink, dev, inode uint64, err error) {
	d := t.Sys().(*syscall.Stat_t)
	atime := d.Atimespec
	mtime := d.Mtimespec
	ctime := d.Ctimespec
	return atime.Sec*1e9 + atime.Nsec, mtime.Sec*1e9 + mtime.Nsec, ctime.Sec*1e9 + ctime.Nsec,
		uint64(d.Nlink), uint64(d.Dev), uint64(d.Ino), nil
}
