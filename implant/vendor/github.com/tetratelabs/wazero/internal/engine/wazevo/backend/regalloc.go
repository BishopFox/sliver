package backend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// RegAllocFunctionMachine is the interface for the machine specific logic that will be used in RegAllocFunction.
type RegAllocFunctionMachine[I regalloc.InstrConstraint] interface {
	// InsertMoveBefore inserts the move instruction from src to dst before the given instruction.
	InsertMoveBefore(dst, src regalloc.VReg, instr I)
	// InsertStoreRegisterAt inserts the instruction(s) to store the given virtual register at the given instruction.
	// If after is true, the instruction(s) will be inserted after the given instruction, otherwise before.
	InsertStoreRegisterAt(v regalloc.VReg, instr I, after bool) I
	// InsertReloadRegisterAt inserts the instruction(s) to reload the given virtual register at the given instruction.
	// If after is true, the instruction(s) will be inserted after the given instruction, otherwise before.
	InsertReloadRegisterAt(v regalloc.VReg, instr I, after bool) I
	// ClobberedRegisters is called when the register allocation is done and the clobbered registers are known.
	ClobberedRegisters(regs []regalloc.VReg)
	// Swap swaps the two virtual registers after the given instruction.
	Swap(cur I, x1, x2, tmp regalloc.VReg)
	// LastInstrForInsertion implements LastInstrForInsertion of regalloc.Function. See its comment for details.
	LastInstrForInsertion(begin, end I) I
	// SSABlockLabel returns the label of the given ssa.BasicBlockID.
	SSABlockLabel(id ssa.BasicBlockID) Label
}

type (
	// RegAllocFunction implements regalloc.Function.
	RegAllocFunction[I regalloc.InstrConstraint, m RegAllocFunctionMachine[I]] struct {
		m   m
		ssb ssa.Builder
		c   Compiler
		// iter is the iterator for reversePostOrderBlocks
		iter                   int
		reversePostOrderBlocks []RegAllocBlock[I, m]
		// labelToRegAllocBlockIndex maps label to the index of reversePostOrderBlocks.
		labelToRegAllocBlockIndex map[Label]int
		loopNestingForestRoots    []ssa.BasicBlock
	}

	// RegAllocBlock implements regalloc.Block.
	RegAllocBlock[I regalloc.InstrConstraint, m RegAllocFunctionMachine[I]] struct {
		// f is the function this instruction belongs to. Used to reuse the regAllocFunctionImpl.predsSlice slice for Defs() and Uses().
		f                           *RegAllocFunction[I, m]
		sb                          ssa.BasicBlock
		l                           Label
		begin, end                  I
		loopNestingForestChildren   []ssa.BasicBlock
		cur                         I
		id                          int
		cachedLastInstrForInsertion I
	}
)

// NewRegAllocFunction returns a new RegAllocFunction.
func NewRegAllocFunction[I regalloc.InstrConstraint, M RegAllocFunctionMachine[I]](m M, ssb ssa.Builder, c Compiler) *RegAllocFunction[I, M] {
	return &RegAllocFunction[I, M]{
		m:                         m,
		ssb:                       ssb,
		c:                         c,
		labelToRegAllocBlockIndex: make(map[Label]int),
	}
}

// AddBlock adds a new block to the function.
func (f *RegAllocFunction[I, M]) AddBlock(sb ssa.BasicBlock, l Label, begin, end I) {
	i := len(f.reversePostOrderBlocks)
	f.reversePostOrderBlocks = append(f.reversePostOrderBlocks, RegAllocBlock[I, M]{
		f:     f,
		sb:    sb,
		l:     l,
		begin: begin,
		end:   end,
		id:    int(sb.ID()),
	})
	f.labelToRegAllocBlockIndex[l] = i
}

// Reset resets the function for the next compilation.
func (f *RegAllocFunction[I, M]) Reset() {
	f.reversePostOrderBlocks = f.reversePostOrderBlocks[:0]
	f.iter = 0
}

// StoreRegisterAfter implements regalloc.Function StoreRegisterAfter.
func (f *RegAllocFunction[I, M]) StoreRegisterAfter(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.InsertStoreRegisterAt(v, instr.(I), true)
}

// ReloadRegisterBefore implements regalloc.Function ReloadRegisterBefore.
func (f *RegAllocFunction[I, M]) ReloadRegisterBefore(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.InsertReloadRegisterAt(v, instr.(I), false)
}

