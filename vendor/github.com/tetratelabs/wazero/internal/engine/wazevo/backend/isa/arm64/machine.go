package arm64

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

type (
	// machine implements backend.Machine.
	machine struct {
		compiler      backend.Compiler
		currentABI    *abiImpl
		currentSSABlk ssa.BasicBlock
		// abis maps ssa.SignatureID to the ABI implementation.
		abis      []abiImpl
		instrPool wazevoapi.Pool[instruction]
		// rootInstr is the root instruction of the currently-compiled function.
		rootInstr *instruction
		// perBlockHead and perBlockEnd are the head and tail of the instruction list per currently-compiled ssa.BasicBlock.
		perBlockHead, perBlockEnd *instruction
		// pendingInstructions are the instructions which are not yet emitted into the instruction list.
		pendingInstructions []*instruction
		regAllocFn          regAllocFunctionImpl
		nextLabel           label

		// ssaBlockIDToLabels maps an SSA block ID to the label.
		ssaBlockIDToLabels []label
		// labelToInstructions maps a label to the instructions of the region which the label represents.
		labelPositions     map[label]*labelPosition
		orderedBlockLabels []*labelPosition
		labelPositionPool  wazevoapi.Pool[labelPosition]

		// addendsWorkQueue is used during address lowering, defined here for reuse.
		addendsWorkQueue queue[ssa.Value]
		addends32        queue[addend32]
		// addends64 is used during address lowering, defined here for reuse.
		addends64              queue[regalloc.VReg]
		unresolvedAddressModes []*instruction

		// condBrRelocs holds the conditional branches which need offset relocation.
		condBrRelocs []condBrReloc

		// spillSlotSize is the size of the stack slot in bytes used for spilling registers.
		// During the execution of the function, the stack looks like:
		//
		//
		//            (high address)
		//          +-----------------+
		//          |     .......     |
		//          |      ret Y      |
		//          |     .......     |
		//          |      ret 0      |
		//          |      arg X      |
		//          |     .......     |
		//          |      arg 1      |
		//          |      arg 0      |
		//          |      xxxxx      |
		//          |   ReturnAddress |
		//          +-----------------+   <<-|
		//          |   ...........   |      |
		//          |   spill slot M  |      | <--- spillSlotSize
		//          |   ............  |      |
		//          |   spill slot 2  |      |
		//          |   spill slot 1  |   <<-+
		//          |   clobbered N   |
		//          |   ...........   |
		//          |   clobbered 1   |
		//          |   clobbered 0   |
		//   SP---> +-----------------+
		//             (low address)
		//
		// and it represents the size of the space between FP and the first spilled slot. This must be a multiple of 16.
		// Also note that this is only known after register allocation.
		spillSlotSize int64
		spillSlots    map[regalloc.VRegID]int64 // regalloc.VRegID to offset.
		// clobberedRegs holds real-register backed VRegs saved at the function prologue, and restored at the epilogue.
		clobberedRegs []regalloc.VReg

		maxRequiredStackSizeForCalls int64
		stackBoundsCheckDisabled     bool

		regAllocStarted bool
	}

	addend32 struct {
		r   regalloc.VReg
		ext extendOp
	}

	// label represents a position in the generated code which is either
	// a real instruction or the constant pool (e.g. jump tables).
	//
	// This is exactly the same as the traditional "label" in assembly code.
	label uint32

	// labelPosition represents the regions of the generated code which the label represents.
	labelPosition struct {
		l            label
		begin, end   *instruction
		binarySize   int64
		binaryOffset int64
	}

	condBrReloc struct {
		cbr *instruction
		// currentLabelPos is the labelPosition within which condBr is defined.
		currentLabelPos *labelPosition
		// Next block's labelPosition.
		nextLabel label
		offset    int64
	}
)

const (
	invalidLabel = 0
	returnLabel  = math.MaxUint32
)

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	m := &machine{
		instrPool:         wazevoapi.NewPool[instruction](resetInstruction),
		labelPositionPool: wazevoapi.NewPool[labelPosition](resetLabelPosition),
		labelPositions:    make(map[label]*labelPosition),
		spillSlots:        make(map[regalloc.VRegID]int64),
		nextLabel:         invalidLabel,
	}
	m.regAllocFn.m = m
	m.regAllocFn.labelToRegAllocBlockIndex = make(map[label]int)
	return m
}

