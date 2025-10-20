// Package regalloc performs register allocation. The algorithm can work on any ISA by implementing the interfaces in
// api.go.
//
// References:
//   - https://web.stanford.edu/class/archive/cs/cs143/cs143.1128/lectures/17/Slides17.pdf
//   - https://en.wikipedia.org/wiki/Chaitin%27s_algorithm
//   - https://llvm.org/ProjectsWithLLVM/2004-Fall-CS426-LS.pdf
//   - https://pfalcon.github.io/ssabook/latest/book-full.pdf: Chapter 9. for liveness analysis.
//   - https://github.com/golang/go/blob/release-branch.go1.21/src/cmd/compile/internal/ssa/regalloc.go
package regalloc

import (
	"fmt"
	"math"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

// NewAllocator returns a new Allocator.
func NewAllocator(allocatableRegs *RegisterInfo) Allocator {
	a := Allocator{
		regInfo:            allocatableRegs,
		phiDefInstListPool: wazevoapi.NewPool[phiDefInstList](resetPhiDefInstList),
		blockStates:        wazevoapi.NewIDedPool[blockState](resetBlockState),
	}
	a.state.vrStates = wazevoapi.NewIDedPool[vrState](resetVrState)
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
		CalleeSavedRegisters RegSet
		CallerSavedRegisters RegSet
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
		allocatableSet           RegSet
		allocatedCalleeSavedRegs []VReg
		vs                       []VReg
		vs2                      []VRegID
		phiDefInstListPool       wazevoapi.Pool[phiDefInstList]

		// Followings are re-used during various places.
		blks             []Block
		reals            []RealReg
		currentOccupants regInUseSet

		// Following two fields are updated while iterating the blocks in the reverse postorder.
		state       state
		blockStates wazevoapi.IDedPool[blockState]
	}

	// programCounter represents an opaque index into the program which is used to represents a LiveInterval of a VReg.
	programCounter int32

	state struct {
		argRealRegs []VReg
		regsInUse   regInUseSet
		vrStates    wazevoapi.IDedPool[vrState]

		currentBlockID int32

		// allocatedRegSet is a set of RealReg that are allocated during the allocation phase. This is reset per function.
		allocatedRegSet RegSet
	}

	blockState struct {
		// liveIns is a list of VReg that are live at the beginning of the block.
		liveIns []VRegID
		// seen is true if the block is visited during the liveness analysis.
		seen bool
		// visited is true if the block is visited during the allocation phase.
		visited            bool
		startFromPredIndex int
		// startRegs is a list of RealReg that are used at the beginning of the block. This is used to fix the merge edges.
		startRegs regInUseSet
		// endRegs is a list of RealReg that are used at the end of the block. This is used to fix the merge edges.
		endRegs regInUseSet
	}

	vrState struct {
		v VReg
		r RealReg
		// defInstr is the instruction that defines this value. If this is the phi value and not the entry block, this is nil.
		defInstr Instr
		// defBlk is the block that defines this value. If this is the phi value, this is the block whose arguments contain this value.
		defBlk Block
		// lca = lowest common ancestor. This is the block that is the lowest common ancestor of all the blocks that
		// reloads this value. This is used to determine the spill location. Only valid if spilled=true.
		lca Block
		// lastUse is the program counter of the last use of this value. This changes while iterating the block, and
		// should not be used across the blocks as it becomes invalid. To check the validity, use lastUseUpdatedAtBlockID.
		lastUse                 programCounter
		lastUseUpdatedAtBlockID int32
		// spilled is true if this value is spilled i.e. the value is reload from the stack somewhere in the program.
		//
		// Note that this field is used during liveness analysis for different purpose. This is used to determine the
		// value is live-in or not.
		spilled bool
		// isPhi is true if this is a phi value.
		isPhi      bool
		desiredLoc desiredLoc
		// phiDefInstList is a list of instructions that defines this phi value.
		// This is used to determine the spill location, and only valid if isPhi=true.
		*phiDefInstList
	}

	// phiDefInstList is a linked list of instructions that defines a phi value.
	phiDefInstList struct {
		instr Instr
		v     VReg
		next  *phiDefInstList
	}

	// desiredLoc represents a desired location for a VReg.
	desiredLoc uint16
	// desiredLocKind is a kind of desired location for a VReg.
	desiredLocKind uint16
)

