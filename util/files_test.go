package util

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomData() []byte {
	buf := make([]byte, 128)
	rand.Read(buf)
	return buf
}

func TestGzip(t *testing.T) {
	sample := randomData()
	gzipData := bytes.NewBuffer([]byte{})
	gz := new(Gzip)
	gz.Encode(gzipData, sample)
	data, err := gz.Decode(gzipData.Bytes())
	if err != nil {
		t.Errorf("gzip decode returned an error %v", err)
		return
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}
