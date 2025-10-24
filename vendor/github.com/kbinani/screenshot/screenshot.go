// Package screenshot captures screen-shot image as image.RGBA.
// Mac, Windows, Linux, FreeBSD, OpenBSD and NetBSD are supported.
package screenshot

import (
	"errors"
	"image"
)

// ErrUnsupported is returned when the platform or architecture used to compile the program
// does not support screenshot, e.g. if you're compiling without CGO on Darwin
var ErrUnsupported = errors.New("screenshot does not support your platform")

// CaptureDisplay captures whole region of displayIndex'th display, starts at 0 for primary display.
func CaptureDisplay(displayIndex int) (*image.RGBA, error) {
	rect := GetDisplayBounds(displayIndex)
	return CaptureRect(rect)
}

// CaptureRect captures specified region of desktop.
func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	return Capture(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
}

func createImage(rect image.Rectangle) (img *image.RGBA, e error) {
	img = nil
	e = errors.New("Cannot create image.RGBA")

	defer func() {
		err := recover()
		if err == nil {
			e = nil
		}
	}()
	// image.NewRGBA may panic if rect is too large.
	img = image.NewRGBA(rect)

	return img, e
}
