package printer_test

import (
	"strings"
	"testing"

	"github.com/mmcloughlin/avo/build"
	"github.com/mmcloughlin/avo/printer"
)

func AssertPrintsLines(t *testing.T, ctx *build.Context, pb printer.Builder, expect []string) {
	t.Helper()

	output := Print(t, ctx, pb)
	lines := strings.Split(output, "\n")

	if len(expect) != len(lines) {
		t.Fatalf("have %d lines of output; expected %d", len(lines), len(expect))
	}

	for i := range expect {
		if expect[i] != lines[i] {
			t.Errorf("mismatch on line %d:\n\tgot\t%s\n\texpect\t%s\n", i, lines[i], expect[i])
		}
	}
}

func Print(t *testing.T, ctx *build.Context, pb printer.Builder) string {
	t.Helper()

	f, errs := ctx.Result()
	if errs != nil {
		t.Fatal(errs)
	}

	p := pb(printer.NewDefaultConfig())
	b, err := p.Print(f)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}
