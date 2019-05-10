package gen

import (
	"fmt"
	"strings"
)

const (
	pkg         = "github.com/mmcloughlin/avo"
	operandType = "operand.Op"
)

// implicitRegister returns avo syntax for the given implicit register type (from Opcodes XML).
func implicitRegister(t string) string {
	r := strings.Replace(t, "mm", "", 1) // handles the "xmm0" type
	return fmt.Sprintf("reg.%s", strings.ToUpper(r))
}

// checkername returns the name of the function that checks an operand of type t.
func checkername(t string) string {
	return "operand.Is" + strings.ToUpper(t)
}
