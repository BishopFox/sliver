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
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
)

const (
	// The Alpha channel is not used, as any values other
	// than 255 (no transparency) will cause the RGB value blending,
	// resulting in modifications to RGB values to compensate. For our
	// use case (lossless data), we cannot use the alpha channel.
	immutableAlpha = 255

	// we can shove three bytes into each pixel: R, G, and B.
	bytesPerPixel = 3

	// DefaultMaxPNGDecodeLen is the maximum amount of pixel data Decode will
	// process. Callers handling untrusted data should use DecodeWithMaxLen with
	// a limit appropriate for the request they are processing.
	DefaultMaxPNGDecodeLen = 2 * 1024 * 1024 * 1024 // 2GB
)

// ErrPNGTooLarge is returned when a PNG's dimensions or decoded payload exceed
// the decode limit.
var ErrPNGTooLarge = errors.New("png decoded payload exceeds maximum size")

// PNGEncoder - PNG image object
type PNGEncoder struct{}

var _ LimitedDecoder = PNGEncoder{}

// Encode outputs a valid PNG file
func (p PNGEncoder) Encode(data []byte) ([]byte, error) {
	img := imageFromBytes(data)
	encoder := &png.Encoder{
		CompressionLevel: png.NoCompression,
	}
	var buf bytes.Buffer
	encoder.Encode(&buf, img)
	return buf.Bytes(), nil
}

// Decode reads a encoded PNG to get the original binary data
func (p PNGEncoder) Decode(data []byte) ([]byte, error) {
	return p.DecodeWithMaxLen(data, DefaultMaxPNGDecodeLen)
}

// DecodeWithMaxLen reads an encoded PNG while limiting both the returned
// payload and the three-byte-per-pixel representation used during decoding.
// Since padding and escape sequences are removed after extracting the pixels,
// this may conservatively reject an image whose final payload would fit.
func (p PNGEncoder) DecodeWithMaxLen(data []byte, maxLen int64) ([]byte, error) {
	if maxLen < 0 {
		return nil, fmt.Errorf("invalid max decode length: %d", maxLen)
	}

	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := validatePNGDimensions(config.Width, config.Height, maxLen); err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	decoded := bytesFromImage(img)
	if int64(len(decoded)) > maxLen {
		return nil, fmt.Errorf("%w: decoded payload is %d bytes, limit is %d", ErrPNGTooLarge, len(decoded), maxLen)
	}
	return decoded, nil
}

func validatePNGDimensions(width int, height int, maxLen int64) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid PNG dimensions: %dx%d", width, height)
	}

	// bytesFromImage materializes three bytes for every pixel before removing
	// padding and escape sequences. Bound that intermediate representation
	// before png.Decode allocates storage based on attacker-controlled dimensions.
	maxPixels := maxLen / int64(bytesPerPixel)
	if int64(width) > maxPixels/int64(height) {
		return fmt.Errorf("%w: %dx%d image exceeds %d bytes", ErrPNGTooLarge, width, height, maxLen)
	}
	return nil
}

// imageFromBytes returns a valid image with data encoded in each pixel
func imageFromBytes(data []byte) image.Image {

	// The data cannot contain null bytes in order to be valid, so
	// we escape 0x0 and 0x1 as such:
	data = bytes.Replace(data, []byte{0x1}, []byte{0x1, 0x1, 0x1}, -1)
	data = bytes.Replace(data, []byte{0x0}, []byte{0x1, 0x0, 0x1}, -1)

	nearestSquareRoot := math.Sqrt(float64(len(data)/bytesPerPixel)) + 1 // rounding up
	width := int(nearestSquareRoot)
	height := int(nearestSquareRoot)
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	b := bytes.NewBuffer(data)
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			w := b.Next(bytesPerPixel)
			if len(w) < bytesPerPixel {
				padding := []byte(strings.Repeat(string(rune(0)), bytesPerPixel))
				w = append(w, padding...)
			}
			img.Set(x, y, color.NRGBA{
				// Three bytes per pixel, informing bytesPerPixel.
				R: w[0],
				G: w[1],
				B: w[2],
				A: immutableAlpha,
			})
		}
	}
	return img
}

func bytesFromImage(img image.Image) []byte {
	data := new(bytes.Buffer)
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			r, g, b, _ := img.At(x, y).RGBA()
			data.WriteByte(byte(r))
			data.WriteByte(byte(g))
			data.WriteByte(byte(b))
		}
	}

	buf := bytes.Trim(data.Bytes(), "\x00") // May still contain escaped null bytes

	// Unescape null bytes
	buf = bytes.Replace(buf, []byte{0x1, 0x1, 0x1}, []byte{0x1}, -1)
	buf = bytes.Replace(buf, []byte{0x1, 0x0, 0x1}, []byte{0x0}, -1)

	return buf // lopping off null padding
}
