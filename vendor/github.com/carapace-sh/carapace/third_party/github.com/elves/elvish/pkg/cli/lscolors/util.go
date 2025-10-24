package lscolors

import "os"

// IsExecutable returns whether the FileInfo refers to an executable file.
//
// This is determined by permission bits on UNIX, and by file name on Windows.
func IsExecutable(stat os.FileInfo) bool {
	return isExecutable(stat)
}

func isExecutable(stat os.FileInfo) bool {
	return !stat.IsDir() && stat.Mode()&0o111 != 0
}
