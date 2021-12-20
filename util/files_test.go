package util

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
)

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
