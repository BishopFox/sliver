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
	"log"
	insecureRand "math/rand"
	"os"
	"slices"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
)

const (

	// EncoderModulus - The modulus used to calculate the encoder ID from a C2 request nonce
	// *** IMPORTANT *** ENCODER IDs MUST BE LESS THAN THE MODULUS
	EncoderModulus = uint64(65537)
	MaxN           = uint64(9999999)
)

var (
	// These were chosen at random other than the "No Encoder" ID (0)
	UnavailableID    = populateID()
	Base32EncoderID  = uint64(SetupDefaultEncoders("Base32Encoder"))
	Base58EncoderID  = uint64(SetupDefaultEncoders("Base58EncoderID"))
	Base64EncoderID  = uint64(SetupDefaultEncoders("Base64EncoderID"))
	EnglishEncoderID = uint64(SetupDefaultEncoders("EnglishEncoderID"))
	GzipEncoderID    = uint64(SetupDefaultEncoders("GzipEncoderID"))
	HexEncoderID     = uint64(SetupDefaultEncoders("HexEncoderID"))
	PNGEncoderID     = uint64(SetupDefaultEncoders("PNGEncoderID"))
	NoEncoderID      = uint64(0)
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

func SetupDefaultEncoders(name string) uint64 {

	encoders, err := db.ResourceIDByType("encoder")
	if err != nil {
		log.Printf("Error:\n%s", err)
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
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	return id
}

// generate unavailable id array on startup
func populateID() []uint64 {
	// remove already used prime numbers from available pool
	resourceIDs, err := db.ResourceIDs()
	if err != nil {
		log.Printf("Error:\n%s", err)
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
