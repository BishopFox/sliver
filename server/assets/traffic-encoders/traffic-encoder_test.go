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
	_ "embed"
	"encoding/hex"
	"fmt"
	"testing"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders/traffic"
	serverEncoders "github.com/bishopfox/sliver/util/encoders/traffic"
)

//go:embed hex.wasm
var hexWASM []byte

func TestTrafficEncoderCompatibility_hex(t *testing.T) {
	implantSideHex, err := implantEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := implantSideHex.Close(); err != nil {
			t.Errorf("close implant encoder: %v", err)
		}
	})
	serverSideHex, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(msg string) {
		t.Log(msg)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := serverSideHex.Close(); err != nil {
			t.Errorf("close server encoder: %v", err)
		}
	})

	directions := []struct {
		name   string
		encode func([]byte) ([]byte, error)
		decode func([]byte) ([]byte, error)
	}{
		{"implant_to_server", implantSideHex.Encode, serverSideHex.Decode},
		{"server_to_implant", serverSideHex.Encode, implantSideHex.Decode},
	}
	for _, direction := range directions {
		t.Run(direction.name, func(t *testing.T) {
			// Exercise empty and binary inputs, both sides of a 64 KiB WASM
			// page boundary, and the largest payload from the old timing test.
			for _, size := range []int{0, 1, 256, 65535, 65536, 65537, 4 * 1024 * 1024} {
				t.Run(fmt.Sprintf("bytes_%d", size), func(t *testing.T) {
					payload := hexTestPayload(size)
					encoded, err := direction.encode(payload)
					if err != nil {
						t.Fatalf("encode: %v", err)
					}
					if !bytes.Equal(encoded, []byte(hex.EncodeToString(payload))) {
						t.Fatal("WASM output does not match standard hex encoding")
					}
					decoded, err := direction.decode(encoded)
					if err != nil {
						t.Fatalf("decode: %v", err)
					}
					if !bytes.Equal(payload, decoded) {
						t.Fatalf("round trip changed the %d-byte payload (decoded %d bytes)", len(payload), len(decoded))
					}
				})
			}
		})
	}
}

func BenchmarkHex(b *testing.B) {
	for _, size := range []int{1024, 1024 * 1024, 2 * 1024 * 1024, 4 * 1024 * 1024} {
		payload := hexTestPayload(size)
		b.Run(fmt.Sprintf("bytes_%d", size), func(b *testing.B) {
			b.Run("standard_round_trip", func(b *testing.B) {
				b.SetBytes(int64(size))
				b.ReportAllocs()
				for b.Loop() {
					encoded := hex.EncodeToString(payload)
					decoded, err := hex.DecodeString(encoded)
					if err != nil {
						b.Fatal(err)
					}
					if !bytes.Equal(payload, decoded) {
						b.Fatal("decoded data does not match original")
					}
				}
			})
			b.Run("wasm_fresh_runtime_round_trip", func(b *testing.B) {
				b.SetBytes(int64(size))
				b.ReportAllocs()
				// Include runtime creation and close in each iteration. The WASM
				// API retains output buffers until the runtime is closed, so an
				// unbounded loop over one runtime would accumulate guest memory.
				for b.Loop() {
					benchmarkHexWASMLifecycle(b, payload)
				}
			})
		})
	}
}

func benchmarkHexWASMLifecycle(b *testing.B, payload []byte) {
	b.Helper()
	encoder, err := serverEncoders.CreateTrafficEncoder("hex", hexWASM, func(string) {})
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := encoder.Close(); err != nil {
			b.Errorf("close encoder: %v", err)
		}
	}()
	encoded, err := encoder.Encode(payload)
	if err != nil {
		b.Fatal(err)
	}
	decoded, err := encoder.Decode(encoded)
	if err != nil {
		b.Fatal(err)
	}
	if !bytes.Equal(payload, decoded) {
		b.Fatal("decoded data does not match original")
	}
}

func hexTestPayload(size int) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(i)
	}
	return payload
}
