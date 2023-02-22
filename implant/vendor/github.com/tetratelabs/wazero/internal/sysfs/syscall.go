//go:build !windows

package sysfs

import (
	"io/fs"
)

func adjustErrno(err error) error {
	return err
}

func adjustRmdirError(err error) error {
	return err
}

func adjustTruncateError(err error) error {
	return err
}

func maybeWrapFile(f file, _ FS, _ string, _ int, _ fs.FileMode) file {
	return f
}
