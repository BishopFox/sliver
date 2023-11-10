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
	"io/fs"
	insecureRand "math/rand"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/log"
	util "github.com/bishopfox/sliver/util/encoders"
	"github.com/bishopfox/sliver/util/encoders/traffic"
)

const (

	// EncoderModulus - The modulus used to calculate the encoder ID from a C2 request nonce
	// *** IMPORTANT *** ENCODER IDs MUST BE LESS THAN THE MODULUS
	EncoderModulus = uint64(65537)
	MaxN           = uint64(9999999)
)

var (
	encodersLog       = log.NamedLogger("encoders", "")
	trafficEncoderLog = log.NamedLogger("encoders", "traffic-encoders")

	TrafficEncoderFS = PassthroughEncoderFS{}

	Base64  = util.Base64{}
	Base58  = util.Base58{}
	Base32  = util.Base32{}
	Hex     = util.Hex{}
	English = util.English{}
	Gzip    = util.Gzip{}
	PNG     = util.PNGEncoder{}
	Nop     = util.NoEncoder{}

	NoEncoderID      = uint64(0)
	Base64EncoderID  = SetupDefaultEncoders("base64")
	Base58EncoderID  = SetupDefaultEncoders("base58")
	Base32EncoderID  = SetupDefaultEncoders("base32")
	HexEncoderID     = SetupDefaultEncoders("hex")
	EnglishEncoderID = SetupDefaultEncoders("english")
	GzipEncoderID    = SetupDefaultEncoders("gzip")
	PNGEncoderID     = SetupDefaultEncoders("png")
	NopEncoderID     = SetupDefaultEncoders("nop")
	UnavailableID    = PopulateID()
)

func SetupDefaultEncoders(name string) uint64 {

	encoders, err := db.ResourceIDByType("encoder")
	if err != nil {
		encodersLog.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	for _, encoder := range encoders {
		if encoder.Name == name {
			return encoder.Value
		}
	}

	id := GetRandomID()
	err = db.SaveResourceID(&clientpb.ResourceID{
		Type:  "encoder",
		Name:  name,
		Value: id,
	})
	if err != nil {
		encodersLog.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	return id
}

// generate unavailable id array on startup
func PopulateID() []uint64 {
	// remove already used prime numbers from available pool
	resourceIDs, err := db.ResourceIDs()
	if err != nil {
		encodersLog.Printf("Error:\n%s", err)
		os.Exit(-1)
	}
	var UnavailableID []uint64
	for _, resourceID := range resourceIDs {
		UnavailableID = append(UnavailableID, resourceID.Value)
	}

	return UnavailableID
}

// generate a random id and ensure it is not in use
func GetRandomID() uint64 {
	id := insecureRand.Intn(int(EncoderModulus))
	for slices.Contains(UnavailableID, uint64(id)) {
		id = insecureRand.Intn(int(EncoderModulus))
	}
	UnavailableID = append(UnavailableID, uint64(id))
	return uint64(id)
}

func init() {
	util.SetEnglishDictionary(assets.English())
	TrafficEncoderFS = PassthroughEncoderFS{
		rootDir: filepath.Join(assets.GetRootAppDir(), "traffic-encoders"),
	}
	loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// EncoderMap - A map of all available encoders (native and traffic/wasm)
var EncoderMap = map[uint64]util.Encoder{
	Base64EncoderID:  Base64,
	Base58EncoderID:  Base58,
	Base32EncoderID:  Base32,
	HexEncoderID:     Hex,
	EnglishEncoderID: English,
	GzipEncoderID:    Gzip,
	PNGEncoderID:     PNG,
}

// TrafficEncoderMap - Keeps track of the loaded traffic encoders (i.e., wasm-based encoder functions)
var TrafficEncoderMap = map[uint64]*traffic.TrafficEncoder{}

// FastEncoderMap - Keeps track of fast native encoders that can be used for large messages
var FastEncoderMap = map[uint64]util.Encoder{
	Base64EncoderID: Base64,
	Base58EncoderID: Base58,
	Base32EncoderID: Base32,
	HexEncoderID:    Hex,
	GzipEncoderID:   Gzip,
}

// SaveTrafficEncoder - Save a traffic encoder to the filesystem
func SaveTrafficEncoder(name string, wasmBin []byte) error {
	if !strings.HasSuffix(name, ".wasm") {
		return fmt.Errorf("invalid encoder name, must end with .wasm")
	}
	wasmFilePath := filepath.Join(assets.GetTrafficEncoderDir(), filepath.Base(name))
	err := os.WriteFile(wasmFilePath, wasmBin, 0600)
	if err != nil {
		return err
	}
	return loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// RemoveTrafficEncoder - Save a traffic encoder to the filesystem
func RemoveTrafficEncoder(name string) error {
	if !strings.HasSuffix(name, ".wasm") {
		return fmt.Errorf("invalid encoder name, must end with .wasm")
	}
	wasmFilePath := filepath.Join(assets.GetTrafficEncoderDir(), filepath.Base(name))
	info, err := os.Stat(wasmFilePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		panic("wasmFilePath is a directory, this should never happen")
	}
	err = os.Remove(wasmFilePath)
	if err != nil {
		return err
	}
	return loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// loadTrafficEncodersFromFS - Loads the wasm traffic encoders from the filesystem, for the
// server these will be loaded from: <app root>/traffic-encoders/*.wasm
func loadTrafficEncodersFromFS(encodersFS util.EncoderFS, logger func(string)) error {

	// Reset references pointing to traffic encoders
	for _, encoder := range TrafficEncoderMap {
		delete(EncoderMap, encoder.ID)
	}
	TrafficEncoderMap = map[uint64]*traffic.TrafficEncoder{}

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
		trafficEncoder.FileName = wasmEncoderFile.Name()
		if _, ok := EncoderMap[uint64(wasmEncoderID)]; ok {
			encodersLog.Errorf(fmt.Sprintf("duplicate encoder id: %d", wasmEncoderID))
			return fmt.Errorf("duplicate encoder id: %d", wasmEncoderID)
		}
		EncoderMap[uint64(wasmEncoderID)] = trafficEncoder
		TrafficEncoderMap[uint64(wasmEncoderID)] = trafficEncoder
		encodersLog.Info(fmt.Sprintf("loading %s (id: %d, bytes: %d)", wasmEncoderModuleName, wasmEncoderID, len(wasmEncoderData)))
	}
	encodersLog.Info(fmt.Sprintf("loaded %d traffic encoders", len(wasmEncoderFiles)))
	return nil
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce uint64) (uint64, util.Encoder, error) {
	encoderID := uint64(nonce) % EncoderModulus
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
	nonce := (randomUint64(MaxN) * EncoderModulus) + encoderID
	return nonce, EncoderMap[encoderID]
}

func randomUint64(max uint64) uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf) % max
}

// PassthroughEncoderFS - Creates an encoder.EncoderFS object from a single local directory
type PassthroughEncoderFS struct {
	rootDir string
}

func (p PassthroughEncoderFS) Open(name string) (fs.File, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if stat, err := os.Stat(localPath); os.IsNotExist(err) || stat.IsDir() {
		return nil, os.ErrNotExist
	}
	return os.Open(localPath)
}

func (p PassthroughEncoderFS) ReadDir(_ string) ([]fs.DirEntry, error) {
	if _, err := os.Stat(p.rootDir); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	ls, err := os.ReadDir(p.rootDir)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for _, entry := range ls {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".wasm") {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (p PassthroughEncoderFS) ReadFile(name string) ([]byte, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if stat, err := os.Stat(localPath); os.IsNotExist(err) || stat.IsDir() {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(localPath)
}
