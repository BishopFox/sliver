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
	"crypto/rand"
	"embed"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/implant/sliver/util"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	// {{if .Config.TrafficEncodersEnabled}}
	"path"
	// {{end}}

	// {{if .Config.TrafficEncodersEnabled}}
	"github.com/bishopfox/sliver/implant/sliver/encoders/traffic"
	// {{end}}
)

var (
	EncoderModulus = uint64(65537)
	MaxN           = uint64(9999999)

	Base32EncoderID, _  = strconv.ParseUint(`{{.Encoders.Base32EncoderID}}`, 10, 64)
	Base58EncoderID, _  = strconv.ParseUint(`{{.Encoders.Base58EncoderID}}`, 10, 64)
	Base64EncoderID, _  = strconv.ParseUint(`{{.Encoders.Base64EncoderID}}`, 10, 64)
	EnglishEncoderID, _ = strconv.ParseUint(`{{.Encoders.EnglishEncoderID}}`, 10, 64)
	GzipEncoderID, _    = strconv.ParseUint(`{{.Encoders.GzipEncoderID}}`, 10, 64)
	HexEncoderID, _     = strconv.ParseUint(`{{.Encoders.HexEncoderID}}`, 10, 64)
	PNGEncoderID, _     = strconv.ParseUint(`{{.Encoders.PNGEncoderID}}`, 10, 64)
)

func init() {
	err := loadWasmEncodersFromAssets()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to load WASM encoders: %v", err)
		// {{end}}
		return
	}
	err = loadEnglishDictionaryFromAssets()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to load english dictionary: %v", err)
		// {{end}}
		return
	}
}

var (
	//go:embed assets/*
	implantAssetsFS embed.FS // files will be gzip'd

	Base64  = Base64Encoder{}
	Base58  = Base58Encoder{}
	Base32  = Base32Encoder{}
	Hex     = HexEncoder{}
	English = EnglishEncoder{}
	Gzip    = GzipEncoder{}
	PNG     = PNGEncoder{}

	// {{if .Config.Debug}}
	Nop = NoEncoder{}
	// {{end}}
)

// EncoderMap - Maps EncoderIDs to Encoders
var EncoderMap = map[uint64]Encoder{}

// EncoderMap - Maps EncoderIDs to Encoders
var NativeEncoderMap = map[uint64]Encoder{
	Base64EncoderID:  Base64,
	HexEncoderID:     Hex,
	EnglishEncoderID: English,
	GzipEncoderID:    Gzip,
	PNGEncoderID:     PNG,

	// {{if .Config.Debug}}
	0: NoEncoder{},
	// {{end}}
}

// Encoder - Can lossless-ly encode arbitrary binary data
type Encoder interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce uint64) (uint64, Encoder, error) {
	encoderID := nonce % EncoderModulus
	if encoderID == 0 {
		return 0, new(NoEncoder), nil
	}
	if encoder, ok := EncoderMap[encoderID]; ok {
		return encoderID, encoder, nil
	}
	return 0, nil, errors.New("invalid encoder nonce")
}

func getMaxEncoderSize() int {
	return 16 * 1024 * 1024 // 16MB
}

// RandomEncoder - Get a random nonce identifier and a matching encoder
func RandomEncoder(size int) (uint64, Encoder) {
	if size < getMaxEncoderSize() && len(EncoderMap) > 0 {
		return randomEncoderFromMap(EncoderMap) // Small message, use any encoder
	} else {
		return randomEncoderFromMap(NativeEncoderMap) // Large message, use native encoders
	}
}

func randomEncoderFromMap(encoderMap map[uint64]Encoder) (uint64, Encoder) {
	keys := make([]uint64, 0, len(encoderMap))
	for k := range encoderMap {
		keys = append(keys, k)
	}
	encoderID := keys[util.Intn(len(keys))]
	nonce := (randomUint64(MaxN) * EncoderModulus) + encoderID
	return nonce, encoderMap[encoderID]
}

func randomUint64(max uint64) uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf) % max
}

func loadWasmEncodersFromAssets() error {

	// *** {{if .Config.TrafficEncodersEnabled}} ***

	// {{if .Config.Debug}}}
	log.Printf("initializing traffic encoder map...")
	// {{end}}

	assetFiles, err := implantAssetsFS.ReadDir("assets")
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to read assets directory: %v", err)
		// {{end}}
		return err
	}
	for _, assetFile := range assetFiles {
		// {{if .Config.Debug}}
		log.Printf("Unpacking asset: %s", assetFile.Name())
		// {{end}}
		if assetFile.IsDir() {
			continue
		}
		if !strings.HasSuffix(assetFile.Name(), ".wasm") {
			continue
		}
		// WASM Module name should be equal to file name without the extension
		wasmEncoderModuleName := strings.TrimSuffix(assetFile.Name(), ".wasm")
		wasmEncoderData, err := implantAssetsFS.ReadFile(path.Join("assets", assetFile.Name()))
		if err != nil {
			return err
		}
		wasmEncoderData, err = Gzip.Decode(wasmEncoderData)
		if err != nil {
			return err
		}
		wasmEncoderID := traffic.CalculateWasmEncoderID(wasmEncoderData)
		trafficEncoder, err := traffic.CreateTrafficEncoder(wasmEncoderModuleName, wasmEncoderData, func(msg string) {
			// {{if .Config.Debug}}
			log.Printf("[Traffic Encoder] %s", msg)
			// {{end}}
		})
		if err != nil {
			return err
		}
		EncoderMap[wasmEncoderID] = trafficEncoder
		// {{if .Config.Debug}}
		log.Printf("loading %s (id: %d, bytes: %d)", wasmEncoderModuleName, wasmEncoderID, len(wasmEncoderData))
		// {{end}}
	}
	// {{if .Config.Debug}}
	log.Printf("completed loading traffic encoders")
	log.Printf("current encoder map:")
	for encoderID, encoder := range EncoderMap {
		log.Printf("encoder %d -> %#v", encoderID, encoder)
	}
	//	{{end}}

	// *** {{end}} ***
	return nil
}

func loadEnglishDictionaryFromAssets() error {
	englishData, err := implantAssetsFS.ReadFile("assets/english.gz")
	if err != nil {
		return err
	}
	englishData, err = Gzip.Decode(englishData)
	if err != nil {
		return err
	}
	for _, word := range strings.Split(string(englishData), "\n") {
		rawEnglishDictionary = append(rawEnglishDictionary, strings.TrimSpace(word))
	}
	return nil
}
