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
	"errors"
	"fmt"
	insecureRand "math/rand"
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
var EncoderMap = map[int]util.Encoder{
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
	wasmEncoderFiles, err := encodersFS.ReadDir(".")
	if err != nil {
		return err
	}
	for _, wasmEncoderFile := range wasmEncoderFiles {
		if wasmEncoderFile.IsDir() {
			continue
		}
		// WASM Module name should be equal to file name without the extension
		wasmEncoderModuleName := strings.TrimSuffix(wasmEncoderFile.Name(), ".wasm")
		wasmEncoderData, err := encodersFS.ReadFile(wasmEncoderFile.Name())
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
		EncoderMap[int(wasmEncoderID)] = trafficEncoder
		encodersLog.Info(fmt.Sprintf("Loading %s (id: %d, bytes: %d)", wasmEncoderModuleName, wasmEncoderID, len(wasmEncoderData)))
	}
	encodersLog.Info(fmt.Sprintf("Loaded %d traffic encoders", len(wasmEncoderFiles)))
	return nil
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
