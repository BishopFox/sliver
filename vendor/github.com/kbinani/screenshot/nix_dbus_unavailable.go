//go:build freebsd

package screenshot

import (
	"image"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of primary display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(x, y, width, height int) (img *image.RGBA, e error) {
	return captureXinerama(x, y, width, height)
}
