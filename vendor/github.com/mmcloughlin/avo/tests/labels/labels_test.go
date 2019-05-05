package labels

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out labels.s -stubs stub.go

func TestLabels(t *testing.T) {
	expect := func() uint64 { return 7 }
	if err := quick.CheckEqual(Labels, expect, nil); err != nil {
		t.Fatal(err)
	}
}
