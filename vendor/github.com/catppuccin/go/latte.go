package catppuccingo

// Latte variant
type latte struct{}

// Latte flavor variant
var Latte Flavor = latte{}

// Latte
func (latte) Name() string { return "latte" }

// Rosewater
func (latte) Rosewater() Color {
	return Color{
		Hex: "#dc8a78",
		RGB: [3]uint8{220, 138, 120},
		HSL: [3]float32{11, 0.59, 0.67},
	}
}

// Flamingo
func (latte) Flamingo() Color {
	return Color{
		Hex: "#dd7878",
		RGB: [3]uint8{221, 120, 120},
		HSL: [3]float32{0, 0.6, 0.67},
	}
}

// Pink
func (latte) Pink() Color {
	return Color{
		Hex: "#ea76cb",
		RGB: [3]uint8{234, 118, 203},
		HSL: [3]float32{316, 0.73, 0.69},
	}
}

// Mauve
func (latte) Mauve() Color {
	return Color{
		Hex: "#8839ef",
		RGB: [3]uint8{136, 57, 239},
		HSL: [3]float32{266, 0.85, 0.58},
	}
}

// Red
func (latte) Red() Color {
	return Color{
		Hex: "#d20f39",
		RGB: [3]uint8{210, 15, 57},
		HSL: [3]float32{347, 0.87, 0.44},
	}
}

// Maroon
func (latte) Maroon() Color {
	return Color{
		Hex: "#e64553",
		RGB: [3]uint8{230, 69, 83},
		HSL: [3]float32{355, 0.76, 0.59},
	}
}

// Peach
func (latte) Peach() Color {
	return Color{
		Hex: "#fe640b",
		RGB: [3]uint8{254, 100, 11},
		HSL: [3]float32{22, 0.99, 0.52},
	}
}

// Yellow
func (latte) Yellow() Color {
	return Color{
		Hex: "#df8e1d",
		RGB: [3]uint8{223, 142, 29},
		HSL: [3]float32{35, 0.77, 0.49},
	}
}

// Green
func (latte) Green() Color {
	return Color{
		Hex: "#40a02b",
		RGB: [3]uint8{64, 160, 43},
		HSL: [3]float32{109, 0.58, 0.4},
	}
}

// Teal
func (latte) Teal() Color {
	return Color{
		Hex: "#179299",
		RGB: [3]uint8{23, 146, 153},
		HSL: [3]float32{183, 0.74, 0.35},
	}
}

// Sky
func (latte) Sky() Color {
	return Color{
		Hex: "#04a5e5",
		RGB: [3]uint8{4, 165, 229},
		HSL: [3]float32{197, 0.97, 0.46},
	}
}

// Sapphire
func (latte) Sapphire() Color {
	return Color{
		Hex: "#209fb5",
		RGB: [3]uint8{32, 159, 181},
		HSL: [3]float32{189, 0.7, 0.42},
	}
}

// Blue
func (latte) Blue() Color {
	return Color{
		Hex: "#1e66f5",
		RGB: [3]uint8{30, 102, 245},
		HSL: [3]float32{220, 0.91, 0.54},
	}
}

// Lavender
func (latte) Lavender() Color {
	return Color{
		Hex: "#7287fd",
		RGB: [3]uint8{114, 135, 253},
		HSL: [3]float32{231, 0.97, 0.72},
	}
}

// Text
func (latte) Text() Color {
	return Color{
		Hex: "#4c4f69",
		RGB: [3]uint8{76, 79, 105},
		HSL: [3]float32{234, 0.16, 0.35},
	}
}

// Subtext 1
func (latte) Subtext1() Color {
	return Color{
		Hex: "#5c5f77",
		RGB: [3]uint8{92, 95, 119},
		HSL: [3]float32{233, 0.13, 0.41},
	}
}

// Subtext 0
func (latte) Subtext0() Color {
	return Color{
		Hex: "#6c6f85",
		RGB: [3]uint8{108, 111, 133},
		HSL: [3]float32{233, 0.1, 0.47},
	}
}

// Overlay 2
func (latte) Overlay2() Color {
	return Color{
		Hex: "#7c7f93",
		RGB: [3]uint8{124, 127, 147},
		HSL: [3]float32{232, 0.1, 0.53},
	}
}

// Overlay 1
func (latte) Overlay1() Color {
	return Color{
		Hex: "#8c8fa1",
		RGB: [3]uint8{140, 143, 161},
		HSL: [3]float32{231, 0.1, 0.59},
	}
}

// Overlay 0
func (latte) Overlay0() Color {
	return Color{
		Hex: "#9ca0b0",
		RGB: [3]uint8{156, 160, 176},
		HSL: [3]float32{228, 0.11, 0.65},
	}
}

// Surface 2
func (latte) Surface2() Color {
	return Color{
		Hex: "#acb0be",
		RGB: [3]uint8{172, 176, 190},
		HSL: [3]float32{227, 0.12, 0.71},
	}
}

// Surface 1
func (latte) Surface1() Color {
	return Color{
		Hex: "#bcc0cc",
		RGB: [3]uint8{188, 192, 204},
		HSL: [3]float32{225, 0.14, 0.77},
	}
}

// Surface 0
func (latte) Surface0() Color {
	return Color{
		Hex: "#ccd0da",
		RGB: [3]uint8{204, 208, 218},
		HSL: [3]float32{223, 0.16, 0.83},
	}
}

// Base
func (latte) Base() Color {
	return Color{
		Hex: "#eff1f5",
		RGB: [3]uint8{239, 241, 245},
		HSL: [3]float32{220, 0.23, 0.95},
	}
}

// Mantle
func (latte) Mantle() Color {
	return Color{
		Hex: "#e6e9ef",
		RGB: [3]uint8{230, 233, 239},
		HSL: [3]float32{220, 0.22, 0.92},
	}
}

// Crust
func (latte) Crust() Color {
	return Color{
		Hex: "#dce0e8",
		RGB: [3]uint8{220, 224, 232},
		HSL: [3]float32{220, 0.21, 0.89},
	}
}
