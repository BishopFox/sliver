package screenshot

import (
	"image"

	"github.com/kbinani/screenshot/internal/xwindow"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of main display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(x, y, width, height int) (*image.RGBA, error) {
	return xwindow.Capture(x, y, width, height)
}

// NumActiveDisplays returns the number of active displays.
func NumActiveDisplays() int {
	return xwindow.NumActiveDisplays()
}

// GetDisplayBounds returns the bounds of displayIndex'th display.
// The main display is displayIndex = 0.
func GetDisplayBounds(displayIndex int) image.Rectangle {
	return xwindow.GetDisplayBounds(displayIndex)
}
