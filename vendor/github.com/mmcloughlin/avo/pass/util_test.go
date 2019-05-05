package pass_test

import (
	"testing"

	"github.com/mmcloughlin/avo/build"
	"github.com/mmcloughlin/avo/internal/test"
	"github.com/mmcloughlin/avo/ir"
	"github.com/mmcloughlin/avo/pass"
)

// BuildFunction is a helper to compile a build context containing a single
// function and (optionally) apply a list of FunctionPasses to it.
func BuildFunction(t *testing.T, ctx *build.Context, passes ...pass.FunctionPass) *ir.Function {
	t.Helper()

	f, err := ctx.Result()
	if err != nil {
		build.LogError(test.Logger(t), err, 0)
		t.FailNow()
	}

	fns := f.Functions()
	if len(fns) != 1 {
		t.Fatalf("expect 1 function")
	}
	fn := fns[0]

	for _, p := range passes {
		if err := p(fn); err != nil {
			t.Fatal(err)
		}
	}

	return fn
}
