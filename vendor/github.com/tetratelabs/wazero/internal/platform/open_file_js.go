package platform

import (
	"io/fs"
	"os"
	"syscall"
)

// See the comments on the same constants in open_file_windows.go
const (
	O_DIRECTORY = 1 << 29
	O_NOFOLLOW  = 1 << 30
)

func OpenFile(path string, flag int, perm fs.FileMode) (File, syscall.Errno) {
	flag &= ^(O_DIRECTORY | O_NOFOLLOW) // erase placeholders
	f, err := os.OpenFile(path, flag, perm)
	return f, UnwrapOSError(err)
}
