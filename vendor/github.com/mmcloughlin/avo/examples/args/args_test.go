package args

import (
	"math"
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out args.s -stubs stub.go

func TestFunctionsEqual(t *testing.T) {
	cases := []struct {
		f, g interface{}
	}{
		{Second, func(x, y int32) int32 { return y }},
		{StringLen, func(s string) int { return len(s) }},
		{SliceLen, func(s []int) int { return len(s) }},
		{SliceCap, func(s []int) int { return cap(s) }},
		{ArrayThree, func(a [7]uint64) uint64 { return a[3] }},
		{FieldByte, func(s Struct) byte { return s.Byte }},
		{FieldInt8, func(s Struct) int8 { return s.Int8 }},
		{FieldUint16, func(s Struct) uint16 { return s.Uint16 }},
		{FieldInt32, func(s Struct) int32 { return s.Int32 }},
		{FieldUint64, func(s Struct) uint64 { return s.Uint64 }},
		{FieldFloat32, func(s Struct) float32 { return s.Float32 }},
		{FieldFloat64, func(s Struct) float64 { return s.Float64 }},
		{FieldStringLen, func(s Struct) int { return len(s.String) }},
		{FieldSliceCap, func(s Struct) int { return cap(s.Slice) }},
		{FieldArrayTwoBTwo, func(s Struct) byte { return s.Array[2].B[2] }},
		{FieldArrayOneC, func(s Struct) uint16 { return s.Array[1].C }},
		{FieldComplex64Imag, func(s Struct) float32 { return imag(s.Complex64) }},
		{FieldComplex128Real, func(s Struct) float64 { return real(s.Complex128) }},
	}
	for _, c := range cases {
		if err := quick.CheckEqual(c.f, c.g, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDereferenceFloat32(t *testing.T) {
	expect := float32(math.Pi)
	s := Struct{Float32: expect}
	got := DereferenceFloat32(&s)
	if got != expect {
		t.Errorf("DereferenceFloat32(%v) = %v; expect %v", s, got, expect)
	}
}
