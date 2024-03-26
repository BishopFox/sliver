//go:build !(linux || darwin) || sqlite3_flock || sqlite3_nosys

package vfs

import "os"

func osSync(file *os.File, _ /*fullsync*/, _ /*dataonly*/ bool) error {
	return file.Sync()
}
