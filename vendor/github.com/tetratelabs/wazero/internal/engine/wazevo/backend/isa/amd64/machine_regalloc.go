package amd64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// InsertMoveBefore implements backend.RegAllocFunctionMachine.
func (m *machine) InsertMoveBefore(dst, src regalloc.VReg, instr *instruction) {
	typ := src.RegType()
	if typ != dst.RegType() {
		panic("BUG: src and dst must have the same type")
	}

	mov := m.allocateInstr()
	if typ == regalloc.RegTypeInt {
		mov.asMovRR(src, dst, true)
	} else {
		mov.asXmmUnaryRmR(sseOpcodeMovdqu, newOperandReg(src), dst)
	}

	cur := instr.prev
	prevNext := cur.next
	cur = linkInstr(cur, mov)
	linkInstr(cur, prevNext)
}

// InsertStoreRegisterAt implements backend.RegAllocFunctionMachine.
func (m *machine) InsertStoreRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
	if !v.IsRealReg() {
		panic("BUG: VReg must be backed by real reg to be stored")
	}

	typ := m.c.TypeOf(v)

	var prevNext, cur *instruction
	if after {
		cur, prevNext = instr, instr.next
	} else {
		cur, prevNext = instr.prev, instr
	}

	offsetFromSP := m.getVRegSpillSlotOffsetFromSP(v.ID(), typ.Size())
	store := m.allocateInstr()
	mem := newOperandMem(m.newAmodeImmReg(uint32(offsetFromSP), rspVReg))
	switch typ {
	case ssa.TypeI32:
		store.asMovRM(v, mem, 4)
	case ssa.TypeI64:
		store.asMovRM(v, mem, 8)
	case ssa.TypeF32:
		store.asXmmMovRM(sseOpcodeMovss, v, mem)
	case ssa.TypeF64:
		store.asXmmMovRM(sseOpcodeMovsd, v, mem)
	case ssa.TypeV128:
		store.asXmmMovRM(sseOpcodeMovdqu, v, mem)
	}

	cur = linkInstr(cur, store)
	return linkInstr(cur, prevNext)
}

// InsertReloadRegisterAt implements backend.RegAllocFunctionMachine.
func (m *machine) InsertReloadRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
	if !v.IsRealReg() {
		panic("BUG: VReg must be backed by real reg to be stored")
	}

	typ := m.c.TypeOf(v)
	var prevNext, cur *instruction
	if after {
		cur, prevNext = instr, instr.next
	} else {
		cur, prevNext = instr.prev, instr
	}

	// Load the value to the temporary.
	load := m.allocateInstr()
	offsetFromSP := m.getVRegSpillSlotOffsetFromSP(v.ID(), typ.Size())
	a := newOperandMem(m.newAmodeImmReg(uint32(offsetFromSP), rspVReg))
	switch typ {
	case ssa.TypeI32:
		load.asMovzxRmR(extModeLQ, a, v)
	case ssa.TypeI64:
		load.asMov64MR(a, v)
	case ssa.TypeF32:
		load.asXmmUnaryRmR(sseOpcodeMovss, a, v)
	case ssa.TypeF64:
		load.asXmmUnaryRmR(sseOpcodeMovsd, a, v)
	case ssa.TypeV128:
		load.asXmmUnaryRmR(sseOpcodeMovdqu, a, v)
	default:
		panic("BUG")
	}

	cur = linkInstr(cur, load)
	return linkInstr(cur, prevNext)
}

// ClobberedRegisters implements backend.RegAllocFunctionMachine.
func (m *machine) ClobberedRegisters(regs []regalloc.VReg) {
	m.clobberedRegs = append(m.clobberedRegs[:0], regs...)
}

// Swap implements backend.RegAllocFunctionMachine.
func (m *machine) Swap(cur *instruction, x1, x2, tmp regalloc.VReg) {
	if x1.RegType() == regalloc.RegTypeInt {
		prevNext := cur.next
		xc := m.allocateInstr().asXCHG(x1, newOperandReg(x2), 8)
		cur = linkInstr(cur, xc)
		linkInstr(cur, prevNext)
	} else {
		if tmp.Valid() {
			prevNext := cur.next
			m.InsertMoveBefore(tmp, x1, prevNext)
			m.InsertMoveBefore(x1, x2, prevNext)
			m.InsertMoveBefore(x2, tmp, prevNext)
		} else {
			prevNext := cur.next
			r2 := x2.RealReg()
			// Temporarily spill x1 to stack.
			cur = m.InsertStoreRegisterAt(x1, cur, true).prev
			// Then move x2 to x1.
			cur = linkInstr(cur, m.allocateInstr().asXmmUnaryRmR(sseOpcodeMovdqa, newOperandReg(x2), x1))
			linkInstr(cur, prevNext)
			// Then reload the original value on x1 from stack to r2.
			m.InsertReloadRegisterAt(x1.SetRealReg(r2), cur, true)
		}
	}
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
	case jmp:
		return cur
	default:
		return end
	}
}

// SSABlockLabel implements backend.RegAllocFunctionMachine.
func (m *machine) SSABlockLabel(id ssa.BasicBlockID) backend.Label {
	return m.ectx.SsaBlockIDToLabels[id]
}
