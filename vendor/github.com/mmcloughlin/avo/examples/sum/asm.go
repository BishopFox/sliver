// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func main() {
	TEXT("Sum", NOSPLIT, "func(xs []uint64) uint64")
	Doc("Sum returns the sum of the elements in xs.")
	ptr := Load(Param("xs").Base(), GP64())
	n := Load(Param("xs").Len(), GP64())

	Comment("Initialize sum register to zero.")
	s := GP64()
	XORQ(s, s)

	Label("loop")
	Comment("Loop until zero bytes remain.")
	CMPQ(n, Imm(0))
	JE(LabelRef("done"))

	Comment("Load from pointer and add to running sum.")
	ADDQ(Mem{Base: ptr}, s)

	Comment("Advance pointer, decrement byte count.")
	ADDQ(Imm(8), ptr)
	DECQ(n)
	JMP(LabelRef("loop"))

	Label("done")
	Comment("Store sum to return value.")
	Store(s, ReturnIndex(0))
	RET()
	Generate()
}
