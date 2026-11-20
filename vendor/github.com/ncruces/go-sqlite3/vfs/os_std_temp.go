//go:build !linux || sqlite3_flock

package vfs

import "os"

func osCreateTemp(flags OpenFlag) (*os.File, error) {
	f, err := os.CreateTemp(os.Getenv("SQLITE_TMPDIR"), "*.db")
	if err != nil {
		return nil, sysError{err, _IOERR_GETTEMPPATH}
	}
	if isUnix && flags&OPEN_DELETEONCLOSE != 0 {
		os.Remove(f.Name())
	}
	return f, nil
}
