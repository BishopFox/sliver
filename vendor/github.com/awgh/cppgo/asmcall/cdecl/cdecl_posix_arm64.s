// +build !windows

TEXT ·call0(SB),4,$16-8 
  MOVD    addr+0(FP), R0
  CALL    R0
  MOVD    R0, ret+8(FP)
  RET

TEXT ·call1(SB),4,$24-16
  MOVD    addr+0(FP), R1
  MOVD    arg1+8(FP), R0 
  CALL    R1
  MOVD    R0, ret+16(FP)
  RET

TEXT ·call2(SB),4,$32-24
  MOVD    addr+0(FP), R2
  MOVD    arg1+8(FP), R0
  MOVD    arg2+16(FP), R1
  CALL    R2
  MOVD    R0, ret+24(FP)
  RET

TEXT ·call3(SB),4,$40-32
  MOVD    addr+0(FP), R3
  MOVD    arg1+8(FP), R0
  MOVD    arg2+16(FP), R1
  MOVD    arg3+24(FP), R2
  CALL    R3
  MOVD    R0, ret+32(FP)
  RET

TEXT ·call4(SB),4,$48-40
  MOVD    addr+0(FP), R4
  MOVD    arg1+8(FP), R0
  MOVD    arg2+16(FP), R1
  MOVD    arg3+24(FP), R2
  MOVD    arg4+32(FP), R3
  CALL    R4
  MOVD    R0, ret+40(FP)
  RET

TEXT ·call5(SB),4,$56-48
  MOVD    addr+0(FP), R5
  MOVD    arg1+8(FP), R0
  MOVD    arg2+16(FP), R1
  MOVD    arg3+24(FP), R2
  MOVD    arg4+32(FP), R3
  MOVD    arg5+40(FP), R4
  CALL    R5
  MOVD    R0, ret+48(FP)
  RET

TEXT ·call6(SB),4,$64-56
  MOVD    addr+0(FP), R6
  MOVD    arg1+8(FP), R0
  MOVD    arg2+16(FP), R1
  MOVD    arg3+24(FP), R2
  MOVD    arg4+32(FP), R3
  MOVD    arg5+40(FP), R4
  MOVD    arg6+48(FP), R5
  CALL    R6
  MOVD    R0, ret+56(FP)
  RET
