//go:build !unix || sqlite3_nosys

package vfs

import "os"

func osSetMode(file *os.File, modeof string) error {
	fi, err := os.Stat(modeof)
	if err != nil {
		return err
	}
	file.Chmod(fi.Mode())
	return nil
}