// Reset implements backend.Machine.
func (m *machine) Reset() {
	m.regAllocStarted = false
	m.instrPool.Reset()
	m.labelPositionPool.Reset()
	m.currentSSABlk = nil
	for l := label(0); l <= m.nextLabel; l++ {
		delete(m.labelPositions, l)
	}
	m.pendingInstructions = m.pendingInstructions[:0]
	m.clobberedRegs = m.clobberedRegs[:0]
	for key := range m.spillSlots {
		m.clobberedRegs = append(m.clobberedRegs, regalloc.VReg(key))
	}
	for _, key := range m.clobberedRegs {
		delete(m.spillSlots, regalloc.VRegID(key))
	}
	m.clobberedRegs = m.clobberedRegs[:0]
	m.orderedBlockLabels = m.orderedBlockLabels[:0]
	m.regAllocFn.reset()
	m.spillSlotSize = 0
	m.unresolvedAddressModes = m.unresolvedAddressModes[:0]
	m.rootInstr = nil
	m.ssaBlockIDToLabels = m.ssaBlockIDToLabels[:0]
	m.perBlockHead, m.perBlockEnd = nil, nil
	m.maxRequiredStackSizeForCalls = 0
	m.nextLabel = invalidLabel
}

// InitializeABI implements backend.Machine InitializeABI.
func (m *machine) InitializeABI(sig *ssa.Signature) {
	m.currentABI = m.getOrCreateABIImpl(sig)
}

// DisableStackCheck implements backend.Machine DisableStackCheck.
func (m *machine) DisableStackCheck() {
	m.stackBoundsCheckDisabled = true
}

// ABI implements backend.Machine.
func (m *machine) ABI() backend.FunctionABI {
	return m.currentABI
}

// allocateLabel allocates an unused label.
func (m *machine) allocateLabel() label {
	m.nextLabel++
	return m.nextLabel
}

// SetCompiler implements backend.Machine.
func (m *machine) SetCompiler(ctx backend.Compiler) {
	m.compiler = ctx
}

// StartLoweringFunction implements backend.Machine.
func (m *machine) StartLoweringFunction(max ssa.BasicBlockID) {
	imax := int(max)
	if len(m.ssaBlockIDToLabels) <= imax {
		// Eagerly allocate labels for the blocks since the underlying slice will be used for the next iteration.
		m.ssaBlockIDToLabels = append(m.ssaBlockIDToLabels, make([]label, imax+1)...)
	}
}

// EndLoweringFunction implements backend.Machine.
func (m *machine) EndLoweringFunction() {}

// StartBlock implements backend.Machine.
func (m *machine) StartBlock(blk ssa.BasicBlock) {
	m.currentSSABlk = blk

	l := m.ssaBlockIDToLabels[m.currentSSABlk.ID()]
	if l == invalidLabel {
		l = m.allocateLabel()
		m.ssaBlockIDToLabels[blk.ID()] = l
	}

	end := m.allocateNop()
	m.perBlockHead, m.perBlockEnd = end, end

	labelPos, ok := m.labelPositions[l]
	if !ok {
		labelPos = m.allocateLabelPosition(l)
		m.labelPositions[l] = labelPos
	}
	m.orderedBlockLabels = append(m.orderedBlockLabels, labelPos)
	labelPos.begin, labelPos.end = end, end
	m.regAllocFn.addBlock(blk, l, labelPos)
}

// EndBlock implements backend.Machine.
func (m *machine) EndBlock() {
	// Insert nop0 as the head of the block for convenience to simplify the logic of inserting instructions.
	m.insertAtPerBlockHead(m.allocateNop())

	l := m.ssaBlockIDToLabels[m.currentSSABlk.ID()]
	m.labelPositions[l].begin = m.perBlockHead

	if m.currentSSABlk.EntryBlock() {
		m.rootInstr = m.perBlockHead
	}
}

func (m *machine) insert(i *instruction) {
	m.pendingInstructions = append(m.pendingInstructions, i)
}

func (m *machine) insertBrTargetLabel() label {
	nop, l := m.allocateBrTarget()
	m.insert(nop)
	return l
}

func (m *machine) allocateBrTarget() (nop *instruction, l label) {
	l = m.allocateLabel()
	nop = m.allocateInstr()
	nop.asNop0WithLabel(l)
	pos := m.allocateLabelPosition(l)
	pos.begin, pos.end = nop, nop
	m.labelPositions[l] = pos
	return
}

