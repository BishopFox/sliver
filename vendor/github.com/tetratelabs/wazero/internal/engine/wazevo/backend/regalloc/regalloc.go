// Package regalloc performs register allocation. The algorithm can work on any ISA by implementing the interfaces in
// api.go.
package regalloc

// References:
// * https://web.stanford.edu/class/archive/cs/cs143/cs143.1128/lectures/17/Slides17.pdf
// * https://en.wikipedia.org/wiki/Chaitin%27s_algorithm
// * https://llvm.org/ProjectsWithLLVM/2004-Fall-CS426-LS.pdf
// * https://pfalcon.github.io/ssabook/latest/book-full.pdf: Chapter 9. for liveness analysis.

import (
	"fmt"
	"math"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

// NewAllocator returns a new Allocator.
func NewAllocator(allocatableRegs *RegisterInfo) Allocator {
	a := Allocator{
		regInfo:               allocatableRegs,
		blockLivenessDataPool: wazevoapi.NewPool[blockLivenessData](resetBlockLivenessData),
		phiDefInstListPool:    wazevoapi.NewPool[phiDefInstList](resetPhiDefInstList),
	}
	a.state.reset()
	for _, regs := range allocatableRegs.AllocatableRegisters {
		for _, r := range regs {
			a.allocatableSet = a.allocatableSet.add(r)
		}
	}
	return a
}

type (
	// RegisterInfo holds the statically-known ISA-specific register information.
	RegisterInfo struct {
		// AllocatableRegisters is a 2D array of allocatable RealReg, indexed by regTypeNum and regNum.
		// The order matters: the first element is the most preferred one when allocating.
		AllocatableRegisters [NumRegType][]RealReg
		CalleeSavedRegisters [RealRegsNumMax]bool
		CallerSavedRegisters [RealRegsNumMax]bool
		RealRegToVReg        []VReg
		// RealRegName returns the name of the given RealReg for debugging.
		RealRegName func(r RealReg) string
		RealRegType func(r RealReg) RegType
	}

	// Allocator is a register allocator.
	Allocator struct {
		// regInfo is static per ABI/ISA, and is initialized by the machine during Machine.PrepareRegisterAllocator.
		regInfo *RegisterInfo
		// allocatableSet is a set of allocatable RealReg derived from regInfo. Static per ABI/ISA.
		allocatableSet           regSet
		allocatedCalleeSavedRegs []VReg
		blockLivenessDataPool    wazevoapi.Pool[blockLivenessData]
		blockLivenessData        [] /* blockID to */ *blockLivenessData
		vs                       []VReg
		maxBlockID               int
		phiDefInstListPool       wazevoapi.Pool[phiDefInstList]

		// Followings are re-used during various places e.g. coloring.
		blks             []Block
		reals            []RealReg
		currentOccupants regInUseSet

		// Following two fields are updated while iterating the blocks in the reverse postorder.
		state       state
		blockStates [] /* blockID to */ blockState
	}

	// blockLivenessData is a per-block information used during the register allocation.
	blockLivenessData struct {
		seen     bool
		liveOuts map[VReg]struct{}
		liveIns  map[VReg]struct{}
	}

	// programCounter represents an opaque index into the program which is used to represents a LiveInterval of a VReg.
	programCounter int32

	state struct {
		argRealRegs          []VReg
		regsInUse            regInUseSet
		vrStates             []vrState
		maxVRegIDEncountered int

		// allocatedRegSet is a set of RealReg that are allocated during the allocation phase. This is reset per function.
		allocatedRegSet regSet
	}

	blockState struct {
		visited            bool
		startFromPredIndex int
		// startRegs is a list of RealReg that are used at the beginning of the block. This is used to fix the merge edges.
		startRegs regInUseSet
		// endRegs is a list of RealReg that are used at the end of the block. This is used to fix the merge edges.
		endRegs regInUseSet
		init    bool
	}

	vrState struct {
		v VReg
		r RealReg
		// defInstr is the instruction that defines this value. If this is the phi value and not the entry block, this is nil.
		defInstr Instr
		// defBlk is the block that defines this value. If this is the phi value, this is the block whose arguments contain this value.
		defBlk Block
		// spilled is true if this value is spilled i.e. the value is reload from the stack somewhere in the program.
		spilled bool
		// lca = lowest common ancestor. This is the block that is the lowest common ancestor of all the blocks that
		// reloads this value. This is used to determine the spill location. Only valid if spilled=true.
		lca Block
		// lastUse is the program counter of the last use of this value. This changes while iterating the block, and
		// should not be used across the blocks as it becomes invalid.
		lastUse programCounter
		// isPhi is true if this is a phi value.
		isPhi bool
		// phiDefInstList is a list of instructions that defines this phi value.
		// This is used to determine the spill location, and only valid if isPhi=true.
		*phiDefInstList
	}

	// phiDefInstList is a linked list of instructions that defines a phi value.
	phiDefInstList struct {
		instr Instr
		next  *phiDefInstList
	}
)

func resetPhiDefInstList(l *phiDefInstList) {
	l.instr = nil
	l.next = nil
}

func (s *state) dump(info *RegisterInfo) { //nolint:unused
	fmt.Println("\t\tstate:")
	fmt.Println("\t\t\targRealRegs:", s.argRealRegs)
	fmt.Println("\t\t\tregsInUse", s.regsInUse.format(info))
	fmt.Println("\t\t\tallocatedRegSet:", s.allocatedRegSet.format(info))
	fmt.Println("\t\t\tused:", s.regsInUse.format(info))
	fmt.Println("\t\t\tmaxVRegIDEncountered:", s.maxVRegIDEncountered)
	var strs []string
	for i, v := range s.vrStates {
		if v.r != RealRegInvalid {
			strs = append(strs, fmt.Sprintf("(v%d: %s)", i, info.RealRegName(v.r)))
		}
	}
	fmt.Println("\t\t\tvrStates:", strings.Join(strs, ", "))
}

func (s *state) reset() {
	s.argRealRegs = s.argRealRegs[:0]
	for i, l := 0, len(s.vrStates); i <= s.maxVRegIDEncountered && i < l; i++ {
		s.vrStates[i].reset()
	}
	s.maxVRegIDEncountered = -1
	s.allocatedRegSet = regSet(0)
	s.regsInUse.reset()
}

func (a *Allocator) getBlockState(bID int) *blockState {
	if bID >= len(a.blockStates) {
		a.blockStates = append(a.blockStates, make([]blockState, (bID+1)-len(a.blockStates))...)
		a.blockStates = a.blockStates[:cap(a.blockStates)]
	}
	ret := &a.blockStates[bID]
	if !ret.init {
		ret.reset()
		ret.init = true
	}
	return ret
}

func (s *state) setVRegState(v VReg, r RealReg) {
	id := int(v.ID())
	if id >= len(s.vrStates) {
		s.vrStates = append(s.vrStates, make([]vrState, id+1-len(s.vrStates))...)
		s.vrStates = s.vrStates[:cap(s.vrStates)]
	}

	st := &s.vrStates[id]
	st.r = r
	st.v = v
}

func (vs *vrState) reset() {
	vs.r = RealRegInvalid
	vs.defInstr = nil
	vs.defBlk = nil
	vs.spilled = false
	vs.lca = nil
	vs.isPhi = false
	vs.phiDefInstList = nil
}

func (s *state) getVRegState(v VReg) *vrState {
	id := int(v.ID())
	if id >= len(s.vrStates) {
		s.setVRegState(v, RealRegInvalid)
	}
	if s.maxVRegIDEncountered < id {
		s.maxVRegIDEncountered = id
	}
	return &s.vrStates[id]
}

func (s *state) useRealReg(r RealReg, v VReg) {
	if s.regsInUse.has(r) {
		panic("BUG: useRealReg: the given real register is already used")
	}
	s.regsInUse.add(r, v)
	s.setVRegState(v, r)
	s.allocatedRegSet = s.allocatedRegSet.add(r)
}

func (s *state) releaseRealReg(r RealReg) {
	current := s.regsInUse.get(r)
	if current.Valid() {
		s.regsInUse.remove(r)
		s.setVRegState(current, RealRegInvalid)
	}
}

// recordReload records that the given VReg is reloaded in the given block.
// This is used to determine the spill location by tracking the lowest common ancestor of all the blocks that reloads the value.
func (vs *vrState) recordReload(f Function, blk Block) {
	vs.spilled = true
	if vs.lca == nil {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d is reloaded in blk%d,\n", vs.v.ID(), blk.ID())
		}
		vs.lca = blk
	} else {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d is reloaded in blk%d, lca=%d\n", vs.v.ID(), blk.ID(), vs.lca.ID())
		}
		vs.lca = f.LowestCommonAncestor(vs.lca, blk)
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("updated lca=%d\n", vs.lca.ID())
		}
	}
}

