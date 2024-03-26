package backend

import (
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

type ExecutableContext interface {
	// StartLoweringFunction is called when the lowering of the given function is started.
	// maximumBlockID is the maximum value of ssa.BasicBlockID existing in the function.
	StartLoweringFunction(maximumBlockID ssa.BasicBlockID)

	// LinkAdjacentBlocks is called after finished lowering all blocks in order to create one single instruction list.
	LinkAdjacentBlocks(prev, next ssa.BasicBlock)

	// StartBlock is called when the compilation of the given block is started.
	// The order of this being called is the reverse post order of the ssa.BasicBlock(s) as we iterate with
	// ssa.Builder BlockIteratorReversePostOrderBegin and BlockIteratorReversePostOrderEnd.
	StartBlock(ssa.BasicBlock)

	// EndBlock is called when the compilation of the current block is finished.
	EndBlock()

	// FlushPendingInstructions flushes the pending instructions to the buffer.
	// This will be called after the lowering of each SSA Instruction.
	FlushPendingInstructions()
}

type ExecutableContextT[Instr any] struct {
	CurrentSSABlk ssa.BasicBlock

	// InstrPool is the InstructionPool of instructions.
	InstructionPool wazevoapi.Pool[Instr]
	asNop           func(*Instr)
	setNext         func(*Instr, *Instr)
	setPrev         func(*Instr, *Instr)

	// RootInstr is the root instruction of the executable.
	RootInstr         *Instr
	labelPositionPool wazevoapi.Pool[LabelPosition[Instr]]
	NextLabel         Label
	// LabelPositions maps a label to the instructions of the region which the label represents.
	LabelPositions     map[Label]*LabelPosition[Instr]
	OrderedBlockLabels []*LabelPosition[Instr]

	// PerBlockHead and PerBlockEnd are the head and tail of the instruction list per currently-compiled ssa.BasicBlock.
	PerBlockHead, PerBlockEnd *Instr
	// PendingInstructions are the instructions which are not yet emitted into the instruction list.
	PendingInstructions []*Instr

	// SsaBlockIDToLabels maps an SSA block ID to the label.
	SsaBlockIDToLabels []Label
}

func NewExecutableContextT[Instr any](
	resetInstruction func(*Instr),
	setNext func(*Instr, *Instr),
	setPrev func(*Instr, *Instr),
	asNop func(*Instr),
) *ExecutableContextT[Instr] {
	return &ExecutableContextT[Instr]{
		InstructionPool:   wazevoapi.NewPool[Instr](resetInstruction),
		asNop:             asNop,
		setNext:           setNext,
		setPrev:           setPrev,
		labelPositionPool: wazevoapi.NewPool[LabelPosition[Instr]](resetLabelPosition[Instr]),
		LabelPositions:    make(map[Label]*LabelPosition[Instr]),
		NextLabel:         LabelInvalid,
	}
}

func resetLabelPosition[T any](l *LabelPosition[T]) {
	*l = LabelPosition[T]{}
}

// StartLoweringFunction implements ExecutableContext.
func (e *ExecutableContextT[Instr]) StartLoweringFunction(max ssa.BasicBlockID) {
	imax := int(max)
	if len(e.SsaBlockIDToLabels) <= imax {
		// Eagerly allocate labels for the blocks since the underlying slice will be used for the next iteration.
		e.SsaBlockIDToLabels = append(e.SsaBlockIDToLabels, make([]Label, imax+1)...)
	}
}

func (e *ExecutableContextT[Instr]) StartBlock(blk ssa.BasicBlock) {
	e.CurrentSSABlk = blk

	l := e.SsaBlockIDToLabels[e.CurrentSSABlk.ID()]
	if l == LabelInvalid {
		l = e.AllocateLabel()
		e.SsaBlockIDToLabels[blk.ID()] = l
	}

	end := e.allocateNop0()
	e.PerBlockHead, e.PerBlockEnd = end, end

	labelPos, ok := e.LabelPositions[l]
	if !ok {
		labelPos = e.AllocateLabelPosition(l)
		e.LabelPositions[l] = labelPos
	}
	e.OrderedBlockLabels = append(e.OrderedBlockLabels, labelPos)
	labelPos.Begin, labelPos.End = end, end
	labelPos.SB = blk
}

// EndBlock implements ExecutableContext.
func (e *ExecutableContextT[T]) EndBlock() {
	// Insert nop0 as the head of the block for convenience to simplify the logic of inserting instructions.
	e.insertAtPerBlockHead(e.allocateNop0())

	l := e.SsaBlockIDToLabels[e.CurrentSSABlk.ID()]
	e.LabelPositions[l].Begin = e.PerBlockHead

	if e.CurrentSSABlk.EntryBlock() {
		e.RootInstr = e.PerBlockHead
	}
}

func (e *ExecutableContextT[T]) insertAtPerBlockHead(i *T) {
	if e.PerBlockHead == nil {
		e.PerBlockHead = i
		e.PerBlockEnd = i
		return
	}
	e.setNext(i, e.PerBlockHead)
	e.setPrev(e.PerBlockHead, i)
	e.PerBlockHead = i
}

// FlushPendingInstructions implements ExecutableContext.
func (e *ExecutableContextT[T]) FlushPendingInstructions() {
	l := len(e.PendingInstructions)
	if l == 0 {
		return
	}
	for i := l - 1; i >= 0; i-- { // reverse because we lower instructions in reverse order.
		e.insertAtPerBlockHead(e.PendingInstructions[i])
	}
	e.PendingInstructions = e.PendingInstructions[:0]
}

func (e *ExecutableContextT[T]) Reset() {
	e.labelPositionPool.Reset()
	e.InstructionPool.Reset()
	for l := Label(0); l <= e.NextLabel; l++ {
		delete(e.LabelPositions, l)
	}
	e.PendingInstructions = e.PendingInstructions[:0]
	e.OrderedBlockLabels = e.OrderedBlockLabels[:0]
	e.RootInstr = nil
	e.SsaBlockIDToLabels = e.SsaBlockIDToLabels[:0]
	e.PerBlockHead, e.PerBlockEnd = nil, nil
	e.NextLabel = LabelInvalid
}

// AllocateLabel allocates an unused label.
func (e *ExecutableContextT[T]) AllocateLabel() Label {
	e.NextLabel++
	return e.NextLabel
}

func (e *ExecutableContextT[T]) AllocateLabelPosition(la Label) *LabelPosition[T] {
	l := e.labelPositionPool.Allocate()
	l.L = la
	return l
}

func (e *ExecutableContextT[T]) GetOrAllocateSSABlockLabel(blk ssa.BasicBlock) Label {
	if blk.ReturnBlock() {
		return LabelReturn
	}
	l := e.SsaBlockIDToLabels[blk.ID()]
	if l == LabelInvalid {
		l = e.AllocateLabel()
		e.SsaBlockIDToLabels[blk.ID()] = l
	}
	return l
}

func (e *ExecutableContextT[T]) allocateNop0() *T {
	i := e.InstructionPool.Allocate()
	e.asNop(i)
	return i
}

// LinkAdjacentBlocks implements backend.Machine.
func (e *ExecutableContextT[T]) LinkAdjacentBlocks(prev, next ssa.BasicBlock) {
	prevLabelPos := e.LabelPositions[e.GetOrAllocateSSABlockLabel(prev)]
	nextLabelPos := e.LabelPositions[e.GetOrAllocateSSABlockLabel(next)]
	e.setNext(prevLabelPos.End, nextLabelPos.Begin)
}

// LabelPosition represents the regions of the generated code which the label represents.
type LabelPosition[Instr any] struct {
	SB           ssa.BasicBlock
	L            Label
	Begin, End   *Instr
	BinaryOffset int64
}

// Label represents a position in the generated code which is either
// a real instruction or the constant InstructionPool (e.g. jump tables).
//
// This is exactly the same as the traditional "label" in assembly code.
type Label uint32

const (
	LabelInvalid Label = 0
	LabelReturn  Label = math.MaxUint32
)

// String implements backend.Machine.
func (l Label) String() string {
	return fmt.Sprintf("L%d", l)
}
