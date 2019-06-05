package encoders

import (
	"bytes"
	"testing"
)

var (
	imageTests = []struct {
		Input []byte
	}{
		{[]byte("abc")},   // byte count on image pixel allignment
		{[]byte("abcde")}, // byte count offset of image pixel allignment
		{randomData()},    // random binary data, will fail if contains leading/trailing nulls
	}
)

func TestPNG(t *testing.T) {
	for _, test := range imageTests {
		png := new(PNG)
		b := new(bytes.Buffer)
		if err := png.Encode(b, test.Input); err != nil {
			t.Errorf("png encode returned error: %q", err)
		}
		decodeOutput, err := png.Decode(b.Bytes())
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(test.Input, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, test.Input)
		}
	}
}
