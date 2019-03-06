package encoders

import (
	"bytes"
	"compress/gzip"
	"io"
)

// GzipEncode - Compress data with gzip
func GzipEncode(w io.Writer, data []byte) error {
	gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	defer gw.Close()
	gw.Write(data)
	return err
}

// GzipDecode - Uncompress data with gzip
func GzipDecode(data []byte) ([]byte, error) {
	bytes.NewReader(data)
	reader, _ := gzip.NewReader(bytes.NewReader(data))
	var buf bytes.Buffer
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