func (s *state) findOrSpillAllocatable(a *Allocator, allocatable []RealReg, forbiddenMask regSet) (r RealReg) {
	r = RealRegInvalid
	var lastUseAt programCounter = math.MinInt32
	var spillVReg VReg
	for _, candidateReal := range allocatable {
		if forbiddenMask.has(candidateReal) {
			continue
		}

		using := s.regsInUse.get(candidateReal)
		if using == VRegInvalid {
			// This is not used at this point.
			return candidateReal
		}

		if last := s.getVRegState(using).lastUse; last > lastUseAt {
			lastUseAt = last
			r = candidateReal
			spillVReg = using
		}
	}

	if r == RealRegInvalid {
		panic("not found any allocatable register")
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Printf("\tspilling v%d when: %s\n", spillVReg.ID(), forbiddenMask.format(a.regInfo))
	}
	s.releaseRealReg(r)
	return r
}

func (s *state) findAllocatable(allocatable []RealReg, forbiddenMask regSet) RealReg {
	for _, r := range allocatable {
		if !s.regsInUse.has(r) && !forbiddenMask.has(r) {
			return r
		}
	}
	return RealRegInvalid
}

func (s *state) resetAt(bs *blockState, liveIns map[VReg]struct{}) {
	s.regsInUse.range_(func(_ RealReg, vr VReg) {
		s.setVRegState(vr, RealRegInvalid)
	})
	s.regsInUse.reset()
	bs.endRegs.range_(func(r RealReg, v VReg) {
		if _, ok := liveIns[v]; ok {
			s.regsInUse.add(r, v)
			s.setVRegState(v, r)
		}
	})
}

