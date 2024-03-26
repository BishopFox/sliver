//go:build !windows || sqlite3_nosys

package vfs

import "time"

func osSleep(d time.Duration) {
	time.Sleep(d)
}