func (m *machine) allocateLabelPosition(la label) *labelPosition {
	l := m.labelPositionPool.Allocate()
	l.l = la
	return l
}

func resetLabelPosition(l *labelPosition) {
	*l = labelPosition{}
}

// FlushPendingInstructions implements backend.Machine.
func (m *machine) FlushPendingInstructions() {
	l := len(m.pendingInstructions)
	if l == 0 {
		return
	}
	for i := l - 1; i >= 0; i-- { // reverse because we lower instructions in reverse order.
		m.insertAtPerBlockHead(m.pendingInstructions[i])
	}
	m.pendingInstructions = m.pendingInstructions[:0]
}

func (m *machine) insertAtPerBlockHead(i *instruction) {
	if m.perBlockHead == nil {
		m.perBlockHead = i
		m.perBlockEnd = i
		return
	}
	i.next = m.perBlockHead
	m.perBlockHead.prev = i
	m.perBlockHead = i
}

// String implements backend.Machine.
func (l label) String() string {
	return fmt.Sprintf("L%d", l)
}

// allocateInstr allocates an instruction.
func (m *machine) allocateInstr() *instruction {
	instr := m.instrPool.Allocate()
	if !m.regAllocStarted {
		instr.addedBeforeRegAlloc = true
	}
	return instr
}

func resetInstruction(i *instruction) {
	*i = instruction{}
}

func (m *machine) allocateNop() *instruction {
	instr := m.allocateInstr()
	instr.asNop0()
	return instr
}

func (m *machine) resolveAddressingMode(arg0offset, ret0offset int64, i *instruction) {
	amode := &i.amode
	switch amode.kind {
	case addressModeKindResultStackSpace:
		amode.imm += ret0offset
	case addressModeKindArgStackSpace:
		amode.imm += arg0offset
	default:
		panic("BUG")
	}

	var sizeInBits byte
	switch i.kind {
	case store8, uLoad8:
		sizeInBits = 8
	case store16, uLoad16:
		sizeInBits = 16
	case store32, fpuStore32, uLoad32, fpuLoad32:
		sizeInBits = 32
	case store64, fpuStore64, uLoad64, fpuLoad64:
		sizeInBits = 64
	case fpuStore128, fpuLoad128:
		sizeInBits = 128
	default:
		panic("BUG")
	}

	if offsetFitsInAddressModeKindRegUnsignedImm12(sizeInBits, amode.imm) {
		amode.kind = addressModeKindRegUnsignedImm12
	} else {
		// This case, we load the offset into the temporary register,
		// and then use it as the index register.
		newPrev := m.lowerConstantI64AndInsert(i.prev, tmpRegVReg, amode.imm)
		linkInstr(newPrev, i)
		*amode = addressMode{kind: addressModeKindRegReg, rn: amode.rn, rm: tmpRegVReg, extOp: extendOpUXTX /* indicates rm reg is 64-bit */}
	}
}

