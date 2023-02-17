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

func TestHex(t *testing.T) {
	sample := randomData()

	// Server-side
	x := new(Hex)
	output := x.Encode(sample)
	data, err := x.Decode(output)
	if err != nil {
		t.Errorf("hex decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	// Implant-side
	implantHex := new(implantEncoders.Hex)
	output2 := implantHex.Encode(sample)
	data2, err := implantHex.Decode(output2)
	if err != nil {
		t.Errorf("implant hex decode returned an error %v", err)
	}
	if !bytes.Equal(sample, data2) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	// Interoperability
	if !bytes.Equal(output, output2) {
		t.Errorf("implant encoder does not match server-side encoder %s", err)
	}

	data3, err := implantHex.Decode(output)
	if err != nil {
		t.Errorf("implant hex decode could not decode server data %v", err)
	}
	if !bytes.Equal(sample, data3) {
		t.Errorf("implant decoded sample of server data does not match returned\n%#v != %#v", sample, data)
	}
}
