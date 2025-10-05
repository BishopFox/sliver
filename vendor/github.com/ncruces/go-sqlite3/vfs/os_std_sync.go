//go:build !(linux || darwin) || sqlite3_flock

package vfs

import "os"

func osSync(file *os.File, _ OpenFlag, _ SyncFlag) error {
	return file.Sync()
}