// ReloadRegisterAfter implements regalloc.Function ReloadRegisterAfter.
func (f *RegAllocFunction[I, M]) ReloadRegisterAfter(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.InsertReloadRegisterAt(v, instr.(I), true)
}

// StoreRegisterBefore implements regalloc.Function StoreRegisterBefore.
func (f *RegAllocFunction[I, M]) StoreRegisterBefore(v regalloc.VReg, instr regalloc.Instr) {
	m := f.m
	m.InsertStoreRegisterAt(v, instr.(I), false)
}

// ClobberedRegisters implements regalloc.Function ClobberedRegisters.
func (f *RegAllocFunction[I, M]) ClobberedRegisters(regs []regalloc.VReg) {
	f.m.ClobberedRegisters(regs)
}

// SwapBefore implements regalloc.Function SwapBefore.
func (f *RegAllocFunction[I, M]) SwapBefore(x1, x2, tmp regalloc.VReg, instr regalloc.Instr) {
	f.m.Swap(instr.Prev().(I), x1, x2, tmp)
}

// PostOrderBlockIteratorBegin implements regalloc.Function PostOrderBlockIteratorBegin.
func (f *RegAllocFunction[I, M]) PostOrderBlockIteratorBegin() regalloc.Block {
	f.iter = len(f.reversePostOrderBlocks) - 1
	return f.PostOrderBlockIteratorNext()
}

// PostOrderBlockIteratorNext implements regalloc.Function PostOrderBlockIteratorNext.
func (f *RegAllocFunction[I, M]) PostOrderBlockIteratorNext() regalloc.Block {
	if f.iter < 0 {
		return nil
	}
	b := &f.reversePostOrderBlocks[f.iter]
	f.iter--
	return b
}

// ReversePostOrderBlockIteratorBegin implements regalloc.Function ReversePostOrderBlockIteratorBegin.
func (f *RegAllocFunction[I, M]) ReversePostOrderBlockIteratorBegin() regalloc.Block {
	f.iter = 0
	return f.ReversePostOrderBlockIteratorNext()
}

// ReversePostOrderBlockIteratorNext implements regalloc.Function ReversePostOrderBlockIteratorNext.
func (f *RegAllocFunction[I, M]) ReversePostOrderBlockIteratorNext() regalloc.Block {
	if f.iter >= len(f.reversePostOrderBlocks) {
		return nil
	}
	b := &f.reversePostOrderBlocks[f.iter]
	f.iter++
	return b
}

// LoopNestingForestRoots implements regalloc.Function LoopNestingForestRoots.
func (f *RegAllocFunction[I, M]) LoopNestingForestRoots() int {
	f.loopNestingForestRoots = f.ssb.LoopNestingForestRoots()
	return len(f.loopNestingForestRoots)
}

