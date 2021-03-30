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

func TestEnglish(t *testing.T) {
	sample := randomData()

	english := new(English)
	output := english.Encode(sample)
	data, err := english.Decode(output)
	if err != nil {
		t.Error("Failed to encode sample data into english")
		return
	}
	if !bytes.Equal(sample, data) {
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}

	implantEnglish := new(implantEncoders.English)
	output2 := implantEnglish.Encode(sample)
	data2, err := implantEnglish.Decode(output2)
	if err != nil {
		t.Error("Failed to encode sample data into english")
		return
	}
	if !bytes.Equal(sample, data2) {
		t.Errorf("implant sample does not match returned\n%#v != %#v", sample, data)
	}

	if !bytes.Equal(data, data2) {
		t.Errorf("implant/server sample does not match\n%#v != %#v", data, data2)
	}
}
