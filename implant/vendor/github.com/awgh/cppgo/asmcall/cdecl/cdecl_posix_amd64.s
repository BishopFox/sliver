// +build !windows

TEXT ·call0(SB),4,$0-16
  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+8(FP)
  RET

TEXT ·call1(SB),4,$0-24
  MOVQ a+8(FP), DI

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+16(FP)
  RET

TEXT ·call2(SB),4,$0-32
  MOVQ a+8(FP), DI
  MOVQ b+16(FP), SI

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+24(FP)
  RET

TEXT ·call3(SB),4,$0-40
  MOVQ a+8(FP), DI
  MOVQ b+16(FP), SI
  MOVQ c+24(FP), DX

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+32(FP)
  RET

TEXT ·call4(SB),4,$0-48
  MOVQ a+8(FP), DI
  MOVQ b+16(FP), SI
  MOVQ c+24(FP), DX
  MOVQ d+32(FP), CX

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+40(FP)
  RET

TEXT ·call5(SB),4,$0-56
  MOVQ a+8(FP), DI
  MOVQ b+16(FP), SI
  MOVQ c+24(FP), DX
  MOVQ d+32(FP), CX
  MOVQ e+40(FP), R8

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+48(FP)
  RET

TEXT ·call6(SB),4,$0-64
  MOVQ a+8(FP), DI
  MOVQ b+16(FP), SI
  MOVQ c+24(FP), DX
  MOVQ d+32(FP), CX
  MOVQ e+40(FP), R8
  MOVQ f+48(FP), R9

  MOVQ addr+0(FP), AX
  CALL AX
  MOVQ AX, ret+56(FP)
  RET
