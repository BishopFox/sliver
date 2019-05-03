package gen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mmcloughlin/avo/internal/inst"
	"github.com/mmcloughlin/avo/internal/prnt"
	"github.com/mmcloughlin/avo/printer"
)

type ctors struct {
	cfg printer.Config
	prnt.Generator
}

// NewCtors will build instruction constructors. Each constructor will check
// that the provided operands match one of the allowed instruction forms. If so
// it will return an Instruction object that can be added to an avo Function.
func NewCtors(cfg printer.Config) Interface {
	return GoFmt(&ctors{cfg: cfg})
}

func (c *ctors) Generate(is []inst.Instruction) ([]byte, error) {
	c.Printf("// %s\n\n", c.cfg.GeneratedWarning())
	c.Printf("package x86\n\n")
	c.Printf("import (\n")
	c.Printf("\t\"errors\"\n")
	c.Printf("\tintrep \"%s/ir\"\n", pkg)
	c.Printf("\t\"%s/reg\"\n", pkg)
	c.Printf("\t\"%s/operand\"\n", pkg)
	c.Printf(")\n\n")

	for _, i := range is {
		c.instruction(i)
	}

	return c.Result()
}

func (c *ctors) instruction(i inst.Instruction) {
	c.Comment(doc(i)...)

	s := params(i)

	c.Printf("func %s(%s) (*intrep.Instruction, error) {\n", i.Opcode, s.ParameterList())
	c.forms(i, s)
	c.Printf("}\n\n")
}

func (c *ctors) forms(i inst.Instruction, s signature) {
	if i.IsNiladic() {
		if len(i.Forms) != 1 {
			c.AddError(fmt.Errorf("%s breaks assumption that niladic instructions have one form", i.Opcode))
		}
		c.Printf("return &%s, nil\n", construct(i, i.Forms[0], s))
		return
	}

	c.Printf("switch {\n")

	for _, f := range i.Forms {
		var conds []string

		if i.IsVariadic() {
			checklen := fmt.Sprintf("%s == %d", s.Length(), len(f.Operands))
			conds = append(conds, checklen)
		}

		for j, op := range f.Operands {
			checktype := fmt.Sprintf("%s(%s)", checkername(op.Type), s.ParameterName(j))
			conds = append(conds, checktype)
		}

		c.Printf("case %s:\n", strings.Join(conds, " && "))
		c.Printf("return &%s, nil\n", construct(i, f, s))
	}

	c.Printf("}\n")
	c.Printf("return nil, errors.New(\"%s: bad operands\")\n", i.Opcode)
}

func construct(i inst.Instruction, f inst.Form, s signature) string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "intrep.Instruction{\n")
	fmt.Fprintf(buf, "\tOpcode: %#v,\n", i.Opcode)
	fmt.Fprintf(buf, "\tOperands: %s,\n", s.ParameterSlice())

	// Input output.
	fmt.Fprintf(buf, "\tInputs: %s,\n", operandsWithAction(f, inst.R, s))
	fmt.Fprintf(buf, "\tOutputs: %s,\n", operandsWithAction(f, inst.W, s))

	// Branch variables.
	if i.IsBranch() {
		fmt.Fprintf(buf, "\tIsBranch: true,\n")
		fmt.Fprintf(buf, "\tIsConditional: %#v,\n", i.IsConditionalBranch())
	}

	fmt.Fprintf(buf, "}")
	return buf.String()
}

func operandsWithAction(f inst.Form, a inst.Action, s signature) string {
	opexprs := []string{}
	for i, op := range f.Operands {
		if op.Action.Contains(a) {
			opexprs = append(opexprs, s.ParameterName(i))
		}
	}
	for _, op := range f.ImplicitOperands {
		if op.Action.Contains(a) {
			opexprs = append(opexprs, implicitRegister(op.Register))
		}
	}
	return fmt.Sprintf("[]%s{%s}", operandType, strings.Join(opexprs, ", "))
}
