TEXT ·call0(SB),4,$0-8
  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+4(FP)
  RET

TEXT ·call1(SB),4,$4-12
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+8(FP)
  RET

TEXT ·call2(SB),4,$8-16
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)
  MOVL b+8(FP), BX
  MOVL BX, 4(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+12(FP)
  RET

TEXT ·call3(SB),4,$12-20
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)
  MOVL b+8(FP), BX
  MOVL BX, 4(SP)
  MOVL c+12(FP), BX
  MOVL BX, 8(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+16(FP)
  RET

TEXT ·call4(SB),4,$16-24
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)
  MOVL b+8(FP), BX
  MOVL BX, 4(SP)
  MOVL c+12(FP), BX
  MOVL BX, 8(SP)
  MOVL d+16(FP), BX
  MOVL BX, 12(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+20(FP)
  RET

TEXT ·call5(SB),4,$20-28
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)
  MOVL b+8(FP), BX
  MOVL BX, 4(SP)
  MOVL c+12(FP), BX
  MOVL BX, 8(SP)
  MOVL d+16(FP), BX
  MOVL BX, 12(SP)
  MOVL e+20(FP), BX
  MOVL BX, 16(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+24(FP)
  RET

TEXT ·call6(SB),4,$24-32
  MOVL a+4(FP), BX
  MOVL BX, 0(SP)
  MOVL b+8(FP), BX
  MOVL BX, 4(SP)
  MOVL c+12(FP), BX
  MOVL BX, 8(SP)
  MOVL d+16(FP), BX
  MOVL BX, 12(SP)
  MOVL e+20(FP), BX
  MOVL BX, 16(SP)
  MOVL f+24(FP), BX
  MOVL BX, 20(SP)

  MOVL addr+0(FP), AX
  CALL AX
  MOVL AX, ret+28(FP)
  RET
