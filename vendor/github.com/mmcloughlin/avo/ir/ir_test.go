package ir

import (
	"reflect"
	"testing"
)

func TestFunctionLabels(t *testing.T) {
	f := NewFunction("labels")
	f.AddInstruction(&Instruction{})
	f.AddLabel("a")
	f.AddInstruction(&Instruction{})
	f.AddLabel("b")
	f.AddInstruction(&Instruction{})
	f.AddLabel("c")
	f.AddInstruction(&Instruction{})

	expect := []Label{"a", "b", "c"}
	if got := f.Labels(); !reflect.DeepEqual(expect, got) {
		t.Fatalf("f.Labels() = %v; expect %v", got, expect)
	}
}
