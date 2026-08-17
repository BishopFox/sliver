//go:build darwin && amd64 && !cgo

#include "textflag.h"

TEXT ·cCall10(SB), NOSPLIT, $0-96
	MOVQ fn+0(FP), AX
	MOVQ a0+8(FP), DI
	MOVQ a1+16(FP), SI
	MOVQ a2+24(FP), DX
	MOVQ a3+32(FP), CX
	MOVQ a4+40(FP), R8
	MOVQ a5+48(FP), R9

	// Keep SysV stack alignment before calling C++ dyld internals:
	// reserve 32 bytes for stack args (a6-a9) plus 8 bytes padding.
	SUBQ $40, SP
	MOVQ a6+56(FP), R10
	MOVQ R10, 0(SP)
	MOVQ a7+64(FP), R10
	MOVQ R10, 8(SP)
	MOVQ a8+72(FP), R10
	MOVQ R10, 16(SP)
	MOVQ a9+80(FP), R10
	MOVQ R10, 24(SP)

	CALL AX

	ADDQ $40, SP
	MOVQ AX, ret+88(FP)
	RET