func (b *blockState) reset() {
	b.visited = false
	b.endRegs.reset()
	b.startRegs.reset()
	b.startFromPredIndex = -1
	b.init = false
}

func (b *blockState) dump(a *RegisterInfo) {
	fmt.Println("\t\tblockState:")
	fmt.Println("\t\t\tstartRegs:", b.startRegs.format(a))
	fmt.Println("\t\t\tendRegs:", b.endRegs.format(a))
	fmt.Println("\t\t\tstartFromPredIndex:", b.startFromPredIndex)
	fmt.Println("\t\t\tvisited:", b.visited)
	fmt.Println("\t\t\tinit:", b.init)
}

// DoAllocation performs register allocation on the given Function.
func (a *Allocator) DoAllocation(f Function) {
	a.livenessAnalysis(f)
	a.alloc(f)
	a.determineCalleeSavedRealRegs(f)
	f.Done()
}

func (a *Allocator) determineCalleeSavedRealRegs(f Function) {
	a.allocatedCalleeSavedRegs = a.allocatedCalleeSavedRegs[:0]
	a.state.allocatedRegSet.range_(func(allocatedRealReg RealReg) {
		if a.regInfo.isCalleeSaved(allocatedRealReg) {
			a.allocatedCalleeSavedRegs = append(a.allocatedCalleeSavedRegs, a.regInfo.RealRegToVReg[allocatedRealReg])
		}
	})
	f.ClobberedRegisters(a.allocatedCalleeSavedRegs)
}

// phiBlk returns the block that defines the given phi value, nil otherwise.
func (s *state) phiBlk(v VReg) Block {
	vs := s.getVRegState(v)
	if vs.isPhi {
		return vs.defBlk
	}
	return nil
}

