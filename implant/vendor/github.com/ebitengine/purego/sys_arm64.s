// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || (!cgo && linux) || windows

#include "textflag.h"
#include "go_asm.h"
#include "funcdata.h"

// syscall9X calls a function in libc on behalf of the syscall package.
// syscall9X takes a pointer to a struct like:
// struct {
//	fn    uintptr
//	a1    uintptr
//	a2    uintptr
//	a3    uintptr
//	a4    uintptr
//	a5    uintptr
//	a6    uintptr
//	a7    uintptr
//	a8    uintptr
//	a9    uintptr
//	r1    uintptr
//	r2    uintptr
//	err   uintptr
// }
// syscall9X must be called on the g0 stack with the
// C calling convention (use libcCall).
GLOBL ·syscall9XABI0(SB), NOPTR|RODATA, $8
DATA ·syscall9XABI0(SB)/8, $syscall9X(SB)
TEXT syscall9X(SB), NOSPLIT, $0
	SUB  $16, RSP   // push structure pointer
	MOVD R0, 8(RSP)

	FMOVD syscall9Args_f1(R0), F0 // f1
	FMOVD syscall9Args_f2(R0), F1 // f2
	FMOVD syscall9Args_f3(R0), F2 // f3
	FMOVD syscall9Args_f4(R0), F3 // f4
	FMOVD syscall9Args_f5(R0), F4 // f5
	FMOVD syscall9Args_f6(R0), F5 // f6
	FMOVD syscall9Args_f7(R0), F6 // f7
	FMOVD syscall9Args_f8(R0), F7 // f8

	MOVD syscall9Args_fn(R0), R12 // fn
	MOVD syscall9Args_a2(R0), R1  // a2
	MOVD syscall9Args_a3(R0), R2  // a3
	MOVD syscall9Args_a4(R0), R3  // a4
	MOVD syscall9Args_a5(R0), R4  // a5
	MOVD syscall9Args_a6(R0), R5  // a6
	MOVD syscall9Args_a7(R0), R6  // a7
	MOVD syscall9Args_a8(R0), R7  // a8
	MOVD syscall9Args_a9(R0), R8  // a9
	MOVD syscall9Args_a1(R0), R0  // a1

	MOVD R8, (RSP) // push a9 onto stack

	BL (R12)

	MOVD  8(RSP), R2              // pop structure pointer
	ADD   $16, RSP
	MOVD  R0, syscall9Args_r1(R2) // save r1
	FMOVD F0, syscall9Args_r2(R2) // save r2
	RET
