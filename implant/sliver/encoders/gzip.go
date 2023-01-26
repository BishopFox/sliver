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
	"compress/gzip"
	"sync"
)

// Gzip - Gzip compression encoder
type GzipEncoder struct{}

var gzipWriterPools = &sync.Pool{}

func init() {
	gzipWriterPools = &sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gzip.BestCompression)
			return w
		},
	}
}

// Encode - Compress data with gzip
func (g GzipEncoder) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzipWriterPools.Get().(*gzip.Writer)
	gzipWriter.Reset(&buf)
	gzipWriter.Write(data)
	gzipWriter.Close()
	gzipWriterPools.Put(gzipWriter)
	return buf.Bytes(), nil
}

// Decode - Uncompressed data with gzip
func (g GzipEncoder) Decode(data []byte) ([]byte, error) {
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
