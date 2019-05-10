package inst

import (
	"sort"
	"strings"
)

// Instruction represents an x86 instruction.
type Instruction struct {
	Opcode  string // Golang assembly mnemonic
	AliasOf string // Opcode of instruction that this is an alias for
	Summary string // Description of the instruction
	Forms   []Form // Accepted operand forms
}

// IsTerminal reports whether the instruction exits a function.
func (i Instruction) IsTerminal() bool {
	// TODO(mbm): how about the RETF* instructions
	return i.Opcode == "RET"
}

// IsBranch reports whether the instruction is a branch; that is, if it can
// cause control flow to jump to another location.
func (i Instruction) IsBranch() bool {
	if i.Opcode == "CALL" {
		return false
	}
	for _, f := range i.Forms {
		for _, op := range f.Operands {
			if strings.HasPrefix(op.Type, "rel") {
				return true
			}
		}
	}
	return false
}

// IsConditionalBranch reports whether the instruction branches dependent on some condition.
func (i Instruction) IsConditionalBranch() bool {
	return i.IsBranch() && i.Opcode != "JMP"
}

// Arities returns the unique arities among the instruction forms.
func (i Instruction) Arities() []int {
	s := map[int]bool{}
	for _, f := range i.Forms {
		s[f.Arity()] = true
	}
	a := make([]int, 0, len(s))
	for n := range s {
		a = append(a, n)
	}
	sort.Ints(a)
	return a
}

// Arity is a convenience for returning the unique instruction arity when you
// know it is not variadic. Panics for a variadic instruction.
func (i Instruction) Arity() int {
	if i.IsVariadic() {
		panic("variadic")
	}
	a := i.Arities()
	return a[0]
}

// IsVariadic reports whether the instruction has more than one arity.
func (i Instruction) IsVariadic() bool {
	return len(i.Arities()) > 1
}

// IsNiladic reports whether the instruction takes no operands.
func (i Instruction) IsNiladic() bool {
	a := i.Arities()
	return len(a) == 1 && a[0] == 0
}

// Form specifies one accepted set of operands for an instruction.
type Form struct {
	// Instruction sets this instruction form requires.
	ISA []string
	// Operands required for this form.
	Operands []Operand
	// Registers read or written but not explicitly passed to the instruction.
	ImplicitOperands []ImplicitOperand
}

// Arity returns the number of operands this form expects.
func (f Form) Arity() int {
	return len(f.Operands)
}

// Signature returns the list of operand types.
func (f Form) Signature() []string {
	s := make([]string, f.Arity())
	for i, op := range f.Operands {
		s[i] = op.Type
	}
	return s
}

// Operand is an operand to an instruction, describing the expected type and read/write action.
type Operand struct {
	Type   string
	Action Action
}

// ImplicitOperand describes a register that is implicitly read/written by an instruction.
type ImplicitOperand struct {
	Register string
	Action   Action
}

// Action specifies the read/write operation of an instruction on an operand.
type Action uint8

// Possible Action types.
const (
	R  Action = 0x1
	W  Action = 0x2
	RW Action = R | W
)

// ActionFromReadWrite builds an Action from boolean flags.
func ActionFromReadWrite(r, w bool) Action {
	var a Action
	if r {
		a |= R
	}
	if w {
		a |= W
	}
	return a
}

// Contains reports whether a supports all actions in s.
func (a Action) Contains(s Action) bool {
	return (a & s) == s
}

// Read reports whether a supports read.
func (a Action) Read() bool {
	return a.Contains(R)
}

// Write reports whether a supports write.
func (a Action) Write() bool {
	return a.Contains(W)
}

// String represents a as a human-readable string.
func (a Action) String() string {
	s := ""
	if a.Read() {
		s += "r"
	}
	if a.Write() {
		s += "w"
	}
	return s
}
