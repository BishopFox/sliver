package traffic_encoders

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
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"testing"
	"time"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders/traffic"
	serverEncoders "github.com/bishopfox/sliver/util/encoders/traffic"
)

//go:embed hex.wasm
var hexWASM []byte

func TestTrafficEncoderCompatibility_hex(t *testing.T) {

	// Hex

	implantSideHex, err := implantEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	serverSideHex, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 1024)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	encodedData, err := implantSideHex.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	decodedData, err := serverSideHex.Decode(encodedData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data does not match original")
	}

	data = make([]byte, 1024)
	_, err = rand.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	encodedData, err = serverSideHex.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	decodedData, err = implantSideHex.Decode(encodedData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decodedData) {
		t.Fatal("Decoded data does not match original")
	}
}

// Encoder specific tests
func TestHexPerformance(t *testing.T) {

	sizes := []int{1024, 1024 * 1024, 2 * 1024 * 1024, 4 * 1024 * 1024}

	// Stock encoder
	for i := 0; i < len(sizes); i++ {
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		stock := time.Now()
		encodedValue := hex.EncodeToString(originalValue)
		decodedValue, err := hex.DecodeString(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Stock encoder took %v (%d bytes)", time.Since(stock), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}

	// Traffic encoder
	for i := 0; i < len(sizes); i++ {
		encoder, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
			t.Log(msg)
		})
		if err != nil {
			t.Fatal(err)
		}
		defer encoder.Close()
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		start := time.Now()
		encodedValue, err := encoder.Encode(originalValue)
		if err != nil {
			t.Fatal(err)
		}
		// t.Logf("Got encoded value (%d bytes)", len(encodedValue))
		decodedValue, err := encoder.Decode(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("WASM Hex encoder took %v (%d bytes)", time.Since(start), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}
}
