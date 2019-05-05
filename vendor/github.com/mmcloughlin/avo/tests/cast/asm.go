// +build ignore

package main

import . "github.com/mmcloughlin/avo/build"

func main() {
	TEXT("Split", NOSPLIT, "func(x uint64) (q uint64, l uint32, w uint16, b uint8)")
	Doc(
		"Split returns the low 64, 32, 16 and 8 bits of x.",
		"Tests the As() methods of virtual general-purpose registers.",
	)
	x := GP64()
	Load(Param("x"), x)
	Store(x, Return("q"))
	Store(x.As32(), Return("l"))
	Store(x.As16(), Return("w"))
	Store(x.As8(), Return("b"))
	RET()

	Generate()
}
