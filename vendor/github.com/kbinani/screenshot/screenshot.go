// Package screenshot captures screen-shot image as image.RGBA.
// Mac, Windows, Linux, FreeBSD, OpenBSD, NetBSD, and Solaris are supported.
package screenshot

import (
	"image"
)

// CaptureDisplay captures whole region of displayIndex'th display.
func CaptureDisplay(displayIndex int) (*image.RGBA, error) {
	rect := GetDisplayBounds(displayIndex)
	return CaptureRect(rect)
}

// CaptureRect captures specified region of desktop.
func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	return Capture(rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy())
}