// liveAnalysis constructs Allocator.blockLivenessData.
// The algorithm here is described in https://pfalcon.github.io/ssabook/latest/book-full.pdf Chapter 9.2.
func (a *Allocator) livenessAnalysis(f Function) {
	// First, we need to allocate blockLivenessData.
	s := &a.state
	for blk := f.PostOrderBlockIteratorBegin(); blk != nil; blk = f.PostOrderBlockIteratorNext() { // Order doesn't matter.
		a.allocateBlockLivenessData(blk.ID())

		// We should gather phi value data.
		for _, p := range blk.BlockParams(&a.vs) {
			vs := s.getVRegState(p)
			vs.isPhi = true
			vs.defBlk = blk
		}
		if blk.ID() > a.maxBlockID {
			a.maxBlockID = blk.ID()
		}
	}

	// Run the Algorithm 9.2 in the bool.
	for blk := f.PostOrderBlockIteratorBegin(); blk != nil; blk = f.PostOrderBlockIteratorNext() {
		blkID := blk.ID()
		info := a.livenessDataAt(blkID)

		ns := blk.Succs()
		for i := 0; i < ns; i++ {
			succ := blk.Succ(i)
			if succ == nil {
				continue
			}

			succID := succ.ID()
			succInfo := a.livenessDataAt(succID)
			if !succInfo.seen { // This means the back edge.
				continue
			}

			for v := range succInfo.liveIns {
				if s.phiBlk(v) != succ {
					info.liveOuts[v] = struct{}{}
					info.liveIns[v] = struct{}{}
				}
			}
		}

		for instr := blk.InstrRevIteratorBegin(); instr != nil; instr = blk.InstrRevIteratorNext() {

			var use, def VReg
			for _, def = range instr.Defs(&a.vs) {
				if !def.IsRealReg() {
					delete(info.liveIns, def)
				}
			}
			for _, use = range instr.Uses(&a.vs) {
				if !use.IsRealReg() {
					info.liveIns[use] = struct{}{}
				}
			}

			// If the destination is a phi value, and ...
			if def.Valid() && s.phiBlk(def) != nil {
				if use.Valid() && use.IsRealReg() {
					// If the source is a real register, this is the beginning of the function.
					a.state.argRealRegs = append(a.state.argRealRegs, use)
				} else {
					// Otherwise, this is the definition of the phi value for the successor block.
					// So we need to make it outlive the block.
					info.liveOuts[def] = struct{}{}
				}
			}
		}
		info.seen = true
	}

	nrs := f.LoopNestingForestRoots()
	for i := 0; i < nrs; i++ {
		root := f.LoopNestingForestRoot(i)
		a.loopTreeDFS(root)
	}
}

// loopTreeDFS implements the Algorithm 9.3 in the book in an iterative way.
func (a *Allocator) loopTreeDFS(entry Block) {
	a.blks = a.blks[:0]
	a.blks = append(a.blks, entry)

	s := &a.state
	for len(a.blks) > 0 {
		tail := len(a.blks) - 1
		loop := a.blks[tail]
		a.blks = a.blks[:tail]
		a.vs = a.vs[:0]

		info := a.livenessDataAt(loop.ID())
		for v := range info.liveIns {
			if s.phiBlk(v) != loop {
				a.vs = append(a.vs, v)
				info.liveOuts[v] = struct{}{}
			}
		}

		cn := loop.LoopNestingForestChildren()
		for i := 0; i < cn; i++ {
			child := loop.LoopNestingForestChild(i)
			childID := child.ID()
			childInfo := a.livenessDataAt(childID)
			for _, v := range a.vs {
				childInfo.liveIns[v] = struct{}{}
				childInfo.liveOuts[v] = struct{}{}
			}
			if child.LoopHeader() {
				a.blks = append(a.blks, child)
			}
		}
	}
}

// alloc allocates registers for the given function by iterating the blocks in the reverse postorder.
// The algorithm here is derived from the Go compiler's allocator https://github.com/golang/go/blob/release-branch.go1.21/src/cmd/compile/internal/ssa/regalloc.go
// In short, this is a simply linear scan register allocation where each block inherits the register allocation state from
// one of its predecessors. Each block inherits the selected state and starts allocation from there.
// If there's a discrepancy in the end states between predecessors, the adjustments are made to ensure consistency after allocation is done (which we call "fixing merge state").
// The spill instructions (store into the dedicated slots) are inserted after all the allocations and fixing merge states. That is because
// at the point, we all know where the reloads happen, and therefore we can know the best place to spill the values. More precisely,
// the spill happens in the block that is the lowest common ancestor of all the blocks that reloads the value.
//
// All of these logics are almost the same as Go's compiler which has a dedicated description in the source file ^^.
func (a *Allocator) alloc(f Function) {
	// First we allocate each block in the reverse postorder (at least one predecessor should be allocated for each block).
	for blk := f.ReversePostOrderBlockIteratorBegin(); blk != nil; blk = f.ReversePostOrderBlockIteratorNext() {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("========== allocating blk%d ========\n", blk.ID())
		}
		a.allocBlock(f, blk)
	}
	// After the allocation, we all know the start and end state of each block. So we can fix the merge states.
	for blk := f.ReversePostOrderBlockIteratorBegin(); blk != nil; blk = f.ReversePostOrderBlockIteratorNext() {
		a.fixMergeState(f, blk)
	}
	// Finally, we insert the spill instructions as we know all the places where the reloads happen.
	a.scheduleSpills(f)
}

