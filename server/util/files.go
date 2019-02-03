package util

import (
	"io"
	"os"
)

// CopyFileContents - Copy/overwrite src to dst
func CopyFileContents(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	out.Sync()
	return nil
}
