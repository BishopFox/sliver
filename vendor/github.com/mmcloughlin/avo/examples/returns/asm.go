// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func main() {
	Package("github.com/mmcloughlin/avo/examples/returns")

	TEXT("Interval", NOSPLIT, "func(start, size uint64) (uint64, uint64)")
	Doc(
		"Interval returns the (start, end) of an interval with the given start and size.",
		"Demonstrates multiple unnamed return values.",
	)
	start := Load(Param("start"), GP64())
	size := Load(Param("size"), GP64())
	end := size
	ADDQ(start, end)
	Store(start, ReturnIndex(0))
	Store(end, ReturnIndex(1))
	RET()

	TEXT("Butterfly", NOSPLIT, "func(x0, x1 float64) (y0, y1 float64)")
	Doc(
		"Butterfly performs a 2-dimensional butterfly operation: computes (x0+x1, x0-x1).",
		"Demonstrates multiple named return values.",
	)
	x0 := Load(Param("x0"), XMM())
	x1 := Load(Param("x1"), XMM())
	y0, y1 := XMM(), XMM()
	MOVSD(x0, y0)
	ADDSD(x1, y0)
	MOVSD(x0, y1)
	SUBSD(x1, y1)
	Store(y0, Return("y0"))
	Store(y1, Return("y1"))
	RET()

	TEXT("Septuple", NOSPLIT, "func(byte) [7]byte")
	Doc(
		"Septuple returns an array of seven of the given byte.",
		"Demonstrates returning array values.",
	)
	b := Load(ParamIndex(0), GP8())
	for i := 0; i < 7; i++ {
		Store(b, ReturnIndex(0).Index(i))
	}
	RET()

	TEXT("CriticalLine", NOSPLIT, "func(t float64) complex128")
	Doc(
		"CriticalLine returns the complex value 0.5 + it on Riemann's critical line.",
		"Demonstrates returning complex values.",
	)
	t := Load(Param("t"), XMM())
	half := XMM()
	MOVSD(ConstData("half", F64(0.5)), half)
	Store(half, ReturnIndex(0).Real())
	Store(t, ReturnIndex(0).Imag())
	RET()

	TEXT("NewStruct", NOSPLIT, "func(w uint16, p [2]float64, q uint64) Struct")
	Doc(
		"NewStruct initializes a Struct value.",
		"Demonstrates returning struct values.",
	)
	w := Load(Param("w"), GP16())
	x := Load(Param("p").Index(0), XMM())
	y := Load(Param("p").Index(1), XMM())
	q := Load(Param("q"), GP64())
	Store(w, ReturnIndex(0).Field("Word"))
	Store(x, ReturnIndex(0).Field("Point").Index(0))
	Store(y, ReturnIndex(0).Field("Point").Index(1))
	Store(q, ReturnIndex(0).Field("Quad"))
	RET()

	Generate()
}
