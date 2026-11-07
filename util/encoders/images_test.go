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
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
)

func TestPNG(t *testing.T) {
	allBytes := make([]byte, 256)
	for i := range allBytes {
		allBytes[i] = byte(i)
	}
	tests := []struct {
		name    string
		payload []byte
	}{
		{"empty", nil},
		{"one_zero", []byte{0}},
		{"one_escape", []byte{1}},
		{"partial_pixel", []byte{2, 3}},
		{"pixel_aligned", []byte("abc")},
		{"pixel_overflow", []byte("abcde")},
		{"before_square_boundary", bytes.Repeat([]byte{2}, 11)},
		{"at_square_boundary", bytes.Repeat([]byte{2}, 12)},
		{"after_square_boundary", bytes.Repeat([]byte{2}, 13)},
		{"leading_zero", []byte{0, 1, 2, 3, 4}},
		{"trailing_zero", []byte{1, 2, 3, 4, 0}},
		{"adjacent_escapes", []byte{0, 1, 0, 1, 1, 0}},
		{"all_bytes", allBytes},
		{"one_megabyte", bytes.Repeat(allBytes, 4096)},
	}
	pngEncoder := new(PNGEncoder)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := pngEncoder.Encode(test.payload)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			decoded, err := pngEncoder.Decode(encoded)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !bytes.Equal(test.payload, decoded) {
				t.Fatalf("round trip changed the %d-byte payload (decoded %d bytes)", len(test.payload), len(decoded))
			}
		})
	}
}

func TestPNGDecodeWithMaxLen(t *testing.T) {
	const maxLen = 4 * bytesPerPixel
	payload := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	pngEncoder := new(PNGEncoder)
	encoded, err := pngEncoder.Encode(payload)
	if err != nil {
		t.Fatalf("png encode failed: %v", err)
	}

	decoded, err := pngEncoder.DecodeWithMaxLen(encoded, maxLen)
	if err != nil {
		t.Fatalf("png decode failed: %v", err)
	}
	if !bytes.Equal(payload, decoded) {
		t.Fatalf("decoded payload does not match original: %q != %q", decoded, payload)
	}
}

func TestPNGDecodeWithMaxLenRejectsOversizedPixelData(t *testing.T) {
	payload := bytes.Repeat([]byte("A"), 8192)
	pngEncoder := new(PNGEncoder)
	encoded, err := pngEncoder.Encode(payload)
	if err != nil {
		t.Fatalf("png encode failed: %v", err)
	}

	_, err = pngEncoder.DecodeWithMaxLen(encoded, int64(len(payload)-1))
	if !errors.Is(err, ErrPNGTooLarge) {
		t.Fatalf("expected ErrPNGTooLarge, got %v", err)
	}
}

func TestPNGDecodeWithMaxLenRejectsOversizedDimensionsBeforeDecode(t *testing.T) {
	const maxLen = 8 * 1024 * 1024
	pngEncoder := new(PNGEncoder)
	encoded, err := pngEncoder.Encode([]byte("abc"))
	if err != nil {
		t.Fatalf("png encode failed: %v", err)
	}

	// A PNG starts with an eight-byte signature followed by its IHDR chunk.
	// Rewrite only the CRC-valid header and omit IDAT so this test remains safe
	// even if the preflight check regresses and png.Decode is called directly.
	const ihdrEnd = 33
	if len(encoded) < ihdrEnd || string(encoded[12:16]) != "IHDR" {
		t.Fatal("encoded PNG does not contain the expected IHDR chunk")
	}
	encoded = encoded[:ihdrEnd]
	binary.BigEndian.PutUint32(encoded[16:20], 4096)
	binary.BigEndian.PutUint32(encoded[20:24], 4096)
	binary.BigEndian.PutUint32(encoded[29:33], crc32.ChecksumIEEE(encoded[12:29]))

	_, err = pngEncoder.DecodeWithMaxLen(encoded, maxLen)
	if !errors.Is(err, ErrPNGTooLarge) {
		t.Fatalf("expected ErrPNGTooLarge, got %v", err)
	}
}

func TestValidatePNGDimensionsRejectsReportedBomb(t *testing.T) {
	err := validatePNGDimensions(32767, 32767, 8*1024*1024)
	if !errors.Is(err, ErrPNGTooLarge) {
		t.Fatalf("expected ErrPNGTooLarge, got %v", err)
	}
}

func TestValidatePNGDimensionsPixelDataBoundary(t *testing.T) {
	if err := validatePNGDimensions(1, 1, bytesPerPixel); err != nil {
		t.Fatalf("expected one pixel to fit: %v", err)
	}
	if err := validatePNGDimensions(1, 1, bytesPerPixel-1); !errors.Is(err, ErrPNGTooLarge) {
		t.Fatalf("expected ErrPNGTooLarge, got %v", err)
	}
}
