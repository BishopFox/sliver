package issue50

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out issue50.s -stubs stub.go

func TestIssue50(t *testing.T) {
	expect := func(x uint32) uint32 { return x }
	if err := quick.CheckEqual(Issue50, expect, nil); err != nil {
		t.Fatal(err)
	}
}
