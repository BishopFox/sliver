// +build ignore

package main

import (
	"math"

	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func main() {
	bytes := GLOBL("bytes", RODATA|NOPTR)
	DATA(0, U64(0x0011223344556677))
	DATA(8, String("strconst"))
	DATA(16, F32(math.Pi))
	DATA(24, F64(math.Pi))
	DATA(32, U32(0x00112233))
	DATA(36, U16(0x4455))
	DATA(38, U8(0x66))
	DATA(39, U8(0x77))

	TEXT("DataAt", NOSPLIT, "func(i int) byte")
	Doc("DataAt returns byte i in the 'bytes' global data section.")
	i := Load(Param("i"), GP64())
	ptr := Mem{Base: GP64()}
	LEAQ(bytes, ptr.Base)
	b := GP8()
	MOVB(ptr.Idx(i, 1), b)
	Store(b, ReturnIndex(0))
	RET()

	Generate()
}
