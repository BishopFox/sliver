//go:build !windows

package osutil

import (
	"io/fs"
	"os"
)

// OpenFile behaves the same as [os.OpenFile],
// except on Windows it sets [syscall.FILE_SHARE_DELETE].
//
// See: https://go.dev/issue/32088#issuecomment-502850674
func OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}
