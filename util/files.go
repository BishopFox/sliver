package util

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ChmodR - Recursively chmod
func ChmodR(path string, filePerm, dirPerm os.FileMode) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			if info.IsDir() {
				err = os.Chmod(name, dirPerm)
			} else {
				err = os.Chmod(name, filePerm)
			}
		}
		return err
	})
}

// CopyFileContents - Copy/overwrite src to dst
func CopyFileContents(src string, dst string) error {
	// Calling f.Sync() should be necessary as long as the
	// returned err is properly checked. The only reason
	// this would fail implicitly (meaning the file isn't
	// available to a Stat() called immediately after calling
	// this function) would be because the kernel or filesystem
	// is inherently broken.
	contents, err := ioutil.ReadFile(filepath.Clean(src))
	if err != nil {
		return err
	}
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Clean(dst), contents, stat.Mode())
}

// ByteCountBinary - Pretty print byte size
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
