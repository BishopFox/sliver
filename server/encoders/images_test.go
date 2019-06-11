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
