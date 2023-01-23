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

func TestGzipEnglish(t *testing.T) {
	InitEnglishDictionary(getTestEnglishDictionary())
	sample := randomData()

	gzEnglishEncoder := new(GzipEnglish)
	output, _ := gzEnglishEncoder.Encode(sample)
	data, err := gzEnglishEncoder.Decode(output)
	if err != nil {
		t.Errorf("gzEnglishEncoder decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}

func TestBase64Gzip(t *testing.T) {
	sample := randomData()

	b64Gz := new(Base64Gzip)
	output, _ := b64Gz.Encode(sample)
	data, err := b64Gz.Decode(output)
	if err != nil {
		t.Errorf("b64Gz decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("b64Gz sample does not match returned\n%#v != %#v", sample, data)
	}
}
