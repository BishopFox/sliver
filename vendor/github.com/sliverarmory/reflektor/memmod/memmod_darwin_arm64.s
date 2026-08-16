//go:build darwin && arm64 && !cgo

#include "textflag.h"

TEXT ·cCall10(SB), NOSPLIT, $0-96
	MOVD fn+0(FP), R16
	MOVD a0+8(FP), R0
	MOVD a1+16(FP), R1
	MOVD a2+24(FP), R2
	MOVD a3+32(FP), R3
	MOVD a4+40(FP), R4
	MOVD a5+48(FP), R5
	MOVD a6+56(FP), R6
	MOVD a7+64(FP), R7
	SUB $16, RSP
	MOVD a8+72(FP), R10
	MOVD a9+80(FP), R11
	MOVD R10, 0(RSP)
	MOVD R11, 8(RSP)
	BL (R16)
	ADD $16, RSP
	MOVD R0, ret+88(FP)
	RET
