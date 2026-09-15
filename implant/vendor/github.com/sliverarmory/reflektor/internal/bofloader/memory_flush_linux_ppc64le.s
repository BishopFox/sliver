//go:build linux && !android && ppc64le

#include "textflag.h"

// Power implementations may combine multiple 32-byte cache blocks into a
// larger line. Walking this compiler-runtime minimum stride is conservative
// across implementations, though it can repeat work on newer processors.
// The caller rounds both endpoints to complete blocks.
//
// The ordering is the conservative self-modifying-code sequence documented
// by IBM's PowerPC processor manuals: DCBF, SYNC, ICBI, SYNC, ISYNC. LLVM's
// compiler-rt uses the same 32-byte walks and permits the second SYNC to be
// elided on modern Power implementations.
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
	// Complete every ICBI before discarding prefetched instructions. POWER8+
	// permits ICBI followed directly by ISYNC, but the full sequence is safe
	// across the Linux ppc64le ISA levels supported by Go.
	SYNC
	ISYNC
	RET
