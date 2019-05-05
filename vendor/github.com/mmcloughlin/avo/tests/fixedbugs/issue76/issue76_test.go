package issue76

import (
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out issue76.s -stubs stub.go

func TestIssue76(t *testing.T) {
	expect := func(x, y uint64) uint64 { return x + y }
	if err := quick.CheckEqual(Issue76, expect, nil); err != nil {
		t.Fatal(err)
	}
}
