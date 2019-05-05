# complex

Demonstrates how to access complex types in `avo`.

The `Real()` and `Imag()` parameter methods may be used to load the sub-components of complex arguments. The following function uses these to implement the [complex norm](http://mathworld.wolfram.com/ComplexModulus.html).

[embedmd]:# (asm.go go /.*TEXT.*Norm/ /RET.*/)
```go
	TEXT("Norm", NOSPLIT, "func(z complex128) float64")
	Doc("Norm returns the complex norm of z.")
	r = Load(Param("z").Real(), XMM())
	i = Load(Param("z").Imag(), XMM())
	MULSD(r, r)
	MULSD(i, i)
	ADDSD(i, r)
	n := XMM()
	SQRTSD(r, n)
	Store(n, ReturnIndex(0))
	RET()
```

Generated assembly:

[embedmd]:# (complex.s s /.*func Norm/ /RET/)
```s
// func Norm(z complex128) float64
TEXT ·Norm(SB), NOSPLIT, $0-24
	MOVSD  z_real+0(FP), X0
	MOVSD  z_imag+8(FP), X1
	MULSD  X0, X0
	MULSD  X1, X1
	ADDSD  X1, X0
	SQRTSD X0, X2
	MOVSD  X2, ret+16(FP)
	RET
```
