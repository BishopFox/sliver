package returns

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out returns.s -stubs stub.go

func TestFunctionsEqual(t *testing.T) {
	cases := []struct {
		f, g interface{}
	}{
		{Interval, func(s, n uint64) (uint64, uint64) { return s, s + n }},
		{Butterfly, func(x0, x1 float64) (float64, float64) { return x0 + x1, x0 - x1 }},
		{Septuple, func(b byte) [7]byte { return [...]byte{b, b, b, b, b, b, b} }},
		{CriticalLine, func(t float64) complex128 { return complex(0.5, t) }},
		{NewStruct, func(w uint16, p [2]float64, q uint64) Struct { return Struct{Word: w, Point: p, Quad: q} }},
	}
	for _, c := range cases {
		if err := quick.CheckEqual(c.f, c.g, nil); err != nil {
			t.Fatal(err)
		}
	}
}
