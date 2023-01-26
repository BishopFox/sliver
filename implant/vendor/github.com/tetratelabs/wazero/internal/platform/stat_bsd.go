//go:build (amd64 || arm64) && (darwin || freebsd)

package platform

import (
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
