package cast

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out cast.s -stubs stub.go

func TestSplit(t *testing.T) {
	expect := func(x uint64) (uint64, uint32, uint16, uint8) {
		return x, uint32(x), uint16(x), uint8(x)
	}
	if err := quick.CheckEqual(Split, expect, nil); err != nil {
		t.Fatal(err)
	}
}
