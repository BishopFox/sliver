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
	"io/fs"
)

type EncodersList struct {
	Base32EncoderID  uint64
	Base58EncoderID  uint64
	Base64EncoderID  uint64
	EnglishEncoderID uint64
	GzipEncoderID    uint64
	HexEncoderID     uint64
	PNGEncoderID     uint64
}

// Encoder - Can losslessly encode arbitrary binary data
type Encoder interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

// EncoderFS - Generic interface to read wasm encoders from a filesystem
type EncoderFS interface {
	Open(name string) (fs.File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}