const (
	// desiredLocKindUnspecified is a kind of desired location for a VReg that is not specified.
	desiredLocKindUnspecified desiredLocKind = iota
	// desiredLocKindStack is a kind of desired location for a VReg that is on the stack, only used for the phi values.
	desiredLocKindStack
	// desiredLocKindReg is a kind of desired location for a VReg that is in a register.
	desiredLocKindReg
	desiredLocUnspecified = desiredLoc(desiredLocKindUnspecified)
	desiredLocStack       = desiredLoc(desiredLocKindStack)
)

func newDesiredLocReg(r RealReg) desiredLoc {
	return desiredLoc(desiredLocKindReg) | desiredLoc(r<<2)
}

func (d desiredLoc) realReg() RealReg {
	return RealReg(d >> 2)
}

func (d desiredLoc) stack() bool {
	return d&3 == desiredLoc(desiredLocKindStack)
}

func resetPhiDefInstList(l *phiDefInstList) {
	l.instr = nil
	l.next = nil
	l.v = VRegInvalid
}

func (s *state) dump(info *RegisterInfo) { //nolint:unused
	fmt.Println("\t\tstate:")
	fmt.Println("\t\t\targRealRegs:", s.argRealRegs)
	fmt.Println("\t\t\tregsInUse", s.regsInUse.format(info))
	fmt.Println("\t\t\tallocatedRegSet:", s.allocatedRegSet.format(info))
	fmt.Println("\t\t\tused:", s.regsInUse.format(info))
	var strs []string
	for i := 0; i <= s.vrStates.MaxIDEncountered(); i++ {
		vs := s.vrStates.Get(i)
		if vs == nil {
			continue
		}
		if vs.r != RealRegInvalid {
			strs = append(strs, fmt.Sprintf("(v%d: %s)", vs.v.ID(), info.RealRegName(vs.r)))
		}
	}
	fmt.Println("\t\t\tvrStates:", strings.Join(strs, ", "))
}

func (s *state) reset() {
	s.argRealRegs = s.argRealRegs[:0]
	s.vrStates.Reset()
	s.allocatedRegSet = RegSet(0)
	s.regsInUse.reset()
	s.currentBlockID = -1
}

func (s *state) setVRegState(v VReg, r RealReg) {
	id := int(v.ID())
	st := s.vrStates.GetOrAllocate(id)
	st.r = r
	st.v = v
}

func resetVrState(vs *vrState) {
	vs.v = VRegInvalid
	vs.r = RealRegInvalid
	vs.defInstr = nil
	vs.defBlk = nil
	vs.spilled = false
	vs.lastUse = -1
	vs.lastUseUpdatedAtBlockID = -1
	vs.lca = nil
	vs.isPhi = false
	vs.phiDefInstList = nil
	vs.desiredLoc = desiredLocUnspecified
}

func (s *state) getVRegState(v VRegID) *vrState {
	return s.vrStates.GetOrAllocate(int(v))
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

func (s *state) findOrSpillAllocatable(a *Allocator, allocatable []RealReg, forbiddenMask RegSet, preferred RealReg) (r RealReg) {
	r = RealRegInvalid
	// First, check if the preferredMask has any allocatable register.
	if preferred != RealRegInvalid && !forbiddenMask.has(preferred) && !s.regsInUse.has(preferred) {
		for _, candidateReal := range allocatable {
			// TODO: we should ensure the preferred register is in the allocatable set in the first place,
			//  but right now, just in case, we check it here.
			if candidateReal == preferred {
				return preferred
			}
		}
	}

	var lastUseAt programCounter
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

		// Real registers in use should not be spilled, so we skip them.
		// For example, if the register is used as an argument register, and it might be
		// spilled and not reloaded when it ends up being used as a temporary to pass
		// stack based argument.
		if using.IsRealReg() {
			continue
		}

		isPreferred := candidateReal == preferred

		// last == -1 means the value won't be used anymore.
		if last := s.getVRegState(using.ID()).lastUse; r == RealRegInvalid || isPreferred || last == -1 || (lastUseAt != -1 && last > lastUseAt) {
			lastUseAt = last
			r = candidateReal
			spillVReg = using
			if isPreferred {
				break
			}
		}
	}

	if r == RealRegInvalid {
		panic("not found any allocatable register")
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Printf("\tspilling v%d when lastUseAt=%d and regsInUse=%s\n", spillVReg.ID(), lastUseAt, s.regsInUse.format(a.regInfo))
	}
	s.releaseRealReg(r)
	return r
}

