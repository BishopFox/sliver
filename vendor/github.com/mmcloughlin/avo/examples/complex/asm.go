// +build ignore

package main

import . "github.com/mmcloughlin/avo/build"

func main() {
	TEXT("Real", NOSPLIT, "func(z complex128) float64")
	Doc("Real returns the real part of z.")
	r := Load(Param("z").Real(), XMM())
	Store(r, ReturnIndex(0))
	RET()

	TEXT("Imag", NOSPLIT, "func(z complex128) float64")
	Doc("Imag returns the imaginary part of z.")
	i := Load(Param("z").Imag(), XMM())
	Store(i, ReturnIndex(0))
	RET()

	TEXT("Norm", NOSPLIT, "func(z complex128) float64")
	Doc("Norm returns the complex norm of z.")
	r = Load(Param("z").Real(), XMM())
	i = Load(Param("z").Imag(), XMM())
	MULSD(r, r)
	MULSD(i, i)
	ADDSD(i, r)
	n := XMM()
	SQRTSD(r, n)
	Store(n, ReturnIndex(0))
	RET()

	Generate()
}
