package gen

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/mmcloughlin/avo/internal/inst"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/printer"
)

type asmtest struct {
	cfg   printer.Config
	sym   string // reference to the test function symbol
	rel8  string // label to be used for near jumps
	rel32 string // label for far jumps
	prnt.Generator
}

// NewAsmTest prints one massive assembly function containing a line for every
// instruction form in the database. The intention is to pass this to the Go
// assembler and confirm there are no errors, thus helping to ensure our
// database is compatible.
func NewAsmTest(cfg printer.Config) Interface {
	return &asmtest{cfg: cfg}
}

func (a *asmtest) Generate(is []inst.Instruction) ([]byte, error) {
	a.Printf("// %s\n\n", a.cfg.GeneratedWarning())

	a.sym = "\u00b7loadertest(SB)"
	a.Printf("TEXT %s, 0, $0\n", a.sym)

	// Define a label for far jumps.
	a.Printf("rel32:\n")
	a.rel32 = "rel32"

	counts := map[string]int{}

	for _, i := range is {
		a.Printf("\t// %s %s\n", i.Opcode, i.Summary)
		if skip, msg := a.skip(i.Opcode); skip {
			a.Printf("\t// SKIP: %s\n", msg)
			counts["skip"]++
			continue
		}

		if i.Opcode[0] == 'J' {
			label := fmt.Sprintf("rel8_%s", strings.ToLower(i.Opcode))
			a.Printf("%s:\n", label)
			a.rel8 = label
		}

		for _, f := range i.Forms {
			as := a.args(i.Opcode, f.Operands)
			if as == nil {
				a.Printf("\t// TODO: %s %#v\n", i.Opcode, f.Operands)
				counts["todo"]++
				continue
			}
			a.Printf("\t%s\t%s\n", i.Opcode, strings.Join(as, ", "))
			counts["total"]++
		}
		a.Printf("\n")
	}

	a.Printf("\tRET\n")

	for m, c := range counts {
		a.Printf("// %s: %d\n", m, c)
	}

	return a.Result()
}

func (a asmtest) skip(opcode string) (bool, string) {
	prefixes := map[string]string{
		"PUSH": "PUSH can produce 'unbalanced PUSH/POP' assembler error",
		"POP":  "POP can produce 'unbalanced PUSH/POP' assembler error",
	}
	for p, m := range prefixes {
		if strings.HasPrefix(opcode, p) {
			return true, m
		}
	}
	return false, ""
}

func (a asmtest) args(opcode string, ops []inst.Operand) []string {
	// Special case for CALL, since it needs a different type of rel32 argument than others.
	if opcode == "CALL" {
		return []string{a.sym}
	}

	as := make([]string, len(ops))
	for i, op := range ops {
		a := a.arg(op.Type, i)
		if a == "" {
			return nil
		}
		as[i] = a
	}
	return as
}

