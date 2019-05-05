package util

import (
	"bytes"
	"compress/gzip"
	"fmt"
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
