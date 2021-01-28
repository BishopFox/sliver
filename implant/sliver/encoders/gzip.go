package encoders

import (
	"bytes"
	"compress/gzip"
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

// GzipEncoderID - EncoderID
const GzipEncoderID = 49

// Gzip - Gzip compression encoder
type Gzip struct{}

// Encode - Compress data with gzip
func (g Gzip) Encode(data []byte) []byte {
	var buf bytes.Buffer
	gzipWriter, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	gzipWriter.Write(data)
	gzipWriter.Close()
	return buf.Bytes()
}

// Decode - Uncompressed data with gzip
func (g Gzip) Decode(data []byte) ([]byte, error) {
	bytes.NewReader(data)
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
