package gen

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mmcloughlin/avo/internal/inst"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/printer"
)

type mov struct {
	cfg printer.Config
	prnt.Generator
}

// NewMOV generates a function that will auto-select the correct MOV instruction
// based on operand types and and sizes.
func NewMOV(cfg printer.Config) Interface {
	return GoFmt(&mov{cfg: cfg})
}

func (m *mov) Generate(is []inst.Instruction) ([]byte, error) {
	m.Printf("// %s\n\n", m.cfg.GeneratedWarning())
	m.Printf("package build\n\n")

	m.Printf("import (\n")
	m.Printf("\t\"go/types\"\n")
	m.Printf("\t\"%s/operand\"\n", pkg)
	m.Printf(")\n\n")

	m.Printf("func (c *Context) mov(a, b operand.Op, an, bn int, t *types.Basic) {\n")
	m.Printf("switch {\n")
	for _, i := range is {
		if ismov(i) {
			m.instruction(i)
		}
	}
	m.Printf("default:\n")
	m.Printf("c.adderrormessage(\"could not deduce mov instruction\")\n")
	m.Printf("}\n")
	m.Printf("}\n")
	return m.Result()
}

func (m *mov) instruction(i inst.Instruction) {
	f := flags(i)
	sizes, err := formsizes(i)
	if err != nil {
		m.AddError(err)
		return
	}
	for _, size := range sizes {
		conds := []string{
			fmt.Sprintf("an == %d", size.A),
			fmt.Sprintf("bn == %d", size.B),
		}
		for c, on := range f {
			cmp := map[bool]string{true: "!=", false: "=="}
			cond := fmt.Sprintf("(t.Info() & %s) %s 0", c, cmp[on])
			conds = append(conds, cond)
		}
		sort.Strings(conds)
		m.Printf("case %s:\n", strings.Join(conds, " && "))
		m.Printf("c.%s(a, b)\n", i.Opcode)
	}
}

// ismov decides whether the given instruction is a plain move instruction.
func ismov(i inst.Instruction) bool {
	if i.AliasOf != "" || !strings.HasPrefix(i.Opcode, "MOV") {
		return false
	}
	exclude := []string{"Packed", "Duplicate", "Aligned", "Hint", "Swapping"}
	for _, substring := range exclude {
		if strings.Contains(i.Summary, substring) {
			return false
		}
	}
	return true
}

func flags(i inst.Instruction) map[string]bool {
	f := map[string]bool{}
	switch {
	case strings.Contains(i.Summary, "Floating-Point"):
		f["types.IsFloat"] = true
	case strings.Contains(i.Summary, "Zero-Extend"):
		f["types.IsInteger"] = true
		f["types.IsUnsigned"] = true
	case strings.Contains(i.Summary, "Sign-Extension"):
		f["types.IsInteger"] = true
		f["types.IsUnsigned"] = false
	default:
		f["types.IsInteger"] = true
	}
	return f
}

type movsize struct{ A, B int8 }

func (s movsize) sortkey() uint16 { return (uint16(s.A) << 8) | uint16(s.B) }

func formsizes(i inst.Instruction) ([]movsize, error) {
	set := map[movsize]bool{}
	for _, f := range i.Forms {
		if f.Arity() != 2 {
			continue
		}
		s := movsize{
			A: opsize[f.Operands[0].Type],
			B: opsize[f.Operands[1].Type],
		}
		if s.A < 0 || s.B < 0 {
			continue
		}
		if s.A == 0 || s.B == 0 {
			return nil, errors.New("unknown operand type")
		}
		set[s] = true
	}
	var ss []movsize
	for s := range set {
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].sortkey() < ss[j].sortkey() })
	return ss, nil
}

var opsize = map[string]int8{
	"imm8":  -1,
	"imm16": -1,
	"imm32": -1,
	"imm64": -1,
	"r8":    1,
	"r16":   2,
	"r32":   4,
	"r64":   8,
	"xmm":   16,
	"m8":    1,
	"m16":   2,
	"m32":   4,
	"m64":   8,
	"m128":  16,
}
