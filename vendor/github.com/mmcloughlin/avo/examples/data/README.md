# data

Constructing data sections in `avo`.

A data section is declared with the `GLOBL` function, and values are specified with `DATA`. The size of the data section will grow as values are added to it.

[embedmd]:# (asm.go go /.*GLOBL\(/ /^$/)
```go
	bytes := GLOBL("bytes", RODATA|NOPTR)
	DATA(0, U64(0x0011223344556677))
	DATA(8, String("strconst"))
	DATA(16, F32(math.Pi))
	DATA(24, F64(math.Pi))
	DATA(32, U32(0x00112233))
	DATA(36, U16(0x4455))
	DATA(38, U8(0x66))
	DATA(39, U8(0x77))
```

The `GLOBL` function returns a reference which may be used in assembly code. The following function indexes into the data section just created.

[embedmd]:# (asm.go go /.*TEXT.*DataAt/ /RET.*/)
```go
	TEXT("DataAt", NOSPLIT, "func(i int) byte")
	Doc("DataAt returns byte i in the 'bytes' global data section.")
	i := Load(Param("i"), GP64())
	ptr := Mem{Base: GP64()}
	LEAQ(bytes, ptr.Base)
	b := GP8()
	MOVB(ptr.Idx(i, 1), b)
	Store(b, ReturnIndex(0))
	RET()
```
