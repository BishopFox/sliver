//go:build linux && !android && ppc64le && !cgo

#include "textflag.h"

TEXT ·flushPPC64InstructionCache(SB),NOSPLIT|NOFRAME,$0-16
	MOVD	start+0(FP), R3
	MOVD	end+8(FP), R4
	MOVD	R3, R5

data:
	CMPU	R3, R4
	BGE	dataDone
	DCBF	(R3)
	ADD	$32, R3
	BR	data

dataDone:
	SYNC

instruction:
	CMPU	R5, R4
	BGE	instructionDone
	ICBI	(R5)
	ADD	$32, R5
	BR	instruction

instructionDone:
	SYNC
	ISYNC
	RET
