// +build go1.10

package screenshot


import (
	"errors"
	"image"
)

func Capture(x, y, width, height int) (*image.RGBA, error) {

		return nil, errors.New("Not Implemented")
}



func NumActiveDisplays() int {
		return 0
}


func GetDisplayBounds(displayIndex int) image.Rectangle {
	return image.Rectangle{}
}
