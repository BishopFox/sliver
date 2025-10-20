package arm64

import (
	"context"
	"fmt"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

type (
	// machine implements backend.Machine.
	machine struct {
		compiler          backend.Compiler
		executableContext *backend.ExecutableContextT[instruction]
		currentABI        *backend.FunctionABI

		regAlloc   regalloc.Allocator
		regAllocFn *backend.RegAllocFunction[*instruction, *machine]

		// addendsWorkQueue is used during address lowering, defined here for reuse.
		addendsWorkQueue wazevoapi.Queue[ssa.Value]
		addends32        wazevoapi.Queue[addend32]
		// addends64 is used during address lowering, defined here for reuse.
		addends64              wazevoapi.Queue[regalloc.VReg]
		unresolvedAddressModes []*instruction

		// condBrRelocs holds the conditional branches which need offset relocation.
		condBrRelocs []condBrReloc

		// jmpTableTargets holds the labels of the jump table targets.
		jmpTableTargets [][]uint32

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

	condBrReloc struct {
		cbr *instruction
		// currentLabelPos is the labelPosition within which condBr is defined.
		currentLabelPos *labelPosition
		// Next block's labelPosition.
		nextLabel label
		offset    int64
	}

	labelPosition = backend.LabelPosition[instruction]
	label         = backend.Label
)

const (
	labelReturn  = backend.LabelReturn
	labelInvalid = backend.LabelInvalid
)

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	m := &machine{
		spillSlots:        make(map[regalloc.VRegID]int64),
		executableContext: newExecutableContext(),
		regAlloc:          regalloc.NewAllocator(regInfo),
	}
	return m
}

func newExecutableContext() *backend.ExecutableContextT[instruction] {
	return backend.NewExecutableContextT[instruction](resetInstruction, setNext, setPrev, asNop0)
}

// ExecutableContext implements backend.Machine.
func (m *machine) ExecutableContext() backend.ExecutableContext {
	return m.executableContext
}

// RegAlloc implements backend.Machine Function.
func (m *machine) RegAlloc() {
	rf := m.regAllocFn
	for _, pos := range m.executableContext.OrderedBlockLabels {
		rf.AddBlock(pos.SB, pos.L, pos.Begin, pos.End)
	}

	m.regAllocStarted = true
	m.regAlloc.DoAllocation(rf)
	// Now that we know the final spill slot size, we must align spillSlotSize to 16 bytes.
	m.spillSlotSize = (m.spillSlotSize + 15) &^ 15
}

// Reset implements backend.Machine.
func (m *machine) Reset() {
	m.clobberedRegs = m.clobberedRegs[:0]
	for key := range m.spillSlots {
		m.clobberedRegs = append(m.clobberedRegs, regalloc.VReg(key))
	}
	for _, key := range m.clobberedRegs {
		delete(m.spillSlots, regalloc.VRegID(key))
	}
	m.clobberedRegs = m.clobberedRegs[:0]
	m.regAllocStarted = false
	m.regAlloc.Reset()
	m.regAllocFn.Reset()
	m.spillSlotSize = 0
	m.unresolvedAddressModes = m.unresolvedAddressModes[:0]
	m.maxRequiredStackSizeForCalls = 0
	m.executableContext.Reset()
	m.jmpTableTargets = m.jmpTableTargets[:0]
}

// SetCurrentABI implements backend.Machine SetCurrentABI.
func (m *machine) SetCurrentABI(abi *backend.FunctionABI) {
	m.currentABI = abi
}

// DisableStackCheck implements backend.Machine DisableStackCheck.
func (m *machine) DisableStackCheck() {
	m.stackBoundsCheckDisabled = true
}

// SetCompiler implements backend.Machine.
func (m *machine) SetCompiler(ctx backend.Compiler) {
	m.compiler = ctx
	m.regAllocFn = backend.NewRegAllocFunction[*instruction, *machine](m, ctx.SSABuilder(), ctx)
}

func (m *machine) insert(i *instruction) {
	ectx := m.executableContext
	ectx.PendingInstructions = append(ectx.PendingInstructions, i)
}

func (m *machine) insertBrTargetLabel() label {
	nop, l := m.allocateBrTarget()
	m.insert(nop)
	return l
}

func (m *machine) allocateBrTarget() (nop *instruction, l label) {
	ectx := m.executableContext
	l = ectx.AllocateLabel()
	nop = m.allocateInstr()
	nop.asNop0WithLabel(l)
	pos := ectx.AllocateLabelPosition(l)
	pos.Begin, pos.End = nop, nop
	ectx.LabelPositions[l] = pos
	return
}

// allocateInstr allocates an instruction.
func (m *machine) allocateInstr() *instruction {
	instr := m.executableContext.InstructionPool.Allocate()
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

// resolveRelativeAddresses resolves the relative addresses before encoding.
func (m *machine) resolveRelativeAddresses(ctx context.Context) {
	ectx := m.executableContext
	for {
		if len(m.unresolvedAddressModes) > 0 {
			arg0offset, ret0offset := m.arg0OffsetFromSP(), m.ret0OffsetFromSP()
			for _, i := range m.unresolvedAddressModes {
				m.resolveAddressingMode(arg0offset, ret0offset, i)
			}
		}

		// Reuse the slice to gather the unresolved conditional branches.
		m.condBrRelocs = m.condBrRelocs[:0]

		var fn string
		var fnIndex int
		var labelToSSABlockID map[label]ssa.BasicBlockID
		if wazevoapi.PerfMapEnabled {
			fn = wazevoapi.GetCurrentFunctionName(ctx)
			labelToSSABlockID = make(map[label]ssa.BasicBlockID)
			for i, l := range ectx.SsaBlockIDToLabels {
				labelToSSABlockID[l] = ssa.BasicBlockID(i)
			}
			fnIndex = wazevoapi.GetCurrentFunctionIndex(ctx)
		}

		// Next, in order to determine the offsets of relative jumps, we have to calculate the size of each label.
		var offset int64
		for i, pos := range ectx.OrderedBlockLabels {
			pos.BinaryOffset = offset
			var size int64
			for cur := pos.Begin; ; cur = cur.next {
				switch cur.kind {
				case nop0:
					l := cur.nop0Label()
					if pos, ok := ectx.LabelPositions[l]; ok {
						pos.BinaryOffset = offset + size
					}
				case condBr:
					if !cur.condBrOffsetResolved() {
						var nextLabel label
						if i < len(ectx.OrderedBlockLabels)-1 {
							// Note: this is only used when the block ends with fallthrough,
							// therefore can be safely assumed that the next block exists when it's needed.
							nextLabel = ectx.OrderedBlockLabels[i+1].L
						}
						m.condBrRelocs = append(m.condBrRelocs, condBrReloc{
							cbr: cur, currentLabelPos: pos, offset: offset + size,
							nextLabel: nextLabel,
						})
					}
				}
				size += cur.size()
				if cur == pos.End {
					break
				}
			}

			if wazevoapi.PerfMapEnabled {
				if size > 0 {
					l := pos.L
					var labelStr string
					if blkID, ok := labelToSSABlockID[l]; ok {
						labelStr = fmt.Sprintf("%s::SSA_Block[%s]", l, blkID)
					} else {
						labelStr = l.String()
					}
					wazevoapi.PerfMap.AddModuleEntry(fnIndex, offset, uint64(size), fmt.Sprintf("%s:::::%s", fn, labelStr))
				}
			}
			offset += size
		}

		// Before resolving any offsets, we need to check if all the conditional branches can be resolved.
		var needRerun bool
		for i := range m.condBrRelocs {
			reloc := &m.condBrRelocs[i]
			cbr := reloc.cbr
			offset := reloc.offset

			target := cbr.condBrLabel()
			offsetOfTarget := ectx.LabelPositions[target].BinaryOffset
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
			if wazevoapi.PerfMapEnabled {
				wazevoapi.PerfMap.Clear()
			}
		} else {
			break
		}
	}

	var currentOffset int64
	for cur := ectx.RootInstr; cur != nil; cur = cur.next {
		switch cur.kind {
		case br:
			target := cur.brLabel()
			offsetOfTarget := ectx.LabelPositions[target].BinaryOffset
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
				offsetOfTarget := ectx.LabelPositions[target].BinaryOffset
				diff := offsetOfTarget - currentOffset
				if divided := diff >> 2; divided < minSignedInt19 || divided > maxSignedInt19 {
					panic("BUG: branch relocation for large conditional branch larger than 19-bit range must be handled properly")
				}
				cur.condBrOffsetResolve(diff)
			}
		case brTableSequence:
			tableIndex := cur.u1
			targets := m.jmpTableTargets[tableIndex]
			for i := range targets {
				l := label(targets[i])
				offsetOfTarget := ectx.LabelPositions[l].BinaryOffset
				diff := offsetOfTarget - (currentOffset + brTableSequenceOffsetTableBegin)
				targets[i] = uint32(diff)
			}
			cur.brTableSequenceOffsetsResolved()
		case emitSourceOffsetInfo:
			m.compiler.AddSourceOffsetInfo(currentOffset, cur.sourceOffsetInfo())
		}
		currentOffset += cur.size()
	}
}

const (
	maxSignedInt26 = 1<<25 - 1
	minSignedInt26 = -(1 << 25)

	maxSignedInt19 = 1<<18 - 1
	minSignedInt19 = -(1 << 18)
)

func (m *machine) insertConditionalJumpTrampoline(cbr *instruction, currentBlk *labelPosition, nextLabel label) {
	cur := currentBlk.End
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
	currentBlk.End = cur

	linkInstr(cur, endNext)
}

// Format implements backend.Machine.
func (m *machine) Format() string {
	ectx := m.executableContext
	begins := map[*instruction]label{}
	for l, pos := range ectx.LabelPositions {
		begins[pos.Begin] = l
	}

	irBlocks := map[label]ssa.BasicBlockID{}
	for i, l := range ectx.SsaBlockIDToLabels {
		irBlocks[l] = ssa.BasicBlockID(i)
	}

	var lines []string
	for cur := ectx.RootInstr; cur != nil; cur = cur.next {
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
	i.asRet()
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
	return m.arg0OffsetFromSP() + m.currentABI.ArgStackSize
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

func (m *machine) addJmpTableTarget(targets []ssa.BasicBlock) (index int) {
	// TODO: reuse the slice!
	labels := make([]uint32, len(targets))
	for j, target := range targets {
		labels[j] = uint32(m.executableContext.GetOrAllocateSSABlockLabel(target))
	}
	index = len(m.jmpTableTargets)
	m.jmpTableTargets = append(m.jmpTableTargets, labels)
	return
}
