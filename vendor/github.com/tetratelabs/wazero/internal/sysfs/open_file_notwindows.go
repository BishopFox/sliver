//go:build !windows

package sysfs

import (
	"io/fs"
	"os"

	"github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
)

// openFile is like os.OpenFile except it accepts a fsapi.Oflag and returns
// sys.Errno. A zero sys.Errno is success.
func openFile(path string, oflag fsapi.Oflag, perm fs.FileMode) (*os.File, sys.Errno) {
	f, err := os.OpenFile(path, toOsOpenFlag(oflag), perm)
	// Note: This does not return a fsapi.File because fsapi.FS that returns
	// one may want to hide the real OS path. For example, this is needed for
	// pre-opens.
	return f, sys.UnwrapOSError(err)
}
