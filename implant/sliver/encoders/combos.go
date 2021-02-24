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

// GzipEnglishEncoderID - EncoderID
const GzipEnglishEncoderID = 45

// GzipEnglish - Gzip+English encoder
type GzipEnglish struct{}

// Encode - Compress english data with gzip
func (g GzipEnglish) Encode(data []byte) []byte {
	gzip := EncoderMap[GzipEncoderID]
	english := EncoderMap[EnglishEncoderID]
	return gzip.Encode(english.Encode(data))
}

// Decode - Uncompressed english data with gzip
func (g GzipEnglish) Decode(data []byte) ([]byte, error) {
	gzip := EncoderMap[GzipEncoderID]
	english := EncoderMap[EnglishEncoderID]
	unzipped, err := gzip.Decode(data)
	if err != nil {
		return nil, err
	}
	return english.Decode(unzipped)
}

// Base64GzipEncoderID - EncoderID
const Base64GzipEncoderID = 64

// Base64Gzip - Base64+Gzip encoder
type Base64Gzip struct{}

// Encode - Base64 encode gzip data
func (g Base64Gzip) Encode(data []byte) []byte {
	gzip := EncoderMap[GzipEncoderID]
	b64 := EncoderMap[Base64EncoderID]
	return b64.Encode(gzip.Encode(data))
}

// Decode - Un-base64 gzip data
func (g Base64Gzip) Decode(data []byte) ([]byte, error) {
	gzip := EncoderMap[GzipEncoderID]
	b64 := EncoderMap[Base64EncoderID]
	raw, err := b64.Decode(data)
	if err != nil {
		return nil, err
	}
	return gzip.Decode(raw)
}
