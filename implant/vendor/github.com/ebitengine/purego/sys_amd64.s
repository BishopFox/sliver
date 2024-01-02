// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || (!cgo && linux)

#include "textflag.h"
#include "abi_amd64.h"
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
TEXT syscall9X(SB), NOSPLIT|NOFRAME, $0
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $32, SP
	MOVQ  DI, 24(BP) // save the pointer

	MOVQ syscall9Args_f1(DI), X0 // f1
	MOVQ syscall9Args_f2(DI), X1 // f2
	MOVQ syscall9Args_f3(DI), X2 // f3
	MOVQ syscall9Args_f4(DI), X3 // f4
	MOVQ syscall9Args_f5(DI), X4 // f5
	MOVQ syscall9Args_f6(DI), X5 // f6
	MOVQ syscall9Args_f7(DI), X6 // f7
	MOVQ syscall9Args_f8(DI), X7 // f8

	MOVQ syscall9Args_fn(DI), R10 // fn
	MOVQ syscall9Args_a2(DI), SI  // a2
	MOVQ syscall9Args_a3(DI), DX  // a3
	MOVQ syscall9Args_a4(DI), CX  // a4
	MOVQ syscall9Args_a5(DI), R8  // a5
	MOVQ syscall9Args_a6(DI), R9  // a6
	MOVQ syscall9Args_a7(DI), R11 // a7
	MOVQ syscall9Args_a8(DI), R12 // a8
	MOVQ syscall9Args_a9(DI), R13 // a9
	MOVQ syscall9Args_a1(DI), DI  // a1

	// push the remaining paramters onto the stack
	MOVQ R11, 0(SP)  // push a7
	MOVQ R12, 8(SP)  // push a8
	MOVQ R13, 16(SP) // push a9
	XORL AX, AX      // vararg: say "no float args"

	CALL R10

	MOVQ 24(BP), DI              // get the pointer back
	MOVQ AX, syscall9Args_r1(DI) // r1
	MOVQ X0, syscall9Args_r2(DI) // r2

	XORL AX, AX  // no error (it's ignored anyway)
	ADDQ $32, SP
	MOVQ BP, SP
	POPQ BP
	RET

TEXT callbackasm1(SB), NOSPLIT|NOFRAME, $0
	// remove return address from stack, we are not returning to callbackasm, but to its caller.
	MOVQ 0(SP), AX
	ADDQ $8, SP

	MOVQ 0(SP), R10 // get the return SP so that we can align register args with stack args

	// make space for first six int and 8 float arguments below the frame
	ADJSP $14*8, SP
	MOVSD X0, (1*8)(SP)
	MOVSD X1, (2*8)(SP)
	MOVSD X2, (3*8)(SP)
	MOVSD X3, (4*8)(SP)
	MOVSD X4, (5*8)(SP)
	MOVSD X5, (6*8)(SP)
	MOVSD X6, (7*8)(SP)
	MOVSD X7, (8*8)(SP)
	MOVQ  DI, (9*8)(SP)
	MOVQ  SI, (10*8)(SP)
	MOVQ  DX, (11*8)(SP)
	MOVQ  CX, (12*8)(SP)
	MOVQ  R8, (13*8)(SP)
	MOVQ  R9, (14*8)(SP)
	LEAQ  8(SP), R8      // R8 = address of args vector

	MOVQ R10, 0(SP) // push the stack pointer below registers

	// determine index into runtime·cbs table
	MOVQ $callbackasm(SB), DX
	SUBQ DX, AX
	MOVQ $0, DX
	MOVQ $5, CX               // divide by 5 because each call instruction in ·callbacks is 5 bytes long
	DIVL CX
	SUBQ $1, AX               // subtract 1 because return PC is to the next slot

	// Switch from the host ABI to the Go ABI.
	PUSH_REGS_HOST_TO_ABI0()

	// Create a struct callbackArgs on our stack to be passed as
	// the "frame" to cgocallback and on to callbackWrap.
	// $24 to make enough room for the arguments to runtime.cgocallback
	SUBQ $(24+callbackArgs__size), SP
	MOVQ AX, (24+callbackArgs_index)(SP)  // callback index
	MOVQ R8, (24+callbackArgs_args)(SP)   // address of args vector
	MOVQ $0, (24+callbackArgs_result)(SP) // result
	LEAQ 24(SP), AX                       // take the address of callbackArgs

	// Call cgocallback, which will call callbackWrap(frame).
	MOVQ ·callbackWrap_call(SB), DI // Get the ABIInternal function pointer
	MOVQ (DI), DI                   // without <ABIInternal> by using a closure.
	MOVQ AX, SI                     // frame (address of callbackArgs)
	MOVQ $0, CX                     // context

	CALL crosscall2(SB) // runtime.cgocallback(fn, frame, ctxt uintptr)

	// Get callback result.
	MOVQ (24+callbackArgs_result)(SP), AX
	ADDQ $(24+callbackArgs__size), SP     // remove callbackArgs struct

	POP_REGS_HOST_TO_ABI0()

	MOVQ 0(SP), R10 // get the SP back

	ADJSP $-14*8, SP // remove arguments

	MOVQ R10, 0(SP)

	RET