func (a *Allocator) allocBlock(f Function, blk Block) {
	bID := blk.ID()
	liveness := a.livenessDataAt(bID)
	s := &a.state
	currentBlkState := a.getBlockState(bID)

	preds := blk.Preds()
	var predState *blockState
	switch preds {
	case 0: // This is the entry block.
	case 1:
		predID := blk.Pred(0).ID()
		predState = a.getBlockState(predID)
		currentBlkState.startFromPredIndex = 0
	default:
		// TODO: there should be some better heuristic to choose the predecessor.
		for i := 0; i < preds; i++ {
			predID := blk.Pred(i).ID()
			if _predState := a.getBlockState(predID); _predState.visited {
				predState = _predState
				currentBlkState.startFromPredIndex = i
				break
			}
		}
	}
	if predState == nil {
		if !blk.Entry() {
			panic(fmt.Sprintf("BUG: at lease one predecessor should be visited for blk%d", blk.ID()))
		}
		for _, u := range s.argRealRegs {
			s.useRealReg(u.RealReg(), u)
		}
	} else if predState != nil {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("allocating blk%d starting from blk%d (on index=%d) \n",
				bID, blk.Pred(currentBlkState.startFromPredIndex).ID(), currentBlkState.startFromPredIndex)
		}
		s.resetAt(predState, liveness.liveIns)
	}

	s.regsInUse.range_(func(allocated RealReg, v VReg) {
		currentBlkState.startRegs.add(allocated, v)
	})

	// Update the last use of each VReg.
	var pc programCounter
	for instr := blk.InstrIteratorBegin(); instr != nil; instr = blk.InstrIteratorNext() {
		for _, use := range instr.Uses(&a.vs) {
			if !use.IsRealReg() {
				s.getVRegState(use).lastUse = pc
			}
		}
		pc++
	}
	// Reset the last use of the liveOuts.
	for outlive := range liveness.liveOuts {
		s.getVRegState(outlive).lastUse = math.MaxInt32
	}

	pc = 0
	for instr := blk.InstrIteratorBegin(); instr != nil; instr = blk.InstrIteratorNext() {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Println(instr)
		}

		var currentUsedSet regSet
		killSet := a.reals[:0]

		// Gather the set of registers that will be used in the current instruction.
		for _, use := range instr.Uses(&a.vs) {
			if use.IsRealReg() {
				r := use.RealReg()
				currentUsedSet = currentUsedSet.add(r)
				if a.allocatableSet.has(r) {
					killSet = append(killSet, r)
				}
			} else {
				vs := s.getVRegState(use)
				if r := vs.r; r != RealRegInvalid {
					currentUsedSet = currentUsedSet.add(r)
				}
			}
		}

		for i, use := range instr.Uses(&a.vs) {
			if !use.IsRealReg() {
				vs := s.getVRegState(use)
				killed := liveness.isKilledAt(vs, pc)
				r := vs.r

				if r == RealRegInvalid {
					r = s.findOrSpillAllocatable(a, a.regInfo.AllocatableRegisters[use.RegType()], currentUsedSet)
					vs.recordReload(f, blk)
					f.ReloadRegisterBefore(use.SetRealReg(r), instr)
					s.useRealReg(r, use)
				}
				if wazevoapi.RegAllocLoggingEnabled {
					fmt.Printf("\ttrying to use v%v on %s\n", use.ID(), a.regInfo.RealRegName(r))
				}
				instr.AssignUse(i, use.SetRealReg(r))
				currentUsedSet = currentUsedSet.add(r)
				if killed {
					if wazevoapi.RegAllocLoggingEnabled {
						fmt.Printf("\tkill v%d with %s\n", use.ID(), a.regInfo.RealRegName(r))
					}
					killSet = append(killSet, r)
				}
			}
		}

		isIndirect := instr.IsIndirectCall()
		call := instr.IsCall() || isIndirect
		if call {
			addr := RealRegInvalid
			if instr.IsIndirectCall() {
				addr = a.vs[0].RealReg()
			}
			a.releaseCallerSavedRegs(addr)
		}

		for _, r := range killSet {
			s.releaseRealReg(r)
		}
		a.reals = killSet

		defs := instr.Defs(&a.vs)
		switch {
		case len(defs) > 1:
			if !call {
				panic("only call can have multiple defs")
			}
			// Call's defining register are all caller-saved registers.
			// Therefore, we can assume that all of them are allocatable.
			for _, def := range defs {
				s.useRealReg(def.RealReg(), def)
			}
		case len(defs) == 1:
			def := defs[0]
			if def.IsRealReg() {
				r := def.RealReg()
				if a.allocatableSet.has(r) {
					if s.regsInUse.has(r) {
						s.releaseRealReg(r)
					}
					s.useRealReg(r, def)
				}
			} else {
				vState := s.getVRegState(def)
				r := vState.r
				// Allocate a new real register if `def` is not currently assigned one.
				// It can happen when multiple instructions define the same VReg (e.g. const loads).
				if r == RealRegInvalid {
					if instr.IsCopy() {
						copySrc := instr.Uses(&a.vs)[0].RealReg()
						if a.allocatableSet.has(copySrc) && !s.regsInUse.has(copySrc) {
							r = copySrc
						}
					}
					if r == RealRegInvalid {
						typ := def.RegType()
						r = s.findOrSpillAllocatable(a, a.regInfo.AllocatableRegisters[typ], regSet(0))
					}
					s.useRealReg(r, def)
				}
				instr.AssignDef(def.SetRealReg(r))
				if wazevoapi.RegAllocLoggingEnabled {
					fmt.Printf("\tdefining v%d with %s\n", def.ID(), a.regInfo.RealRegName(r))
				}
				if vState.isPhi {
					n := a.phiDefInstListPool.Allocate()
					n.instr = instr
					n.next = vState.phiDefInstList
					vState.phiDefInstList = n
				} else {
					vState.defInstr = instr
					vState.defBlk = blk
				}
			}
		}
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Println(instr)
		}
		pc++
	}

	s.regsInUse.range_(func(allocated RealReg, v VReg) {
		currentBlkState.endRegs.add(allocated, v)
	})

	currentBlkState.visited = true
	if wazevoapi.RegAllocLoggingEnabled {
		currentBlkState.dump(a.regInfo)
	}
}

