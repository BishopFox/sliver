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

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/util"
)

const (

	// EncoderModulus - The modulus used to calculate the encoder ID from a C2 request nonce
	// *** IMPORTANT *** ENCODER IDs MUST BE LESS THAN THE MODULUS
	EncoderModulus = uint64(65537)
	MaxN           = uint64(9999999)
)

var (
	// These were chosen at random other than the "No Encoder" ID (0)
	PrimeNumbers     = generateDefaultPrimeNumbers()
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

	prime := GetPrimeNumber()
	err = db.SaveResourceID(&clientpb.ResourceID{
		Type:  "encoder",
		Name:  name,
		Value: prime,
	})
	if err != nil {
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}

	return prime
}

func generateDefaultPrimeNumbers() []uint64 {
	// remove already used prime numbers from available pool
	resourceIDs, err := db.ResourceIDs()
	if err != nil {
		log.Printf("Error:\n%s", err)
		os.Exit(-1)
	}
	pool := util.DefaultPrimeNumbers
	for _, resourceID := range resourceIDs {
		pool = util.RemoveElement(pool, resourceID.Value)
	}

	return pool
}

func GetPrimeNumber() uint64 {
	prime := PrimeNumbers[insecureRand.Intn(len(PrimeNumbers))]
	PrimeNumbers = util.RemoveElement(PrimeNumbers, prime)
	return prime
}
