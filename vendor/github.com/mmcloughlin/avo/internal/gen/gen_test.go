package gen

import "testing"

func TestBuilderInterfaces(t *testing.T) {
	var _ = []Builder{
		NewAsmTest,
		NewGoData,
		NewGoDataTest,
		NewCtors,
		NewCtorsTest,
		NewBuild,
		NewMOV,
	}
}
