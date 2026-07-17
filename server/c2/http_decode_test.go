package c2

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"image"
	"image/png"
	"testing"

	sliverEncoders "github.com/bishopfox/sliver/util/encoders"
)

func TestDecodeReqBodyWithMaxLenRejectsOversizedPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1700, 1700))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}

	_, err := decodeReqBodyWithMaxLen(sliverEncoders.PNGEncoder{}, encoded.Bytes(), DefaultMaxUnauthBodyLength)
	if !errors.Is(err, sliverEncoders.ErrPNGTooLarge) {
		t.Fatalf("expected ErrPNGTooLarge, got %v", err)
	}
}
