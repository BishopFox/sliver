//go:build !unix || sqlite3_nosys

package vfs

import (
	"io/fs"
	"os"
)

func osAccess(path string, flags AccessFlag) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if flags == ACCESS_EXISTS {
		return nil
	}

	const (
		S_IREAD  = 0400
		S_IWRITE = 0200
		S_IEXEC  = 0100
	)

	var want fs.FileMode = S_IREAD
	if flags == ACCESS_READWRITE {
		want |= S_IWRITE
	}
	if fi.IsDir() {
		want |= S_IEXEC
	}
	if fi.Mode()&want != want {
		return fs.ErrPermission
	}
	return nil
}
