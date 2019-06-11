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
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
)

// CopyFileContents - Copy/overwrite src to dst
func CopyFileContents(src string, dst string) error {
	// Calling f.Sync() should be unecessary as long as the
	// returned err is properly checked. The only reason
	// this would fail implictly (meaning the file isn't
	// available to a Stat() called immediately after calling
	// this function) would be because the kernel or filesystem
	// is inherently broken.
	contents, err := ioutil.ReadFile(filepath.Clean(src))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Clean(dst), contents, 0775)
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

// Gzip - Gzip compression encoder
type Gzip struct{}

// Encode - Compress data with gzip
func (g Gzip) Encode(w io.Writer, data []byte) error {
	gw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	defer gw.Close()
	_, err := gw.Write(data)
	return err
}

// Decode - Uncompress data with gzip
func (g Gzip) Decode(data []byte) ([]byte, error) {
	bytes.NewReader(data)
	reader, _ := gzip.NewReader(bytes.NewReader(data))
	var buf bytes.Buffer
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
