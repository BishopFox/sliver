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

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
)

func TestBase32(t *testing.T) {
	sample := randomData()

	b32 := new(Base32)
	output := b32.Encode(sample)
	data, err := b32.Decode(output)
	if err != nil {
		t.Errorf("b32 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	implantBase32 := new(implantEncoders.Base32)
	output2 := implantBase32.Encode(sample)
	data2, err := implantBase32.Decode(output2)
	if err != nil {
		t.Errorf("implant b32 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data2) {
		t.Logf("sample  = %#v", sample)
		t.Logf("output2 = %#v", output2)
		t.Logf("  data2 = %#v", data2)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	output = b32.Encode(sample)
	data, err = implantBase32.Decode(output)
	if err != nil {
		t.Errorf("b32 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	output = implantBase32.Encode(sample)
	data, err = b32.Decode(output)
	if err != nil {
		t.Errorf("b32 decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}
