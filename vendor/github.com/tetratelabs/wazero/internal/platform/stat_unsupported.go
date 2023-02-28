//go:build !((amd64 || arm64 || riscv64) && linux) && !((amd64 || arm64) && (darwin || freebsd)) && !((amd64 || arm64) && windows)

package platform

import (
	"io/fs"
	"os"
)

func statTimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	atimeNsec, mtimeNsec, ctimeNsec = mtimes(t)
	return
}

func stat(_ fs.File, t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64, nlink, dev, inode uint64, err error) {
	atimeNsec, mtimeNsec, ctimeNsec = mtimes(t)
	return
}
