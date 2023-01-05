//go:build (amd64 || arm64) && linux

package platform

import (
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
