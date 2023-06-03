package encoders

import "encoding/base64"

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

// Base64Encoder Encoder
type Base64Encoder struct{}

var base64Alphabet = "a0b2c5def6hijklmnopqr_st-uvwxyzA1B3C4DEFGHIJKLM7NO9PQR8ST+UVWXYZ"
var sliverBase64 = base64.NewEncoding(base64Alphabet).WithPadding(base64.NoPadding)

// Encode - Base64 Encode
func (e Base64Encoder) Encode(data []byte) ([]byte, error) {
	return []byte(sliverBase64.EncodeToString(data)), nil
}

// Decode - Base64 Decode
func (e Base64Encoder) Decode(data []byte) ([]byte, error) {
	return sliverBase64.DecodeString(string(data))
}
