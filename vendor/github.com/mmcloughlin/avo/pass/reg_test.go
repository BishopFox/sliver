package pass_test

import (
	"testing"

	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/reg"

	"github.com/mmcloughlin/avo/pass"

	"github.com/mmcloughlin/avo/build"
	"github.com/mmcloughlin/avo/operand"
)

func TestLivenessBasic(t *testing.T) {
	// Build: a = 1, b = 2, a = a+b
	ctx := build.NewContext()
	ctx.Function("add")
	a := ctx.GP64()
	b := ctx.GP64()
	ctx.MOVQ(operand.U64(1), a)
	ctx.MOVQ(operand.U64(2), b)
	ctx.ADDQ(a, b)

	AssertLiveness(t, ctx,
		[][]reg.Register{
			{},
			{a},
			{a, b},
		},
		[][]reg.Register{
			{a},
			{a, b},
			{},
		},
	)
}

func AssertLiveness(t *testing.T, ctx *build.Context, in, out [][]reg.Register) {
	fn := ConstructLiveness(t, ctx)
	is := fn.Instructions()

	if len(in) != len(is) || len(out) != len(is) {
		t.Fatalf("%d instructions: %d/%d in/out expectations", len(is), len(in), len(out))
	}

	for idx, i := range is {
		AssertRegistersMatchSet(t, in[idx], i.LiveIn)
		AssertRegistersMatchSet(t, out[idx], i.LiveOut)
	}
}

func AssertRegistersMatchSet(t *testing.T, rs []reg.Register, s reg.Set) {
	if !s.Equals(reg.NewSetFromSlice(rs)) {
		t.Fatalf("register slice does not match set: %#v and %#v", rs, s)
	}
}

func ConstructLiveness(t *testing.T, ctx *build.Context) *ir.Function {
	return BuildFunction(t, ctx, pass.LabelTarget, pass.CFG, pass.Liveness)
}
