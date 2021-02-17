package encoders

import "encoding/hex"

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

// HexEncoderID - EncoderID
const HexEncoderID = 92

// Hex Encoder
type Hex struct{}

// Encode - Hex Encode
func (e Hex) Encode(data []byte) []byte {
	return []byte(hex.EncodeToString(data))
}

// Decode - Hex Decode
func (e Hex) Decode(data []byte) ([]byte, error) {
	return hex.DecodeString(string(data))
}