func (a *Allocator) releaseCallerSavedRegs(addrReg RealReg) {
	s := &a.state

	for i := 0; i < 64; i++ {
		allocated := RealReg(i)
		if allocated == addrReg { // If this is the call indirect, we should not touch the addr register.
			continue
		}
		if v := s.regsInUse.get(allocated); v.Valid() {
			if v.IsRealReg() {
				continue // This is the argument register as it's already used by VReg backed by the corresponding RealReg.
			}
			if !a.regInfo.isCallerSaved(allocated) {
				// If this is not a caller-saved register, it is safe to keep it across the call.
				continue
			}
			s.releaseRealReg(allocated)
		}
	}
}

func (a *Allocator) fixMergeState(f Function, blk Block) {
	preds := blk.Preds()
	if preds <= 1 {
		return
	}

	s := &a.state

	// Restores the state at the beginning of the block.
	bID := blk.ID()
	blkSt := a.getBlockState(bID)
	desiredOccupants := &blkSt.startRegs
	aliveOnRegVRegs := make(map[VReg]RealReg)
	for i := 0; i < 64; i++ {
		r := RealReg(i)
		if v := blkSt.startRegs.get(r); v.Valid() {
			aliveOnRegVRegs[v] = r
		}
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Println("fixMergeState", blk.ID(), ":", desiredOccupants.format(a.regInfo))
	}

	currentOccupants := &a.currentOccupants
	for i := 0; i < preds; i++ {
		currentOccupants.reset()
		if i == blkSt.startFromPredIndex {
			continue
		}

		currentOccupantsRev := make(map[VReg]RealReg)
		pred := blk.Pred(i)
		predSt := a.getBlockState(pred.ID())
		for ii := 0; ii < 64; ii++ {
			r := RealReg(ii)
			if v := predSt.endRegs.get(r); v.Valid() {
				if _, ok := aliveOnRegVRegs[v]; !ok {
					continue
				}
				currentOccupants.add(r, v)
				currentOccupantsRev[v] = r
			}
		}

		s.resetAt(predSt, a.livenessDataAt(bID).liveIns)

		// Finds the free registers if any.
		intTmp, floatTmp := VRegInvalid, VRegInvalid
		if intFree := s.findAllocatable(
			a.regInfo.AllocatableRegisters[RegTypeInt], desiredOccupants.set,
		); intFree != RealRegInvalid {
			intTmp = FromRealReg(intFree, RegTypeInt)
		}
		if floatFree := s.findAllocatable(
			a.regInfo.AllocatableRegisters[RegTypeFloat], desiredOccupants.set,
		); floatFree != RealRegInvalid {
			floatTmp = FromRealReg(floatFree, RegTypeFloat)
		}

		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Println("\t", pred.ID(), ":", currentOccupants.format(a.regInfo))
		}

		for ii := 0; ii < 64; ii++ {
			r := RealReg(ii)
			desiredVReg := desiredOccupants.get(r)
			if !desiredVReg.Valid() {
				continue
			}

			currentVReg := currentOccupants.get(r)
			if desiredVReg.ID() == currentVReg.ID() {
				continue
			}

			typ := desiredVReg.RegType()
			var tmpRealReg VReg
			if typ == RegTypeInt {
				tmpRealReg = intTmp
			} else {
				tmpRealReg = floatTmp
			}
			a.reconcileEdge(f, r, pred, currentOccupants, currentOccupantsRev, currentVReg, desiredVReg, tmpRealReg, typ)
		}
	}
}

