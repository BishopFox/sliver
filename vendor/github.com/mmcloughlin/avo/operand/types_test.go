package operand

import (
	"reflect"
	"testing"

	"github.com/mmcloughlin/avo/reg"
)

func TestSymbolString(t *testing.T) {
	cases := []struct {
		Symbol Symbol
		Expect string
	}{
		{Symbol{}, ""},
		{Symbol{Name: "name"}, "name"},
		{Symbol{Name: "static", Static: true}, "static<>"},
	}
	for _, c := range cases {
		got := c.Symbol.String()
		if got != c.Expect {
			t.Errorf("%#v.String() = %s expected %s", c.Symbol, got, c.Expect)
		}
	}
}

func TestMemAsm(t *testing.T) {
	cases := []struct {
		Mem    Mem
		Expect string
	}{
		{Mem{Base: reg.EAX}, "(AX)"},
		{Mem{Disp: 16, Base: reg.RAX}, "16(AX)"},
		{Mem{Disp: -7, Base: reg.RAX}, "-7(AX)"},
		{Mem{Base: reg.R11, Index: reg.RAX, Scale: 4}, "(R11)(AX*4)"},
		{Mem{Base: reg.R11, Index: reg.RAX, Scale: 1}, "(R11)(AX*1)"},
		{Mem{Base: reg.R11, Index: reg.RAX}, "(R11)"},
		{Mem{Base: reg.R11, Scale: 8}, "(R11)"},
		{Mem{Disp: 2048, Base: reg.R11, Index: reg.RAX, Scale: 8}, "2048(R11)(AX*8)"},
		{Mem{Symbol: Symbol{Name: "foo"}, Base: reg.StaticBase}, "foo+0(SB)"},
		{Mem{Symbol: Symbol{Name: "foo"}, Base: reg.StaticBase, Disp: 4}, "foo+4(SB)"},
		{Mem{Symbol: Symbol{Name: "foo"}, Base: reg.StaticBase, Disp: -7}, "foo-7(SB)"},
		{Mem{Symbol: Symbol{Name: "bar", Static: true}, Base: reg.StaticBase, Disp: 4, Index: reg.R11, Scale: 4}, "bar<>+4(SB)(R11*4)"},
		{NewParamAddr("param", 16), "param+16(FP)"},
		{NewStackAddr(42), "42(SP)"},
		{NewDataAddr(Symbol{Name: "data", Static: true}, 13), "data<>+13(SB)"},
	}
	for _, c := range cases {
		got := c.Mem.Asm()
		if got != c.Expect {
			t.Errorf("%#v.Asm() = %s expected %s", c.Mem, got, c.Expect)
		}
	}
}

func TestRegisters(t *testing.T) {
	cases := []struct {
		Op     Op
		Expect []reg.Register
	}{
		{reg.R11, []reg.Register{reg.R11}},
		{Mem{Base: reg.EAX}, []reg.Register{reg.EAX}},
		{Mem{Base: reg.RBX, Index: reg.R10}, []reg.Register{reg.RBX, reg.R10}},
		{Imm(42), nil},
		{Rel(42), nil},
		{LabelRef("idk"), nil},
	}
	for _, c := range cases {
		got := Registers(c.Op)
		if !reflect.DeepEqual(got, c.Expect) {
			t.Errorf("Registers(%#v) = %#v expected %#v", c.Op, got, c.Expect)
		}
	}
}
