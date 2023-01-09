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
	"crypto/sha256"
	"embed"
	"errors"
	insecureRand "math/rand"
)

const (
	// EncoderModulus - Nonce % EncoderModulus = EncoderID, and needs to be equal
	//                  to or greater than the largest EncoderID value.
	EncoderModulus = 101
	maxN           = 999999
)

// Encoder - Can losslessly encode arbitrary binary data to ASCII
type Encoder interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

// EncoderMap - Maps EncoderIDs to Encoders
var EncoderMap = map[int]Encoder{
	Base64EncoderID:      Base64{},
	HexEncoderID:         Hex{},
	EnglishEncoderID:     English{},
	GzipEncoderID:        Gzip{},
	GzipEnglishEncoderID: GzipEnglish{},
	Base64GzipEncoderID:  Base64Gzip{},
}

func InitEncoderMap(encodersFS embed.FS) error {
	// Load WASM encoders
	wasmEncoderFiles, err := encodersFS.ReadDir(".")
	if err != nil {
		return err
	}
	for _, wasmEncoderFile := range wasmEncoderFiles {
		if wasmEncoderFile.IsDir() {
			continue
		}
		wasmEncoderName := wasmEncoderFile.Name()
		wasmEncoderData, err := encodersFS.ReadFile(wasmEncoderName)
		if err != nil {
			return err
		}
		wasmEncoderID := calculateWasmEncoderID(wasmEncoderData)
		trafficEncoder, err := CreateTrafficEncoder(wasmEncoderName, wasmEncoderData, func(s string) {})
		if err != nil {
			return err
		}
		EncoderMap[int(wasmEncoderID)] = trafficEncoder
	}
	return nil
}

func calculateWasmEncoderID(wasmEncoderData []byte) uint16 {
	digest := sha256.Sum256(wasmEncoderData)
	return uint16(digest[0])<<8 + uint16(digest[1])
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce int) (int, Encoder, error) {
	encoderID := nonce % EncoderModulus
	if encoderID == 0 {
		return 0, new(NoEncoder), nil
	}
	if encoder, ok := EncoderMap[encoderID]; ok {
		return encoderID, encoder, nil
	}
	return -1, nil, errors.New("invalid encoder nonce")
}

// RandomEncoder - Get a random nonce identifier and a matching encoder
func RandomEncoder() (int, Encoder) {
	keys := make([]int, 0, len(EncoderMap))
	for k := range EncoderMap {
		keys = append(keys, k)
	}
	encoderID := keys[insecureRand.Intn(len(keys))]
	nonce := (insecureRand.Intn(maxN) * EncoderModulus) + encoderID
	return nonce, EncoderMap[encoderID]
}

// NopNonce - A NOP nonce identifies a request with no encoder/payload
//
//	any value where mod = 0
func NopNonce() int {
	return insecureRand.Intn(maxN) * EncoderModulus
}

// NoEncoder - A NOP encoder
type NoEncoder struct{}

// Encode - Don't do anything
func (n NoEncoder) Encode(data []byte) ([]byte, error) {
	return data, nil
}

// Decode - Don't do anything
func (n NoEncoder) Decode(data []byte) ([]byte, error) {
	return data, nil
}
