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
	insecureRand "math/rand"
	"testing"

	"github.com/bishopfox/sliver/util/encoders"
)

//go:embed base64.wasm
var base64WASM []byte

func TestTrafficEncoder_base64_basic(t *testing.T) {
	encoder, err := encoders.CreateTrafficEncoder("base64", base64WASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
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

func TestTrafficEncoder_base64_random(t *testing.T) {
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
