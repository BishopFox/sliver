// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

func main() {
	TEXT("Labels", NOSPLIT, "func() uint64")
	XORQ(RAX, RAX)
	INCQ(RAX)
	Label("neverused")
	INCQ(RAX)
	INCQ(RAX)
	INCQ(RAX)
	INCQ(RAX)
	JMP(LabelRef("next"))
	Label("next")
	INCQ(RAX)
	INCQ(RAX)
	Store(RAX, ReturnIndex(0))
	RET()

	Generate()
}
