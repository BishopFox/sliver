package gen

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/mmcloughlin/avo/internal/inst"
)

// signature provides access to details about the signature of an instruction function.
type signature interface {
	ParameterList() string
	Arguments() string
	ParameterName(int) string
	ParameterSlice() string
	Length() string
}

// argslist is the signature for a function with the given named parameters.
type argslist []string

func (a argslist) ParameterList() string      { return strings.Join(a, ", ") + " " + operandType }
func (a argslist) Arguments() string          { return strings.Join(a, ", ") }
func (a argslist) ParameterName(i int) string { return a[i] }
func (a argslist) ParameterSlice() string {
	return fmt.Sprintf("[]%s{%s}", operandType, strings.Join(a, ", "))
}
func (a argslist) Length() string { return strconv.Itoa(len(a)) }

// variadic is the signature for a variadic function.
type variadic struct {
	name string
}

func (v variadic) ParameterList() string      { return v.name + " ..." + operandType }
func (v variadic) Arguments() string          { return v.name + "..." }
func (v variadic) ParameterName(i int) string { return fmt.Sprintf("%s[%d]", v.name, i) }
func (v variadic) ParameterSlice() string     { return v.name }
func (v variadic) Length() string             { return fmt.Sprintf("len(%s)", v.name) }

// niladic is the signature for a function with no arguments.
type niladic struct{}

func (n niladic) ParameterList() string      { return "" }
func (n niladic) Arguments() string          { return "" }
func (n niladic) ParameterName(i int) string { panic("niladic function has no parameters") }
func (n niladic) ParameterSlice() string     { return "nil" }
func (n niladic) Length() string             { return "0" }

// params generates the function parameters and a function.
func params(i inst.Instruction) signature {
	// Handle the case of forms with multiple arities.
	switch {
	case i.IsVariadic():
		return variadic{name: "ops"}
	case i.IsNiladic():
		return niladic{}
	}

	// Generate nice-looking variable names.
	n := i.Arity()
	ops := make([]string, n)
	count := map[string]int{}
	for j := 0; j < n; j++ {
		// Collect unique lowercase bytes from first characters of operand types.
		s := map[byte]bool{}
		for _, f := range i.Forms {
			c := f.Operands[j].Type[0]
			if 'a' <= c && c <= 'z' {
				s[c] = true
			}
		}

		// Operand name is the sorted bytes.
		var b []byte
		for c := range s {
			b = append(b, c)
		}
		sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
		name := string(b)

		// Append a counter if we've seen it already.
		m := count[name]
		count[name]++
		if m > 0 {
			name += strconv.Itoa(m)
		}
		ops[j] = name
	}

	return argslist(ops)
}

// doc generates the lines of the function comment.
func doc(i inst.Instruction) []string {
	lines := []string{
		fmt.Sprintf("%s: %s.", i.Opcode, i.Summary),
		"",
		"Forms:",
		"",
	}

	// Write a table of instruction forms.
	buf := bytes.NewBuffer(nil)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	for _, f := range i.Forms {
		row := i.Opcode + "\t" + strings.Join(f.Signature(), "\t") + "\n"
		fmt.Fprint(w, row)
	}
	w.Flush()

	tbl := strings.TrimSpace(buf.String())
	for _, line := range strings.Split(tbl, "\n") {
		lines = append(lines, "\t"+line)
	}

	return lines
}
