//go:build !(amd64 || arm64) || !(darwin || linux || freebsd || windows)

package platform

import "os"

func statTimes(t os.FileInfo) (atimeNsec, mtimeNsec, ctimeNsec int64) {
	return mtimes(t)
}
