package sum

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out sum.s -stubs stub.go

func TestSum(t *testing.T) {
	expect := func(xs []uint64) uint64 {
		var s uint64
		for _, x := range xs {
			s += x
		}
		return s
	}
	if err := quick.CheckEqual(Sum, expect, nil); err != nil {
		t.Fatal(err)
	}
}
