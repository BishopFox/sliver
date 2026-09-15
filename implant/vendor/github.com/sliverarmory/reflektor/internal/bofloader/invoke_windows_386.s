//go:build windows && 386

#include "textflag.h"
#include "go_asm.h"

GLOBL ·windows386InvokeABI0(SB), NOPTR|RODATA, $4
DATA ·windows386InvokeABI0(SB)/4, $windows386Invoke(SB)

// windows386Invoke is entered by runtime.cgocall using the i386 C ABI. BOF
// entry points are __cdecl, so this caller removes their two stack arguments;
// syscall.SyscallN cannot be used because its i386 bridge is stdcall.
TEXT windows386Invoke(SB), NOSPLIT|NOFRAME, $0
	PUSHL BX
	MOVL 8(SP), BX
	MOVL windows386InvokeFrame_entry(BX), AX
	MOVL windows386InvokeFrame_address(BX), CX
	MOVL windows386InvokeFrame_length(BX), DX
	SUBL $8, SP
	MOVL CX, 0(SP)
	MOVL DX, 4(SP)
	CALL AX
	ADDL $8, SP
	POPL BX
	RET