// ResolveRelativeAddresses implements backend.Machine.
func (m *machine) ResolveRelativeAddresses(ctx context.Context) {
	if len(m.unresolvedAddressModes) > 0 {
		arg0offset, ret0offset := m.arg0OffsetFromSP(), m.ret0OffsetFromSP()
		for _, i := range m.unresolvedAddressModes {
			m.resolveAddressingMode(arg0offset, ret0offset, i)
		}
	}

	// Reuse the slice to gather the unresolved conditional branches.
	cbrs := m.condBrRelocs[:0]

	var fn string
	var fnIndex int
	var labelToSSABlockID map[label]ssa.BasicBlockID
	if wazevoapi.PerfMapEnabled {
		fn = wazevoapi.GetCurrentFunctionName(ctx)
		labelToSSABlockID = make(map[label]ssa.BasicBlockID)
		for i, l := range m.ssaBlockIDToLabels {
			labelToSSABlockID[l] = ssa.BasicBlockID(i)
		}
		fnIndex = wazevoapi.GetCurrentFunctionIndex(ctx)
	}

	// Next, in order to determine the offsets of relative jumps, we have to calculate the size of each label.
	var offset int64
	for i, pos := range m.orderedBlockLabels {
		pos.binaryOffset = offset
		var size int64
		for cur := pos.begin; ; cur = cur.next {
			switch cur.kind {
			case nop0:
				l := cur.nop0Label()
				if pos, ok := m.labelPositions[l]; ok {
					pos.binaryOffset = offset + size
				}
			case condBr:
				if !cur.condBrOffsetResolved() {
					var nextLabel label
					if i < len(m.orderedBlockLabels)-1 {
						// Note: this is only used when the block ends with fallthrough,
						// therefore can be safely assumed that the next block exists when it's needed.
						nextLabel = m.orderedBlockLabels[i+1].l
					}
					cbrs = append(cbrs, condBrReloc{
						cbr: cur, currentLabelPos: pos, offset: offset + size,
						nextLabel: nextLabel,
					})
				}
			}
			size += cur.size()
			if cur == pos.end {
				break
			}
		}

		if wazevoapi.PerfMapEnabled {
			if size > 0 {
				l := pos.l
				var labelStr string
				if blkID, ok := labelToSSABlockID[l]; ok {
					labelStr = fmt.Sprintf("%s::SSA_Block[%s]", l, blkID)
				} else {
					labelStr = l.String()
				}
				wazevoapi.PerfMap.AddModuleEntry(fnIndex, offset, uint64(size), fmt.Sprintf("%s:::::%s", fn, labelStr))
			}
		}

		pos.binarySize = size
		offset += size
	}

	// Before resolving any offsets, we need to check if all the conditional branches can be resolved.
	var needRerun bool
	for i := range cbrs {
		reloc := &cbrs[i]
		cbr := reloc.cbr
		offset := reloc.offset

		target := cbr.condBrLabel()
		offsetOfTarget := m.labelPositions[target].binaryOffset
		diff := offsetOfTarget - offset
		if divided := diff >> 2; divided < minSignedInt19 || divided > maxSignedInt19 {
			// This case the conditional branch is too huge. We place the trampoline instructions at the end of the current block,
			// and jump to it.
			m.insertConditionalJumpTrampoline(cbr, reloc.currentLabelPos, reloc.nextLabel)
			// Then, we need to recall this function to fix up the label offsets
			// as they have changed after the trampoline is inserted.
			needRerun = true
		}
	}
	if needRerun {
		m.ResolveRelativeAddresses(ctx)
		if wazevoapi.PerfMapEnabled {
			wazevoapi.PerfMap.Clear()
		}
		return
	}

	var currentOffset int64
	for cur := m.rootInstr; cur != nil; cur = cur.next {
		switch cur.kind {
		case br:
			target := cur.brLabel()
			offsetOfTarget := m.labelPositions[target].binaryOffset
			diff := offsetOfTarget - currentOffset
			divided := diff >> 2
			if divided < minSignedInt26 || divided > maxSignedInt26 {
				// This means the currently compiled single function is extremely large.
				panic("too large function that requires branch relocation of large unconditional branch larger than 26-bit range")
			}
			cur.brOffsetResolve(diff)
		case condBr:
			if !cur.condBrOffsetResolved() {
				target := cur.condBrLabel()
				offsetOfTarget := m.labelPositions[target].binaryOffset
				diff := offsetOfTarget - currentOffset
				if divided := diff >> 2; divided < minSignedInt19 || divided > maxSignedInt19 {
					panic("BUG: branch relocation for large conditional branch larger than 19-bit range must be handled properly")
				}
				cur.condBrOffsetResolve(diff)
			}
		case brTableSequence:
			for i := range cur.targets {
				l := label(cur.targets[i])
				offsetOfTarget := m.labelPositions[l].binaryOffset
				diff := offsetOfTarget - (currentOffset + brTableSequenceOffsetTableBegin)
				cur.targets[i] = uint32(diff)
			}
			cur.brTableSequenceOffsetsResolved()
		case emitSourceOffsetInfo:
			m.compiler.AddSourceOffsetInfo(currentOffset, cur.sourceOffsetInfo())
		}
		currentOffset += cur.size()
	}
}

const (
	maxSignedInt26 int64 = 1<<25 - 1
	minSignedInt26 int64 = -(1 << 25)

	maxSignedInt19 int64 = 1<<19 - 1
	minSignedInt19 int64 = -(1 << 19)
)

