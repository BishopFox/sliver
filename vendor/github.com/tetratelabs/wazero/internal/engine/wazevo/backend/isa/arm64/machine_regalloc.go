package arm64

// This file implements the interfaces required for register allocations. See backend.RegAllocFunctionMachine.

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// ClobberedRegisters implements backend.RegAllocFunctionMachine.
func (m *machine) ClobberedRegisters(regs []regalloc.VReg) {
	m.clobberedRegs = append(m.clobberedRegs[:0], regs...)
}

// Swap implements backend.RegAllocFunctionMachine.
func (m *machine) Swap(cur *instruction, x1, x2, tmp regalloc.VReg) {
	prevNext := cur.next
	var mov1, mov2, mov3 *instruction
	if x1.RegType() == regalloc.RegTypeInt {
		if !tmp.Valid() {
			tmp = tmpRegVReg
		}
		mov1 = m.allocateInstr().asMove64(tmp, x1)
		mov2 = m.allocateInstr().asMove64(x1, x2)
		mov3 = m.allocateInstr().asMove64(x2, tmp)
		cur = linkInstr(cur, mov1)
		cur = linkInstr(cur, mov2)
		cur = linkInstr(cur, mov3)
		linkInstr(cur, prevNext)
	} else {
		if !tmp.Valid() {
			r2 := x2.RealReg()
			// Temporarily spill x1 to stack.
			cur = m.InsertStoreRegisterAt(x1, cur, true).prev
			// Then move x2 to x1.
			cur = linkInstr(cur, m.allocateInstr().asFpuMov128(x1, x2))
			linkInstr(cur, prevNext)
			// Then reload the original value on x1 from stack to r2.
			m.InsertReloadRegisterAt(x1.SetRealReg(r2), cur, true)
		} else {
			mov1 = m.allocateInstr().asFpuMov128(tmp, x1)
			mov2 = m.allocateInstr().asFpuMov128(x1, x2)
			mov3 = m.allocateInstr().asFpuMov128(x2, tmp)
			cur = linkInstr(cur, mov1)
			cur = linkInstr(cur, mov2)
			cur = linkInstr(cur, mov3)
			linkInstr(cur, prevNext)
		}
	}
}

// InsertMoveBefore implements backend.RegAllocFunctionMachine.
func (m *machine) InsertMoveBefore(dst, src regalloc.VReg, instr *instruction) {
	typ := src.RegType()
	if typ != dst.RegType() {
		panic("BUG: src and dst must have the same type")
	}

	mov := m.allocateInstr()
	if typ == regalloc.RegTypeInt {
		mov.asMove64(dst, src)
	} else {
		mov.asFpuMov128(dst, src)
	}

	cur := instr.prev
	prevNext := cur.next
	cur = linkInstr(cur, mov)
	linkInstr(cur, prevNext)
}

// SSABlockLabel implements backend.RegAllocFunctionMachine.
func (m *machine) SSABlockLabel(id ssa.BasicBlockID) backend.Label {
	return m.executableContext.SsaBlockIDToLabels[id]
}

// InsertStoreRegisterAt implements backend.RegAllocFunctionMachine.
func (m *machine) InsertStoreRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
	if !v.IsRealReg() {
		panic("BUG: VReg must be backed by real reg to be stored")
	}

	typ := m.compiler.TypeOf(v)

	var prevNext, cur *instruction
	if after {
		cur, prevNext = instr, instr.next
	} else {
		cur, prevNext = instr.prev, instr
	}

	offsetFromSP := m.getVRegSpillSlotOffsetFromSP(v.ID(), typ.Size())
	var amode addressMode
	cur, amode = m.resolveAddressModeForOffsetAndInsert(cur, offsetFromSP, typ.Bits(), spVReg, true)
	store := m.allocateInstr()
	store.asStore(operandNR(v), amode, typ.Bits())

	cur = linkInstr(cur, store)
	return linkInstr(cur, prevNext)
}

// InsertReloadRegisterAt implements backend.RegAllocFunctionMachine.
func (m *machine) InsertReloadRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
	if !v.IsRealReg() {
		panic("BUG: VReg must be backed by real reg to be stored")
	}

	typ := m.compiler.TypeOf(v)

	var prevNext, cur *instruction
	if after {
		cur, prevNext = instr, instr.next
	} else {
		cur, prevNext = instr.prev, instr
	}

	offsetFromSP := m.getVRegSpillSlotOffsetFromSP(v.ID(), typ.Size())
	var amode addressMode
	cur, amode = m.resolveAddressModeForOffsetAndInsert(cur, offsetFromSP, typ.Bits(), spVReg, true)
	load := m.allocateInstr()
	switch typ {
	case ssa.TypeI32, ssa.TypeI64:
		load.asULoad(operandNR(v), amode, typ.Bits())
	case ssa.TypeF32, ssa.TypeF64:
		load.asFpuLoad(operandNR(v), amode, typ.Bits())
	case ssa.TypeV128:
		load.asFpuLoad(operandNR(v), amode, 128)
	default:
		panic("TODO")
	}

	cur = linkInstr(cur, load)
	return linkInstr(cur, prevNext)
}

// LastInstrForInsertion implements backend.RegAllocFunctionMachine.
func (m *machine) LastInstrForInsertion(begin, end *instruction) *instruction {
	cur := end
	for cur.kind == nop0 {
		cur = cur.prev
		if cur == begin {
			return end
		}
	}
	switch cur.kind {
	case br:
		return cur
	default:
		return end
	}
}
