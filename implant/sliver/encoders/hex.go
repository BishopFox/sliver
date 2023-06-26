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

import "encoding/hex"

// Hex Encoder
type HexEncoder struct{}

// Encode - Hex Encode
func (e HexEncoder) Encode(data []byte) ([]byte, error) {
	return []byte(hex.EncodeToString(data)), nil
}

// Decode - Hex Decode
func (e HexEncoder) Decode(data []byte) ([]byte, error) {
	return hex.DecodeString(string(data))
}
