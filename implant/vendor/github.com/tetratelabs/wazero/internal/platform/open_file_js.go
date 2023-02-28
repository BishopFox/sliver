//go:build js

package platform

import (
	"io/fs"
	"os"
)

// See the comments on the same constants in open_file_windows.go
const (
	O_DIRECTORY = 1 << 29
	O_NOFOLLOW  = 1 << 30
)

func OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	flag &= ^(O_DIRECTORY | O_NOFOLLOW) // erase placeholders
	return os.OpenFile(name, flag, perm)
}