func (m *machine) insertConditionalJumpTrampoline(cbr *instruction, currentBlk *labelPosition, nextLabel label) {
	cur := currentBlk.end
	originalTarget := cbr.condBrLabel()
	endNext := cur.next

	if cur.kind != br {
		// If the current block ends with a conditional branch, we can just insert the trampoline after it.
		// Otherwise, we need to insert "skip" instruction to skip the trampoline instructions.
		skip := m.allocateInstr()
		skip.asBr(nextLabel)
		cur = linkInstr(cur, skip)
	}

	cbrNewTargetInstr, cbrNewTargetLabel := m.allocateBrTarget()
	cbr.setCondBrTargets(cbrNewTargetLabel)
	cur = linkInstr(cur, cbrNewTargetInstr)

	// Then insert the unconditional branch to the original, which should be possible to get encoded
	// as 26-bit offset should be enough for any practical application.
	br := m.allocateInstr()
	br.asBr(originalTarget)
	cur = linkInstr(cur, br)

	// Update the end of the current block.
	currentBlk.end = cur

	linkInstr(cur, endNext)
}

func (m *machine) getOrAllocateSSABlockLabel(blk ssa.BasicBlock) label {
	if blk.ReturnBlock() {
		return returnLabel
	}
	l := m.ssaBlockIDToLabels[blk.ID()]
	if l == invalidLabel {
		l = m.allocateLabel()
		m.ssaBlockIDToLabels[blk.ID()] = l
	}
	return l
}

// LinkAdjacentBlocks implements backend.Machine.
func (m *machine) LinkAdjacentBlocks(prev, next ssa.BasicBlock) {
	prevLabelPos := m.labelPositions[m.getOrAllocateSSABlockLabel(prev)]
	nextLabelPos := m.labelPositions[m.getOrAllocateSSABlockLabel(next)]
	prevLabelPos.end.next = nextLabelPos.begin
}

// Format implements backend.Machine.
func (m *machine) Format() string {
	begins := map[*instruction]label{}
	for l, pos := range m.labelPositions {
		begins[pos.begin] = l
	}

	irBlocks := map[label]ssa.BasicBlockID{}
	for i, l := range m.ssaBlockIDToLabels {
		irBlocks[l] = ssa.BasicBlockID(i)
	}

	var lines []string
	for cur := m.rootInstr; cur != nil; cur = cur.next {
		if l, ok := begins[cur]; ok {
			var labelStr string
			if blkID, ok := irBlocks[l]; ok {
				labelStr = fmt.Sprintf("%s (SSA Block: %s):", l, blkID)
			} else {
				labelStr = fmt.Sprintf("%s:", l)
			}
			lines = append(lines, labelStr)
		}
		if cur.kind == nop0 {
			continue
		}
		lines = append(lines, "\t"+cur.String())
	}
	return "\n" + strings.Join(lines, "\n") + "\n"
}

// InsertReturn implements backend.Machine.
func (m *machine) InsertReturn() {
	i := m.allocateInstr()
	i.asRet(m.currentABI)
	m.insert(i)
}

func (m *machine) getVRegSpillSlotOffsetFromSP(id regalloc.VRegID, size byte) int64 {
	offset, ok := m.spillSlots[id]
	if !ok {
		offset = m.spillSlotSize
		// TODO: this should be aligned depending on the `size` to use Imm12 offset load/store as much as possible.
		m.spillSlots[id] = offset
		m.spillSlotSize += int64(size)
	}
	return offset + 16 // spill slot starts above the clobbered registers and the frame size.
}

func (m *machine) clobberedRegSlotSize() int64 {
	return int64(len(m.clobberedRegs) * 16)
}

func (m *machine) arg0OffsetFromSP() int64 {
	return m.frameSize() +
		16 + // 16-byte aligned return address
		16 // frame size saved below the clobbered registers.
}

func (m *machine) ret0OffsetFromSP() int64 {
	return m.arg0OffsetFromSP() + m.currentABI.argStackSize
}

func (m *machine) requiredStackSize() int64 {
	return m.maxRequiredStackSizeForCalls +
		m.frameSize() +
		16 + // 16-byte aligned return address.
		16 // frame size saved below the clobbered registers.
}

func (m *machine) frameSize() int64 {
	s := m.clobberedRegSlotSize() + m.spillSlotSize
	if s&0xf != 0 {
		panic(fmt.Errorf("BUG: frame size %d is not 16-byte aligned", s))
	}
	return s
}
