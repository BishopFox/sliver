package backend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// SSAValueDefinition represents a definition of an SSA value.
type SSAValueDefinition struct {
	// BlockParamValue is valid if Instr == nil
	BlockParamValue ssa.Value

	// BlkParamVReg is valid if Instr == nil
	BlkParamVReg regalloc.VReg

	// Instr is not nil if this is a definition from an instruction.
	Instr *ssa.Instruction
	// N is the index of the return value in the instr's return values list.
	N int
	// RefCount is the number of references to the result.
	RefCount int
}

func (d *SSAValueDefinition) IsFromInstr() bool {
	return d.Instr != nil
}

func (d *SSAValueDefinition) IsFromBlockParam() bool {
	return d.Instr == nil
}

func (d *SSAValueDefinition) SSAValue() ssa.Value {
	if d.IsFromBlockParam() {
		return d.BlockParamValue
	} else {
		r, rs := d.Instr.Returns()
		if d.N == 0 {
			return r
		} else {
			return rs[d.N-1]
		}
	}
}
