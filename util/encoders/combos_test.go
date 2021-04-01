package encoders

import (
	"bytes"
	"testing"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
)

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

func TestGzipEnglish(t *testing.T) {
	sample := randomData()

	gzEnglishEncoder := new(GzipEnglish)
	output := gzEnglishEncoder.Encode(sample)
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

	implantGzEnglishEncoder := new(implantEncoders.GzipEnglish)
	data2, err := implantGzEnglishEncoder.Decode(output)
	if err != nil {
		t.Errorf("implant gzEnglish decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data2) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output  = %#v", output)
		t.Logf("  data2 = %#v", data2)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data2)
	}

	output2 := implantGzEnglishEncoder.Encode(sample)
	data3, err := gzEnglishEncoder.Decode(output2)
	if err != nil {
		t.Errorf("gzEnglish decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data3) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output2 = %#v", output2)
		t.Logf("  data3 = %#v", data3)
		t.Errorf("gzEnglish does not match returned\n%#v != %#v", sample, data3)
	}

	output4 := gzEnglishEncoder.Encode(sample)
	data4, err := implantGzEnglishEncoder.Decode(output4)
	if err != nil {
		t.Errorf("gzEnglish decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data4) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output4  = %#v", output4)
		t.Logf("  data4 = %#v", data4)
		t.Errorf("gzEnglish does not match returned\n%#v != %#v", sample, data3)
	}
}

func TestBase64Gzip(t *testing.T) {
	sample := randomData()

	b64Gz := new(Base64Gzip)
	output := b64Gz.Encode(sample)
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

	implantBase64Gz := new(implantEncoders.Base64Gzip)
	data2, err := implantBase64Gz.Decode(output)
	if err != nil {
		t.Errorf("implant implantBase64Gz decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data2) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output  = %#v", output)
		t.Logf("  data2 = %#v", data2)
		t.Errorf("implantB64Gz sample does not match returned\n%#v != %#v", sample, data)
	}
	output2 := implantBase64Gz.Encode(sample)
	if !bytes.Equal(output, output2) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output1 = %#v", output)
		t.Logf("output2 = %#v", output2)
		t.Errorf("server and implant outputs differ\n%#v != %#v", sample, data)
	}
}