func (s *state) findAllocatable(allocatable []RealReg, forbiddenMask RegSet) RealReg {
	for _, r := range allocatable {
		if !s.regsInUse.has(r) && !forbiddenMask.has(r) {
			return r
		}
	}
	return RealRegInvalid
}

func (s *state) resetAt(bs *blockState) {
	s.regsInUse.range_(func(_ RealReg, vr VReg) {
		s.setVRegState(vr, RealRegInvalid)
	})
	s.regsInUse.reset()
	bs.endRegs.range_(func(r RealReg, v VReg) {
		id := int(v.ID())
		st := s.vrStates.GetOrAllocate(id)
		if st.lastUseUpdatedAtBlockID == s.currentBlockID && st.lastUse == programCounterLiveIn {
			s.regsInUse.add(r, v)
			s.setVRegState(v, r)
		}
	})
}

func resetBlockState(b *blockState) {
	b.seen = false
	b.visited = false
	b.endRegs.reset()
	b.startRegs.reset()
	b.startFromPredIndex = -1
	b.liveIns = b.liveIns[:0]
}

func (b *blockState) dump(a *RegisterInfo) {
	fmt.Println("\t\tblockState:")
	fmt.Println("\t\t\tstartRegs:", b.startRegs.format(a))
	fmt.Println("\t\t\tendRegs:", b.endRegs.format(a))
	fmt.Println("\t\t\tstartFromPredIndex:", b.startFromPredIndex)
	fmt.Println("\t\t\tvisited:", b.visited)
}

// DoAllocation performs register allocation on the given Function.
func (a *Allocator) DoAllocation(f Function) {
	a.livenessAnalysis(f)
	a.alloc(f)
	a.determineCalleeSavedRealRegs(f)
}

func (a *Allocator) determineCalleeSavedRealRegs(f Function) {
	a.allocatedCalleeSavedRegs = a.allocatedCalleeSavedRegs[:0]
	a.state.allocatedRegSet.Range(func(allocatedRealReg RealReg) {
		if a.regInfo.CalleeSavedRegisters.has(allocatedRealReg) {
			a.allocatedCalleeSavedRegs = append(a.allocatedCalleeSavedRegs, a.regInfo.RealRegToVReg[allocatedRealReg])
		}
	})
	f.ClobberedRegisters(a.allocatedCalleeSavedRegs)
}

func (a *Allocator) getOrAllocateBlockState(blockID int32) *blockState {
	return a.blockStates.GetOrAllocate(int(blockID))
}

// phiBlk returns the block that defines the given phi value, nil otherwise.
func (s *state) phiBlk(v VRegID) Block {
	vs := s.getVRegState(v)
	if vs.isPhi {
		return vs.defBlk
	}
	return nil
}

const (
	programCounterLiveIn  = math.MinInt32
	programCounterLiveOut = math.MaxInt32
)

