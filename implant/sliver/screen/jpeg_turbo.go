// +build jpegturbo
//go:build jpegturbo
// +build jpegturbo

package screen

import (
	"image"
	"io"

	jpegturbo "github.com/pixiv/go-libjpeg/jpeg"
)

func jpegQuality(q int) *jpegturbo.EncoderOptions {
	return &jpegturbo.EncoderOptions{Quality: q}
}

func encodeJpeg(w io.Writer, src image.Image, opts *jpegturbo.EncoderOptions) {
	jpegturbo.Encode(w, src, opts)
}
