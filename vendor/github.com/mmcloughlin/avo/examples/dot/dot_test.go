package dot

import (
	"math/rand"
	"testing"
)

//go:generate go run asm.go -out dot.s -stubs stub.go

func TestEmpty(t *testing.T) {
	if Dot(nil, nil) != 0.0 {
		t.Fatal("expect dot product of empty vectors to be zero")
	}
}

func TestLengths(t *testing.T) {
	const epsilon = 0.00001
	for n := 0; n < 1000; n++ {
		x, y := RandomVector(n), RandomVector(n)
		got := Dot(x, y)
		expect := Expect(x, y)
		relerr := got/expect - 1.0
		if Abs(relerr) > epsilon {
			t.Fatalf("bad result on vector length %d: got %v expect %v relative error %f", n, got, expect, relerr)
		}
	}
}

func Expect(x, y []float32) float32 {
	var p float32
	for i := range x {
		p += x[i] * y[i]
	}
	return p
}

func RandomVector(n int) []float32 {
	x := make([]float32, n)
	for i := 0; i < n; i++ {
		x[i] = rand.Float32() * 100
	}
	return x
}

func Abs(x float32) float32 {
	if x < 0.0 {
		return -x
	}
	return x
}
