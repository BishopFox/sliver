// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	TEXT("Issue76", NOSPLIT, "func(x, y uint64) uint64")
	x := Load(Param("x"), GP64())
	y := Load(Param("y"), GP64())
	s := add(x, y)
	Store(s, ReturnIndex(0))
	RET()
	Generate()
}

// add generates code to add x and y. The intent here is to demonstrate how a
// natural subroutine in avo typically requires temporary registers, which in
// turn can be "optimized out" by the register allocator and result in redundant
// self-moves.
func add(x, y Register) Register {
	s := GP64()
	MOVQ(x, s) // likely to become a self move
	ADDQ(y, s)
	return s
}
