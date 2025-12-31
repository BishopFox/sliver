package catppuccingo

// Mocha variant
type mocha struct{}

// Mocha flavor variant
var Mocha Flavor = mocha{}

// Mocha
func (mocha) Name() string { return "mocha" }

// Rosewater
func (mocha) Rosewater() Color {
	return Color{
		Hex: "#f5e0dc",
		RGB: [3]uint8{245, 224, 220},
		HSL: [3]float32{10, 0.56, 0.91},
	}
}

// Flamingo
func (mocha) Flamingo() Color {
	return Color{
		Hex: "#f2cdcd",
		RGB: [3]uint8{242, 205, 205},
		HSL: [3]float32{0, 0.59, 0.88},
	}
}

// Pink
func (mocha) Pink() Color {
	return Color{
		Hex: "#f5c2e7",
		RGB: [3]uint8{245, 194, 231},
		HSL: [3]float32{316, 0.72, 0.86},
	}
}

// Mauve
func (mocha) Mauve() Color {
	return Color{
		Hex: "#cba6f7",
		RGB: [3]uint8{203, 166, 247},
		HSL: [3]float32{267, 0.84, 0.81},
	}
}

// Red
func (mocha) Red() Color {
	return Color{
		Hex: "#f38ba8",
		RGB: [3]uint8{243, 139, 168},
		HSL: [3]float32{343, 0.81, 0.75},
	}
}

// Maroon
func (mocha) Maroon() Color {
	return Color{
		Hex: "#eba0ac",
		RGB: [3]uint8{235, 160, 172},
		HSL: [3]float32{350, 0.65, 0.77},
	}
}

// Peach
func (mocha) Peach() Color {
	return Color{
		Hex: "#fab387",
		RGB: [3]uint8{250, 179, 135},
		HSL: [3]float32{23, 0.92, 0.75},
	}
}

// Yellow
func (mocha) Yellow() Color {
	return Color{
		Hex: "#f9e2af",
		RGB: [3]uint8{249, 226, 175},
		HSL: [3]float32{41, 0.86, 0.83},
	}
}

// Green
func (mocha) Green() Color {
	return Color{
		Hex: "#a6e3a1",
		RGB: [3]uint8{166, 227, 161},
		HSL: [3]float32{115, 0.54, 0.76},
	}
}

// Teal
func (mocha) Teal() Color {
	return Color{
		Hex: "#94e2d5",
		RGB: [3]uint8{148, 226, 213},
		HSL: [3]float32{170, 0.57, 0.73},
	}
}

// Sky
func (mocha) Sky() Color {
	return Color{
		Hex: "#89dceb",
		RGB: [3]uint8{137, 220, 235},
		HSL: [3]float32{189, 0.71, 0.73},
	}
}

// Sapphire
func (mocha) Sapphire() Color {
	return Color{
		Hex: "#74c7ec",
		RGB: [3]uint8{116, 199, 236},
		HSL: [3]float32{199, 0.76, 0.69},
	}
}

// Blue
func (mocha) Blue() Color {
	return Color{
		Hex: "#89b4fa",
		RGB: [3]uint8{137, 180, 250},
		HSL: [3]float32{217, 0.92, 0.76},
	}
}

// Lavender
func (mocha) Lavender() Color {
	return Color{
		Hex: "#b4befe",
		RGB: [3]uint8{180, 190, 254},
		HSL: [3]float32{232, 0.97, 0.85},
	}
}

// Text
func (mocha) Text() Color {
	return Color{
		Hex: "#cdd6f4",
		RGB: [3]uint8{205, 214, 244},
		HSL: [3]float32{226, 0.64, 0.88},
	}
}

// Subtext 1
func (mocha) Subtext1() Color {
	return Color{
		Hex: "#bac2de",
		RGB: [3]uint8{186, 194, 222},
		HSL: [3]float32{227, 0.35, 0.8},
	}
}

// Subtext 0
func (mocha) Subtext0() Color {
	return Color{
		Hex: "#a6adc8",
		RGB: [3]uint8{166, 173, 200},
		HSL: [3]float32{228, 0.24, 0.72},
	}
}

// Overlay 2
func (mocha) Overlay2() Color {
	return Color{
		Hex: "#9399b2",
		RGB: [3]uint8{147, 153, 178},
		HSL: [3]float32{228, 0.17, 0.64},
	}
}

// Overlay 1
func (mocha) Overlay1() Color {
	return Color{
		Hex: "#7f849c",
		RGB: [3]uint8{127, 132, 156},
		HSL: [3]float32{230, 0.13, 0.55},
	}
}

// Overlay 0
func (mocha) Overlay0() Color {
	return Color{
		Hex: "#6c7086",
		RGB: [3]uint8{108, 112, 134},
		HSL: [3]float32{231, 0.11, 0.47},
	}
}

// Surface 2
func (mocha) Surface2() Color {
	return Color{
		Hex: "#585b70",
		RGB: [3]uint8{88, 91, 112},
		HSL: [3]float32{233, 0.12, 0.39},
	}
}

// Surface 1
func (mocha) Surface1() Color {
	return Color{
		Hex: "#45475a",
		RGB: [3]uint8{69, 71, 90},
		HSL: [3]float32{234, 0.13, 0.31},
	}
}

// Surface 0
func (mocha) Surface0() Color {
	return Color{
		Hex: "#313244",
		RGB: [3]uint8{49, 50, 68},
		HSL: [3]float32{237, 0.16, 0.23},
	}
}

// Base
func (mocha) Base() Color {
	return Color{
		Hex: "#1e1e2e",
		RGB: [3]uint8{30, 30, 46},
		HSL: [3]float32{240, 0.21, 0.15},
	}
}

// Mantle
func (mocha) Mantle() Color {
	return Color{
		Hex: "#181825",
		RGB: [3]uint8{24, 24, 37},
		HSL: [3]float32{240, 0.21, 0.12},
	}
}

// Crust
func (mocha) Crust() Color {
	return Color{
		Hex: "#11111b",
		RGB: [3]uint8{17, 17, 27},
		HSL: [3]float32{240, 0.23, 0.09},
	}
}
