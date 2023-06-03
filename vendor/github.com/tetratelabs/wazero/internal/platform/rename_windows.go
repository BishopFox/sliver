package platform

import (
	"errors"
	"os"
	"syscall"
)

func Rename(from, to string) syscall.Errno {
	if from == to {
		return 0
	}

	fromStat, err := os.Stat(from)
	if err != nil {
		return syscall.ENOENT
	}

	if toStat, err := os.Stat(to); err == nil {
		fromIsDir, toIsDir := fromStat.IsDir(), toStat.IsDir()
		if fromIsDir && !toIsDir { // dir to file
			return syscall.ENOTDIR
		} else if !fromIsDir && toIsDir { // file to dir
			return syscall.EISDIR
		} else if !fromIsDir && !toIsDir { // file to file
			// Use os.Rename instead of syscall.Rename in order to allow the overrides of the existing file.
			// Underneath os.Rename, it uses MoveFileEx instead of MoveFile (used by syscall.Rename).
			return UnwrapOSError(os.Rename(from, to))
		} else { // dir to dir
			if dirs, _ := os.ReadDir(to); len(dirs) == 0 {
				// On Windows, renaming to the empty dir will be rejected,
				// so first we remove the empty dir, and then rename to it.
				if err := os.Remove(to); err != nil {
					return UnwrapOSError(err)
				}
				return UnwrapOSError(syscall.Rename(from, to))
			}
			return syscall.ENOTEMPTY
		}
	} else if !errors.Is(err, syscall.ENOENT) { // Failed to stat the destination.
		return UnwrapOSError(err)
	} else { // Destination not-exist.
		return UnwrapOSError(syscall.Rename(from, to))
	}
}
