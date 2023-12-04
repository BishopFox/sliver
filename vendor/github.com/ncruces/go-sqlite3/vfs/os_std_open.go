//go:build !windows || sqlite3_nosys

package vfs

import (
	"io/fs"
	"os"
)

func osOpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}
