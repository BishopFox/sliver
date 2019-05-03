package pass_test

import (
	"reflect"
	"testing"

	"github.com/mmcloughlin/avo/build"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/pass"
	"github.com/mmcloughlin/avo/reg"
)

func TestPruneSelfMoves(t *testing.T) {
	// Construct a function containing a self-move.
	ctx := build.NewContext()
	ctx.Function("add")
	ctx.MOVQ(operand.U64(1), reg.RAX)
	ctx.MOVQ(operand.U64(2), reg.RCX)
	ctx.MOVQ(reg.RAX, reg.RAX) // self move
	ctx.MOVQ(reg.RCX, reg.R8)
	ctx.ADDQ(reg.R8, reg.RAX)

	// Build the function without the pass and save the nodes.
	fn := BuildFunction(t, ctx)
	pre := append([]ir.Node{}, fn.Nodes...)

	// Apply the pass.
	if err := pass.PruneSelfMoves(fn); err != nil {
		t.Fatal(err)
	}

	// Confirm the self-move was removed and everything else was untouched.
	expect := []ir.Node{}
	for i, n := range pre {
		if i != 2 {
			expect = append(expect, n)
		}
	}

	if !reflect.DeepEqual(fn.Nodes, expect) {
		t.Fatal("unexpected result from self-move pruning")
	}
}

func TestPruneJumpToFollowingLabel(t *testing.T) {
	// Construct a function containing a jump to following.
	ctx := build.NewContext()
	ctx.Function("add")
	ctx.XORQ(reg.RAX, reg.RAX)
	ctx.JMP(operand.LabelRef("next"))
	ctx.Label("next")
	ctx.XORQ(reg.RAX, reg.RAX)

	// Build the function with the PruneJumpToFollowingLabel pass.
	fn := BuildFunction(t, ctx, pass.PruneJumpToFollowingLabel)

	// Confirm no JMP instruction remains.
	for _, i := range fn.Instructions() {
		if i.Opcode == "JMP" {
			t.Fatal("JMP instruction not removed")
		}
	}
}

func TestPruneDanglingLabels(t *testing.T) {
	// Construct a function containing an unreferenced label.
	ctx := build.NewContext()
	ctx.Function("add")
	ctx.XORQ(reg.RAX, reg.RAX)
	ctx.JMP(operand.LabelRef("referenced"))
	ctx.XORQ(reg.RAX, reg.RAX)
	ctx.Label("dangling")
	ctx.XORQ(reg.RAX, reg.RAX)
	ctx.Label("referenced")
	ctx.XORQ(reg.RAX, reg.RAX)

	// Build the function with the PruneDanglingLabels pass.
	fn := BuildFunction(t, ctx, pass.PruneDanglingLabels)

	// Confirm the only label remaining is "referenced".
	expect := []ir.Label{"referenced"}
	if !reflect.DeepEqual(expect, fn.Labels()) {
		t.Fatal("expected dangling label to be removed")
	}
}
