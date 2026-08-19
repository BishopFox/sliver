//go:build linux && !cgo && 386

#include "textflag.h"
#include "go_asm.h"

GLOBL ·linux386CallABI0(SB), NOPTR|RODATA, $4
DATA ·linux386CallABI0(SB)/4, $linux386Call(SB)

// linux386Call is entered by runtime.cgocall using the i386 SysV C ABI. Its
// sole argument is a five-word linux386CallFrame. It intentionally handles
// only integer arguments and results; in particular it does not touch the x87
// stack after calling an integer-returning export.
TEXT linux386Call(SB), NOSPLIT|NOFRAME, $0
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

TEXT ·cCall0(SB), NOSPLIT, $0-8
	MOVL fn+0(FP), AX
	CALL AX
	MOVL AX, ret+4(FP)
	RET

TEXT ·cCall1(SB), NOSPLIT, $0-12
	MOVL fn+0(FP), AX
	MOVL a0+4(FP), BX
	SUBL $4, SP
	MOVL BX, 0(SP)
	CALL AX
	ADDL $4, SP
	MOVL AX, ret+8(FP)
	RET

TEXT ·cCall2(SB), NOSPLIT, $0-16
	MOVL fn+0(FP), AX
	MOVL a0+4(FP), BX
	MOVL a1+8(FP), CX
	SUBL $8, SP
	MOVL BX, 0(SP)
	MOVL CX, 4(SP)
	CALL AX
	ADDL $8, SP
	MOVL AX, ret+12(FP)
	RET

TEXT ·cCall3(SB), NOSPLIT, $0-20
	MOVL fn+0(FP), AX
	MOVL a0+4(FP), BX
	MOVL a1+8(FP), CX
	MOVL a2+12(FP), DX
	SUBL $12, SP
	MOVL BX, 0(SP)
	MOVL CX, 4(SP)
	MOVL DX, 8(SP)
	CALL AX
	ADDL $12, SP
	MOVL AX, ret+16(FP)
	RET
