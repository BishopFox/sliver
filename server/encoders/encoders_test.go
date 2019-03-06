package encoders

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomData() []byte {
	buf := make([]byte, 1024)
	rand.Read(buf)
	return buf
}

func TestGzip(t *testing.T) {
	sample := randomData()
	gzipData := bytes.NewBuffer([]byte{})
	GzipEncode(gzipData, sample)
	data, err := GzipDecode(gzipData.Bytes())
	if err != nil {
		t.Errorf("gzip decode returned an error %v", err)
	}
	if bytes.Compare(sample, data) != 0 {
		t.Errorf("sample data does not match data returned from gzip decode")
	}
}
