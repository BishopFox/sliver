//go:build !unix || sqlite3_nosys

package vfs

import (
	"io/fs"
	"os"
)

const (
	_S_IREAD  = 0400
	_S_IWRITE = 0200
	_S_IEXEC  = 0100
)

func osAccess(path string, flags AccessFlag) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if flags == ACCESS_EXISTS {
		return nil
	}

	var want fs.FileMode = _S_IREAD
	if flags == ACCESS_READWRITE {
		want |= _S_IWRITE
	}
	if fi.IsDir() {
		want |= _S_IEXEC
	}
	if fi.Mode()&want != want {
		return fs.ErrPermission
	}
	return nil
}
