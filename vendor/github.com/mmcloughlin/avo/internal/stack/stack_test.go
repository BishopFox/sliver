package stack_test

import (
	"runtime"
	"testing"

	"github.com/mmcloughlin/avo/internal/stack"
)

const pkg = "github.com/mmcloughlin/avo/internal/stack_test"

func TestFramesFirst(t *testing.T) {
	fs := stack.Frames(0, 1)
	if len(fs) == 0 {
		t.Fatalf("empty slice")
	}
	got := fs[0].Function
	expect := pkg + ".TestFramesFirst"
	if got != expect {
		t.Fatalf("bad function name %s; expect %s", got, expect)
	}
}

func TestMatchFirst(t *testing.T) {
	first := stack.Match(0, func(_ runtime.Frame) bool { return true })
	if first == nil {
		t.Fatalf("nil match")
	}
	got := first.Function
	expect := pkg + ".TestMatchFirst"
	if got != expect {
		t.Fatalf("bad function name %s; expect %s", got, expect)
	}
}