func (a *Allocator) reconcileEdge(f Function,
	r RealReg,
	pred Block,
	currentOccupants *regInUseSet,
	currentOccupantsRev map[VReg]RealReg,
	currentVReg, desiredVReg VReg,
	freeReg VReg,
	typ RegType,
) {
	s := &a.state
	if currentVReg.Valid() {
		// Both are on reg.
		er, ok := currentOccupantsRev[desiredVReg]
		if !ok {
			if wazevoapi.RegAllocLoggingEnabled {
				fmt.Printf("\t\tv%d is desired to be on %s, but currently on the stack\n",
					desiredVReg.ID(), a.regInfo.RealRegName(r),
				)
			}
			// This case is that the desired value is on the stack, but currentVReg is on the target register.
			// We need to move the current value to the stack, and reload the desired value.
			// TODO: we can do better here.
			f.StoreRegisterBefore(currentVReg.SetRealReg(r), pred.LastInstr())
			delete(currentOccupantsRev, currentVReg)

			s.getVRegState(desiredVReg).recordReload(f, pred)
			f.ReloadRegisterBefore(desiredVReg.SetRealReg(r), pred.LastInstr())
			currentOccupants.add(r, desiredVReg)
			currentOccupantsRev[desiredVReg] = r
			return
		}

		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d is desired to be on %s, but currently on %s\n",
				desiredVReg.ID(), a.regInfo.RealRegName(r), a.regInfo.RealRegName(er),
			)
		}
		f.SwapAtEndOfBlock(
			currentVReg.SetRealReg(r),
			desiredVReg.SetRealReg(er),
			freeReg,
			pred,
		)
		s.allocatedRegSet = s.allocatedRegSet.add(freeReg.RealReg())
		currentOccupantsRev[desiredVReg] = r
		currentOccupantsRev[currentVReg] = er
		currentOccupants.add(r, desiredVReg)
		currentOccupants.add(er, currentVReg)
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d previously on %s moved to %s\n", currentVReg.ID(), a.regInfo.RealRegName(r), a.regInfo.RealRegName(er))
		}
	} else {
		// Desired is on reg, but currently the target register is not used.
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d is desired to be on %s, current not used\n",
				desiredVReg.ID(), a.regInfo.RealRegName(r),
			)
		}
		if currentReg, ok := currentOccupantsRev[desiredVReg]; ok {
			f.InsertMoveBefore(
				FromRealReg(r, typ),
				desiredVReg.SetRealReg(currentReg),
				pred.LastInstr(),
			)
			currentOccupants.remove(currentReg)
		} else {
			s.getVRegState(desiredVReg).recordReload(f, pred)
			f.ReloadRegisterBefore(desiredVReg.SetRealReg(r), pred.LastInstr())
		}
		currentOccupantsRev[desiredVReg] = r
		currentOccupants.add(r, desiredVReg)
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Println("\t", pred.ID(), ":", currentOccupants.format(a.regInfo))
	}
}

func (a *Allocator) scheduleSpills(f Function) {
	vrStates := a.state.vrStates
	for i := 0; i <= a.state.maxVRegIDEncountered; i++ {
		vs := &vrStates[i]
		if vs.spilled {
			a.scheduleSpill(f, vs)
		}
	}
}

