package backend

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// FunctionABI represents an ABI for the specific target combined with the function signature.
type FunctionABI interface {
	// CalleeGenFunctionArgsToVRegs generates instructions to move arguments to virtual registers.
	CalleeGenFunctionArgsToVRegs(regs []ssa.Value)
	// CalleeGenVRegsToFunctionReturns generates instructions to move virtual registers to a return value locations.
	CalleeGenVRegsToFunctionReturns(regs []ssa.Value)
}

type (
	// ABIArg represents either argument or return value's location.
	ABIArg struct {
		// Index is the index of the argument.
		Index int
		// Kind is the kind of the argument.
		Kind ABIArgKind
		// Reg is valid if Kind == ABIArgKindReg.
		// This VReg must be based on RealReg.
		Reg regalloc.VReg
		// Offset is valid if Kind == ABIArgKindStack.
		// This is the offset from the beginning of either arg or ret stack slot.
		Offset int64
		// Type is the type of the argument.
		Type ssa.Type
	}

	// ABIArgKind is the kind of ABI argument.
	ABIArgKind byte
)

const (
	// ABIArgKindReg represents an argument passed in a register.
	ABIArgKindReg = iota
	// ABIArgKindStack represents an argument passed in the stack.
	ABIArgKindStack
)

// String implements fmt.Stringer.
func (a *ABIArg) String() string {
	return fmt.Sprintf("args[%d]: %s", a.Index, a.Kind)
}

// String implements fmt.Stringer.
func (a ABIArgKind) String() string {
	switch a {
	case ABIArgKindReg:
		return "reg"
	case ABIArgKindStack:
		return "stack"
	default:
		panic("BUG")
	}
}
