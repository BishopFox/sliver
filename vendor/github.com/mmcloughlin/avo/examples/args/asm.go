// +build ignore

package main

import . "github.com/mmcloughlin/avo/build"

func main() {
	Package("github.com/mmcloughlin/avo/examples/args")

	TEXT("Second", NOSPLIT, "func(x, y int32) int32")
	y := Load(Param("y"), GP32())
	Store(y, ReturnIndex(0))
	RET()

	TEXT("StringLen", NOSPLIT, "func(s string) int")
	strlen := Load(Param("s").Len(), GP64())
	Store(strlen, ReturnIndex(0))
	RET()

	TEXT("SliceLen", NOSPLIT, "func(s []int) int")
	slicelen := Load(Param("s").Len(), GP64())
	Store(slicelen, ReturnIndex(0))
	RET()

	TEXT("SliceCap", NOSPLIT, "func(s []int) int")
	slicecap := Load(Param("s").Cap(), GP64())
	Store(slicecap, ReturnIndex(0))
	RET()

	TEXT("ArrayThree", NOSPLIT, "func(a [7]uint64) uint64")
	a3 := Load(Param("a").Index(3), GP64())
	Store(a3, ReturnIndex(0))
	RET()

	TEXT("FieldByte", NOSPLIT, "func(s Struct) byte")
	b := Load(Param("s").Field("Byte"), GP8())
	Store(b, ReturnIndex(0))
	RET()

	TEXT("FieldInt8", NOSPLIT, "func(s Struct) int8")
	i8 := Load(Param("s").Field("Int8"), GP8())
	Store(i8, ReturnIndex(0))
	RET()

	TEXT("FieldUint16", NOSPLIT, "func(s Struct) uint16")
	u16 := Load(Param("s").Field("Uint16"), GP16())
	Store(u16, ReturnIndex(0))
	RET()

	TEXT("FieldInt32", NOSPLIT, "func(s Struct) int32")
	i32 := Load(Param("s").Field("Int32"), GP32())
	Store(i32, ReturnIndex(0))
	RET()

	TEXT("FieldUint64", NOSPLIT, "func(s Struct) uint64")
	u64 := Load(Param("s").Field("Uint64"), GP64())
	Store(u64, ReturnIndex(0))
	RET()

	TEXT("FieldFloat32", NOSPLIT, "func(s Struct) float32")
	f32 := Load(Param("s").Field("Float32"), XMM())
	Store(f32, ReturnIndex(0))
	RET()

	TEXT("FieldFloat64", NOSPLIT, "func(s Struct) float64")
	f64 := Load(Param("s").Field("Float64"), XMM())
	Store(f64, ReturnIndex(0))
	RET()

	TEXT("FieldStringLen", NOSPLIT, "func(s Struct) int")
	l := Load(Param("s").Field("String").Len(), GP64())
	Store(l, ReturnIndex(0))
	RET()

	TEXT("FieldSliceCap", NOSPLIT, "func(s Struct) int")
	c := Load(Param("s").Field("Slice").Cap(), GP64())
	Store(c, ReturnIndex(0))
	RET()

	TEXT("FieldArrayTwoBTwo", NOSPLIT, "func(s Struct) byte")
	b2 := Load(Param("s").Field("Array").Index(2).Field("B").Index(2), GP8())
	Store(b2, ReturnIndex(0))
	RET()

	TEXT("FieldArrayOneC", NOSPLIT, "func(s Struct) uint16")
	c1 := Load(Param("s").Field("Array").Index(1).Field("C"), GP16())
	Store(c1, ReturnIndex(0))
	RET()

	TEXT("FieldComplex64Imag", NOSPLIT, "func(s Struct) float32")
	c64i := Load(Param("s").Field("Complex64").Imag(), XMM())
	Store(c64i, ReturnIndex(0))
	RET()

	TEXT("FieldComplex128Real", NOSPLIT, "func(s Struct) float64")
	c128r := Load(Param("s").Field("Complex128").Real(), XMM())
	Store(c128r, ReturnIndex(0))
	RET()

	TEXT("DereferenceFloat32", NOSPLIT, "func(s *Struct) float32")
	s := Dereference(Param("s"))
	f := Load(s.Field("Float32"), XMM())
	Store(f, ReturnIndex(0))

	RET()
	Generate()
}
