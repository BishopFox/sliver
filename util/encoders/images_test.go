package encoders

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
	"testing"
)

var (
	imageTests = []struct {
		Input []byte
	}{
		{[]byte("abc")},   // byte count on image pixel alignment
		{[]byte("abcde")}, // byte count offset of image pixel alignment
		{[]byte{0x0, 0x01, 0x02, 0x03, 0x04}},
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x0}},
	}
)

func TestPNG(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for _, test := range imageTests {
		buf, _ := pngEncoder.Encode(test.Input)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(test.Input, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, test.Input)
		}
	}
}

func TestPNGRandomDataRandomSize(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(1024 * 1024)
		buf, _ := pngEncoder.Encode(sample)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(sample, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, sample)
		}
	}
}
