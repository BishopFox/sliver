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
	"encoding/base64"
	insecureRand "math/rand"
	"testing"
	"time"

	"github.com/bishopfox/sliver/util/encoders"
)

//go:embed base64.wasm
var base64WASM []byte

func TestTrafficEncoder_base64Basic(t *testing.T) {
	encoder, err := encoders.CreateTrafficEncoder("base64", base64WASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	originalValue := []byte("hello world")
	encodedValue, err := encoder.Encode(originalValue)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Encoded value: %s", string(encodedValue))
	decodedValue, err := encoder.Decode(encodedValue)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalValue, decodedValue) {
		t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
	}
}

func TestTrafficEncoder_base64RandomSmall(t *testing.T) {
	encoder, err := encoders.CreateTrafficEncoder("base64", base64WASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	for i := 0; i < 1000; i++ {
		originalValue := make([]byte, insecureRand.Intn(1024)+1)
		rand.Read(originalValue)
		encodedValue, err := encoder.Encode(originalValue)
		if err != nil {
			t.Fatal(err)
		}
		// t.Logf("Encoded value: %s", string(encodedValue))
		decodedValue, err := encoder.Decode(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}
}

func TestTrafficEncoder_base64RandomLarge(t *testing.T) {
	encoder, err := encoders.CreateTrafficEncoder("base64", base64WASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	originalValue := make([]byte, 2*1024*1024)
	rand.Read(originalValue)
	encodedValue, err := encoder.Encode(originalValue)
	if err != nil {
		t.Fatal(err)
	}
	decodedValue, err := encoder.Decode(encodedValue)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(originalValue, decodedValue) {
		t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
	}
}

func TestPerformance(t *testing.T) {

	sizes := []int{1024, 1024 * 1024, 2 * 1024 * 1024, 4 * 1024 * 1024}

	// Stock encoder
	for i := 0; i < len(sizes); i++ {
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		stock := time.Now()
		encodedValue := base64.StdEncoding.EncodeToString(originalValue)
		decodedValue, err := base64.StdEncoding.DecodeString(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Stock encoder took %v (%d bytes)", time.Since(stock), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}

	// Traffic encoder
	encoder, err := encoders.CreateTrafficEncoder("base64", base64WASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()

	for i := 0; i < len(sizes); i++ {
		originalValue := make([]byte, sizes[i])
		rand.Read(originalValue)
		start := time.Now()
		encodedValue, err := encoder.Encode(originalValue)
		if err != nil {
			t.Fatal(err)
		}
		decodedValue, err := encoder.Decode(encodedValue)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Traffic encoder took %v (%d bytes)", time.Since(start), sizes[i])
		if !bytes.Equal(originalValue, decodedValue) {
			t.Fatalf("Expected %v but got %v", originalValue, decodedValue)
		}
	}
}
