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
	"crypto/rand"
	insecureRand "math/rand"
	"testing"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
)

func TestGzip(t *testing.T) {
	sample := randomData()

	gzip := new(Gzip)
	output := gzip.Encode(sample)
	data, err := gzip.Decode(output)
	if err != nil {
		t.Errorf("gzip decode returned an error %v", err)
	}
	if !bytes.Equal(data, sample) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	implantGzip := new(implantEncoders.Gzip)
	output2 := implantGzip.Encode(sample)
	data2, err := implantGzip.Decode(output)
	if err != nil {
		t.Errorf("implant gzip decode returned an error %v", err)
	}
	if !bytes.Equal(data2, sample) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output2)
		t.Logf("  data = %#v", data2)
		t.Errorf("sample does not match implant returned\n%#v != %#v", sample, data)
	}

	if !bytes.Equal(output, output2) {
		t.Logf("output1 = %#v", output)
		t.Logf("output2 = %#v", output2)
		t.Errorf("server/implant outputs do not match returned\n%#v != %#v", sample, data)
	}
}

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize))
	rand.Read(buf)
	return buf
}

func TestGzipGunzip(t *testing.T) {
	for i := 0; i < 100; i++ {
		data := randomDataRandomSize(8192)
		gzipData := GzipBuf(data)
		gunzipData := GunzipBuf(gzipData)
		if !bytes.Equal(data, gunzipData) {
			t.Fatalf("Data does not match")
		}
	}
}
