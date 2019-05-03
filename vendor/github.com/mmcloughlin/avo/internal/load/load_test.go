package load_test

import (
	"testing"

	"github.com/mmcloughlin/avo/internal/gen"
	"github.com/mmcloughlin/avo/internal/inst"
	"github.com/mmcloughlin/avo/internal/load"
	"github.com/mmcloughlin/avo/internal/test"
	"github.com/mmcloughlin/avo/printer"
)

func Load(t *testing.T) []inst.Instruction {
	t.Helper()
	l := load.NewLoaderFromDataDir("testdata")
	is, err := l.Load()
	if err != nil {
		t.Fatal(err)
	}
	return is
}

func TestAssembles(t *testing.T) {
	is := Load(t)
	g := gen.NewAsmTest(printer.NewArgvConfig())
	b, err := g.Generate(is)
	if err != nil {
		t.Fatal(err)
	}
	test.Assembles(t, b)
}
