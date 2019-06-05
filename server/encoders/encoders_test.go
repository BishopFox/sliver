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

func TestEnglish(t *testing.T) {
	sample := randomData()
	english := new(English)
	encoded := english.Encode(sample)
	data, err := english.Decode(encoded)
	if err != nil {
		t.Error("Failed to encode sample data into english")
		return
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