// LoopNestingForestRoot implements regalloc.Function LoopNestingForestRoot.
func (f *RegAllocFunction[I, M]) LoopNestingForestRoot(i int) regalloc.Block {
	blk := f.loopNestingForestRoots[i]
	l := f.m.SSABlockLabel(blk.ID())
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

// InsertMoveBefore implements regalloc.Function InsertMoveBefore.
func (f *RegAllocFunction[I, M]) InsertMoveBefore(dst, src regalloc.VReg, instr regalloc.Instr) {
	f.m.InsertMoveBefore(dst, src, instr.(I))
}

// LowestCommonAncestor implements regalloc.Function LowestCommonAncestor.
func (f *RegAllocFunction[I, M]) LowestCommonAncestor(blk1, blk2 regalloc.Block) regalloc.Block {
	ret := f.ssb.LowestCommonAncestor(blk1.(*RegAllocBlock[I, M]).sb, blk2.(*RegAllocBlock[I, M]).sb)
	l := f.m.SSABlockLabel(ret.ID())
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

// Idom implements regalloc.Function Idom.
func (f *RegAllocFunction[I, M]) Idom(blk regalloc.Block) regalloc.Block {
	builder := f.ssb
	idom := builder.Idom(blk.(*RegAllocBlock[I, M]).sb)
	if idom == nil {
		panic("BUG: idom must not be nil")
	}
	l := f.m.SSABlockLabel(idom.ID())
	index := f.labelToRegAllocBlockIndex[l]
	return &f.reversePostOrderBlocks[index]
}

// ID implements regalloc.Block.
func (r *RegAllocBlock[I, m]) ID() int32 { return int32(r.id) }

// BlockParams implements regalloc.Block.
func (r *RegAllocBlock[I, m]) BlockParams(regs *[]regalloc.VReg) []regalloc.VReg {
	c := r.f.c
	*regs = (*regs)[:0]
	for i := 0; i < r.sb.Params(); i++ {
		v := c.VRegOf(r.sb.Param(i))
		*regs = append(*regs, v)
	}
	return *regs
}

// InstrIteratorBegin implements regalloc.Block.
func (r *RegAllocBlock[I, m]) InstrIteratorBegin() regalloc.Instr {
	r.cur = r.begin
	return r.cur
}

// InstrIteratorNext implements regalloc.Block.
func (r *RegAllocBlock[I, m]) InstrIteratorNext() regalloc.Instr {
	for {
		if r.cur == r.end {
			return nil
		}
		instr := r.cur.Next()
		r.cur = instr.(I)
		if instr == nil {
			return nil
		} else if instr.AddedBeforeRegAlloc() {
			// Only concerned about the instruction added before regalloc.
			return instr
		}
	}
}

// InstrRevIteratorBegin implements regalloc.Block.
func (r *RegAllocBlock[I, m]) InstrRevIteratorBegin() regalloc.Instr {
	r.cur = r.end
	return r.cur
}

// InstrRevIteratorNext implements regalloc.Block.
func (r *RegAllocBlock[I, m]) InstrRevIteratorNext() regalloc.Instr {
	for {
		if r.cur == r.begin {
			return nil
		}
		instr := r.cur.Prev()
		r.cur = instr.(I)
		if instr == nil {
			return nil
		} else if instr.AddedBeforeRegAlloc() {
			// Only concerned about the instruction added before regalloc.
			return instr
		}
	}
}

// FirstInstr implements regalloc.Block.
func (r *RegAllocBlock[I, m]) FirstInstr() regalloc.Instr {
	return r.begin
}

// EndInstr implements regalloc.Block.
func (r *RegAllocBlock[I, m]) EndInstr() regalloc.Instr {
	return r.end
}

// LastInstrForInsertion implements regalloc.Block.
func (r *RegAllocBlock[I, m]) LastInstrForInsertion() regalloc.Instr {
	var nil I
	if r.cachedLastInstrForInsertion == nil {
		r.cachedLastInstrForInsertion = r.f.m.LastInstrForInsertion(r.begin, r.end)
	}
	return r.cachedLastInstrForInsertion
}

// Preds implements regalloc.Block.
func (r *RegAllocBlock[I, m]) Preds() int { return r.sb.Preds() }

// Pred implements regalloc.Block.
func (r *RegAllocBlock[I, m]) Pred(i int) regalloc.Block {
	sb := r.sb
	pred := sb.Pred(i)
	l := r.f.m.SSABlockLabel(pred.ID())
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}

// Entry implements regalloc.Block.
func (r *RegAllocBlock[I, m]) Entry() bool { return r.sb.EntryBlock() }

// Succs implements regalloc.Block.
func (r *RegAllocBlock[I, m]) Succs() int {
	return r.sb.Succs()
}

// Succ implements regalloc.Block.
func (r *RegAllocBlock[I, m]) Succ(i int) regalloc.Block {
	sb := r.sb
	succ := sb.Succ(i)
	if succ.ReturnBlock() {
		return nil
	}
	l := r.f.m.SSABlockLabel(succ.ID())
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}

// LoopHeader implements regalloc.Block.
func (r *RegAllocBlock[I, m]) LoopHeader() bool {
	return r.sb.LoopHeader()
}

// LoopNestingForestChildren implements regalloc.Block.
func (r *RegAllocBlock[I, m]) LoopNestingForestChildren() int {
	r.loopNestingForestChildren = r.sb.LoopNestingForestChildren()
	return len(r.loopNestingForestChildren)
}

// LoopNestingForestChild implements regalloc.Block.
func (r *RegAllocBlock[I, m]) LoopNestingForestChild(i int) regalloc.Block {
	blk := r.loopNestingForestChildren[i]
	l := r.f.m.SSABlockLabel(blk.ID())
	index := r.f.labelToRegAllocBlockIndex[l]
	return &r.f.reversePostOrderBlocks[index]
}