// arg generates an argument for an operand of the given type.
func (a asmtest) arg(t string, i int) string {
	m := map[string]string{
		"1":     "$1", // <xs:enumeration value="1" />
		"3":     "$3", // <xs:enumeration value="3" />
		"imm2u": "$3",
		// <xs:enumeration value="imm4" />
		"imm8":  fmt.Sprintf("$%d", math.MaxInt8),  // <xs:enumeration value="imm8" />
		"imm16": fmt.Sprintf("$%d", math.MaxInt16), // <xs:enumeration value="imm16" />
		"imm32": fmt.Sprintf("$%d", math.MaxInt32), // <xs:enumeration value="imm32" />
		"imm64": fmt.Sprintf("$%d", math.MaxInt64), // <xs:enumeration value="imm64" />

		"al":   "AL",                    // <xs:enumeration value="al" />
		"cl":   "CL",                    // <xs:enumeration value="cl" />
		"r8":   "CH",                    // <xs:enumeration value="r8" />
		"ax":   "AX",                    // <xs:enumeration value="ax" />
		"r16":  "SI",                    // <xs:enumeration value="r16" />
		"eax":  "AX",                    // <xs:enumeration value="eax" />
		"r32":  "DX",                    // <xs:enumeration value="r32" />
		"rax":  "AX",                    // <xs:enumeration value="rax" />
		"r64":  "R15",                   // <xs:enumeration value="r64" />
		"mm":   "M5",                    // <xs:enumeration value="mm" />
		"xmm0": "X0",                    // <xs:enumeration value="xmm0" />
		"xmm":  "X" + strconv.Itoa(7+i), // <xs:enumeration value="xmm" />
		// <xs:enumeration value="xmm{k}" />
		// <xs:enumeration value="xmm{k}{z}" />
		"ymm": "Y" + strconv.Itoa(3+i), // <xs:enumeration value="ymm" />
		// <xs:enumeration value="ymm{k}" />
		// <xs:enumeration value="ymm{k}{z}" />
		// <xs:enumeration value="zmm" />
		// <xs:enumeration value="zmm{k}" />
		// <xs:enumeration value="zmm{k}{z}" />
		// <xs:enumeration value="k" />
		// <xs:enumeration value="k{k}" />
		// <xs:enumeration value="moffs32" />
		// <xs:enumeration value="moffs64" />
		"m":   "0(AX)(CX*2)",  // <xs:enumeration value="m" />
		"m8":  "8(AX)(CX*2)",  // <xs:enumeration value="m8" />
		"m16": "16(AX)(CX*2)", // <xs:enumeration value="m16" />
		// <xs:enumeration value="m16{k}{z}" />
		"m32": "32(AX)(CX*2)", // <xs:enumeration value="m32" />
		// <xs:enumeration value="m32{k}" />
		// <xs:enumeration value="m32{k}{z}" />
		"m64": "64(AX)(CX*2)", // <xs:enumeration value="m64" />
		// <xs:enumeration value="m64{k}" />
		// <xs:enumeration value="m64{k}{z}" />
		"m128": "128(AX)(CX*2)", // <xs:enumeration value="m128" />
		// <xs:enumeration value="m128{k}{z}" />
		"m256": "256(AX)(CX*2)", // <xs:enumeration value="m256" />
		// <xs:enumeration value="m256{k}{z}" />
		// <xs:enumeration value="m512" />
		// <xs:enumeration value="m512{k}{z}" />
		// <xs:enumeration value="m64/m32bcst" />
		// <xs:enumeration value="m128/m32bcst" />
		// <xs:enumeration value="m256/m32bcst" />
		// <xs:enumeration value="m512/m32bcst" />
		// <xs:enumeration value="m128/m64bcst" />
		// <xs:enumeration value="m256/m64bcst" />
		// <xs:enumeration value="m512/m64bcst" />
		"vm32x": "32(X14*8)", // <xs:enumeration value="vm32x" />
		// <xs:enumeration value="vm32x{k}" />
		"vm64x": "64(X14*8)", // <xs:enumeration value="vm64x" />
		// <xs:enumeration value="vm64x{k}" />
		"vm32y": "32(Y13*8)", // <xs:enumeration value="vm32y" />
		// <xs:enumeration value="vm32y{k}" />
		"vm64y": "64(Y13*8)", // <xs:enumeration value="vm64y" />
		// <xs:enumeration value="vm64y{k}" />
		// <xs:enumeration value="vm32z" />
		// <xs:enumeration value="vm32z{k}" />
		// <xs:enumeration value="vm64z" />
		// <xs:enumeration value="vm64z{k}" />
		"rel8":  a.rel8,  // <xs:enumeration value="rel8" />
		"rel32": a.rel32, // <xs:enumeration value="rel32" />
		// <xs:enumeration value="{er}" />
		// <xs:enumeration value="{sae}" />

		// Appear unused:
		"r8l":  "????", // <xs:enumeration value="r8l" />
		"r16l": "????", // <xs:enumeration value="r16l" />
		"r32l": "????", // <xs:enumeration value="r32l" />
	}
	return m[t]
}
