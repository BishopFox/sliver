package catppuccingo

// Frappé variant
type frappe struct{}

// Frappé flavor variant
var Frappe Flavor = frappe{}

// Frappé
func (frappe) Name() string { return "frappe" }

// Rosewater
func (frappe) Rosewater() Color {
	return Color{
		Hex: "#f2d5cf",
		RGB: [3]uint8{242, 213, 207},
		HSL: [3]float32{10, 0.57, 0.88},
	}
}

// Flamingo
func (frappe) Flamingo() Color {
	return Color{
		Hex: "#eebebe",
		RGB: [3]uint8{238, 190, 190},
		HSL: [3]float32{0, 0.59, 0.84},
	}
}

// Pink
func (frappe) Pink() Color {
	return Color{
		Hex: "#f4b8e4",
		RGB: [3]uint8{244, 184, 228},
		HSL: [3]float32{316, 0.73, 0.84},
	}
}

// Mauve
func (frappe) Mauve() Color {
	return Color{
		Hex: "#ca9ee6",
		RGB: [3]uint8{202, 158, 230},
		HSL: [3]float32{277, 0.59, 0.76},
	}
}

// Red
func (frappe) Red() Color {
	return Color{
		Hex: "#e78284",
		RGB: [3]uint8{231, 130, 132},
		HSL: [3]float32{359, 0.68, 0.71},
	}
}

// Maroon
func (frappe) Maroon() Color {
	return Color{
		Hex: "#ea999c",
		RGB: [3]uint8{234, 153, 156},
		HSL: [3]float32{358, 0.66, 0.76},
	}
}

// Peach
func (frappe) Peach() Color {
	return Color{
		Hex: "#ef9f76",
		RGB: [3]uint8{239, 159, 118},
		HSL: [3]float32{20, 0.79, 0.7},
	}
}

// Yellow
func (frappe) Yellow() Color {
	return Color{
		Hex: "#e5c890",
		RGB: [3]uint8{229, 200, 144},
		HSL: [3]float32{40, 0.62, 0.73},
	}
}

// Green
func (frappe) Green() Color {
	return Color{
		Hex: "#a6d189",
		RGB: [3]uint8{166, 209, 137},
		HSL: [3]float32{96, 0.44, 0.68},
	}
}

// Teal
func (frappe) Teal() Color {
	return Color{
		Hex: "#81c8be",
		RGB: [3]uint8{129, 200, 190},
		HSL: [3]float32{172, 0.39, 0.65},
	}
}

// Sky
func (frappe) Sky() Color {
	return Color{
		Hex: "#99d1db",
		RGB: [3]uint8{153, 209, 219},
		HSL: [3]float32{189, 0.48, 0.73},
	}
}

// Sapphire
func (frappe) Sapphire() Color {
	return Color{
		Hex: "#85c1dc",
		RGB: [3]uint8{133, 193, 220},
		HSL: [3]float32{199, 0.55, 0.69},
	}
}

// Blue
func (frappe) Blue() Color {
	return Color{
		Hex: "#8caaee",
		RGB: [3]uint8{140, 170, 238},
		HSL: [3]float32{222, 0.74, 0.74},
	}
}

// Lavender
func (frappe) Lavender() Color {
	return Color{
		Hex: "#babbf1",
		RGB: [3]uint8{186, 187, 241},
		HSL: [3]float32{239, 0.66, 0.84},
	}
}

// Text
func (frappe) Text() Color {
	return Color{
		Hex: "#c6d0f5",
		RGB: [3]uint8{198, 208, 245},
		HSL: [3]float32{227, 0.7, 0.87},
	}
}

// Subtext 1
func (frappe) Subtext1() Color {
	return Color{
		Hex: "#b5bfe2",
		RGB: [3]uint8{181, 191, 226},
		HSL: [3]float32{227, 0.44, 0.8},
	}
}

// Subtext 0
func (frappe) Subtext0() Color {
	return Color{
		Hex: "#a5adce",
		RGB: [3]uint8{165, 173, 206},
		HSL: [3]float32{228, 0.29, 0.73},
	}
}

// Overlay 2
func (frappe) Overlay2() Color {
	return Color{
		Hex: "#949cbb",
		RGB: [3]uint8{148, 156, 187},
		HSL: [3]float32{228, 0.22, 0.66},
	}
}

// Overlay 1
func (frappe) Overlay1() Color {
	return Color{
		Hex: "#838ba7",
		RGB: [3]uint8{131, 139, 167},
		HSL: [3]float32{227, 0.17, 0.58},
	}
}

// Overlay 0
func (frappe) Overlay0() Color {
	return Color{
		Hex: "#737994",
		RGB: [3]uint8{115, 121, 148},
		HSL: [3]float32{229, 0.13, 0.52},
	}
}

// Surface 2
func (frappe) Surface2() Color {
	return Color{
		Hex: "#626880",
		RGB: [3]uint8{98, 104, 128},
		HSL: [3]float32{228, 0.13, 0.44},
	}
}

// Surface 1
func (frappe) Surface1() Color {
	return Color{
		Hex: "#51576d",
		RGB: [3]uint8{81, 87, 109},
		HSL: [3]float32{227, 0.15, 0.37},
	}
}

// Surface 0
func (frappe) Surface0() Color {
	return Color{
		Hex: "#414559",
		RGB: [3]uint8{65, 69, 89},
		HSL: [3]float32{230, 0.16, 0.3},
	}
}

// Base
func (frappe) Base() Color {
	return Color{
		Hex: "#303446",
		RGB: [3]uint8{48, 52, 70},
		HSL: [3]float32{229, 0.19, 0.23},
	}
}

// Mantle
func (frappe) Mantle() Color {
	return Color{
		Hex: "#292c3c",
		RGB: [3]uint8{41, 44, 60},
		HSL: [3]float32{231, 0.19, 0.2},
	}
}

// Crust
func (frappe) Crust() Color {
	return Color{
		Hex: "#232634",
		RGB: [3]uint8{35, 38, 52},
		HSL: [3]float32{229, 0.2, 0.17},
	}
}
