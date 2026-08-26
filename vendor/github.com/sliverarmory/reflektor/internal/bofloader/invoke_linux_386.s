//go:build linux && !android && 386

#include "textflag.h"
#include "go_asm.h"

GLOBL ·linux386InvokeABI0(SB), NOPTR|RODATA, $4
DATA ·linux386InvokeABI0(SB)/4, $linux386Invoke(SB)

// linux386Invoke is entered by runtime.cgocall using the i386 SysV C ABI.
// The BOF entry itself uses the C signature void go(char *args, int length).
TEXT linux386Invoke(SB), NOSPLIT|NOFRAME, $0
	PUSHL BX
	MOVL 8(SP), BX
	MOVL linux386InvokeFrame_entry(BX), AX
	MOVL linux386InvokeFrame_address(BX), CX
	MOVL linux386InvokeFrame_length(BX), DX
	SUBL $8, SP
	MOVL CX, 0(SP)
	MOVL DX, 4(SP)
	CALL AX
	ADDL $8, SP
	POPL BX
	RET
