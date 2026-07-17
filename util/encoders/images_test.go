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

var (
	imageTests = []struct {
		Input []byte
	}{
		{[]byte("abc")},   // byte count on image pixel alignment
		{[]byte("abcde")}, // byte count offset of image pixel alignment
		{[]byte{0x0, 0x01, 0x02, 0x03, 0x04}},
		{[]byte{0x01, 0x02, 0x03, 0x04, 0x0}},
	}
)

func TestPNG(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for _, test := range imageTests {
		buf, _ := pngEncoder.Encode(test.Input)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(test.Input, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, test.Input)
		}
	}
}

func TestPNGRandomDataRandomSize(t *testing.T) {
	pngEncoder := new(PNGEncoder)
	for i := 0; i < 100; i++ {
		sample := randomDataRandomSize(1024 * 1024)
		buf, _ := pngEncoder.Encode(sample)
		decodeOutput, err := pngEncoder.Decode(buf)
		if err != nil {
			t.Errorf("png decode returned error: %q", err)
		}
		if !bytes.Equal(sample, decodeOutput) {
			t.Errorf("png Decode(img) => %q, expected %q", decodeOutput, sample)
		}
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
