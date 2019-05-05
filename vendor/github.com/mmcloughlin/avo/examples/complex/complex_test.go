package complex

import (
	"math"
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out complex.s -stubs stub.go

func TestReal(t *testing.T) {
	expect := func(z complex128) float64 {
		return real(z)
	}
	if err := quick.CheckEqual(Real, expect, nil); err != nil {
		t.Fatal(err)
	}
}

func TestImag(t *testing.T) {
	expect := func(z complex128) float64 {
		return imag(z)
	}
	if err := quick.CheckEqual(Imag, expect, nil); err != nil {
		t.Fatal(err)
	}
}

func TestNorm(t *testing.T) {
	expect := func(z complex128) float64 {
		return math.Sqrt(real(z)*real(z) + imag(z)*imag(z))
	}
	if err := quick.CheckEqual(Norm, expect, nil); err != nil {
		t.Fatal(err)
	}
}
