# args

Demonstrates how to reference function parameters in `avo`.

## Basics

Use `Param()` to reference arguments by name. The `Load()` function can be used to load the argument into a register (this will select the correct `MOV` instruction for you). Likewise `Store` and `ReturnIndex` can be used to write the return value. The following function will return its second argument.

[embedmd]:# (asm.go go /.*TEXT.*Second/ /RET.*/)
```go
	TEXT("Second", NOSPLIT, "func(x, y int32) int32")
	y := Load(Param("y"), GP32())
	Store(y, ReturnIndex(0))
	RET()
```

This `avo` code will generate the following assembly. Note that parameter references are named to conform to [`asmdecl`](https://godoc.org/golang.org/x/tools/go/analysis/passes/asmdecl) rules enforced by `go vet`.

[embedmd]:# (args.s s /.*func Second/ /RET/)
```s
// func Second(x int32, y int32) int32
TEXT ·Second(SB), NOSPLIT, $0-12
	MOVL y+4(FP), AX
	MOVL AX, ret+8(FP)
	RET
```

Primitive types can be loaded as above. Other types consist of sub-components which must be loaded into registers independently; for example strings, slices, arrays, structs and complex values.

## Strings and Slices

Strings and slices actually consist of multiple components under the hood: see [`reflect.StringHeader`](https://golang.org/pkg/reflect/#StringHeader) and [`reflect.SliceHeader`](https://golang.org/pkg/reflect/#SliceHeader). The following `avo` code allows you to load the string length.

[embedmd]:# (asm.go go /.*TEXT.*StringLen/ /RET.*/)
```go
	TEXT("StringLen", NOSPLIT, "func(s string) int")
	strlen := Load(Param("s").Len(), GP64())
	Store(strlen, ReturnIndex(0))
	RET()
```

The same code would work for a slice argument. Likewise `Param(...).Base()` and `Param(...).Cap()` will load the base pointer and capacity (slice only).

## Array Indexing

Arrays can be indexed with the `Index()` method. For example, the following returns the third element of the passed array.

[embedmd]:# (asm.go go /.*TEXT.*ArrayThree/ /RET.*/)
```go
	TEXT("ArrayThree", NOSPLIT, "func(a [7]uint64) uint64")
	a3 := Load(Param("a").Index(3), GP64())
	Store(a3, ReturnIndex(0))
	RET()
```

## Struct Fields

Struct fields can be accessed with the `Field()` method. Note that this _requires_ the package to be specified, so that `avo` can parse the type definition. In this example we specify the package with the line:

[embedmd]:# (asm.go go /.*Package\(.*/)
```go
	Package("github.com/mmcloughlin/avo/examples/args")
```

This package contains the struct definition:

[embedmd]:# (args.go go /type Struct/ /^}/)
```go
type Struct struct {
	Byte       byte
	Int8       int8
	Uint16     uint16
	Int32      int32
	Uint64     uint64
	Float32    float32
	Float64    float64
	String     string
	Slice      []Sub
	Array      [5]Sub
	Complex64  complex64
	Complex128 complex128
}
```

The following function will return the `Float64` field from this struct.

[embedmd]:# (asm.go go /.*TEXT.*FieldFloat64/ /RET.*/)
```go
	TEXT("FieldFloat64", NOSPLIT, "func(s Struct) float64")
	f64 := Load(Param("s").Field("Float64"), XMM())
	Store(f64, ReturnIndex(0))
	RET()
```

## Complex Values

Complex types `complex{64,128}` are actually just pairs of `float{32,64}` values. These can be accessed with the `Real()` and `Imag()` methods. For example the following function returns the imaginary part of the `Complex64` struct field.

[embedmd]:# (asm.go go /.*TEXT.*FieldComplex64Imag/ /RET.*/)
```go
	TEXT("FieldComplex64Imag", NOSPLIT, "func(s Struct) float32")
	c64i := Load(Param("s").Field("Complex64").Imag(), XMM())
	Store(c64i, ReturnIndex(0))
	RET()
```

## Nested Data Structures

The above methods may be composed to reference arbitrarily nested data structures. For example, the following returns `s.Array[2].B[2]`.

[embedmd]:# (asm.go go /.*TEXT.*FieldArrayTwoBTwo/ /RET.*/)
```go
	TEXT("FieldArrayTwoBTwo", NOSPLIT, "func(s Struct) byte")
	b2 := Load(Param("s").Field("Array").Index(2).Field("B").Index(2), GP8())
	Store(b2, ReturnIndex(0))
	RET()
```