// liveAnalysis constructs Allocator.blockLivenessData.
// The algorithm here is described in https://pfalcon.github.io/ssabook/latest/book-full.pdf Chapter 9.2.
func (a *Allocator) livenessAnalysis(f Function) {
	s := &a.state
	for blk := f.PostOrderBlockIteratorBegin(); blk != nil; blk = f.PostOrderBlockIteratorNext() { // Order doesn't matter.

		// We should gather phi value data.
		for _, p := range blk.BlockParams(&a.vs) {
			vs := s.getVRegState(p.ID())
			vs.isPhi = true
			vs.defBlk = blk
		}
	}

	for blk := f.PostOrderBlockIteratorBegin(); blk != nil; blk = f.PostOrderBlockIteratorNext() {
		blkID := blk.ID()
		info := a.getOrAllocateBlockState(blkID)

		a.vs2 = a.vs2[:0]
		const (
			flagDeleted = false
			flagLive    = true
		)
		ns := blk.Succs()
		for i := 0; i < ns; i++ {
			succ := blk.Succ(i)
			if succ == nil {
				continue
			}

			succID := succ.ID()
			succInfo := a.getOrAllocateBlockState(succID)
			if !succInfo.seen { // This means the back edge.
				continue
			}

			for _, v := range succInfo.liveIns {
				if s.phiBlk(v) != succ {
					st := s.getVRegState(v)
					// We use .spilled field to store the flag.
					st.spilled = flagLive
					a.vs2 = append(a.vs2, v)
				}
			}
		}

		for instr := blk.InstrRevIteratorBegin(); instr != nil; instr = blk.InstrRevIteratorNext() {

			var use, def VReg
			for _, def = range instr.Defs(&a.vs) {
				if !def.IsRealReg() {
					id := def.ID()
					st := s.getVRegState(id)
					// We use .spilled field to store the flag.
					st.spilled = flagDeleted
					a.vs2 = append(a.vs2, id)
				}
			}
			for _, use = range instr.Uses(&a.vs) {
				if !use.IsRealReg() {
					id := use.ID()
					st := s.getVRegState(id)
					// We use .spilled field to store the flag.
					st.spilled = flagLive
					a.vs2 = append(a.vs2, id)
				}
			}

			if def.Valid() && s.phiBlk(def.ID()) != nil {
				if use.Valid() && use.IsRealReg() {
					// If the destination is a phi value, and the source is a real register, this is the beginning of the function.
					a.state.argRealRegs = append(a.state.argRealRegs, use)
				}
			}
		}

		for _, v := range a.vs2 {
			st := s.getVRegState(v)
			// We use .spilled field to store the flag.
			if st.spilled == flagLive { //nolint:gosimple
				info.liveIns = append(info.liveIns, v)
				st.spilled = false
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
		a.vs2 = a.vs2[:0]
		const (
			flagDone    = false
			flagPending = true
		)
		info := a.getOrAllocateBlockState(loop.ID())
		for _, v := range info.liveIns {
			if s.phiBlk(v) != loop {
				a.vs2 = append(a.vs2, v)
				st := s.getVRegState(v)
				// We use .spilled field to store the flag.
				st.spilled = flagPending
			}
		}

		var siblingAddedView []VRegID
		cn := loop.LoopNestingForestChildren()
		for i := 0; i < cn; i++ {
			child := loop.LoopNestingForestChild(i)
			childID := child.ID()
			childInfo := a.getOrAllocateBlockState(childID)

			if i == 0 {
				begin := len(childInfo.liveIns)
				for _, v := range a.vs2 {
					st := s.getVRegState(v)
					// We use .spilled field to store the flag.
					if st.spilled == flagPending { //nolint:gosimple
						st.spilled = flagDone
						// TODO: deduplicate, though I don't think it has much impact.
						childInfo.liveIns = append(childInfo.liveIns, v)
					}
				}
				siblingAddedView = childInfo.liveIns[begin:]
			} else {
				// TODO: deduplicate, though I don't think it has much impact.
				childInfo.liveIns = append(childInfo.liveIns, siblingAddedView...)
			}

			if child.LoopHeader() {
				a.blks = append(a.blks, child)
			}
		}

		if cn == 0 {
			// If there's no forest child, we haven't cleared the .spilled field at this point.
			for _, v := range a.vs2 {
				st := s.getVRegState(v)
				st.spilled = false
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
		if blk.Entry() {
			a.finalizeStartReg(blk)
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

func (a *Allocator) updateLiveInVRState(liveness *blockState) {
	currentBlockID := a.state.currentBlockID
	for _, v := range liveness.liveIns {
		vs := a.state.getVRegState(v)
		vs.lastUse = programCounterLiveIn
		vs.lastUseUpdatedAtBlockID = currentBlockID
	}
}

func (a *Allocator) finalizeStartReg(blk Block) {
	bID := blk.ID()
	liveness := a.getOrAllocateBlockState(bID)
	s := &a.state
	currentBlkState := a.getOrAllocateBlockState(bID)
	if currentBlkState.startFromPredIndex > -1 {
		return
	}

	s.currentBlockID = bID
	a.updateLiveInVRState(liveness)

	preds := blk.Preds()
	var predState *blockState
	switch preds {
	case 0: // This is the entry block.
	case 1:
		predID := blk.Pred(0).ID()
		predState = a.getOrAllocateBlockState(predID)
		currentBlkState.startFromPredIndex = 0
	default:
		// TODO: there should be some better heuristic to choose the predecessor.
		for i := 0; i < preds; i++ {
			predID := blk.Pred(i).ID()
			if _predState := a.getOrAllocateBlockState(predID); _predState.visited {
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
		currentBlkState.startFromPredIndex = 0
	} else if predState != nil {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("allocating blk%d starting from blk%d (on index=%d) \n",
				bID, blk.Pred(currentBlkState.startFromPredIndex).ID(), currentBlkState.startFromPredIndex)
		}
		s.resetAt(predState)
	}

	s.regsInUse.range_(func(allocated RealReg, v VReg) {
		currentBlkState.startRegs.add(allocated, v)
	})
	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Printf("finalized start reg for blk%d: %s\n", blk.ID(), currentBlkState.startRegs.format(a.regInfo))
	}
}

func (a *Allocator) allocBlock(f Function, blk Block) {
	bID := blk.ID()
	s := &a.state
	currentBlkState := a.getOrAllocateBlockState(bID)
	s.currentBlockID = bID

	if currentBlkState.startFromPredIndex < 0 {
		panic("BUG: startFromPredIndex should be set in finalizeStartReg prior to allocBlock")
	}

	// Clears the previous state.
	s.regsInUse.range_(func(allocatedRealReg RealReg, vr VReg) {
		s.setVRegState(vr, RealRegInvalid)
	})
	s.regsInUse.reset()
	// Then set the start state.
	currentBlkState.startRegs.range_(func(allocatedRealReg RealReg, vr VReg) {
		s.useRealReg(allocatedRealReg, vr)
	})

	desiredUpdated := a.vs2[:0]

	// Update the last use of each VReg.
	var pc programCounter
	for instr := blk.InstrIteratorBegin(); instr != nil; instr = blk.InstrIteratorNext() {
		var use, def VReg
		for _, use = range instr.Uses(&a.vs) {
			if !use.IsRealReg() {
				s.getVRegState(use.ID()).lastUse = pc
			}
		}

		if instr.IsCopy() {
			def = instr.Defs(&a.vs)[0]
			r := def.RealReg()
			if r != RealRegInvalid {
				useID := use.ID()
				vs := s.getVRegState(useID)
				if !vs.isPhi { // TODO: no idea why do we need this.
					vs.desiredLoc = newDesiredLocReg(r)
					desiredUpdated = append(desiredUpdated, useID)
				}
			}
		}
		pc++
	}

	// Mark all live-out values by checking live-in of the successors.
	// While doing so, we also update the desired register values.
	var succ Block
	for i, ns := 0, blk.Succs(); i < ns; i++ {
		succ = blk.Succ(i)
		if succ == nil {
			continue
		}

		succID := succ.ID()
		succState := a.getOrAllocateBlockState(succID)
		for _, v := range succState.liveIns {
			if s.phiBlk(v) != succ {
				st := s.getVRegState(v)
				st.lastUse = programCounterLiveOut
			}
		}

		if succState.startFromPredIndex > -1 {
			if wazevoapi.RegAllocLoggingEnabled {
				fmt.Printf("blk%d -> blk%d: start_regs: %s\n", bID, succID, succState.startRegs.format(a.regInfo))
			}
			succState.startRegs.range_(func(allocatedRealReg RealReg, vr VReg) {
				vs := s.getVRegState(vr.ID())
				vs.desiredLoc = newDesiredLocReg(allocatedRealReg)
				desiredUpdated = append(desiredUpdated, vr.ID())
			})
			for _, p := range succ.BlockParams(&a.vs) {
				vs := s.getVRegState(p.ID())
				if vs.desiredLoc.realReg() == RealRegInvalid {
					vs.desiredLoc = desiredLocStack
					desiredUpdated = append(desiredUpdated, p.ID())
				}
			}
		}
	}

	// Propagate the desired register values from the end of the block to the beginning.
	for instr := blk.InstrRevIteratorBegin(); instr != nil; instr = blk.InstrRevIteratorNext() {
		if instr.IsCopy() {
			def := instr.Defs(&a.vs)[0]
			defState := s.getVRegState(def.ID())
			desired := defState.desiredLoc.realReg()
			if desired == RealRegInvalid {
				continue
			}

			use := instr.Uses(&a.vs)[0]
			useID := use.ID()
			useState := s.getVRegState(useID)
			if s.phiBlk(useID) != succ && useState.desiredLoc == desiredLocUnspecified {
				useState.desiredLoc = newDesiredLocReg(desired)
				desiredUpdated = append(desiredUpdated, useID)
			}
		}
	}

	pc = 0
	for instr := blk.InstrIteratorBegin(); instr != nil; instr = blk.InstrIteratorNext() {
		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Println(instr)
		}

		var currentUsedSet RegSet
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
				vs := s.getVRegState(use.ID())
				if r := vs.r; r != RealRegInvalid {
					currentUsedSet = currentUsedSet.add(r)
				}
			}
		}

		for i, use := range instr.Uses(&a.vs) {
			if !use.IsRealReg() {
				vs := s.getVRegState(use.ID())
				killed := vs.lastUse == pc
				r := vs.r

				if r == RealRegInvalid {
					r = s.findOrSpillAllocatable(a, a.regInfo.AllocatableRegisters[use.RegType()], currentUsedSet,
						// Prefer the desired register if it's available.
						vs.desiredLoc.realReg())
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
			// Some instructions define multiple values on real registers.
			// E.g. call instructions (following calling convention) / div instruction on x64 that defines both rax and rdx.
			//
			// Note that currently I assume that such instructions define only the pre colored real registers, not the VRegs
			// that require allocations. If we need to support such case, we need to add the logic to handle it here,
			// though is there any such instruction?
			for _, def := range defs {
				if !def.IsRealReg() {
					panic("BUG: multiple defs should be on real registers")
				}
				r := def.RealReg()
				if s.regsInUse.has(r) {
					s.releaseRealReg(r)
				}
				s.useRealReg(r, def)
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
				vState := s.getVRegState(def.ID())
				r := vState.r

				if desired := vState.desiredLoc.realReg(); desired != RealRegInvalid {
					if r != desired {
						if (vState.isPhi && vState.defBlk == succ) ||
							// If this is not a phi and it's already assigned a real reg,
							// this value has multiple definitions, hence we cannot assign the desired register.
							(!s.regsInUse.has(desired) && r == RealRegInvalid) {
							// If the phi value is passed via a real register, we force the value to be in the desired register.
							if wazevoapi.RegAllocLoggingEnabled {
								fmt.Printf("\t\tv%d is phi and desiredReg=%s\n", def.ID(), a.regInfo.RealRegName(desired))
							}
							if r != RealRegInvalid {
								// If the value is already in a different real register, we release it to change the state.
								// Otherwise, multiple registers might have the same values at the end, which results in
								// messing up the merge state reconciliation.
								s.releaseRealReg(r)
							}
							r = desired
							s.releaseRealReg(r)
							s.useRealReg(r, def)
						}
					}
				}

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
						r = s.findOrSpillAllocatable(a, a.regInfo.AllocatableRegisters[typ], RegSet(0), RealRegInvalid)
					}
					s.useRealReg(r, def)
				}
				dr := def.SetRealReg(r)
				instr.AssignDef(dr)
				if wazevoapi.RegAllocLoggingEnabled {
					fmt.Printf("\tdefining v%d with %s\n", def.ID(), a.regInfo.RealRegName(r))
				}
				if vState.isPhi {
					if vState.desiredLoc.stack() { // Stack based phi value.
						f.StoreRegisterAfter(dr, instr)
						// Release the real register as it's not used anymore.
						s.releaseRealReg(r)
					} else {
						// Only the register based phis are necessary to track the defining instructions
						// since the stack-based phis are already having stores inserted ^.
						n := a.phiDefInstListPool.Allocate()
						n.instr = instr
						n.next = vState.phiDefInstList
						n.v = dr
						vState.phiDefInstList = n
					}
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

	// Reset the desired end location.
	for _, v := range desiredUpdated {
		vs := s.getVRegState(v)
		vs.desiredLoc = desiredLocUnspecified
	}
	a.vs2 = desiredUpdated[:0]

	for i := 0; i < blk.Succs(); i++ {
		succ := blk.Succ(i)
		if succ == nil {
			continue
		}
		// If the successor is not visited yet, finalize the start state.
		a.finalizeStartReg(succ)
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
			if !a.regInfo.CallerSavedRegisters.has(allocated) {
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
	blkSt := a.getOrAllocateBlockState(bID)
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

	s.currentBlockID = bID
	a.updateLiveInVRState(a.getOrAllocateBlockState(bID))

	currentOccupants := &a.currentOccupants
	for i := 0; i < preds; i++ {
		currentOccupants.reset()
		if i == blkSt.startFromPredIndex {
			continue
		}

		currentOccupantsRev := make(map[VReg]RealReg)
		pred := blk.Pred(i)
		predSt := a.getOrAllocateBlockState(pred.ID())
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

		s.resetAt(predSt)

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
			f.StoreRegisterBefore(currentVReg.SetRealReg(r), pred.LastInstrForInsertion())
			delete(currentOccupantsRev, currentVReg)

			s.getVRegState(desiredVReg.ID()).recordReload(f, pred)
			f.ReloadRegisterBefore(desiredVReg.SetRealReg(r), pred.LastInstrForInsertion())
			currentOccupants.add(r, desiredVReg)
			currentOccupantsRev[desiredVReg] = r
			return
		}

		if wazevoapi.RegAllocLoggingEnabled {
			fmt.Printf("\t\tv%d is desired to be on %s, but currently on %s\n",
				desiredVReg.ID(), a.regInfo.RealRegName(r), a.regInfo.RealRegName(er),
			)
		}
		f.SwapBefore(
			currentVReg.SetRealReg(r),
			desiredVReg.SetRealReg(er),
			freeReg,
			pred.LastInstrForInsertion(),
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
				pred.LastInstrForInsertion(),
			)
			currentOccupants.remove(currentReg)
		} else {
			s.getVRegState(desiredVReg.ID()).recordReload(f, pred)
			f.ReloadRegisterBefore(desiredVReg.SetRealReg(r), pred.LastInstrForInsertion())
		}
		currentOccupantsRev[desiredVReg] = r
		currentOccupants.add(r, desiredVReg)
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Println("\t", pred.ID(), ":", currentOccupants.format(a.regInfo))
	}
}

func (a *Allocator) scheduleSpills(f Function) {
	states := a.state.vrStates
	for i := 0; i <= states.MaxIDEncountered(); i++ {
		vs := states.Get(i)
		if vs == nil {
			continue
		}
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
			f.StoreRegisterAfter(defInstr.v, defInstr.instr)
		}
		return
	}

	pos := vs.lca
	definingBlk := vs.defBlk
	r := RealRegInvalid
	if definingBlk == nil {
		panic(fmt.Sprintf("BUG: definingBlk should not be nil for %s. This is likley a bug in backend lowering logic", vs.v.String()))
	}
	if pos == nil {
		panic(fmt.Sprintf("BUG: pos should not be nil for %s. This is likley a bug in backend lowering logic", vs.v.String()))
	}

	if wazevoapi.RegAllocLoggingEnabled {
		fmt.Printf("v%d is spilled in blk%d, lca=blk%d\n", v.ID(), definingBlk.ID(), pos.ID())
	}
	for pos != definingBlk {
		st := a.getOrAllocateBlockState(pos.ID())
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
	a.blockStates.Reset()
	a.phiDefInstListPool.Reset()
	a.vs = a.vs[:0]
}
