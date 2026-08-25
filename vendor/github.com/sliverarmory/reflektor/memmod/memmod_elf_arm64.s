//go:build !cgo && arm64 && ((linux && !android) || freebsd)

#include "textflag.h"

TEXT ·cCall0(SB), NOSPLIT, $0-16
	MOVD fn+0(FP), R16
	BL (R16)
	MOVD R0, ret+8(FP)
	RET

TEXT ·cCall1(SB), NOSPLIT, $0-24
	MOVD fn+0(FP), R16
	MOVD a0+8(FP), R0
	BL (R16)
	MOVD R0, ret+16(FP)
	RET

TEXT ·cCall2(SB), NOSPLIT, $0-32
	MOVD fn+0(FP), R16
	MOVD a0+8(FP), R0
	MOVD a1+16(FP), R1
	BL (R16)
	MOVD R0, ret+24(FP)
	RET

TEXT ·cCall3(SB), NOSPLIT, $0-40
	MOVD fn+0(FP), R16
	MOVD a0+8(FP), R0
	MOVD a1+16(FP), R1
	MOVD a2+24(FP), R2
	BL (R16)
	MOVD R0, ret+32(FP)
	RET
