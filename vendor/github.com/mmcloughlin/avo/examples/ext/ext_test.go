package ext

import (
	"testing"
	"testing/quick"

	"github.com/mmcloughlin/avo/examples/ext/ext"
)

//go:generate go run asm.go -out ext.s

func TestFunc(t *testing.T) {
	expect := func(e ext.Struct) byte { return e.B }
	if err := quick.CheckEqual(StructFieldB, expect, nil); err != nil {
		t.Fatal(err)
	}
}
