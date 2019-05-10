// +build ignore

package main

import (
	"strconv"

	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	. "github.com/mmcloughlin/avo/reg"
)

const (
	k0U64 = 0xb89b0f8e1655514f
	k1U64 = 0x8c6f736011bd5127
	k2U64 = 0x8f29bd94edce7b39
	k3U64 = 0x9c1b8e1e9628323f

	k2U32 = 0x802910e3
	k3U32 = 0x819b13af
	k4U32 = 0x91cb27e5
	k5U32 = 0xc1a269c1
)

func imul(k uint64, r Register) {
	t := GP64()
	MOVQ(U64(k), t)
	IMULQ(t, r)
}

func makelabels(name string, n int) []string {
	l := make([]string, n)
	for i := 0; i < n; i++ {
		l[i] = name + strconv.Itoa(i)
	}
	return l
}

func main() {
	Package("github.com/mmcloughlin/avo/examples/stadtx")
	TEXT("Hash", NOSPLIT, "func(state *State, key []byte) uint64")
	Doc("Hash computes the Stadtx hash.")

	statePtr := Load(Param("state"), GP64())
	ptr := Load(Param("key").Base(), GP64())
	n := Load(Param("key").Len(), GP64())

	v0 := GP64()
	v1 := GP64()
	MOVQ(Mem{Base: statePtr}, v0)
	MOVQ(Mem{Base: statePtr, Disp: 8}, v1)

	t := GP64()
	MOVQ(n, t)
	ADDQ(U32(1), t)
	imul(k0U64, t)
	XORQ(t, v0)

	MOVQ(n, t)
	ADDQ(U32(2), t)
	imul(k1U64, t)
	XORQ(t, v1)

	long := "coreLong"
	CMPQ(n, U32(32))
	JGE(LabelRef(long))
	//
	u64s := GP64()
	MOVQ(n, u64s)
	SHRQ(U8(3), u64s)
	//
	labels := makelabels("shortCore", 4)
	//
	for i := 0; i < 4; i++ {
		CMPQ(u64s, U32(i))
		JE(LabelRef(labels[i]))
	}
	for i := 3; i > 0; i-- {
		Label(labels[i])
		r := GP64()
		MOVQ(Mem{Base: ptr}, r)
		imul(k3U64, r)
		ADDQ(r, v0)
		RORQ(U8(17), v0)
		XORQ(v1, v0)
		RORQ(U8(53), v1)
		ADDQ(v0, v1)
		ADDQ(U32(8), ptr)
		SUBQ(U32(8), n)
	}
	Label(labels[0])

	labels = makelabels("shortTail", 8)

	for i := 0; i < 8; i++ {
		CMPQ(n, U32(i))
		JE(LabelRef(labels[i]))
	}

	after := "shortAfter"

	ch := GP64()

	Label(labels[7])
	MOVBQZX(Mem{Base: ptr, Disp: 6}, ch)
	SHLQ(U8(32), ch)
	ADDQ(ch, v0)

	Label(labels[6])
	MOVBQZX(Mem{Base: ptr, Disp: 5}, ch)
	SHLQ(U8(48), ch)
	ADDQ(ch, v1)

	Label(labels[5])
	MOVBQZX(Mem{Base: ptr, Disp: 4}, ch)
	SHLQ(U8(16), ch)
	ADDQ(ch, v0)

	Label(labels[4])

	MOVLQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v1)

	JMP(LabelRef(after))

	Label(labels[3])
	MOVBQZX(Mem{Base: ptr, Disp: 2}, ch)
	SHLQ(U8(48), ch)
	ADDQ(ch, v0)

	Label(labels[2])

	MOVWQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v1)

	JMP(LabelRef(after))

	Label(labels[1])
	MOVBQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v0)

	Label(labels[0])
	RORQ(U8(32), v1)
	XORQ(U32(0xff), v1)

	Label(after)

	XORQ(v0, v1)

	RORQ(U8(33), v0)
	ADDQ(v1, v0)

	ROLQ(U8(17), v1)
	XORQ(v0, v1)

	ROLQ(U8(43), v0)
	ADDQ(v1, v0)

	ROLQ(U8(31), v1)
	SUBQ(v0, v1)

	ROLQ(U8(13), v0)
	XORQ(v1, v0)

	SUBQ(v0, v1)

	ROLQ(U8(41), v0)
	ADDQ(v1, v0)

	ROLQ(U8(37), v1)
	XORQ(v0, v1)

	RORQ(U8(39), v0)
	ADDQ(v1, v0)

	RORQ(U8(15), v1)
	ADDQ(v0, v1)

	ROLQ(U8(15), v0)
	XORQ(v1, v0)

	RORQ(U8(5), v1)

	XORQ(v1, v0)

	Store(v0, ReturnIndex(0))
	RET()

	Label(long)

	v2 := GP64()
	v3 := GP64()

	MOVQ(Mem{Base: statePtr, Disp: 16}, v2)
	MOVQ(Mem{Base: statePtr, Disp: 24}, v3)

	MOVQ(n, t)
	ADDQ(U32(3), t)
	imul(k2U64, t)
	XORQ(t, v2)

	MOVQ(n, t)
	ADDQ(U32(4), t)
	imul(k3U64, t)
	XORQ(t, v3)

	r := GP64()
	loop := "block"
	Label(loop)
	MOVQ(Mem{Base: ptr}, r)
	imul(k2U32, r)
	ADDQ(r, v0)
	ROLQ(U8(57), v0)
	XORQ(v3, v0)

	MOVQ(Mem{Base: ptr, Disp: 8}, r)
	imul(k3U32, r)
	ADDQ(r, v1)
	ROLQ(U8(63), v1)
	XORQ(v2, v1)

	MOVQ(Mem{Base: ptr, Disp: 16}, r)
	imul(k4U32, r)
	ADDQ(r, v2)
	RORQ(U8(47), v2)
	ADDQ(v0, v2)

	MOVQ(Mem{Base: ptr, Disp: 24}, r)
	imul(k5U32, r)
	ADDQ(r, v3)
	RORQ(U8(11), v3)
	SUBQ(v1, v3)

	ADDQ(U32(32), ptr)
	SUBQ(U32(32), n)

	CMPQ(n, U32(32))
	JGE(LabelRef(loop))

	nsave := GP64()
	MOVQ(n, nsave)

	MOVQ(n, u64s)
	SHRQ(U8(3), u64s)

	labels = makelabels("longCore", 4)

	for i := 0; i < 4; i++ {
		CMPQ(u64s, U32(i))
		JE(LabelRef(labels[i]))
	}
	Label(labels[3])

	MOVQ(Mem{Base: ptr}, r)
	imul(k2U32, r)
	ADDQ(r, v0)
	ROLQ(U8(57), v0)
	XORQ(v3, v0)
	ADDQ(U32(8), ptr)
	SUBQ(U32(8), n)

	Label(labels[2])

	MOVQ(Mem{Base: ptr}, r)
	imul(k3U32, r)
	ADDQ(r, v1)
	ROLQ(U8(63), v1)
	XORQ(v2, v1)
	ADDQ(U32(8), ptr)
	SUBQ(U32(8), n)

	Label(labels[1])

	MOVQ(Mem{Base: ptr}, r)
	imul(k4U32, r)
	ADDQ(r, v2)
	RORQ(U8(47), v2)
	ADDQ(v0, v2)
	ADDQ(U32(8), ptr)
	SUBQ(U32(8), n)

	Label(labels[0])

	RORQ(U8(11), v3)
	SUBQ(v1, v3)

	ADDQ(U32(1), nsave)
	imul(k3U64, nsave)
	XORQ(nsave, v0)

	labels = makelabels("longTail", 8)

	for i := 0; i < 8; i++ {
		CMPQ(n, U32(i))
		JE(LabelRef(labels[i]))
	}

	after = "longAfter"
	Label(labels[7])
	MOVBQZX(Mem{Base: ptr, Disp: 6}, ch)
	ADDQ(ch, v1)

	Label(labels[6])

	MOVWQZX(Mem{Base: ptr, Disp: 4}, ch)
	ADDQ(ch, v2)
	MOVLQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v3)
	JMP(LabelRef(after))

	Label(labels[5])
	MOVBQZX(Mem{Base: ptr, Disp: 4}, ch)
	ADDQ(ch, v1)

	Label(labels[4])

	MOVLQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v2)

	JMP(LabelRef(after))

	Label(labels[3])
	MOVBQZX(Mem{Base: ptr, Disp: 2}, ch)
	ADDQ(ch, v3)

	Label(labels[2])

	MOVWQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v1)

	JMP(LabelRef(after))

	Label(labels[1])
	MOVBQZX(Mem{Base: ptr}, ch)
	ADDQ(ch, v2)

	Label(labels[0])
	ROLQ(U8(32), v3)
	XORQ(U32(0xff), v3)

	Label(after)

	SUBQ(v2, v1)
	RORQ(U8(19), v0)
	SUBQ(v0, v1)
	RORQ(U8(53), v1)
	XORQ(v1, v3)
	SUBQ(v3, v0)
	ROLQ(U8(43), v3)
	ADDQ(v3, v0)
	RORQ(U8(3), v0)
	SUBQ(v0, v3)
	RORQ(U8(43), v2)
	SUBQ(v3, v2)
	ROLQ(U8(55), v2)
	XORQ(v0, v2)
	SUBQ(v2, v1)
	RORQ(U8(7), v3)
	SUBQ(v2, v3)
	RORQ(U8(31), v2)
	ADDQ(v2, v3)
	SUBQ(v1, v2)
	RORQ(U8(39), v3)
	XORQ(v3, v2)
	RORQ(U8(17), v3)
	XORQ(v2, v3)
	ADDQ(v3, v1)
	RORQ(U8(9), v1)
	XORQ(v1, v2)
	ROLQ(U8(24), v2)
	XORQ(v2, v3)
	RORQ(U8(59), v3)
	RORQ(U8(1), v0)
	SUBQ(v1, v0)

	XORQ(v1, v0)
	XORQ(v3, v2)

	XORQ(v2, v0)

	Store(v0, ReturnIndex(0))
	RET()

	Generate()
}
