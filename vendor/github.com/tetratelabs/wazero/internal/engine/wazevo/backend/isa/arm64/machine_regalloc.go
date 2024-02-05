package arm64

// This file implements the interfaces required for register allocations. See regalloc/api.go.

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

type (
	// regAllocFunctionImpl implements regalloc.Function.
	regAllocFunctionImpl struct {
		m *machine
		// iter is the iterator for reversePostOrderBlocks
		iter                   int
		reversePostOrderBlocks []regAllocBlockImpl
		// labelToRegAllocBlockIndex maps label to the index of reversePostOrderBlocks.
		labelToRegAllocBlockIndex map[label]int
		loopNestingForestRoots    []ssa.BasicBlock
	}

	// regAllocBlockImpl implements regalloc.Block.
	regAllocBlockImpl struct {
		// f is the function this instruction belongs to. Used to reuse the regAllocFunctionImpl.predsSlice slice for Defs() and Uses().
		f                         *regAllocFunctionImpl
		sb                        ssa.BasicBlock
		l                         label
		pos                       *labelPosition
		loopNestingForestChildren []ssa.BasicBlock
		cur                       *instruction
		id                        int
		cachedLastInstr           regalloc.Instr
	}
)

func (f *regAllocFunctionImpl) addBlock(sb ssa.BasicBlock, l label, pos *labelPosition) {
	i := len(f.reversePostOrderBlocks)
	f.reversePostOrderBlocks = append(f.reversePostOrderBlocks, regAllocBlockImpl{
		f:   f,
		sb:  sb,
		l:   l,
		pos: pos,
		id:  int(sb.ID()),
	})
	f.labelToRegAllocBlockIndex[l] = i
}

func (f *regAllocFunctionImpl) reset() {
	f.reversePostOrderBlocks = f.reversePostOrderBlocks[:0]
	f.iter = 0
}

var (
	_ regalloc.Function = (*regAllocFunctionImpl)(nil)
	_ regalloc.Block    = (*regAllocBlockImpl)(nil)
)

// PostOrderBlockIteratorBegin implements regalloc.Function PostOrderBlockIteratorBegin.
func (f *regAllocFunctionImpl) PostOrderBlockIteratorBegin() regalloc.Block {
	f.iter = len(f.reversePostOrderBlocks) - 1
	return f.PostOrderBlockIteratorNext()
}

// PostOrderBlockIteratorNext implements regalloc.Function PostOrderBlockIteratorNext.
func (f *regAllocFunctionImpl) PostOrderBlockIteratorNext() regalloc.Block {
	if f.iter < 0 {
		return nil
	}
	b := &f.reversePostOrderBlocks[f.iter]
	f.iter--
	return b
}

// ReversePostOrderBlockIteratorBegin implements regalloc.Function ReversePostOrderBlockIteratorBegin.
func (f *regAllocFunctionImpl) ReversePostOrderBlockIteratorBegin() regalloc.Block {
	f.iter = 0
	return f.ReversePostOrderBlockIteratorNext()
}

// ReversePostOrderBlockIteratorNext implements regalloc.Function ReversePostOrderBlockIteratorNext.
func (f *regAllocFunctionImpl) ReversePostOrderBlockIteratorNext() regalloc.Block {
	if f.iter >= len(f.reversePostOrderBlocks) {
		return nil
	}
	b := &f.reversePostOrderBlocks[f.iter]
	f.iter++
	return b
}

// ClobberedRegisters implements regalloc.Function ClobberedRegisters.
func (f *regAllocFunctionImpl) ClobberedRegisters(regs []regalloc.VReg) {
	m := f.m
	m.clobberedRegs = append(m.clobberedRegs[:0], regs...)
}

// StoreRegisterBefore implements regalloc.Function StoreRegisterBefore.
func (f *regAllocFunctionImpl) StoreRegisterBefore(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.insertStoreRegisterAt(v, instr.(*instruction), false)
}

// SwapAtEndOfBlock implements regalloc.Function SwapAtEndOfBlock.
func (f *regAllocFunctionImpl) SwapAtEndOfBlock(x1, x2, tmp regalloc.VReg, block regalloc.Block) {
	blk := block.(*regAllocBlockImpl)
	cur := blk.LastInstr().(*instruction)
	cur = cur.prev
	f.m.swap(cur, x1, x2, tmp)
}

