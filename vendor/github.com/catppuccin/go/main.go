//go:generate whiskers go.tera
//go:generate go fmt ./...
package catppuccingo

import (
	"image/color"
	"strings"
)

// Flavor is an interface implemented by all Catppuccin variations.
type Flavor interface {
	Rosewater() Color
	Flamingo() Color
	Pink() Color
	Mauve() Color
	Red() Color
	Maroon() Color
	Peach() Color
	Yellow() Color
	Green() Color
	Teal() Color
	Sky() Color
	Sapphire() Color
	Blue() Color
	Lavender() Color
	Text() Color
	Subtext1() Color
	Subtext0() Color
	Overlay2() Color
	Overlay1() Color
	Overlay0() Color
	Surface2() Color
	Surface1() Color
	Surface0() Color
	Crust() Color
	Mantle() Color
	Base() Color
	Name() string
}

// Theme and Flavour are type aliases of Flavor to keep compatibility with previous versions.
type (
	Theme   = Flavour
	Flavour = Flavor
)

// Color is a color in Hex, RGB, and HSL.
type Color struct {
	Hex string
	RGB [3]uint8
	HSL [3]float32
}

// RGBA implements color.Color
func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(c.RGB[0])
	r |= r << 8
	g = uint32(c.RGB[1])
	g |= g << 8
	b = uint32(c.RGB[2])
	b |= b << 8
	a = 0xffff
	return
}

var _ color.Color = Color{}

// Variant returns the Theme variant by name.
func Variant(flavor string) Theme {
	for _, t := range []Theme{
		Mocha,
		Frappe,
		Macchiato,
		Latte,
	} {
		if strings.EqualFold(t.Name(), flavor) {
			return t
		}
	}
	return nil
}
