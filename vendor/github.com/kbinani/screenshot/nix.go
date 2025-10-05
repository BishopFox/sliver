//go:build !s390x && !ppc64le && !darwin && !windows && (linux || freebsd || openbsd || netbsd)

package screenshot

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xinerama"
	"image"
)

// NumActiveDisplays returns the number of active displays.
func NumActiveDisplays() (num int) {
	defer func() {
		e := recover()
		if e != nil {
			num = 0
		}
	}()

	c, err := xgb.NewConn()
	if err != nil {
		return 0
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return 0
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return 0
	}

	num = int(reply.Number)
	return num
}

// GetDisplayBounds returns the bounds of displayIndex'th display.
// The main display is displayIndex = 0.
func GetDisplayBounds(displayIndex int) (rect image.Rectangle) {
	defer func() {
		e := recover()
		if e != nil {
			rect = image.Rectangle{}
		}
	}()

	c, err := xgb.NewConn()
	if err != nil {
		return image.Rectangle{}
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return image.Rectangle{}
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return image.Rectangle{}
	}

	if displayIndex >= int(reply.Number) {
		return image.Rectangle{}
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)

	screen := reply.ScreenInfo[displayIndex]
	x := int(screen.XOrg) - x0
	y := int(screen.YOrg) - y0
	w := int(screen.Width)
	h := int(screen.Height)
	rect = image.Rect(x, y, x+w, y+h)
	return rect
}
