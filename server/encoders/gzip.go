package encoders

import (
	"bytes"
	"compress/gzip"
	"io"
)

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
