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

	// PNGEncoderID - Encoder ID
	PNGEncoderID = 22
)

// PNGEncoder - PNG image object
type PNGEncoder struct{}

// Encode outputs a valid PNG file
func (p PNGEncoder) Encode(data []byte) []byte {
	payload := new(Hex).Encode(data)
	img := imageFromBytes(payload)
	encoder := &png.Encoder{
		CompressionLevel: png.NoCompression,
	}
	var buf bytes.Buffer
	encoder.Encode(&buf, img)

	return buf.Bytes()
}

// Decode reads a encoded PNG to get the original binary data
func (p PNGEncoder) Decode(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	img, err := png.Decode(buf)
	if err != nil {
		return nil, err
	}
	data = bytesFromImage(img)
	return new(Hex).Decode(data)
}

// imageFromBytes returns a valid image with data encoded in each pixel
func imageFromBytes(data []byte) image.Image {
	// lop off prefix and suffix nulls
	data = bytes.Trim(data, "\x00")
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
	return bytes.Trim(data.Bytes(), "\x00") // lopping off null padding
}