func (a *Allocator) scheduleSpill(f Function, vs *vrState) {
	v := vs.v
	// If the value is the phi value, we need to insert a spill after each phi definition.
	if vs.isPhi {
		for defInstr := vs.phiDefInstList; defInstr != nil; defInstr = defInstr.next {
			def := defInstr.instr.Defs(&a.vs)[0]
			f.StoreRegisterAfter(def, defInstr.instr)
		}
		return
	}

	pos := vs.lca
	definingBlk := vs.defBlk
	r := RealRegInvalid
	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Printf("v%d is spilled in blk%d, lca=blk%d\n", v.ID(), definingBlk.ID(), pos.ID())
	}
	for pos != definingBlk {
		st := a.blockStates[pos.ID()]
		for ii := 0; ii < 64; ii++ {
			rr := RealReg(ii)
			if st.startRegs.get(rr) == v {
				r = rr
				// Already in the register, so we can place the spill at the beginning of the block.
				break
			}
		}

		if r != RealRegInvalid {
			break
		}

		pos = f.Idom(pos)
	}

	if pos == definingBlk {
		defInstr := vs.defInstr
		defInstr.Defs(&a.vs)
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("schedule spill v%d after %v\n", v.ID(), defInstr)
		}
		f.StoreRegisterAfter(a.vs[0], defInstr)
	} else {
		// Found an ancestor block that holds the value in the register at the beginning of the block.
		// We need to insert a spill before the last use.
		first := pos.FirstInstr()
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("schedule spill v%d before %v\n", v.ID(), first)
		}
		f.StoreRegisterAfter(v.SetRealReg(r), first)
	}
}

// Reset resets the allocator's internal state so that it can be reused.
func (a *Allocator) Reset() {
	a.state.reset()
	for i, l := 0, len(a.blockStates); i <= a.maxBlockID && i < l; i++ {
		a.blockLivenessData[i] = nil
		s := &a.blockStates[i]
		s.reset()
	}
	a.blockLivenessDataPool.Reset()
	a.phiDefInstListPool.Reset()

	a.vs = a.vs[:0]
	a.maxBlockID = -1
}

func (a *Allocator) allocateBlockLivenessData(blockID int) *blockLivenessData {
	if blockID >= len(a.blockLivenessData) {
		a.blockLivenessData = append(a.blockLivenessData, make([]*blockLivenessData, (blockID+1)-len(a.blockLivenessData))...)
	}
	info := a.blockLivenessData[blockID]
	if info == nil {
		info = a.blockLivenessDataPool.Allocate()
		a.blockLivenessData[blockID] = info
	}
	return info
}

func (a *Allocator) livenessDataAt(blockID int) (info *blockLivenessData) {
	info = a.blockLivenessData[blockID]
	return
}

func resetBlockLivenessData(i *blockLivenessData) {
	i.seen = false
	i.liveOuts = resetMap(i.liveOuts)
	i.liveIns = resetMap(i.liveIns)
}

func resetMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		m = make(map[K]V)
	} else {
		for v := range m {
			delete(m, v)
		}
	}
	return m
}

// Format is for debugging.
func (i *blockLivenessData) Format(ri *RegisterInfo) string {
	var buf strings.Builder
	buf.WriteString("\t\tblockLivenessData:")
	buf.WriteString("\n\t\t\tliveOuts: ")
	for v := range i.liveOuts {
		if v.IsRealReg() {
			buf.WriteString(fmt.Sprintf("%s ", ri.RealRegName(v.RealReg())))
		} else {
			buf.WriteString(fmt.Sprintf("%v ", v))
		}
	}
	buf.WriteString("\n\t\t\tliveIns: ")
	for v := range i.liveIns {
		if v.IsRealReg() {
			buf.WriteString(fmt.Sprintf("%s ", ri.RealRegName(v.RealReg())))
		} else {
			buf.WriteString(fmt.Sprintf("%v ", v))
		}
	}
	buf.WriteString(fmt.Sprintf("\n\t\t\tseen: %v", i.seen))
	return buf.String()
}

func (i *blockLivenessData) isKilledAt(vs *vrState, pos programCounter) bool {
	v := vs.v
	if vs.lastUse == pos {
		if _, ok := i.liveOuts[v]; !ok {
			return true
		}
	}
	return false
}

func (r *RegisterInfo) isCalleeSaved(reg RealReg) bool {
	return r.CalleeSavedRegisters[reg]
}

func (r *RegisterInfo) isCallerSaved(reg RealReg) bool {
	return r.CallerSavedRegisters[reg]
}
