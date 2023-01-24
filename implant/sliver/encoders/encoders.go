package encoders

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"errors"
	insecureRand "math/rand"

	util "github.com/bishopfox/sliver/util/encoders"
)

var (
	Base64  = util.Base64{}
	Hex     = util.Hex{}
	English = util.English{}
	Gzip    = util.Gzip{}
	PNG     = util.PNGEncoder{}

	// {{if .Config.Debug}}
	Nop = util.NoEncoder{}
	// {{end}}
)

// EncoderMap - Maps EncoderIDs to Encoders
var EncoderMap = map[int]util.Encoder{
	util.Base64EncoderID:  Base64,
	util.HexEncoderID:     Hex,
	util.EnglishEncoderID: English,
	util.GzipEncoderID:    Gzip,
	util.PNGEncoderID:     PNG,

	// {{if .Config.Debug}}
	0: util.NoEncoder{},
	// {{end}}
}

// EncoderMap - Maps EncoderIDs to Encoders
var NativeEncoderMap = map[int]util.Encoder{
	util.Base64EncoderID:  Base64,
	util.HexEncoderID:     Hex,
	util.EnglishEncoderID: English,
	util.GzipEncoderID:    Gzip,
	util.PNGEncoderID:     PNG,

	// {{if .Config.Debug}}
	0: util.NoEncoder{},
	// {{end}}
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce int) (int, util.Encoder, error) {
	encoderID := nonce % util.EncoderModulus
	if encoderID == 0 {
		return 0, new(util.NoEncoder), nil
	}
	if encoder, ok := EncoderMap[encoderID]; ok {
		return encoderID, encoder, nil
	}
	return -1, nil, errors.New("invalid encoder nonce")
}

// RandomEncoder - Get a random nonce identifier and a matching encoder
func RandomEncoder() (int, util.Encoder) {
	keys := make([]int, 0, len(EncoderMap))
	for k := range EncoderMap {
		keys = append(keys, k)
	}
	encoderID := keys[insecureRand.Intn(len(keys))]
	nonce := (insecureRand.Intn(util.MaxN) * util.EncoderModulus) + encoderID
	return nonce, EncoderMap[encoderID]
}
