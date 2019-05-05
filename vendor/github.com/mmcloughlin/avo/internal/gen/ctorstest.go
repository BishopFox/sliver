package gen

import (
	"strings"

	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/printer"

	"github.com/mmcloughlin/avo/internal/inst"
)

type ctorstest struct {
	cfg printer.Config
	prnt.Generator
}

// NewCtorsTest autogenerates tests for the constructors build by NewCtors.
func NewCtorsTest(cfg printer.Config) Interface {
	return GoFmt(&ctorstest{cfg: cfg})
}

func (c *ctorstest) Generate(is []inst.Instruction) ([]byte, error) {
	c.Printf("// %s\n\n", c.cfg.GeneratedWarning())
	c.Printf("package x86\n\n")
	c.Printf("import (\n")
	c.Printf("\t\"testing\"\n")
	c.Printf("\t\"math\"\n")
	c.Printf("\t\"%s/reg\"\n", pkg)
	c.Printf("\t\"%s/operand\"\n", pkg)
	c.Printf(")\n\n")

	for _, i := range is {
		c.instruction(i)
	}

	return c.Result()
}

func (c *ctorstest) instruction(i inst.Instruction) {
	c.Printf("func Test%sValidForms(t *testing.T) {", i.Opcode)

	for _, f := range i.Forms {
		name := strings.Join(f.Signature(), "_")
		c.Printf("t.Run(\"form=%s\", func(t *testing.T) {\n", name)

		for _, args := range validFormArgs(f) {
			c.Printf("if _, err := %s(%s)", i.Opcode, strings.Join(args, ", "))
			c.Printf("; err != nil { t.Fatal(err) }\n")
		}

		c.Printf("})\n")
	}

	c.Printf("}\n\n")
}

func validFormArgs(f inst.Form) [][]string {
	n := len(f.Operands)
	args := make([][]string, n)
	for i, op := range f.Operands {
		valid, ok := validArgs[op.Type]
		if !ok {
			panic("missing operands for type " + op.Type)
		}
		args[i] = valid
	}
	return cross(args)
}

var validArgs = map[string][]string{
	// Immediates
	"1":     {"operand.Imm(1)"},
	"3":     {"operand.Imm(3)"},
	"imm2u": {"operand.Imm(1)", "operand.Imm(3)"},
	"imm8":  {"operand.Imm(math.MaxInt8)"},
	"imm16": {"operand.Imm(math.MaxInt16)"},
	"imm32": {"operand.Imm(math.MaxInt32)"},
	"imm64": {"operand.Imm(math.MaxInt64)"},

	// Registers
	"al":   {"reg.AL"},
	"cl":   {"reg.CL"},
	"ax":   {"reg.AX"},
	"eax":  {"reg.EAX"},
	"rax":  {"reg.RAX"},
	"r8":   {"reg.CH", "reg.BL", "reg.R13B"},
	"r16":  {"reg.CX", "reg.R9W"},
	"r32":  {"reg.R10L"},
	"r64":  {"reg.R11"},
	"xmm0": {"reg.X0"},
	"xmm":  {"reg.X7"},
	"ymm":  {"reg.Y15"},

	// Memory
	"m":    {"operand.Mem{Base: reg.BX, Index: reg.CX, Scale: 2}"},
	"m8":   {"operand.Mem{Base: reg.BL, Index: reg.CH, Scale: 1}"},
	"m16":  {"operand.Mem{Base: reg.BX, Index: reg.CX, Scale: 2}"},
	"m32":  {"operand.Mem{Base: reg.EBX, Index: reg.ECX, Scale: 4}"},
	"m64":  {"operand.Mem{Base: reg.RBX, Index: reg.RCX, Scale: 8}"},
	"m128": {"operand.Mem{Base: reg.RBX, Index: reg.RCX, Scale: 8}"},
	"m256": {"operand.Mem{Base: reg.RBX, Index: reg.RCX, Scale: 8}"},

	// Vector memory
	"vm32x": {"operand.Mem{Base: reg.R13, Index: reg.X4, Scale: 1}"},
	"vm64x": {"operand.Mem{Base: reg.R13, Index: reg.X8, Scale: 1}"},
	"vm32y": {"operand.Mem{Base: reg.R13, Index: reg.Y4, Scale: 1}"},
	"vm64y": {"operand.Mem{Base: reg.R13, Index: reg.Y8, Scale: 1}"},

	// Relative
	"rel8":  {"operand.Rel(math.MaxInt8)"},
	"rel32": {"operand.Rel(math.MaxInt32)", "operand.LabelRef(\"lbl\")"},
}
