package encoders

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
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}

func TestHex(t *testing.T) {
	sample := randomData()
	x := new(Hex)
	output := x.Encode(sample)
	data, err := x.Decode(output)
	if err != nil {
		t.Errorf("hex decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}

func TestBase64(t *testing.T) {
	sample := randomData()
	b64 := new(Base64)
	output := b64.Encode(sample)
	data, err := b64.Decode(output)
	if err != nil {
		t.Errorf("b64 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}