func (m *machine) swap(cur *instruction, x1, x2, tmp regalloc.VReg) {
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
			cur = m.insertStoreRegisterAt(x1, cur, true).prev
			// Then move x2 to x1.
			cur = linkInstr(cur, m.allocateInstr().asFpuMov128(x1, x2))
			linkInstr(cur, prevNext)
			// Then reload the original value on x1 from stack to r2.
			m.insertReloadRegisterAt(x1.SetRealReg(r2), cur, true)
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

// InsertMoveBefore implements regalloc.Function InsertMoveBefore.
func (f *regAllocFunctionImpl) InsertMoveBefore(dst, src regalloc.VReg, instr regalloc.Instr) {
	m := f.m

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

	cur := instr.(*instruction).prev
	prevNext := cur.next
	cur = linkInstr(cur, mov)
	linkInstr(cur, prevNext)
}

// StoreRegisterAfter implements regalloc.Function StoreRegisterAfter.
func (f *regAllocFunctionImpl) StoreRegisterAfter(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.insertStoreRegisterAt(v, instr.(*instruction), true)
}

// ReloadRegisterBefore implements regalloc.Function ReloadRegisterBefore.
func (f *regAllocFunctionImpl) ReloadRegisterBefore(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.insertReloadRegisterAt(v, instr.(*instruction), false)
}

// ReloadRegisterAfter implements regalloc.Function ReloadRegisterAfter.
func (f *regAllocFunctionImpl) ReloadRegisterAfter(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.insertReloadRegisterAt(v, instr.(*instruction), true)
}

// Done implements regalloc.Function Done.
func (f *regAllocFunctionImpl) Done() {
	m := f.m
	// Now that we know the final spill slot size, we must align spillSlotSize to 16 bytes.
	m.spillSlotSize = (m.spillSlotSize + 15) &^ 15
}

// ID implements regalloc.Block ID.
func (r *regAllocBlockImpl) ID() int {
	return r.id
}

// Preds implements regalloc.Block Preds.
func (r *regAllocBlockImpl) Preds() int {
	return r.sb.Preds()
}

// Pred implements regalloc.Block Pred.
func (r *regAllocBlockImpl) Pred(i int) regalloc.Block {
	sb := r.sb
	pred := sb.Pred(i)
	l := r.f.m.ssaBlockIDToLabels[pred.ID()]
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}

// Succs implements regalloc.Block Succs.
func (r *regAllocBlockImpl) Succs() int {
	return r.sb.Succs()
}

// Succ implements regalloc.Block Succ.
func (r *regAllocBlockImpl) Succ(i int) regalloc.Block {
	sb := r.sb
	succ := sb.Succ(i)
	if succ.ReturnBlock() {
		return nil
	}
	l := r.f.m.ssaBlockIDToLabels[succ.ID()]
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}

// LoopHeader implements regalloc.Block LoopHeader.
func (r *regAllocBlockImpl) LoopHeader() bool {
	return r.sb.LoopHeader()
}

// LoopNestingForestRoots implements regalloc.Function LoopNestingForestRoots.
func (f *regAllocFunctionImpl) LoopNestingForestRoots() int {
	f.loopNestingForestRoots = f.m.compiler.SSABuilder().LoopNestingForestRoots()
	return len(f.loopNestingForestRoots)
}

// LoopNestingForestRoot implements regalloc.Function LoopNestingForestRoot.
func (f *regAllocFunctionImpl) LoopNestingForestRoot(i int) regalloc.Block {
	blk := f.loopNestingForestRoots[i]
	l := f.m.ssaBlockIDToLabels[blk.ID()]
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

// LoopNestingForestChildren implements regalloc.Block LoopNestingForestChildren.
func (r *regAllocBlockImpl) LoopNestingForestChildren() int {
	r.loopNestingForestChildren = r.sb.LoopNestingForestChildren()
	return len(r.loopNestingForestChildren)
}

// LoopNestingForestChild implements regalloc.Block LoopNestingForestChild.
func (r *regAllocBlockImpl) LoopNestingForestChild(i int) regalloc.Block {
	blk := r.loopNestingForestChildren[i]
	l := r.f.m.ssaBlockIDToLabels[blk.ID()]
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}

// InstrIteratorBegin implements regalloc.Block InstrIteratorBegin.
func (r *regAllocBlockImpl) InstrIteratorBegin() regalloc.Instr {
	r.cur = r.pos.begin
	return r.cur
}

// InstrIteratorNext implements regalloc.Block InstrIteratorNext.
func (r *regAllocBlockImpl) InstrIteratorNext() regalloc.Instr {
	for {
		if r.cur == r.pos.end {
			return nil
		}
		instr := r.cur.next
		r.cur = instr
		if instr == nil {
			return nil
		} else if instr.addedBeforeRegAlloc {
			// Only concerned about the instruction added before regalloc.
			return instr
		}
	}
}

// InstrRevIteratorBegin implements regalloc.Block InstrRevIteratorBegin.
func (r *regAllocBlockImpl) InstrRevIteratorBegin() regalloc.Instr {
	r.cur = r.pos.end
	return r.cur
}

// InstrRevIteratorNext implements regalloc.Block InstrRevIteratorNext.
func (r *regAllocBlockImpl) InstrRevIteratorNext() regalloc.Instr {
	for {
		if r.cur == r.pos.begin {
			return nil
		}
		instr := r.cur.prev
		r.cur = instr
		if instr == nil {
			return nil
		} else if instr.addedBeforeRegAlloc {
			// Only concerned about the instruction added before regalloc.
			return instr
		}
	}
}

// BlockParams implements regalloc.Block BlockParams.
func (r *regAllocBlockImpl) BlockParams(regs *[]regalloc.VReg) []regalloc.VReg {
	c := r.f.m.compiler
	*regs = (*regs)[:0]
	for i := 0; i < r.sb.Params(); i++ {
		v := c.VRegOf(r.sb.Param(i))
		*regs = append(*regs, v)
	}
	return *regs
}

// Entry implements regalloc.Block Entry.
func (r *regAllocBlockImpl) Entry() bool { return r.sb.EntryBlock() }

// RegisterInfo implements backend.Machine.
func (m *machine) RegisterInfo() *regalloc.RegisterInfo {
	return regInfo
}

// Function implements backend.Machine Function.
func (m *machine) Function() regalloc.Function {
	m.regAllocStarted = true
	return &m.regAllocFn
}

func (m *machine) insertStoreRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
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

func (m *machine) insertReloadRegisterAt(v regalloc.VReg, instr *instruction, after bool) *instruction {
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

// LowestCommonAncestor implements regalloc.Function LowestCommonAncestor.
func (f *regAllocFunctionImpl) LowestCommonAncestor(blk1, blk2 regalloc.Block) regalloc.Block {
	ret := f.m.compiler.SSABuilder().LowestCommonAncestor(blk1.(*regAllocBlockImpl).sb, blk2.(*regAllocBlockImpl).sb)
	l := f.m.ssaBlockIDToLabels[ret.ID()]
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

func (f *regAllocFunctionImpl) Idom(blk regalloc.Block) regalloc.Block {
	builder := f.m.compiler.SSABuilder()
	idom := builder.Idom(blk.(*regAllocBlockImpl).sb)
	if idom == nil {
		panic("BUG: idom must not be nil")
	}
	l := f.m.ssaBlockIDToLabels[idom.ID()]
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

// FirstInstr implements regalloc.Block FirstInstr.
func (r *regAllocBlockImpl) FirstInstr() regalloc.Instr {
	return r.pos.begin
}

// LastInstr implements regalloc.Block LastInstr.
func (r *regAllocBlockImpl) LastInstr() regalloc.Instr {
	if r.cachedLastInstr == nil {
		cur := r.pos.end
		for cur.kind == nop0 {
			cur = cur.prev
			if cur == r.pos.begin {
				r.cachedLastInstr = r.pos.end
				return r.cachedLastInstr
			}
		}
		switch cur.kind {
		case br:
			r.cachedLastInstr = cur
		default:
			r.cachedLastInstr = r.pos.end
		}
	}
	return r.cachedLastInstr
}
