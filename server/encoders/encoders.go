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
	"crypto/rand"
	"encoding/binary"
	"fmt"
	insecureRand "math/rand"
	"path"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	util "github.com/bishopfox/sliver/util/encoders"
	"github.com/bishopfox/sliver/util/encoders/traffic"
)

var (
	encodersLog       = log.NamedLogger("encoders", "")
	trafficEncoderLog = log.NamedLogger("encoders", "traffic-encoders")

	Base64  = util.Base64{}
	Base58  = util.Base58{}
	Base32  = util.Base32{}
	Hex     = util.Hex{}
	English = util.English{}
	Gzip    = util.Gzip{}
	PNG     = util.PNGEncoder{}
	Nop     = util.NoEncoder{}
)

func init() {
	util.SetEnglishDictionary(assets.English())
	LoadTrafficEncodersFromFS(assets.TrafficEncoderFS, func(msg string) {
		trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// EncoderMap - Maps EncoderIDs to Encoders
var EncoderMap = map[uint64]util.Encoder{
	util.Base64EncoderID:  Base64,
	util.HexEncoderID:     Hex,
	util.EnglishEncoderID: English,
	util.GzipEncoderID:    Gzip,
	util.PNGEncoderID:     PNG,
}

// LoadTrafficEncodersFromFS - Loads the wasm traffic encoders from the filesystem, for the
// server these will be loaded from: <app root>/traffic-encoders/*.wasm
func LoadTrafficEncodersFromFS(encodersFS util.EncoderFS, logger func(string)) error {
	// Load WASM encoders
	encodersLog.Info("initializing traffic encoder map...")
	wasmEncoderFiles, err := encodersFS.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, wasmEncoderFile := range wasmEncoderFiles {
		encodersLog.Debugf("checking file: %s", wasmEncoderFile.Name())
		if wasmEncoderFile.IsDir() {
			continue
		}
		if !strings.HasSuffix(wasmEncoderFile.Name(), ".wasm") {
			continue
		}
		// WASM Module name should be equal to file name without the extension
		wasmEncoderModuleName := strings.TrimSuffix(wasmEncoderFile.Name(), ".wasm")
		wasmEncoderData, err := encodersFS.ReadFile(path.Join("traffic-encoders", wasmEncoderFile.Name()))
		if err != nil {
			encodersLog.Errorf(fmt.Sprintf("failed to read file %s (%s)", wasmEncoderModuleName, err.Error()))
			return err
		}
		wasmEncoderID := traffic.CalculateWasmEncoderID(wasmEncoderData)
		trafficEncoder, err := traffic.CreateTrafficEncoder(wasmEncoderModuleName, wasmEncoderData, logger)
		if err != nil {
			encodersLog.Errorf(fmt.Sprintf("failed to create traffic encoder from '%s': %s", wasmEncoderModuleName, err.Error()))
			return err
		}
		EncoderMap[uint64(wasmEncoderID)] = trafficEncoder
		encodersLog.Info(fmt.Sprintf("loading %s (id: %d, bytes: %d)", wasmEncoderModuleName, wasmEncoderID, len(wasmEncoderData)))
	}
	encodersLog.Info(fmt.Sprintf("loaded %d traffic encoders", len(wasmEncoderFiles)))
	return nil
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce uint64) (uint64, util.Encoder, error) {
	encoderID := uint64(nonce) % util.EncoderModulus
	if encoderID == 0 {
		return 0, new(util.NoEncoder), nil
	}
	if encoder, ok := EncoderMap[encoderID]; ok {
		return encoderID, encoder, nil
	}
	return 0, nil, fmt.Errorf("invalid encoder id: %d", encoderID)
}

// RandomEncoder - Get a random nonce identifier and a matching encoder
func RandomEncoder() (uint64, util.Encoder) {
	keys := make([]uint64, 0, len(EncoderMap))
	for k := range EncoderMap {
		keys = append(keys, k)
	}
	encoderID := keys[insecureRand.Intn(len(keys))]
	nonce := (randomUint64(util.MaxN) * util.EncoderModulus) + encoderID
	return nonce, EncoderMap[encoderID]
}

func randomUint64(max uint64) uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf) % max
}
