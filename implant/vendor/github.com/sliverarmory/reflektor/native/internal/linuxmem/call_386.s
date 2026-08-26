//go:build linux && !android && 386

// SPDX-License-Identifier: MIT
// Adapted from Reflektor's memmod Linux call bridge; see
// ../../../memmod/COPYING.

#include "textflag.h"
#include "go_asm.h"

GLOBL ·linux386CallABI0(SB), NOPTR|RODATA, $4
DATA ·linux386CallABI0(SB)/4, $reflektorNativeLinux386Call(SB)

// reflektorNativeLinux386Call is entered by runtime.cgocall using the i386
// SysV C ABI. Its sole argument is a five-word linux386CallFrame. It
// intentionally handles only integer arguments and results; in particular it
// does not touch the x87 stack after calling an integer-returning export. The
// globally unique name prevents collisions when root memmod and native are
// linked into the same binary.
TEXT reflektorNativeLinux386Call(SB), NOSPLIT|NOFRAME, $0
	PUSHL BX
	MOVL 8(SP), BX
	MOVL linux386CallFrame_fn(BX), AX
	MOVL linux386CallFrame_a0(BX), CX
	MOVL linux386CallFrame_a1(BX), DX
	SUBL $24, SP
	MOVL CX, 0(SP)
	MOVL DX, 4(SP)
	MOVL linux386CallFrame_a2(BX), CX
	MOVL CX, 8(SP)
	CALL AX
	ADDL $24, SP
	MOVL AX, linux386CallFrame_result(BX)
	POPL BX
	RET
