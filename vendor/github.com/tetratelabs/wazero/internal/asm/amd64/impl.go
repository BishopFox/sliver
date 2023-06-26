package amd64

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/internal/asm"
)

// nodeImpl implements asm.Node for amd64.
type nodeImpl struct {
	// jumpTarget holds the target node in the linked for the jump-kind instruction.
	jumpTarget *nodeImpl

	// prev and next hold the prev/next node from this node in the assembled linked list.
	prev, next *nodeImpl

	// forwardJumpOrigins hold all the nodes trying to jump into this node as a
	// singly linked list. In other words, all the nodes with .jumpTarget == this.
	forwardJumpOrigins *nodeImpl

	staticConst *asm.StaticConst

	dstConst       asm.ConstantValue
	offsetInBinary asm.NodeOffsetInBinary
	srcConst       asm.ConstantValue
	instruction    asm.Instruction

	// readInstructionAddressBeforeTargetInstruction holds the instruction right before the target of
	// read instruction address instruction. See asm.assemblerBase.CompileReadInstructionAddress.
	readInstructionAddressBeforeTargetInstruction asm.Instruction
	flag                                          nodeFlag
	types                                         operandTypes
	srcReg, dstReg                                asm.Register
	srcMemIndex, dstMemIndex                      asm.Register
	srcMemScale, dstMemScale                      byte
	arg                                           byte

	// staticConstReferrersAdded true if this node is already added into AssemblerImpl.staticConstReferrers.
	// Only used when staticConst is not nil. Through re-assembly, we might end up adding multiple times which causes unnecessary
	// allocations, so we use this flag to do it once.
	staticConstReferrersAdded bool
}

type nodeFlag byte

const (
	// nodeFlagInitializedForEncoding is always set to indicate that node is already initialized. Notably, this is used to judge
	// whether a jump is backward or forward before encoding.
	nodeFlagInitializedForEncoding nodeFlag = 1 << iota
	nodeFlagBackwardJump
	// nodeFlagShortForwardJump is set to false by default and only used by forward branch jumps, which means .jumpTarget != nil and
	// the target node is encoded after this node. False by default means that we Encode all the jumps with jumpTarget
	// as short jump (i.e. relative signed 8-bit integer offset jump) and try to Encode as small as possible.
	nodeFlagShortForwardJump
)

func (n *nodeImpl) isInitializedForEncoding() bool {
	return n.flag&nodeFlagInitializedForEncoding != 0
}

func (n *nodeImpl) isJumpNode() bool {
	return n.jumpTarget != nil
}

func (n *nodeImpl) isBackwardJump() bool {
	return n.isJumpNode() && (n.flag&nodeFlagBackwardJump != 0)
}

func (n *nodeImpl) isForwardJump() bool {
	return n.isJumpNode() && (n.flag&nodeFlagBackwardJump == 0)
}

func (n *nodeImpl) isForwardShortJump() bool {
	return n.isForwardJump() && n.flag&nodeFlagShortForwardJump != 0
}

// AssignJumpTarget implements asm.Node.AssignJumpTarget.
func (n *nodeImpl) AssignJumpTarget(target asm.Node) {
	n.jumpTarget = target.(*nodeImpl)
}

// AssignDestinationConstant implements asm.Node.AssignDestinationConstant.
func (n *nodeImpl) AssignDestinationConstant(value asm.ConstantValue) {
	n.dstConst = value
}

// AssignSourceConstant implements asm.Node.AssignSourceConstant.
func (n *nodeImpl) AssignSourceConstant(value asm.ConstantValue) {
	n.srcConst = value
}

// OffsetInBinary implements asm.Node.OffsetInBinary.
func (n *nodeImpl) OffsetInBinary() asm.NodeOffsetInBinary {
	return n.offsetInBinary
}

// String implements fmt.Stringer.
//
// This is for debugging purpose, and the format is almost same as the AT&T assembly syntax,
// meaning that this should look like "INSTRUCTION ${from}, ${to}" where each operand
// might be embraced by '[]' to represent the memory location.
func (n *nodeImpl) String() (ret string) {
	instName := InstructionName(n.instruction)
	switch n.types {
	case operandTypesNoneToNone:
		ret = instName
	case operandTypesNoneToRegister:
		ret = fmt.Sprintf("%s %s", instName, RegisterName(n.dstReg))
	case operandTypesNoneToMemory:
		if n.dstMemIndex != asm.NilRegister {
			ret = fmt.Sprintf("%s [%s + 0x%x + %s*0x%x]", instName,
				RegisterName(n.dstReg), n.dstConst, RegisterName(n.dstMemIndex), n.dstMemScale)
		} else {
			ret = fmt.Sprintf("%s [%s + 0x%x]", instName, RegisterName(n.dstReg), n.dstConst)
		}
	case operandTypesNoneToBranch:
		ret = fmt.Sprintf("%s {%v}", instName, n.jumpTarget)
	case operandTypesRegisterToNone:
		ret = fmt.Sprintf("%s %s", instName, RegisterName(n.srcReg))
	case operandTypesRegisterToRegister:
		ret = fmt.Sprintf("%s %s, %s", instName, RegisterName(n.srcReg), RegisterName(n.dstReg))
	case operandTypesRegisterToMemory:
		if n.dstMemIndex != asm.NilRegister {
			ret = fmt.Sprintf("%s %s, [%s + 0x%x + %s*0x%x]", instName, RegisterName(n.srcReg),
				RegisterName(n.dstReg), n.dstConst, RegisterName(n.dstMemIndex), n.dstMemScale)
		} else {
			ret = fmt.Sprintf("%s %s, [%s + 0x%x]", instName, RegisterName(n.srcReg), RegisterName(n.dstReg), n.dstConst)
		}
	case operandTypesRegisterToConst:
		ret = fmt.Sprintf("%s %s, 0x%x", instName, RegisterName(n.srcReg), n.dstConst)
	case operandTypesMemoryToRegister:
		if n.srcMemIndex != asm.NilRegister {
			ret = fmt.Sprintf("%s [%s + %#x + %s*%#x], %s", instName,
				RegisterName(n.srcReg), n.srcConst, RegisterName(n.srcMemIndex), n.srcMemScale, RegisterName(n.dstReg))
		} else {
			ret = fmt.Sprintf("%s [%s + 0x%x], %s", instName, RegisterName(n.srcReg), n.srcConst, RegisterName(n.dstReg))
		}
	case operandTypesMemoryToConst:
		if n.srcMemIndex != asm.NilRegister {
			ret = fmt.Sprintf("%s [%s + %#x + %s*0x%x], 0x%x", instName,
				RegisterName(n.srcReg), n.srcConst, RegisterName(n.srcMemIndex), n.srcMemScale, n.dstConst)
		} else {
			ret = fmt.Sprintf("%s [%s + %#x], 0x%x", instName, RegisterName(n.srcReg), n.srcConst, n.dstConst)
		}
	case operandTypesConstToMemory:
		if n.dstMemIndex != asm.NilRegister {
			ret = fmt.Sprintf("%s 0x%x, [%s + 0x%x + %s*0x%x]", instName, n.srcConst,
				RegisterName(n.dstReg), n.dstConst, RegisterName(n.dstMemIndex), n.dstMemScale)
		} else {
			ret = fmt.Sprintf("%s 0x%x, [%s + 0x%x]", instName, n.srcConst, RegisterName(n.dstReg), n.dstConst)
		}
	case operandTypesConstToRegister:
		ret = fmt.Sprintf("%s 0x%x, %s", instName, n.srcConst, RegisterName(n.dstReg))
	case operandTypesStaticConstToRegister:
		ret = fmt.Sprintf("%s $%#x, %s", instName, n.staticConst.Raw, RegisterName(n.dstReg))
	case operandTypesRegisterToStaticConst:
		ret = fmt.Sprintf("%s %s, $%#x", instName, RegisterName(n.srcReg), n.staticConst.Raw)
	}
	return
}

type operandTypes byte

const (
	operandTypesNoneToNone operandTypes = iota
	operandTypesNoneToRegister
	operandTypesNoneToMemory
	operandTypesNoneToBranch
	operandTypesRegisterToNone
	operandTypesRegisterToRegister
	operandTypesRegisterToMemory
	operandTypesRegisterToConst
	operandTypesMemoryToRegister
	operandTypesMemoryToConst
	operandTypesConstToRegister
	operandTypesConstToMemory
	operandTypesStaticConstToRegister
	operandTypesRegisterToStaticConst
)

// String implements fmt.Stringer
func (o operandTypes) String() (ret string) {
	switch o {
	case operandTypesNoneToNone:
		ret = "NoneToNone"
	case operandTypesNoneToRegister:
		ret = "NoneToRegister"
	case operandTypesNoneToMemory:
		ret = "NoneToMemory"
	case operandTypesNoneToBranch:
		ret = "NoneToBranch"
	case operandTypesRegisterToNone:
		ret = "RegisterToNone"
	case operandTypesRegisterToRegister:
		ret = "RegisterToRegister"
	case operandTypesRegisterToMemory:
		ret = "RegisterToMemory"
	case operandTypesRegisterToConst:
		ret = "RegisterToConst"
	case operandTypesMemoryToRegister:
		ret = "MemoryToRegister"
	case operandTypesMemoryToConst:
		ret = "MemoryToConst"
	case operandTypesConstToRegister:
		ret = "ConstToRegister"
	case operandTypesConstToMemory:
		ret = "ConstToMemory"
	case operandTypesStaticConstToRegister:
		ret = "StaticConstToRegister"
	case operandTypesRegisterToStaticConst:
		ret = "RegisterToStaticConst"
	}
	return
}

type (
	// AssemblerImpl implements Assembler.
	AssemblerImpl struct {
		root    *nodeImpl
		current *nodeImpl
		asm.BaseAssemblerImpl
		readInstructionAddressNodes []*nodeImpl

		// staticConstReferrers maintains the list of static const referrers which requires the
		// offset resolution after finalizing the binary layout.
		staticConstReferrers []staticConstReferrer

		nodePool nodePool
		pool     asm.StaticConstPool

		// MaxDisplacementForConstantPool is fixed to defaultMaxDisplacementForConstantPool
		// but have it as an exported field here for testability.
		MaxDisplacementForConstantPool int

		forceReAssemble bool
	}

	// staticConstReferrer represents a referrer of a asm.StaticConst.
	staticConstReferrer struct {
		n *nodeImpl
		// instLen is the encoded length of the instruction for `n`.
		instLen int
	}
)

func NewAssembler() *AssemblerImpl {
	return &AssemblerImpl{
		nodePool:                       nodePool{index: nodePageSize},
		pool:                           asm.NewStaticConstPool(),
		MaxDisplacementForConstantPool: defaultMaxDisplacementForConstantPool,
	}
}

const nodePageSize = 128

type nodePage = [nodePageSize]nodeImpl

// nodePool is the central allocation pool for nodeImpl used by a single AssemblerImpl.
// This reduces the allocations over compilation by reusing AssemblerImpl.
type nodePool struct {
	pages []*nodePage
	index int
}

// allocNode allocates a new nodeImpl for use from the pool.
// This expands the pool if there is no space left for it.
func (n *nodePool) allocNode() *nodeImpl {
	if n.index == nodePageSize {
		if len(n.pages) == cap(n.pages) {
			n.pages = append(n.pages, new(nodePage))
		} else {
			i := len(n.pages)
			n.pages = n.pages[:i+1]
			if n.pages[i] == nil {
				n.pages[i] = new(nodePage)
			}
		}
		n.index = 0
	}
	ret := &n.pages[len(n.pages)-1][n.index]
	n.index++
	return ret
}

func (n *nodePool) reset() {
	for _, ns := range n.pages {
		pages := ns[:]
		for i := range pages {
			pages[i] = nodeImpl{}
		}
	}
	n.pages = n.pages[:0]
	n.index = nodePageSize
}

// AllocateNOP implements asm.AssemblerBase.
func (a *AssemblerImpl) AllocateNOP() asm.Node {
	n := a.nodePool.allocNode()
	n.instruction = NOP
	n.types = operandTypesNoneToNone
	return n
}

// Add implements asm.AssemblerBase.
func (a *AssemblerImpl) Add(n asm.Node) {
	a.addNode(n.(*nodeImpl))
}

// Reset implements asm.AssemblerBase.
func (a *AssemblerImpl) Reset() {
	pool := a.pool
	pool.Reset()
	*a = AssemblerImpl{
		nodePool:                    a.nodePool,
		pool:                        pool,
		readInstructionAddressNodes: a.readInstructionAddressNodes[:0],
		staticConstReferrers:        a.staticConstReferrers[:0],
		BaseAssemblerImpl: asm.BaseAssemblerImpl{
			SetBranchTargetOnNextNodes: a.SetBranchTargetOnNextNodes[:0],
			JumpTableEntries:           a.JumpTableEntries[:0],
		},
	}
	a.nodePool.reset()
}

// newNode creates a new Node and appends it into the linked list.
func (a *AssemblerImpl) newNode(instruction asm.Instruction, types operandTypes) *nodeImpl {
	n := a.nodePool.allocNode()
	n.instruction = instruction
	n.types = types
	a.addNode(n)
	return n
}

// addNode appends the new node into the linked list.
func (a *AssemblerImpl) addNode(node *nodeImpl) {
	if a.root == nil {
		a.root = node
		a.current = node
	} else {
		parent := a.current
		parent.next = node
		node.prev = parent
		a.current = node
	}

	for _, o := range a.SetBranchTargetOnNextNodes {
		origin := o.(*nodeImpl)
		origin.jumpTarget = node
	}
	// Reuse the underlying slice to avoid re-allocations.
	a.SetBranchTargetOnNextNodes = a.SetBranchTargetOnNextNodes[:0]
}

// encodeNode encodes the given node into writer.
func (a *AssemblerImpl) encodeNode(buf asm.Buffer, n *nodeImpl) (err error) {
	switch n.types {
	case operandTypesNoneToNone:
		err = a.encodeNoneToNone(buf, n)
	case operandTypesNoneToRegister:
		err = a.encodeNoneToRegister(buf, n)
	case operandTypesNoneToMemory:
		err = a.encodeNoneToMemory(buf, n)
	case operandTypesNoneToBranch:
		// Branching operand can be encoded as relative jumps.
		err = a.encodeRelativeJump(buf, n)
	case operandTypesRegisterToNone:
		err = a.encodeRegisterToNone(buf, n)
	case operandTypesRegisterToRegister:
		err = a.encodeRegisterToRegister(buf, n)
	case operandTypesRegisterToMemory:
		err = a.encodeRegisterToMemory(buf, n)
	case operandTypesRegisterToConst:
		err = a.encodeRegisterToConst(buf, n)
	case operandTypesMemoryToRegister:
		err = a.encodeMemoryToRegister(buf, n)
	case operandTypesMemoryToConst:
		err = a.encodeMemoryToConst(buf, n)
	case operandTypesConstToRegister:
		err = a.encodeConstToRegister(buf, n)
	case operandTypesConstToMemory:
		err = a.encodeConstToMemory(buf, n)
	case operandTypesStaticConstToRegister:
		err = a.encodeStaticConstToRegister(buf, n)
	case operandTypesRegisterToStaticConst:
		err = a.encodeRegisterToStaticConst(buf, n)
	default:
		err = fmt.Errorf("encoder undefined for [%s] operand type", n.types)
	}
	if err != nil {
		err = fmt.Errorf("%w: %s", err, n) // Ensure the error is debuggable by including the string value of the node.
	}
	return
}

// Assemble implements asm.AssemblerBase
func (a *AssemblerImpl) Assemble(buf asm.Buffer) error {
	a.initializeNodesForEncoding()

	// Continue encoding until we are not forced to re-assemble which happens when
	// a short relative jump ends up the offset larger than 8-bit length.
	for {
		err := a.encode(buf)
		if err != nil {
			return err
		}

		if !a.forceReAssemble {
			break
		} else {
			// We reset the length of buffer but don't delete the underlying slice since
			// the binary size will roughly the same after reassemble.
			buf.Reset()
			// Reset the re-assemble flag in order to avoid the infinite loop!
			a.forceReAssemble = false
		}
	}

	code := buf.Bytes()
	for _, n := range a.readInstructionAddressNodes {
		if err := a.finalizeReadInstructionAddressNode(code, n); err != nil {
			return err
		}
	}

	// Now that we've finished the layout, fill out static consts offsets.
	for i := range a.staticConstReferrers {
		ref := &a.staticConstReferrers[i]
		n, instLen := ref.n, ref.instLen
		// Calculate the displacement between the RIP (the offset _after_ n) and the static constant.
		displacement := int(n.staticConst.OffsetInBinary) - int(n.OffsetInBinary()) - instLen
		// The offset must be stored at the 4 bytes from the tail of this n. See AssemblerImpl.encodeStaticConstImpl for detail.
		displacementOffsetInInstruction := n.OffsetInBinary() + uint64(instLen-4)
		binary.LittleEndian.PutUint32(code[displacementOffsetInInstruction:], uint32(int32(displacement)))
	}

	return a.FinalizeJumpTableEntry(code)
}

// initializeNodesForEncoding initializes nodeImpl.flag and determine all the jumps
// are forward or backward jump.
func (a *AssemblerImpl) initializeNodesForEncoding() {
	for n := a.root; n != nil; n = n.next {
		n.flag |= nodeFlagInitializedForEncoding
		if target := n.jumpTarget; target != nil {
			if target.isInitializedForEncoding() {
				// This means the target exists behind.
				n.flag |= nodeFlagBackwardJump
			} else {
				// Otherwise, this is forward jump.
				// We start with assuming that the jump can be short (8-bit displacement).
				// If it doens't fit, we change this flag in resolveRelativeForwardJump.
				n.flag |= nodeFlagShortForwardJump

				// If the target node is also the branching instruction, we replace the target with the NOP
				// node so that we can avoid the collision of the target.forwardJumpOrigins both as destination and origins.
				if target.types == operandTypesNoneToBranch {
					// Allocate the NOP node from the pool.
					nop := a.nodePool.allocNode()
					nop.instruction = NOP
					nop.types = operandTypesNoneToNone
					// Insert it between target.prev and target: [target.prev, target] -> [target.prev, nop, target]
					prev := target.prev
					nop.prev = prev
					prev.next = nop
					nop.next = target
					target.prev = nop
					n.jumpTarget = nop
					target = nop
				}

				// We add this node `n` into the end of the linked list (.forwardJumpOrigins) beginning from the `target.forwardJumpOrigins`.
				// Insert the current `n` as the head of the list.
				n.forwardJumpOrigins = target.forwardJumpOrigins
				target.forwardJumpOrigins = n
			}
		}
	}
}

func (a *AssemblerImpl) encode(buf asm.Buffer) error {
	for n := a.root; n != nil; n = n.next {
		// If an instruction needs NOP padding, we do so before encoding it.
		//
		// This is necessary to avoid Intel's jump erratum; see in Section 2.1
		// in for when we have to pad NOP:
		// https://www.intel.com/content/dam/support/us/en/documents/processors/mitigations-jump-conditional-code-erratum.pdf
		//
		// This logic used to be implemented in a function called maybeNOPPadding,
		// but the complexity of the logic made it impossible for the compiler to
		// inline. Since this function is on a hot code path, we inlined the
		// initial checks to skip the function call when instructions do not need
		// NOP padding.
		switch info := nopPaddingInfo[n.instruction]; {
		case info.jmp:
			if err := a.encodeJmpNOPPadding(buf, n); err != nil {
				return err
			}
		case info.onNextJmp:
			if err := a.encodeOnNextJmpNOPPAdding(buf, n); err != nil {
				return err
			}
		}

		// After the padding, we can finalize the offset of this instruction in the binary.
		n.offsetInBinary = uint64(buf.Len())

		if err := a.encodeNode(buf, n); err != nil {
			return err
		}

		if n.forwardJumpOrigins != nil {
			if err := a.resolveForwardRelativeJumps(buf, n); err != nil {
				return fmt.Errorf("invalid relative forward jumps: %w", err)
			}
		}

		a.maybeFlushConstants(buf, n.next == nil)
	}
	return nil
}

var nopPaddingInfo = [instructionEnd]struct {
	jmp, onNextJmp bool
}{
	RET: {jmp: true},
	JMP: {jmp: true},
	JCC: {jmp: true},
	JCS: {jmp: true},
	JEQ: {jmp: true},
	JGE: {jmp: true},
	JGT: {jmp: true},
	JHI: {jmp: true},
	JLE: {jmp: true},
	JLS: {jmp: true},
	JLT: {jmp: true},
	JMI: {jmp: true},
	JNE: {jmp: true},
	JPC: {jmp: true},
	JPS: {jmp: true},
	// The possible fused jump instructions if the next node is a conditional jump instruction.
	CMPL:  {onNextJmp: true},
	CMPQ:  {onNextJmp: true},
	TESTL: {onNextJmp: true},
	TESTQ: {onNextJmp: true},
	ADDL:  {onNextJmp: true},
	ADDQ:  {onNextJmp: true},
	SUBL:  {onNextJmp: true},
	SUBQ:  {onNextJmp: true},
	ANDL:  {onNextJmp: true},
	ANDQ:  {onNextJmp: true},
	INCQ:  {onNextJmp: true},
	DECQ:  {onNextJmp: true},
}

func (a *AssemblerImpl) encodeJmpNOPPadding(buf asm.Buffer, n *nodeImpl) error {
	// In order to know the instruction length before writing into the binary,
	// we try encoding it.
	prevLen := buf.Len()

	// Assign the temporary offset which may or may not be correct depending on the padding decision.
	n.offsetInBinary = uint64(prevLen)

	// Encode the node and get the instruction length.
	if err := a.encodeNode(buf, n); err != nil {
		return err
	}
	instructionLen := int32(buf.Len() - prevLen)

	// Revert the written bytes.
	buf.Truncate(prevLen)
	return a.encodeNOPPadding(buf, instructionLen)
}

func (a *AssemblerImpl) encodeOnNextJmpNOPPAdding(buf asm.Buffer, n *nodeImpl) error {
	instructionLen, err := a.fusedInstructionLength(buf, n)
	if err != nil {
		return err
	}
	return a.encodeNOPPadding(buf, instructionLen)
}

// encodeNOPPadding maybe appends NOP instructions before the node `n`.
// This is necessary to avoid Intel's jump erratum:
// https://www.intel.com/content/dam/support/us/en/documents/processors/mitigations-jump-conditional-code-erratum.pdf
func (a *AssemblerImpl) encodeNOPPadding(buf asm.Buffer, instructionLen int32) error {
	const boundaryInBytes int32 = 32
	const mask = boundaryInBytes - 1
	var padNum int
	currentPos := int32(buf.Len())
	if used := currentPos & mask; used+instructionLen >= boundaryInBytes {
		padNum = int(boundaryInBytes - used)
	}
	a.padNOP(buf, padNum)
	return nil
}

// fusedInstructionLength returns the length of "macro fused instruction" if the
// instruction sequence starting from `n` can be fused by processor. Otherwise,
// returns zero.
func (a *AssemblerImpl) fusedInstructionLength(buf asm.Buffer, n *nodeImpl) (ret int32, err error) {
	// Find the next non-NOP instruction.
	next := n.next
	for ; next != nil && next.instruction == NOP; next = next.next {
	}

	if next == nil {
		return
	}

	inst, jmpInst := n.instruction, next.instruction

	if !nopPaddingInfo[jmpInst].jmp {
		// If the next instruction is not jump kind, the instruction will not be fused.
		return
	}

	// How to determine whether the instruction can be fused is described in
	// Section 3.4.2.2 of "Intel Optimization Manual":
	// https://www.intel.com/content/dam/doc/manual/64-ia-32-architectures-optimization-manual.pdf
	isTest := inst == TESTL || inst == TESTQ
	isCmp := inst == CMPQ || inst == CMPL
	isTestCmp := isTest || isCmp
	if isTestCmp && (n.types == operandTypesMemoryToConst || n.types == operandTypesConstToMemory) {
		// The manual says: "CMP and TEST can not be fused when comparing MEM-IMM".
		return
	}

	// Implement the decision according to the table 3-1 in the manual.
	isAnd := inst == ANDL || inst == ANDQ
	if !isTest && !isAnd {
		if jmpInst == JMI || jmpInst == JPL || jmpInst == JPS || jmpInst == JPC {
			// These jumps are only fused for TEST or AND.
			return
		}
		isAdd := inst == ADDL || inst == ADDQ
		isSub := inst == SUBL || inst == SUBQ
		if !isCmp && !isAdd && !isSub {
			if jmpInst == JCS || jmpInst == JCC || jmpInst == JHI || jmpInst == JLS {
				// Thses jumpst are only fused for TEST, AND, CMP, ADD, or SUB.
				return
			}
		}
	}

	// Now the instruction is ensured to be fused by the processor.
	// In order to know the fused instruction length before writing into the binary,
	// we try encoding it.
	savedLen := uint64(buf.Len())

	// Encode the nodes into the buffer.
	if err = a.encodeNode(buf, n); err != nil {
		return
	}
	if err = a.encodeNode(buf, next); err != nil {
		return
	}

	ret = int32(uint64(buf.Len()) - savedLen)

	// Revert the written bytes.
	buf.Truncate(int(savedLen))
	return
}

// nopOpcodes is the multi byte NOP instructions table derived from section 5.8 "Code Padding with Operand-Size Override and Multibyte NOP"
// in "AMD Software Optimization Guide for AMD Family 15h Processors" https://www.amd.com/system/files/TechDocs/47414_15h_sw_opt_guide.pdf
var nopOpcodes = [][11]byte{
	{0x90},
	{0x66, 0x90},
	{0x0f, 0x1f, 0x00},
	{0x0f, 0x1f, 0x40, 0x00},
	{0x0f, 0x1f, 0x44, 0x00, 0x00},
	{0x66, 0x0f, 0x1f, 0x44, 0x00, 0x00},
	{0x0f, 0x1f, 0x80, 0x00, 0x00, 0x00, 0x00},
	{0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x66, 0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x66, 0x66, 0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x66, 0x66, 0x66, 0x0f, 0x1f, 0x84, 0x00, 0x00, 0x00, 0x00, 0x00},
}

func (a *AssemblerImpl) padNOP(buf asm.Buffer, num int) {
	for num > 0 {
		singleNopNum := num
		if singleNopNum > len(nopOpcodes) {
			singleNopNum = len(nopOpcodes)
		}
		buf.AppendBytes(nopOpcodes[singleNopNum-1][:singleNopNum])
		num -= singleNopNum
	}
}

// CompileStandAlone implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileStandAlone(instruction asm.Instruction) asm.Node {
	return a.newNode(instruction, operandTypesNoneToNone)
}

// CompileConstToRegister implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileConstToRegister(
	instruction asm.Instruction,
	value asm.ConstantValue,
	destinationReg asm.Register,
) (inst asm.Node) {
	n := a.newNode(instruction, operandTypesConstToRegister)
	n.srcConst = value
	n.dstReg = destinationReg
	return n
}

// CompileRegisterToRegister implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileRegisterToRegister(instruction asm.Instruction, from, to asm.Register) {
	n := a.newNode(instruction, operandTypesRegisterToRegister)
	n.srcReg = from
	n.dstReg = to
}

// CompileMemoryToRegister implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileMemoryToRegister(
	instruction asm.Instruction,
	sourceBaseReg asm.Register,
	sourceOffsetConst asm.ConstantValue,
	destinationReg asm.Register,
) {
	n := a.newNode(instruction, operandTypesMemoryToRegister)
	n.srcReg = sourceBaseReg
	n.srcConst = sourceOffsetConst
	n.dstReg = destinationReg
}

// CompileRegisterToMemory implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileRegisterToMemory(
	instruction asm.Instruction,
	sourceRegister, destinationBaseRegister asm.Register,
	destinationOffsetConst asm.ConstantValue,
) {
	n := a.newNode(instruction, operandTypesRegisterToMemory)
	n.srcReg = sourceRegister
	n.dstReg = destinationBaseRegister
	n.dstConst = destinationOffsetConst
}

// CompileJump implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileJump(jmpInstruction asm.Instruction) asm.Node {
	return a.newNode(jmpInstruction, operandTypesNoneToBranch)
}

// CompileJumpToMemory implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileJumpToMemory(
	jmpInstruction asm.Instruction,
	baseReg asm.Register,
	offset asm.ConstantValue,
) {
	n := a.newNode(jmpInstruction, operandTypesNoneToMemory)
	n.dstReg = baseReg
	n.dstConst = offset
}

// CompileJumpToRegister implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileJumpToRegister(jmpInstruction asm.Instruction, reg asm.Register) {
	n := a.newNode(jmpInstruction, operandTypesNoneToRegister)
	n.dstReg = reg
}

// CompileReadInstructionAddress implements the same method as documented on asm.AssemblerBase.
func (a *AssemblerImpl) CompileReadInstructionAddress(
	destinationRegister asm.Register,
	beforeAcquisitionTargetInstruction asm.Instruction,
) {
	n := a.newNode(LEAQ, operandTypesMemoryToRegister)
	n.dstReg = destinationRegister
	n.readInstructionAddressBeforeTargetInstruction = beforeAcquisitionTargetInstruction
}

// CompileRegisterToRegisterWithArg implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileRegisterToRegisterWithArg(
	instruction asm.Instruction,
	from, to asm.Register,
	arg byte,
) {
	n := a.newNode(instruction, operandTypesRegisterToRegister)
	n.srcReg = from
	n.dstReg = to
	n.arg = arg
}

// CompileMemoryWithIndexToRegister implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileMemoryWithIndexToRegister(
	instruction asm.Instruction,
	srcBaseReg asm.Register,
	srcOffsetConst asm.ConstantValue,
	srcIndex asm.Register,
	srcScale int16,
	dstReg asm.Register,
) {
	n := a.newNode(instruction, operandTypesMemoryToRegister)
	n.srcReg = srcBaseReg
	n.srcConst = srcOffsetConst
	n.srcMemIndex = srcIndex
	n.srcMemScale = byte(srcScale)
	n.dstReg = dstReg
}

// CompileMemoryWithIndexAndArgToRegister implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileMemoryWithIndexAndArgToRegister(
	instruction asm.Instruction,
	srcBaseReg asm.Register,
	srcOffsetConst asm.ConstantValue,
	srcIndex asm.Register,
	srcScale int16,
	dstReg asm.Register,
	arg byte,
) {
	n := a.newNode(instruction, operandTypesMemoryToRegister)
	n.srcReg = srcBaseReg
	n.srcConst = srcOffsetConst
	n.srcMemIndex = srcIndex
	n.srcMemScale = byte(srcScale)
	n.dstReg = dstReg
	n.arg = arg
}

// CompileRegisterToMemoryWithIndex implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileRegisterToMemoryWithIndex(
	instruction asm.Instruction,
	srcReg, dstBaseReg asm.Register,
	dstOffsetConst asm.ConstantValue,
	dstIndex asm.Register,
	dstScale int16,
) {
	n := a.newNode(instruction, operandTypesRegisterToMemory)
	n.srcReg = srcReg
	n.dstReg = dstBaseReg
	n.dstConst = dstOffsetConst
	n.dstMemIndex = dstIndex
	n.dstMemScale = byte(dstScale)
}

// CompileRegisterToMemoryWithIndexAndArg implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileRegisterToMemoryWithIndexAndArg(
	instruction asm.Instruction,
	srcReg, dstBaseReg asm.Register,
	dstOffsetConst asm.ConstantValue,
	dstIndex asm.Register,
	dstScale int16,
	arg byte,
) {
	n := a.newNode(instruction, operandTypesRegisterToMemory)
	n.srcReg = srcReg
	n.dstReg = dstBaseReg
	n.dstConst = dstOffsetConst
	n.dstMemIndex = dstIndex
	n.dstMemScale = byte(dstScale)
	n.arg = arg
}

// CompileRegisterToConst implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileRegisterToConst(
	instruction asm.Instruction,
	srcRegister asm.Register,
	value asm.ConstantValue,
) asm.Node {
	n := a.newNode(instruction, operandTypesRegisterToConst)
	n.srcReg = srcRegister
	n.dstConst = value
	return n
}

// CompileRegisterToNone implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileRegisterToNone(instruction asm.Instruction, register asm.Register) {
	n := a.newNode(instruction, operandTypesRegisterToNone)
	n.srcReg = register
}

// CompileNoneToRegister implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileNoneToRegister(instruction asm.Instruction, register asm.Register) {
	n := a.newNode(instruction, operandTypesNoneToRegister)
	n.dstReg = register
}

// CompileNoneToMemory implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileNoneToMemory(
	instruction asm.Instruction,
	baseReg asm.Register,
	offset asm.ConstantValue,
) {
	n := a.newNode(instruction, operandTypesNoneToMemory)
	n.dstReg = baseReg
	n.dstConst = offset
}

// CompileConstToMemory implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileConstToMemory(
	instruction asm.Instruction,
	value asm.ConstantValue,
	dstbaseReg asm.Register,
	dstOffset asm.ConstantValue,
) asm.Node {
	n := a.newNode(instruction, operandTypesConstToMemory)
	n.srcConst = value
	n.dstReg = dstbaseReg
	n.dstConst = dstOffset
	return n
}

// CompileMemoryToConst implements the same method as documented on amd64.Assembler.
func (a *AssemblerImpl) CompileMemoryToConst(
	instruction asm.Instruction,
	srcBaseReg asm.Register,
	srcOffset, value asm.ConstantValue,
) asm.Node {
	n := a.newNode(instruction, operandTypesMemoryToConst)
	n.srcReg = srcBaseReg
	n.srcConst = srcOffset
	n.dstConst = value
	return n
}

func errorEncodingUnsupported(n *nodeImpl) error {
	return fmt.Errorf("%s is unsupported for %s type", InstructionName(n.instruction), n.types)
}

func (a *AssemblerImpl) encodeNoneToNone(buf asm.Buffer, n *nodeImpl) (err error) {
	// Throughout the encoding methods, we use this pair of base offset and
	// code buffer to write instructions.
	//
	// The code buffer is allocated at the end of the current buffer to a size
	// large enough to hold all the bytes that may be written by the method.
	//
	// We use Go's append builtin to write to the buffer because it allows the
	// compiler to generate much better code than if we made calls to write
	// methods to mutate an encapsulated byte slice.
	//
	// At the end of the method, we truncate the buffer size back to the base
	// plus the length of the code buffer so the end of the buffer points right
	// after the last byte that was written.
	base := buf.Len()
	code := buf.Append(4)[:0]

	switch n.instruction {
	case CDQ:
		// https://www.felixcloutier.com/x86/cwd:cdq:cqo
		code = append(code, 0x99)
	case CQO:
		// https://www.felixcloutier.com/x86/cwd:cdq:cqo
		code = append(code, rexPrefixW, 0x99)
	case NOP:
		// Simply optimize out the NOP instructions.
	case RET:
		// https://www.felixcloutier.com/x86/ret
		code = append(code, 0xc3)
	case UD2:
		// https://mudongliang.github.io/x86/html/file_module_x86_id_318.html
		code = append(code, 0x0f, 0x0b)
	case REPMOVSQ:
		code = append(code, 0xf3, rexPrefixW, 0xa5)
	case REPSTOSQ:
		code = append(code, 0xf3, rexPrefixW, 0xab)
	case STD:
		code = append(code, 0xfd)
	case CLD:
		code = append(code, 0xfc)
	default:
		err = errorEncodingUnsupported(n)
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeNoneToRegister(buf asm.Buffer, n *nodeImpl) (err error) {
	regBits, prefix := register3bits(n.dstReg, registerSpecifierPositionModRMFieldRM)

	// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
	modRM := 0b11_000_000 | // Specifying that opeand is register.
		regBits
	if n.instruction == JMP {
		// JMP's opcode is defined as "FF /4" meaning that we have to have "4"
		// in 4-6th bits in the ModRM byte. https://www.felixcloutier.com/x86/jmp
		modRM |= 0b00_100_000
	} else if n.instruction == NEGQ {
		prefix |= rexPrefixW
		modRM |= 0b00_011_000
	} else if n.instruction == INCQ {
		prefix |= rexPrefixW
	} else if n.instruction == DECQ {
		prefix |= rexPrefixW
		modRM |= 0b00_001_000
	} else {
		if RegSP <= n.dstReg && n.dstReg <= RegDI {
			// If the destination is one byte length register, we need to have the default prefix.
			// https: //wiki.osdev.org/X86-64_Instruction_Encoding#Registers
			prefix |= rexPrefixDefault
		}
	}

	base := buf.Len()
	code := buf.Append(4)[:0]

	if prefix != rexPrefixNone {
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#Encoding
		code = append(code, prefix)
	}

	switch n.instruction {
	case JMP:
		// https://www.felixcloutier.com/x86/jmp
		code = append(code, 0xff, modRM)
	case SETCC:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x93, modRM)
	case SETCS:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x92, modRM)
	case SETEQ:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x94, modRM)
	case SETGE:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9d, modRM)
	case SETGT:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9f, modRM)
	case SETHI:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x97, modRM)
	case SETLE:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9e, modRM)
	case SETLS:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x96, modRM)
	case SETLT:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9c, modRM)
	case SETNE:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x95, modRM)
	case SETPC:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9b, modRM)
	case SETPS:
		// https://www.felixcloutier.com/x86/setcc
		code = append(code, 0x0f, 0x9a, modRM)
	case NEGQ:
		// https://www.felixcloutier.com/x86/neg
		code = append(code, 0xf7, modRM)
	case INCQ:
		// https://www.felixcloutier.com/x86/inc
		code = append(code, 0xff, modRM)
	case DECQ:
		// https://www.felixcloutier.com/x86/dec
		code = append(code, 0xff, modRM)
	default:
		err = errorEncodingUnsupported(n)
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeNoneToMemory(buf asm.Buffer, n *nodeImpl) (err error) {
	rexPrefix, modRM, sbi, sbiExist, displacementWidth, err := n.getMemoryLocation(true)
	if err != nil {
		return err
	}

	var opcode byte
	switch n.instruction {
	case INCQ:
		// https://www.felixcloutier.com/x86/inc
		rexPrefix |= rexPrefixW
		opcode = 0xff
	case DECQ:
		// https://www.felixcloutier.com/x86/dec
		rexPrefix |= rexPrefixW
		modRM |= 0b00_001_000 // DEC needs "/1" extension in ModRM.
		opcode = 0xff
	case JMP:
		// https://www.felixcloutier.com/x86/jmp
		modRM |= 0b00_100_000 // JMP needs "/4" extension in ModRM.
		opcode = 0xff
	default:
		return errorEncodingUnsupported(n)
	}

	base := buf.Len()
	code := buf.Append(12)[:0]

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode, modRM)

	if sbiExist {
		code = append(code, sbi)
	}

	if displacementWidth != 0 {
		code = appendConst(code, n.dstConst, displacementWidth)
	}

	buf.Truncate(base + len(code))
	return
}

type relativeJumpOpcode struct{ short, long []byte }

func (o relativeJumpOpcode) instructionLen(short bool) int64 {
	if short {
		return int64(len(o.short)) + 1 // 1 byte = 8 bit offset
	} else {
		return int64(len(o.long)) + 4 // 4 byte = 32 bit offset
	}
}

var relativeJumpOpcodes = [...]relativeJumpOpcode{
	// https://www.felixcloutier.com/x86/jcc
	JCC: {short: []byte{0x73}, long: []byte{0x0f, 0x83}},
	JCS: {short: []byte{0x72}, long: []byte{0x0f, 0x82}},
	JEQ: {short: []byte{0x74}, long: []byte{0x0f, 0x84}},
	JGE: {short: []byte{0x7d}, long: []byte{0x0f, 0x8d}},
	JGT: {short: []byte{0x7f}, long: []byte{0x0f, 0x8f}},
	JHI: {short: []byte{0x77}, long: []byte{0x0f, 0x87}},
	JLE: {short: []byte{0x7e}, long: []byte{0x0f, 0x8e}},
	JLS: {short: []byte{0x76}, long: []byte{0x0f, 0x86}},
	JLT: {short: []byte{0x7c}, long: []byte{0x0f, 0x8c}},
	JMI: {short: []byte{0x78}, long: []byte{0x0f, 0x88}},
	JPL: {short: []byte{0x79}, long: []byte{0x0f, 0x89}},
	JNE: {short: []byte{0x75}, long: []byte{0x0f, 0x85}},
	JPC: {short: []byte{0x7b}, long: []byte{0x0f, 0x8b}},
	JPS: {short: []byte{0x7a}, long: []byte{0x0f, 0x8a}},
	// https://www.felixcloutier.com/x86/jmp
	JMP: {short: []byte{0xeb}, long: []byte{0xe9}},
}

func (a *AssemblerImpl) resolveForwardRelativeJumps(buf asm.Buffer, target *nodeImpl) (err error) {
	offsetInBinary := int64(target.OffsetInBinary())
	origin := target.forwardJumpOrigins
	for ; origin != nil; origin = origin.forwardJumpOrigins {
		shortJump := origin.isForwardShortJump()
		op := relativeJumpOpcodes[origin.instruction]
		instructionLen := op.instructionLen(shortJump)

		// Calculate the offset from the EIP (at the time of executing this jump instruction)
		// to the target instruction. This value is always >= 0 as here we only handle forward jumps.
		offset := offsetInBinary - (int64(origin.OffsetInBinary()) + instructionLen)
		if shortJump {
			if offset > math.MaxInt8 {
				// This forces reassemble in the outer loop inside AssemblerImpl.Assemble().
				a.forceReAssemble = true
				// From the next reAssemble phases, this forward jump will be encoded long jump and
				// allocate 32-bit offset bytes by default. This means that this `origin` node
				// will always enter the "long jump offset encoding" block below
				origin.flag ^= nodeFlagShortForwardJump
			} else {
				buf.Bytes()[origin.OffsetInBinary()+uint64(instructionLen)-1] = byte(offset)
			}
		} else { // long jump offset encoding.
			if offset > math.MaxInt32 {
				return fmt.Errorf("too large jump offset %d for encoding %s", offset, InstructionName(origin.instruction))
			}
			binary.LittleEndian.PutUint32(buf.Bytes()[origin.OffsetInBinary()+uint64(instructionLen)-4:], uint32(offset))
		}
	}
	return nil
}

func (a *AssemblerImpl) encodeRelativeJump(buf asm.Buffer, n *nodeImpl) (err error) {
	if n.jumpTarget == nil {
		err = fmt.Errorf("jump target must not be nil for relative %s", InstructionName(n.instruction))
		return
	}

	op := relativeJumpOpcodes[n.instruction]
	var isShortJump bool
	// offsetOfEIP means the offset of EIP register at the time of executing this jump instruction.
	// Relative jump instructions can be encoded with the signed 8-bit or 32-bit integer offsets from the EIP.
	var offsetOfEIP int64 = 0 // We set zero and resolve later once the target instruction is encoded for forward jumps
	if n.isBackwardJump() {
		// If this is the backward jump, we can calculate the exact offset now.
		offsetOfJumpInstruction := int64(n.jumpTarget.OffsetInBinary()) - int64(n.OffsetInBinary())
		isShortJump = offsetOfJumpInstruction-2 >= math.MinInt8
		offsetOfEIP = offsetOfJumpInstruction - op.instructionLen(isShortJump)
	} else {
		// For forward jumps, we resolve the offset when we Encode the target node. See AssemblerImpl.ResolveForwardRelativeJumps.
		isShortJump = n.isForwardShortJump()
	}

	if offsetOfEIP < math.MinInt32 { // offsetOfEIP is always <= 0 as we don't calculate it for forward jump here.
		return fmt.Errorf("too large jump offset %d for encoding %s", offsetOfEIP, InstructionName(n.instruction))
	}

	base := buf.Len()
	code := buf.Append(6)[:0]

	if isShortJump {
		code = append(code, op.short...)
		code = append(code, byte(offsetOfEIP))
	} else {
		code = append(code, op.long...)
		code = appendUint32(code, uint32(offsetOfEIP))
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeRegisterToNone(buf asm.Buffer, n *nodeImpl) (err error) {
	regBits, prefix := register3bits(n.srcReg, registerSpecifierPositionModRMFieldRM)

	// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
	modRM := 0b11_000_000 | // Specifying that opeand is register.
		regBits

	var opcode byte
	switch n.instruction {
	case DIVL:
		// https://www.felixcloutier.com/x86/div
		modRM |= 0b00_110_000
		opcode = 0xf7
	case DIVQ:
		// https://www.felixcloutier.com/x86/div
		prefix |= rexPrefixW
		modRM |= 0b00_110_000
		opcode = 0xf7
	case IDIVL:
		// https://www.felixcloutier.com/x86/idiv
		modRM |= 0b00_111_000
		opcode = 0xf7
	case IDIVQ:
		// https://www.felixcloutier.com/x86/idiv
		prefix |= rexPrefixW
		modRM |= 0b00_111_000
		opcode = 0xf7
	case MULL:
		// https://www.felixcloutier.com/x86/mul
		modRM |= 0b00_100_000
		opcode = 0xf7
	case MULQ:
		// https://www.felixcloutier.com/x86/mul
		prefix |= rexPrefixW
		modRM |= 0b00_100_000
		opcode = 0xf7
	default:
		err = errorEncodingUnsupported(n)
	}

	base := buf.Len()
	code := buf.Append(3)[:0]

	if prefix != rexPrefixNone {
		code = append(code, prefix)
	}

	code = append(code, opcode, modRM)

	buf.Truncate(base + len(code))
	return
}

var registerToRegisterOpcode = [instructionEnd]*struct {
	opcode          []byte
	rPrefix         rexPrefix
	mandatoryPrefix byte
	srcOnModRMReg   bool
	isSrc8bit       bool
	needArg         bool
}{
	// https://www.felixcloutier.com/x86/add
	ADDL: {opcode: []byte{0x1}, srcOnModRMReg: true},
	ADDQ: {opcode: []byte{0x1}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/and
	ANDL: {opcode: []byte{0x21}, srcOnModRMReg: true},
	ANDQ: {opcode: []byte{0x21}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/cmp
	CMPL: {opcode: []byte{0x39}},
	CMPQ: {opcode: []byte{0x39}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/cmovcc
	CMOVQCS: {opcode: []byte{0x0f, 0x42}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/addsd
	ADDSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x58}},
	// https://www.felixcloutier.com/x86/addss
	ADDSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x58}},
	// https://www.felixcloutier.com/x86/addpd
	ANDPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x54}},
	// https://www.felixcloutier.com/x86/addps
	ANDPS: {opcode: []byte{0x0f, 0x54}},
	// https://www.felixcloutier.com/x86/bsr
	BSRL: {opcode: []byte{0xf, 0xbd}},
	BSRQ: {opcode: []byte{0xf, 0xbd}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/comisd
	COMISD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x2f}},
	// https://www.felixcloutier.com/x86/comiss
	COMISS: {opcode: []byte{0x0f, 0x2f}},
	// https://www.felixcloutier.com/x86/cvtsd2ss
	CVTSD2SS: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x5a}},
	// https://www.felixcloutier.com/x86/cvtsi2sd
	CVTSL2SD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x2a}},
	// https://www.felixcloutier.com/x86/cvtsi2sd
	CVTSQ2SD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x2a}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/cvtsi2ss
	CVTSL2SS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x2a}},
	// https://www.felixcloutier.com/x86/cvtsi2ss
	CVTSQ2SS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x2a}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/cvtss2sd
	CVTSS2SD: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5a}},
	// https://www.felixcloutier.com/x86/cvttsd2si
	CVTTSD2SL: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x2c}},
	CVTTSD2SQ: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x2c}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/cvttss2si
	CVTTSS2SL: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x2c}},
	CVTTSS2SQ: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x2c}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/divsd
	DIVSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x5e}},
	// https://www.felixcloutier.com/x86/divss
	DIVSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5e}},
	// https://www.felixcloutier.com/x86/lzcnt
	LZCNTL: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xbd}},
	LZCNTQ: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xbd}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/maxsd
	MAXSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x5f}},
	// https://www.felixcloutier.com/x86/maxss
	MAXSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5f}},
	// https://www.felixcloutier.com/x86/minsd
	MINSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x5d}},
	// https://www.felixcloutier.com/x86/minss
	MINSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5d}},
	// https://www.felixcloutier.com/x86/movsx:movsxd
	MOVBLSX: {opcode: []byte{0x0f, 0xbe}, isSrc8bit: true},
	// https://www.felixcloutier.com/x86/movzx
	MOVBLZX: {opcode: []byte{0x0f, 0xb6}, isSrc8bit: true},
	// https://www.felixcloutier.com/x86/movzx
	MOVWLZX: {opcode: []byte{0x0f, 0xb7}, isSrc8bit: true},
	// https://www.felixcloutier.com/x86/movsx:movsxd
	MOVBQSX: {opcode: []byte{0x0f, 0xbe}, rPrefix: rexPrefixW, isSrc8bit: true},
	// https://www.felixcloutier.com/x86/movsx:movsxd
	MOVLQSX: {opcode: []byte{0x63}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/movsx:movsxd
	MOVWQSX: {opcode: []byte{0x0f, 0xbf}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/movsx:movsxd
	MOVWLSX: {opcode: []byte{0x0f, 0xbf}},
	// https://www.felixcloutier.com/x86/imul
	IMULQ: {opcode: []byte{0x0f, 0xaf}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/mulss
	MULSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x59}},
	// https://www.felixcloutier.com/x86/mulsd
	MULSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x59}},
	// https://www.felixcloutier.com/x86/or
	ORL: {opcode: []byte{0x09}, srcOnModRMReg: true},
	ORQ: {opcode: []byte{0x09}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/orpd
	ORPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x56}},
	// https://www.felixcloutier.com/x86/orps
	ORPS: {opcode: []byte{0x0f, 0x56}},
	// https://www.felixcloutier.com/x86/popcnt
	POPCNTL: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xb8}},
	POPCNTQ: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xb8}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/roundss
	ROUNDSS: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x0a}, needArg: true},
	// https://www.felixcloutier.com/x86/roundsd
	ROUNDSD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x0b}, needArg: true},
	// https://www.felixcloutier.com/x86/sqrtss
	SQRTSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x51}},
	// https://www.felixcloutier.com/x86/sqrtsd
	SQRTSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x51}},
	// https://www.felixcloutier.com/x86/sub
	SUBL: {opcode: []byte{0x29}, srcOnModRMReg: true},
	SUBQ: {opcode: []byte{0x29}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/subss
	SUBSS: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5c}},
	// https://www.felixcloutier.com/x86/subsd
	SUBSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x5c}},
	// https://www.felixcloutier.com/x86/test
	TESTL: {opcode: []byte{0x85}, srcOnModRMReg: true},
	TESTQ: {opcode: []byte{0x85}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/tzcnt
	TZCNTL: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xbc}},
	TZCNTQ: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xbc}, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/ucomisd
	UCOMISD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x2e}},
	// https://www.felixcloutier.com/x86/ucomiss
	UCOMISS: {opcode: []byte{0x0f, 0x2e}},
	// https://www.felixcloutier.com/x86/xchg
	XCHGQ: {opcode: []byte{0x87}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/xor
	XORL: {opcode: []byte{0x31}, srcOnModRMReg: true},
	XORQ: {opcode: []byte{0x31}, rPrefix: rexPrefixW, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/xorpd
	XORPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x57}},
	XORPS: {opcode: []byte{0x0f, 0x57}},
	// https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
	PINSRB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x20}, needArg: true},
	// https://www.felixcloutier.com/x86/pinsrw
	PINSRW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xc4}, needArg: true},
	// https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
	PINSRD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x22}, needArg: true},
	// https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
	PINSRQ: {mandatoryPrefix: 0x66, rPrefix: rexPrefixW, opcode: []byte{0x0f, 0x3a, 0x22}, needArg: true},
	// https://www.felixcloutier.com/x86/movdqu:vmovdqu8:vmovdqu16:vmovdqu32:vmovdqu64
	MOVDQU: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x6f}},
	// https://www.felixcloutier.com/x86/movdqa:vmovdqa32:vmovdqa64
	MOVDQA: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x6f}},
	// https://www.felixcloutier.com/x86/paddb:paddw:paddd:paddq
	PADDB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xfc}},
	PADDW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xfd}},
	PADDD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xfe}},
	PADDQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd4}},
	// https://www.felixcloutier.com/x86/psubb:psubw:psubd
	PSUBB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf8}},
	PSUBW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf9}},
	PSUBD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xfa}},
	// https://www.felixcloutier.com/x86/psubq
	PSUBQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xfb}},
	// https://www.felixcloutier.com/x86/addps
	ADDPS: {opcode: []byte{0x0f, 0x58}},
	// https://www.felixcloutier.com/x86/addpd
	ADDPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x58}},
	// https://www.felixcloutier.com/x86/subps
	SUBPS: {opcode: []byte{0x0f, 0x5c}},
	// https://www.felixcloutier.com/x86/subpd
	SUBPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x5c}},
	// https://www.felixcloutier.com/x86/pxor
	PXOR: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xef}},
	// https://www.felixcloutier.com/x86/pand
	PAND: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xdb}},
	// https://www.felixcloutier.com/x86/por
	POR: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xeb}},
	// https://www.felixcloutier.com/x86/pandn
	PANDN: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xdf}},
	// https://www.felixcloutier.com/x86/pshufb
	PSHUFB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x0}},
	// https://www.felixcloutier.com/x86/pshufd
	PSHUFD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x70}, needArg: true},
	// https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
	PEXTRB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x14}, needArg: true, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/pextrw
	PEXTRW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xc5}, needArg: true},
	// https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
	PEXTRD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x16}, needArg: true, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
	PEXTRQ: {rPrefix: rexPrefixW, mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x16}, needArg: true, srcOnModRMReg: true},
	// https://www.felixcloutier.com/x86/insertps
	INSERTPS: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x21}, needArg: true},
	// https://www.felixcloutier.com/x86/movlhps
	MOVLHPS: {opcode: []byte{0x0f, 0x16}},
	// https://www.felixcloutier.com/x86/ptest
	PTEST: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x17}},
	// https://www.felixcloutier.com/x86/pcmpeqb:pcmpeqw:pcmpeqd
	PCMPEQB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x74}},
	PCMPEQW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x75}},
	PCMPEQD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x76}},
	// https://www.felixcloutier.com/x86/pcmpeqq
	PCMPEQQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x29}},
	// https://www.felixcloutier.com/x86/paddusb:paddusw
	PADDUSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xdc}},
	// https://www.felixcloutier.com/x86/movsd
	MOVSD: {mandatoryPrefix: 0xf2, opcode: []byte{0x0f, 0x10}},
	// https://www.felixcloutier.com/x86/packsswb:packssdw
	PACKSSWB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x63}},
	// https://www.felixcloutier.com/x86/pmovmskb
	PMOVMSKB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd7}},
	// https://www.felixcloutier.com/x86/movmskps
	MOVMSKPS: {opcode: []byte{0x0f, 0x50}},
	// https://www.felixcloutier.com/x86/movmskpd
	MOVMSKPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x50}},
	// https://www.felixcloutier.com/x86/psraw:psrad:psraq
	PSRAD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe2}},
	// https://www.felixcloutier.com/x86/psraw:psrad:psraq
	PSRAW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe1}},
	// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
	PSRLQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd3}},
	// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
	PSRLD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd2}},
	// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
	PSRLW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd1}},
	// https://www.felixcloutier.com/x86/psllw:pslld:psllq
	PSLLW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf1}},
	// https://www.felixcloutier.com/x86/psllw:pslld:psllq
	PSLLD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf2}},
	// https://www.felixcloutier.com/x86/psllw:pslld:psllq
	PSLLQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf3}},
	// https://www.felixcloutier.com/x86/punpcklbw:punpcklwd:punpckldq:punpcklqdq
	PUNPCKLBW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x60}},
	// https://www.felixcloutier.com/x86/punpckhbw:punpckhwd:punpckhdq:punpckhqdq
	PUNPCKHBW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x68}},
	// https://www.felixcloutier.com/x86/cmpps
	CMPPS: {opcode: []byte{0x0f, 0xc2}, needArg: true},
	// https://www.felixcloutier.com/x86/cmppd
	CMPPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xc2}, needArg: true},
	// https://www.felixcloutier.com/x86/pcmpgtq
	PCMPGTQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x37}},
	// https://www.felixcloutier.com/x86/pcmpgtb:pcmpgtw:pcmpgtd
	PCMPGTD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x66}},
	// https://www.felixcloutier.com/x86/pcmpgtb:pcmpgtw:pcmpgtd
	PCMPGTW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x65}},
	// https://www.felixcloutier.com/x86/pcmpgtb:pcmpgtw:pcmpgtd
	PCMPGTB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x64}},
	// https://www.felixcloutier.com/x86/pminsd:pminsq
	PMINSD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x39}},
	// https://www.felixcloutier.com/x86/pmaxsb:pmaxsw:pmaxsd:pmaxsq
	PMAXSD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3d}},
	// https://www.felixcloutier.com/x86/pmaxsb:pmaxsw:pmaxsd:pmaxsq
	PMAXSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xee}},
	// https://www.felixcloutier.com/x86/pmaxsb:pmaxsw:pmaxsd:pmaxsq
	PMAXSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3c}},
	// https://www.felixcloutier.com/x86/pminsb:pminsw
	PMINSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xea}},
	// https://www.felixcloutier.com/x86/pminsb:pminsw
	PMINSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x38}},
	// https://www.felixcloutier.com/x86/pminud:pminuq
	PMINUD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3b}},
	// https://www.felixcloutier.com/x86/pminub:pminuw
	PMINUW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3a}},
	// https://www.felixcloutier.com/x86/pminub:pminuw
	PMINUB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xda}},
	// https://www.felixcloutier.com/x86/pmaxud:pmaxuq
	PMAXUD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3f}},
	// https://www.felixcloutier.com/x86/pmaxub:pmaxuw
	PMAXUW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x3e}},
	// https://www.felixcloutier.com/x86/pmaxub:pmaxuw
	PMAXUB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xde}},
	// https://www.felixcloutier.com/x86/pmullw
	PMULLW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd5}},
	// https://www.felixcloutier.com/x86/pmulld:pmullq
	PMULLD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x40}},
	// https://www.felixcloutier.com/x86/pmuludq
	PMULUDQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf4}},
	// https://www.felixcloutier.com/x86/psubsb:psubsw
	PSUBSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe8}},
	// https://www.felixcloutier.com/x86/psubsb:psubsw
	PSUBSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe9}},
	// https://www.felixcloutier.com/x86/psubusb:psubusw
	PSUBUSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd8}},
	// https://www.felixcloutier.com/x86/psubusb:psubusw
	PSUBUSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xd9}},
	// https://www.felixcloutier.com/x86/paddsb:paddsw
	PADDSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xed}},
	// https://www.felixcloutier.com/x86/paddsb:paddsw
	PADDSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xec}},
	// https://www.felixcloutier.com/x86/paddusb:paddusw
	PADDUSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xdd}},
	// https://www.felixcloutier.com/x86/pavgb:pavgw
	PAVGB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe0}},
	// https://www.felixcloutier.com/x86/pavgb:pavgw
	PAVGW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe3}},
	// https://www.felixcloutier.com/x86/pabsb:pabsw:pabsd:pabsq
	PABSB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x1c}},
	// https://www.felixcloutier.com/x86/pabsb:pabsw:pabsd:pabsq
	PABSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x1d}},
	// https://www.felixcloutier.com/x86/pabsb:pabsw:pabsd:pabsq
	PABSD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x1e}},
	// https://www.felixcloutier.com/x86/blendvpd
	BLENDVPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x15}},
	// https://www.felixcloutier.com/x86/maxpd
	MAXPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x5f}},
	// https://www.felixcloutier.com/x86/maxps
	MAXPS: {opcode: []byte{0x0f, 0x5f}},
	// https://www.felixcloutier.com/x86/minpd
	MINPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x5d}},
	// https://www.felixcloutier.com/x86/minps
	MINPS: {opcode: []byte{0x0f, 0x5d}},
	// https://www.felixcloutier.com/x86/andnpd
	ANDNPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x55}},
	// https://www.felixcloutier.com/x86/andnps
	ANDNPS: {opcode: []byte{0x0f, 0x55}},
	// https://www.felixcloutier.com/x86/mulps
	MULPS: {opcode: []byte{0x0f, 0x59}},
	// https://www.felixcloutier.com/x86/mulpd
	MULPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x59}},
	// https://www.felixcloutier.com/x86/divps
	DIVPS: {opcode: []byte{0x0f, 0x5e}},
	// https://www.felixcloutier.com/x86/divpd
	DIVPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x5e}},
	// https://www.felixcloutier.com/x86/sqrtps
	SQRTPS: {opcode: []byte{0x0f, 0x51}},
	// https://www.felixcloutier.com/x86/sqrtpd
	SQRTPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x51}},
	// https://www.felixcloutier.com/x86/roundps
	ROUNDPS: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x08}, needArg: true},
	// https://www.felixcloutier.com/x86/roundpd
	ROUNDPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x09}, needArg: true},
	// https://www.felixcloutier.com/x86/palignr
	PALIGNR: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x3a, 0x0f}, needArg: true},
	// https://www.felixcloutier.com/x86/punpcklbw:punpcklwd:punpckldq:punpcklqdq
	PUNPCKLWD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x61}},
	// https://www.felixcloutier.com/x86/punpckhbw:punpckhwd:punpckhdq:punpckhqdq
	PUNPCKHWD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x69}},
	// https://www.felixcloutier.com/x86/pmulhuw
	PMULHUW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe4}},
	// https://www.felixcloutier.com/x86/pmuldq
	PMULDQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x28}},
	// https://www.felixcloutier.com/x86/pmulhrsw
	PMULHRSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x0b}},
	// https://www.felixcloutier.com/x86/pmovsx
	PMOVSXBW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x20}},
	// https://www.felixcloutier.com/x86/pmovsx
	PMOVSXWD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x23}},
	// https://www.felixcloutier.com/x86/pmovsx
	PMOVSXDQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x25}},
	// https://www.felixcloutier.com/x86/pmovzx
	PMOVZXBW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x30}},
	// https://www.felixcloutier.com/x86/pmovzx
	PMOVZXWD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x33}},
	// https://www.felixcloutier.com/x86/pmovzx
	PMOVZXDQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x35}},
	// https://www.felixcloutier.com/x86/pmulhw
	PMULHW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe5}},
	// https://www.felixcloutier.com/x86/cmpps
	CMPEQPS: {opcode: []byte{0x0f, 0xc2}, needArg: true},
	// https://www.felixcloutier.com/x86/cmppd
	CMPEQPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xc2}, needArg: true},
	// https://www.felixcloutier.com/x86/cvttps2dq
	CVTTPS2DQ: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x5b}},
	// https://www.felixcloutier.com/x86/cvtdq2ps
	CVTDQ2PS: {opcode: []byte{0x0f, 0x5b}},
	// https://www.felixcloutier.com/x86/cvtdq2pd
	CVTDQ2PD: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0xe6}},
	// https://www.felixcloutier.com/x86/cvtpd2ps
	CVTPD2PS: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x5a}},
	// https://www.felixcloutier.com/x86/cvtps2pd
	CVTPS2PD: {opcode: []byte{0x0f, 0x5a}},
	// https://www.felixcloutier.com/x86/movupd
	MOVUPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x10}},
	// https://www.felixcloutier.com/x86/shufps
	SHUFPS: {opcode: []byte{0x0f, 0xc6}, needArg: true},
	// https://www.felixcloutier.com/x86/pmaddwd
	PMADDWD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xf5}},
	// https://www.felixcloutier.com/x86/unpcklps
	UNPCKLPS: {opcode: []byte{0x0f, 0x14}},
	// https://www.felixcloutier.com/x86/packuswb
	PACKUSWB: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x67}},
	// https://www.felixcloutier.com/x86/packsswb:packssdw
	PACKSSDW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x6b}},
	// https://www.felixcloutier.com/x86/packusdw
	PACKUSDW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x2b}},
	// https://www.felixcloutier.com/x86/pmaddubsw
	PMADDUBSW: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x38, 0x04}},
	// https://www.felixcloutier.com/x86/cvttpd2dq
	CVTTPD2DQ: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0xe6}},
}

var registerToRegisterShiftOpcode = [instructionEnd]*struct {
	opcode         []byte
	rPrefix        rexPrefix
	modRMExtension byte
}{
	// https://www.felixcloutier.com/x86/rcl:rcr:rol:ror
	ROLL: {opcode: []byte{0xd3}},
	ROLQ: {opcode: []byte{0xd3}, rPrefix: rexPrefixW},
	RORL: {opcode: []byte{0xd3}, modRMExtension: 0b00_001_000},
	RORQ: {opcode: []byte{0xd3}, modRMExtension: 0b00_001_000, rPrefix: rexPrefixW},
	// https://www.felixcloutier.com/x86/sal:sar:shl:shr
	SARL: {opcode: []byte{0xd3}, modRMExtension: 0b00_111_000},
	SARQ: {opcode: []byte{0xd3}, modRMExtension: 0b00_111_000, rPrefix: rexPrefixW},
	SHLL: {opcode: []byte{0xd3}, modRMExtension: 0b00_100_000},
	SHLQ: {opcode: []byte{0xd3}, modRMExtension: 0b00_100_000, rPrefix: rexPrefixW},
	SHRL: {opcode: []byte{0xd3}, modRMExtension: 0b00_101_000},
	SHRQ: {opcode: []byte{0xd3}, modRMExtension: 0b00_101_000, rPrefix: rexPrefixW},
}

func (a *AssemblerImpl) encodeRegisterToRegister(buf asm.Buffer, n *nodeImpl) (err error) {
	// Alias for readability
	inst := n.instruction
	base := buf.Len()
	code := buf.Append(8)[:0]

	switch inst {
	case MOVL, MOVQ:
		var (
			opcode          []byte
			mandatoryPrefix byte
			srcOnModRMReg   bool
			rPrefix         rexPrefix
		)
		srcIsFloat, dstIsFloat := isVectorRegister(n.srcReg), isVectorRegister(n.dstReg)
		f2f := srcIsFloat && dstIsFloat
		if f2f {
			// https://www.felixcloutier.com/x86/movq
			opcode, mandatoryPrefix = []byte{0x0f, 0x7e}, 0xf3
		} else if srcIsFloat && !dstIsFloat {
			// https://www.felixcloutier.com/x86/movd:movq
			opcode, mandatoryPrefix, srcOnModRMReg = []byte{0x0f, 0x7e}, 0x66, true
		} else if !srcIsFloat && dstIsFloat {
			// https://www.felixcloutier.com/x86/movd:movq
			opcode, mandatoryPrefix, srcOnModRMReg = []byte{0x0f, 0x6e}, 0x66, false
		} else {
			// https://www.felixcloutier.com/x86/mov
			opcode, srcOnModRMReg = []byte{0x89}, true
		}

		rexPrefix, modRM, err := n.getRegisterToRegisterModRM(srcOnModRMReg)
		if err != nil {
			return err
		}
		rexPrefix |= rPrefix

		if inst == MOVQ && !f2f {
			rexPrefix |= rexPrefixW
		}
		if mandatoryPrefix != 0 {
			code = append(code, mandatoryPrefix)
		}
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		code = append(code, opcode...)
		code = append(code, modRM)
		buf.Truncate(base + len(code))
		return nil
	}

	if op := registerToRegisterOpcode[inst]; op != nil {
		rexPrefix, modRM, err := n.getRegisterToRegisterModRM(op.srcOnModRMReg)
		if err != nil {
			return err
		}
		rexPrefix |= op.rPrefix

		if op.isSrc8bit && RegSP <= n.srcReg && n.srcReg <= RegDI {
			// If an operand register is 8-bit length of SP, BP, DI, or SI register, we need to have the default prefix.
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#Registers
			rexPrefix |= rexPrefixDefault
		}

		if op.mandatoryPrefix != 0 {
			code = append(code, op.mandatoryPrefix)
		}

		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		code = append(code, op.opcode...)
		code = append(code, modRM)

		if op.needArg {
			code = append(code, n.arg)
		}
	} else if op := registerToRegisterShiftOpcode[inst]; op != nil {
		reg3bits, rexPrefix := register3bits(n.dstReg, registerSpecifierPositionModRMFieldRM)
		rexPrefix |= op.rPrefix
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}

		// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
		modRM := 0b11_000_000 |
			(op.modRMExtension) |
			reg3bits
		code = append(code, op.opcode...)
		code = append(code, modRM)
	} else {
		return errorEncodingUnsupported(n)
	}

	buf.Truncate(base + len(code))
	return nil
}

func (a *AssemblerImpl) encodeRegisterToMemory(buf asm.Buffer, n *nodeImpl) (err error) {
	rexPrefix, modRM, sbi, sbiExist, displacementWidth, err := n.getMemoryLocation(true)
	if err != nil {
		return err
	}

	var opcode []byte
	var mandatoryPrefix byte
	var isShiftInstruction bool
	var needArg bool
	switch n.instruction {
	case CMPL:
		// https://www.felixcloutier.com/x86/cmp
		opcode = []byte{0x3b}
	case CMPQ:
		// https://www.felixcloutier.com/x86/cmp
		rexPrefix |= rexPrefixW
		opcode = []byte{0x3b}
	case MOVB:
		// https://www.felixcloutier.com/x86/mov
		opcode = []byte{0x88}
		// 1 byte register operands need default prefix for the following registers.
		if n.srcReg >= RegSP && n.srcReg <= RegDI {
			rexPrefix |= rexPrefixDefault
		}
	case MOVL:
		if isVectorRegister(n.srcReg) {
			// https://www.felixcloutier.com/x86/movd:movq
			opcode = []byte{0x0f, 0x7e}
			mandatoryPrefix = 0x66
		} else {
			// https://www.felixcloutier.com/x86/mov
			opcode = []byte{0x89}
		}
	case MOVQ:
		if isVectorRegister(n.srcReg) {
			// https://www.felixcloutier.com/x86/movq
			opcode = []byte{0x0f, 0xd6}
			mandatoryPrefix = 0x66
		} else {
			// https://www.felixcloutier.com/x86/mov
			rexPrefix |= rexPrefixW
			opcode = []byte{0x89}
		}
	case MOVW:
		// https://www.felixcloutier.com/x86/mov
		// Note: Need 0x66 to indicate that the operand size is 16-bit.
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#Operand-size_and_address-size_override_prefix
		mandatoryPrefix = 0x66
		opcode = []byte{0x89}
	case SARL:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		modRM |= 0b00_111_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case SARQ:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		rexPrefix |= rexPrefixW
		modRM |= 0b00_111_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case SHLL:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		modRM |= 0b00_100_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case SHLQ:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		rexPrefix |= rexPrefixW
		modRM |= 0b00_100_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case SHRL:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		modRM |= 0b00_101_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case SHRQ:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		rexPrefix |= rexPrefixW
		modRM |= 0b00_101_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case ROLL:
		// https://www.felixcloutier.com/x86/rcl:rcr:rol:ror
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case ROLQ:
		// https://www.felixcloutier.com/x86/rcl:rcr:rol:ror
		rexPrefix |= rexPrefixW
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case RORL:
		// https://www.felixcloutier.com/x86/rcl:rcr:rol:ror
		modRM |= 0b00_001_000
		opcode = []byte{0xd3}
		isShiftInstruction = true
	case RORQ:
		// https://www.felixcloutier.com/x86/rcl:rcr:rol:ror
		rexPrefix |= rexPrefixW
		opcode = []byte{0xd3}
		modRM |= 0b00_001_000
		isShiftInstruction = true
	case MOVDQU:
		// https://www.felixcloutier.com/x86/movdqu:vmovdqu8:vmovdqu16:vmovdqu32:vmovdqu64
		mandatoryPrefix = 0xf3
		opcode = []byte{0x0f, 0x7f}
	case PEXTRB: // https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x14}
		needArg = true
	case PEXTRW: // https://www.felixcloutier.com/x86/pextrw
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x15}
		needArg = true
	case PEXTRD: // https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x16}
		needArg = true
	case PEXTRQ: // https://www.felixcloutier.com/x86/pextrb:pextrd:pextrq
		mandatoryPrefix = 0x66
		rexPrefix |= rexPrefixW // REX.W
		opcode = []byte{0x0f, 0x3a, 0x16}
		needArg = true
	default:
		return errorEncodingUnsupported(n)
	}

	if !isShiftInstruction {
		srcReg3Bits, prefix := register3bits(n.srcReg, registerSpecifierPositionModRMFieldReg)

		rexPrefix |= prefix
		modRM |= srcReg3Bits << 3 // Place the source register on ModRM:reg
	} else {
		if n.srcReg != RegCX {
			return fmt.Errorf("shifting instruction %s require CX register as src but got %s", InstructionName(n.instruction), RegisterName(n.srcReg))
		}
	}

	base := buf.Len()
	code := buf.Append(16)[:0]

	if mandatoryPrefix != 0 {
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#Mandatory_prefix
		code = append(code, mandatoryPrefix)
	}

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode...)
	code = append(code, modRM)

	if sbiExist {
		code = append(code, sbi)
	}

	if displacementWidth != 0 {
		code = appendConst(code, n.dstConst, displacementWidth)
	}

	if needArg {
		code = append(code, n.arg)
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeRegisterToConst(buf asm.Buffer, n *nodeImpl) (err error) {
	regBits, prefix := register3bits(n.srcReg, registerSpecifierPositionModRMFieldRM)

	base := buf.Len()
	code := buf.Append(10)[:0]

	switch n.instruction {
	case CMPL, CMPQ:
		if n.instruction == CMPQ {
			prefix |= rexPrefixW
		}
		if prefix != rexPrefixNone {
			code = append(code, prefix)
		}
		is8bitConst := fitInSigned8bit(n.dstConst)
		// https://www.felixcloutier.com/x86/cmp
		if n.srcReg == RegAX && !is8bitConst {
			code = append(code, 0x3d)
		} else {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM := 0b11_000_000 | // Specifying that opeand is register.
				0b00_111_000 | // CMP with immediate needs "/7" extension.
				regBits
			if is8bitConst {
				code = append(code, 0x83, modRM)
			} else {
				code = append(code, 0x81, modRM)
			}
		}
	default:
		err = errorEncodingUnsupported(n)
	}

	if fitInSigned8bit(n.dstConst) {
		code = append(code, byte(n.dstConst))
	} else {
		code = appendUint32(code, uint32(n.dstConst))
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) finalizeReadInstructionAddressNode(code []byte, n *nodeImpl) (err error) {
	// Find the target instruction node.
	targetNode := n
	for ; targetNode != nil; targetNode = targetNode.next {
		if targetNode.instruction == n.readInstructionAddressBeforeTargetInstruction {
			targetNode = targetNode.next
			break
		}
	}

	if targetNode == nil {
		return errors.New("BUG: target instruction not found for read instruction address")
	}

	offset := targetNode.OffsetInBinary() - (n.OffsetInBinary() + 7 /* 7 = the length of the LEAQ instruction */)
	if offset >= math.MaxInt32 {
		return errors.New("BUG: too large offset for LEAQ instruction")
	}

	binary.LittleEndian.PutUint32(code[n.OffsetInBinary()+3:], uint32(int32(offset)))
	return nil
}

func (a *AssemblerImpl) encodeReadInstructionAddress(buf asm.Buffer, n *nodeImpl) error {
	dstReg3Bits, rexPrefix := register3bits(n.dstReg, registerSpecifierPositionModRMFieldReg)

	a.readInstructionAddressNodes = append(a.readInstructionAddressNodes, n)

	// https://www.felixcloutier.com/x86/lea
	opcode := byte(0x8d)
	rexPrefix |= rexPrefixW

	// https://wiki.osdev.org/X86-64_Instruction_Encoding#32.2F64-bit_addressing
	modRM := 0b00_000_101 | // Indicate "LEAQ [RIP + 32bit displacement], dstReg" encoding.
		(dstReg3Bits << 3) // Place the dstReg on ModRM:reg.

	code := buf.Append(7)
	code[0] = rexPrefix
	code[1] = opcode
	code[2] = modRM
	binary.LittleEndian.PutUint32(code[3:], 0) // Preserve
	return nil
}

func (a *AssemblerImpl) encodeMemoryToRegister(buf asm.Buffer, n *nodeImpl) (err error) {
	if n.instruction == LEAQ && n.readInstructionAddressBeforeTargetInstruction != NONE {
		return a.encodeReadInstructionAddress(buf, n)
	}

	rexPrefix, modRM, sbi, sbiExist, displacementWidth, err := n.getMemoryLocation(false)
	if err != nil {
		return err
	}

	dstReg3Bits, prefix := register3bits(n.dstReg, registerSpecifierPositionModRMFieldReg)
	rexPrefix |= prefix
	modRM |= dstReg3Bits << 3 // Place the destination register on ModRM:reg

	var mandatoryPrefix byte
	var opcode []byte
	var needArg bool

	switch n.instruction {
	case ADDL:
		// https://www.felixcloutier.com/x86/add
		opcode = []byte{0x03}
	case ADDQ:
		// https://www.felixcloutier.com/x86/add
		rexPrefix |= rexPrefixW
		opcode = []byte{0x03}
	case CMPL:
		// https://www.felixcloutier.com/x86/cmp
		opcode = []byte{0x39}
	case CMPQ:
		// https://www.felixcloutier.com/x86/cmp
		rexPrefix |= rexPrefixW
		opcode = []byte{0x39}
	case LEAQ:
		// https://www.felixcloutier.com/x86/lea
		rexPrefix |= rexPrefixW
		opcode = []byte{0x8d}
	case MOVBLSX:
		// https://www.felixcloutier.com/x86/movsx:movsxd
		opcode = []byte{0x0f, 0xbe}
	case MOVBLZX:
		// https://www.felixcloutier.com/x86/movzx
		opcode = []byte{0x0f, 0xb6}
	case MOVBQSX:
		// https://www.felixcloutier.com/x86/movsx:movsxd
		rexPrefix |= rexPrefixW
		opcode = []byte{0x0f, 0xbe}
	case MOVBQZX:
		// https://www.felixcloutier.com/x86/movzx
		rexPrefix |= rexPrefixW
		opcode = []byte{0x0f, 0xb6}
	case MOVLQSX:
		// https://www.felixcloutier.com/x86/movsx:movsxd
		rexPrefix |= rexPrefixW
		opcode = []byte{0x63}
	case MOVLQZX:
		// https://www.felixcloutier.com/x86/mov
		// Note: MOVLQZX means zero extending 32bit reg to 64-bit reg and
		// that is semantically equivalent to MOV 32bit to 32bit.
		opcode = []byte{0x8B}
	case MOVL:
		// https://www.felixcloutier.com/x86/mov
		// Note: MOVLQZX means zero extending 32bit reg to 64-bit reg and
		// that is semantically equivalent to MOV 32bit to 32bit.
		if isVectorRegister(n.dstReg) {
			// https://www.felixcloutier.com/x86/movd:movq
			opcode = []byte{0x0f, 0x6e}
			mandatoryPrefix = 0x66
		} else {
			// https://www.felixcloutier.com/x86/mov
			opcode = []byte{0x8B}
		}
	case MOVQ:
		if isVectorRegister(n.dstReg) {
			// https://www.felixcloutier.com/x86/movq
			opcode = []byte{0x0f, 0x7e}
			mandatoryPrefix = 0xf3
		} else {
			// https://www.felixcloutier.com/x86/mov
			rexPrefix |= rexPrefixW
			opcode = []byte{0x8B}
		}
	case MOVWLSX:
		// https://www.felixcloutier.com/x86/movsx:movsxd
		opcode = []byte{0x0f, 0xbf}
	case MOVWLZX:
		// https://www.felixcloutier.com/x86/movzx
		opcode = []byte{0x0f, 0xb7}
	case MOVWQSX:
		// https://www.felixcloutier.com/x86/movsx:movsxd
		rexPrefix |= rexPrefixW
		opcode = []byte{0x0f, 0xbf}
	case MOVWQZX:
		// https://www.felixcloutier.com/x86/movzx
		rexPrefix |= rexPrefixW
		opcode = []byte{0x0f, 0xb7}
	case SUBQ:
		// https://www.felixcloutier.com/x86/sub
		rexPrefix |= rexPrefixW
		opcode = []byte{0x2b}
	case SUBSD:
		// https://www.felixcloutier.com/x86/subsd
		opcode = []byte{0x0f, 0x5c}
		mandatoryPrefix = 0xf2
	case SUBSS:
		// https://www.felixcloutier.com/x86/subss
		opcode = []byte{0x0f, 0x5c}
		mandatoryPrefix = 0xf3
	case UCOMISD:
		// https://www.felixcloutier.com/x86/ucomisd
		opcode = []byte{0x0f, 0x2e}
		mandatoryPrefix = 0x66
	case UCOMISS:
		// https://www.felixcloutier.com/x86/ucomiss
		opcode = []byte{0x0f, 0x2e}
	case MOVDQU:
		// https://www.felixcloutier.com/x86/movdqu:vmovdqu8:vmovdqu16:vmovdqu32:vmovdqu64
		mandatoryPrefix = 0xf3
		opcode = []byte{0x0f, 0x6f}
	case PMOVSXBW: // https://www.felixcloutier.com/x86/pmovsx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x20}
	case PMOVSXWD: // https://www.felixcloutier.com/x86/pmovsx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x23}
	case PMOVSXDQ: // https://www.felixcloutier.com/x86/pmovsx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x25}
	case PMOVZXBW: // https://www.felixcloutier.com/x86/pmovzx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x30}
	case PMOVZXWD: // https://www.felixcloutier.com/x86/pmovzx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x33}
	case PMOVZXDQ: // https://www.felixcloutier.com/x86/pmovzx
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x38, 0x35}
	case PINSRB: // https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x20}
		needArg = true
	case PINSRW: // https://www.felixcloutier.com/x86/pinsrw
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0xc4}
		needArg = true
	case PINSRD: // https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x22}
		needArg = true
	case PINSRQ: // https://www.felixcloutier.com/x86/pinsrb:pinsrd:pinsrq
		rexPrefix |= rexPrefixW
		mandatoryPrefix = 0x66
		opcode = []byte{0x0f, 0x3a, 0x22}
		needArg = true
	default:
		return errorEncodingUnsupported(n)
	}

	base := buf.Len()
	code := buf.Append(16)[:0]

	if mandatoryPrefix != 0 {
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#Mandatory_prefix
		code = append(code, mandatoryPrefix)
	}

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode...)
	code = append(code, modRM)

	if sbiExist {
		code = append(code, sbi)
	}

	if displacementWidth != 0 {
		code = appendConst(code, n.srcConst, displacementWidth)
	}

	if needArg {
		code = append(code, n.arg)
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeConstToRegister(buf asm.Buffer, n *nodeImpl) (err error) {
	regBits, rexPrefix := register3bits(n.dstReg, registerSpecifierPositionModRMFieldRM)

	isFloatReg := isVectorRegister(n.dstReg)
	switch n.instruction {
	case PSLLD, PSLLQ, PSRLD, PSRLQ, PSRAW, PSRLW, PSLLW, PSRAD:
		if !isFloatReg {
			return fmt.Errorf("%s needs float register but got %s", InstructionName(n.instruction), RegisterName(n.dstReg))
		}
	default:
		if isFloatReg {
			return fmt.Errorf("%s needs int register but got %s", InstructionName(n.instruction), RegisterName(n.dstReg))
		}
	}

	if n.instruction != MOVQ && !fitIn32bit(n.srcConst) {
		return fmt.Errorf("constant must fit in 32-bit integer for %s, but got %d", InstructionName(n.instruction), n.srcConst)
	} else if (n.instruction == SHLQ || n.instruction == SHRQ) && (n.srcConst < 0 || n.srcConst > math.MaxUint8) {
		return fmt.Errorf("constant must fit in positive 8-bit integer for %s, but got %d", InstructionName(n.instruction), n.srcConst)
	} else if (n.instruction == PSLLD ||
		n.instruction == PSLLQ ||
		n.instruction == PSRLD ||
		n.instruction == PSRLQ) && (n.srcConst < math.MinInt8 || n.srcConst > math.MaxInt8) {
		return fmt.Errorf("constant must fit in signed 8-bit integer for %s, but got %d", InstructionName(n.instruction), n.srcConst)
	}

	base := buf.Len()
	code := buf.Append(32)[:0]

	isSigned8bitConst := fitInSigned8bit(n.srcConst)
	switch inst := n.instruction; inst {
	case ADDQ:
		// https://www.felixcloutier.com/x86/add
		rexPrefix |= rexPrefixW
		if n.dstReg == RegAX && !isSigned8bitConst {
			code = append(code, rexPrefix, 0x05)
		} else {
			modRM := 0b11_000_000 | // Specifying that opeand is register.
				regBits
			if isSigned8bitConst {
				code = append(code, rexPrefix, 0x83, modRM)
			} else {
				code = append(code, rexPrefix, 0x81, modRM)
			}
		}
		if isSigned8bitConst {
			code = append(code, byte(n.srcConst))
		} else {
			code = appendUint32(code, uint32(n.srcConst))
		}
	case ANDQ:
		// https://www.felixcloutier.com/x86/and
		rexPrefix |= rexPrefixW
		if n.dstReg == RegAX && !isSigned8bitConst {
			code = append(code, rexPrefix, 0x25)
		} else {
			modRM := 0b11_000_000 | // Specifying that opeand is register.
				0b00_100_000 | // AND with immediate needs "/4" extension.
				regBits
			if isSigned8bitConst {
				code = append(code, rexPrefix, 0x83, modRM)
			} else {
				code = append(code, rexPrefix, 0x81, modRM)
			}
		}
		if fitInSigned8bit(n.srcConst) {
			code = append(code, byte(n.srcConst))
		} else {
			code = appendUint32(code, uint32(n.srcConst))
		}
	case TESTQ:
		// https://www.felixcloutier.com/x86/test
		rexPrefix |= rexPrefixW
		if n.dstReg == RegAX && !isSigned8bitConst {
			code = append(code, rexPrefix, 0xa9)
		} else {
			modRM := 0b11_000_000 | // Specifying that operand is register
				regBits
			code = append(code, rexPrefix, 0xf7, modRM)
		}
		code = appendUint32(code, uint32(n.srcConst))
	case MOVL:
		// https://www.felixcloutier.com/x86/mov
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		code = append(code, 0xb8|regBits)
		code = appendUint32(code, uint32(n.srcConst))
	case MOVQ:
		// https://www.felixcloutier.com/x86/mov
		if fitIn32bit(n.srcConst) {
			if n.srcConst > math.MaxInt32 {
				if rexPrefix != rexPrefixNone {
					code = append(code, rexPrefix)
				}
				code = append(code, 0xb8|regBits)
			} else {
				rexPrefix |= rexPrefixW
				modRM := 0b11_000_000 | // Specifying that opeand is register.
					regBits
				code = append(code, rexPrefix, 0xc7, modRM)
			}
			code = appendUint32(code, uint32(n.srcConst))
		} else {
			rexPrefix |= rexPrefixW
			code = append(code, rexPrefix, 0xb8|regBits)
			code = appendUint64(code, uint64(n.srcConst))
		}
	case SHLQ:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		rexPrefix |= rexPrefixW
		modRM := 0b11_000_000 | // Specifying that opeand is register.
			0b00_100_000 | // SHL with immediate needs "/4" extension.
			regBits
		if n.srcConst == 1 {
			code = append(code, rexPrefix, 0xd1, modRM)
		} else {
			code = append(code, rexPrefix, 0xc1, modRM, byte(n.srcConst))
		}
	case SHRQ:
		// https://www.felixcloutier.com/x86/sal:sar:shl:shr
		rexPrefix |= rexPrefixW
		modRM := 0b11_000_000 | // Specifying that opeand is register.
			0b00_101_000 | // SHR with immediate needs "/5" extension.
			regBits
		if n.srcConst == 1 {
			code = append(code, rexPrefix, 0xd1, modRM)
		} else {
			code = append(code, rexPrefix, 0xc1, modRM, byte(n.srcConst))
		}
	case PSLLD:
		// https://www.felixcloutier.com/x86/psllw:pslld:psllq
		modRM := 0b11_000_000 | // Specifying that opeand is register.
			0b00_110_000 | // PSLL with immediate needs "/6" extension.
			regBits
		if rexPrefix != rexPrefixNone {
			code = append(code, 0x66, rexPrefix, 0x0f, 0x72, modRM, byte(n.srcConst))
		} else {
			code = append(code, 0x66, 0x0f, 0x72, modRM, byte(n.srcConst))
		}
	case PSLLQ:
		// https://www.felixcloutier.com/x86/psllw:pslld:psllq
		modRM := 0b11_000_000 | // Specifying that opeand is register.
			0b00_110_000 | // PSLL with immediate needs "/6" extension.
			regBits
		if rexPrefix != rexPrefixNone {
			code = append(code, 0x66, rexPrefix, 0x0f, 0x73, modRM, byte(n.srcConst))
		} else {
			code = append(code, 0x66, 0x0f, 0x73, modRM, byte(n.srcConst))
		}
	case PSRLD:
		// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
		// https://www.felixcloutier.com/x86/psllw:pslld:psllq
		modRM := 0b11_000_000 | // Specifying that operand is register.
			0b00_010_000 | // PSRL with immediate needs "/2" extension.
			regBits
		if rexPrefix != rexPrefixNone {
			code = append(code, 0x66, rexPrefix, 0x0f, 0x72, modRM, byte(n.srcConst))
		} else {
			code = append(code, 0x66, 0x0f, 0x72, modRM, byte(n.srcConst))
		}
	case PSRLQ:
		// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
		modRM := 0b11_000_000 | // Specifying that operand is register.
			0b00_010_000 | // PSRL with immediate needs "/2" extension.
			regBits
		if rexPrefix != rexPrefixNone {
			code = append(code, 0x66, rexPrefix, 0x0f, 0x73, modRM, byte(n.srcConst))
		} else {
			code = append(code, 0x66, 0x0f, 0x73, modRM, byte(n.srcConst))
		}
	case PSRAW, PSRAD:
		// https://www.felixcloutier.com/x86/psraw:psrad:psraq
		modRM := 0b11_000_000 | // Specifying that operand is register.
			0b00_100_000 | // PSRAW with immediate needs "/4" extension.
			regBits
		code = append(code, 0x66)
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}

		var op byte
		if inst == PSRAD {
			op = 0x72
		} else { // PSRAW
			op = 0x71
		}

		code = append(code, 0x0f, op, modRM, byte(n.srcConst))
	case PSRLW:
		// https://www.felixcloutier.com/x86/psrlw:psrld:psrlq
		modRM := 0b11_000_000 | // Specifying that operand is register.
			0b00_010_000 | // PSRLW with immediate needs "/2" extension.
			regBits
		code = append(code, 0x66)
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		code = append(code, 0x0f, 0x71, modRM, byte(n.srcConst))
	case PSLLW:
		// https://www.felixcloutier.com/x86/psllw:pslld:psllq
		modRM := 0b11_000_000 | // Specifying that operand is register.
			0b00_110_000 | // PSLLW with immediate needs "/6" extension.
			regBits
		code = append(code, 0x66)
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		code = append(code, 0x0f, 0x71, modRM, byte(n.srcConst))
	case XORL, XORQ:
		// https://www.felixcloutier.com/x86/xor
		if inst == XORQ {
			rexPrefix |= rexPrefixW
		}
		if rexPrefix != rexPrefixNone {
			code = append(code, rexPrefix)
		}
		if n.dstReg == RegAX && !isSigned8bitConst {
			code = append(code, 0x35)
		} else {
			modRM := 0b11_000_000 | // Specifying that opeand is register.
				0b00_110_000 | // XOR with immediate needs "/6" extension.
				regBits
			if isSigned8bitConst {
				code = append(code, 0x83, modRM)
			} else {
				code = append(code, 0x81, modRM)
			}
		}
		if fitInSigned8bit(n.srcConst) {
			code = append(code, byte(n.srcConst))
		} else {
			code = appendUint32(code, uint32(n.srcConst))
		}
	default:
		err = errorEncodingUnsupported(n)
	}

	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeMemoryToConst(buf asm.Buffer, n *nodeImpl) (err error) {
	if !fitIn32bit(n.dstConst) {
		return fmt.Errorf("too large target const %d for %s", n.dstConst, InstructionName(n.instruction))
	}

	rexPrefix, modRM, sbi, sbiExist, displacementWidth, err := n.getMemoryLocation(false)
	if err != nil {
		return err
	}

	// Alias for readability.
	c := n.dstConst

	var opcode, constWidth byte
	switch n.instruction {
	case CMPL:
		// https://www.felixcloutier.com/x86/cmp
		if fitInSigned8bit(c) {
			opcode = 0x83
			constWidth = 8
		} else {
			opcode = 0x81
			constWidth = 32
		}
		modRM |= 0b00_111_000
	default:
		return errorEncodingUnsupported(n)
	}

	base := buf.Len()
	code := buf.Append(20)[:0]

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode, modRM)

	if sbiExist {
		code = append(code, sbi)
	}

	if displacementWidth != 0 {
		code = appendConst(code, n.srcConst, displacementWidth)
	}

	code = appendConst(code, c, constWidth)
	buf.Truncate(base + len(code))
	return
}

func (a *AssemblerImpl) encodeConstToMemory(buf asm.Buffer, n *nodeImpl) (err error) {
	rexPrefix, modRM, sbi, sbiExist, displacementWidth, err := n.getMemoryLocation(true)
	if err != nil {
		return err
	}

	// Alias for readability.
	inst := n.instruction
	c := n.srcConst

	if inst == MOVB && !fitInSigned8bit(c) {
		return fmt.Errorf("too large load target const %d for MOVB", c)
	} else if !fitIn32bit(c) {
		return fmt.Errorf("too large load target const %d for %s", c, InstructionName(n.instruction))
	}

	var constWidth, opcode byte
	switch inst {
	case MOVB:
		opcode = 0xc6
		constWidth = 8
	case MOVL:
		opcode = 0xc7
		constWidth = 32
	case MOVQ:
		rexPrefix |= rexPrefixW
		opcode = 0xc7
		constWidth = 32
	default:
		return errorEncodingUnsupported(n)
	}

	base := buf.Len()
	code := buf.Append(20)[:0]

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode, modRM)

	if sbiExist {
		code = append(code, sbi)
	}

	if displacementWidth != 0 {
		code = appendConst(code, n.dstConst, displacementWidth)
	}

	code = appendConst(code, c, constWidth)

	buf.Truncate(base + len(code))
	return
}

func appendUint32(code []byte, v uint32) []byte {
	b := [4]byte{}
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	return append(code, b[:]...)
}

func appendUint64(code []byte, v uint64) []byte {
	b := [8]byte{}
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	return append(code, b[:]...)
}

func appendConst(code []byte, v int64, length byte) []byte {
	switch length {
	case 8:
		return append(code, byte(v))
	case 32:
		return appendUint32(code, uint32(v))
	default:
		return appendUint64(code, uint64(v))
	}
}

func (n *nodeImpl) getMemoryLocation(dstMem bool) (p rexPrefix, modRM byte, sbi byte, sbiExist bool, displacementWidth byte, err error) {
	var baseReg, indexReg asm.Register
	var offset asm.ConstantValue
	var scale byte
	if dstMem {
		baseReg, offset, indexReg, scale = n.dstReg, n.dstConst, n.dstMemIndex, n.dstMemScale
	} else {
		baseReg, offset, indexReg, scale = n.srcReg, n.srcConst, n.srcMemIndex, n.srcMemScale
	}

	if !fitIn32bit(offset) {
		err = errors.New("offset does not fit in 32-bit integer")
		return
	}

	if baseReg == asm.NilRegister && indexReg != asm.NilRegister {
		// [(index*scale) + displacement] addressing is possible, but we haven't used it for now.
		err = errors.New("addressing without base register but with index is not implemented")
	} else if baseReg == asm.NilRegister {
		modRM = 0b00_000_100 // Indicate that the memory location is specified by SIB.
		sbi, sbiExist = byte(0b00_100_101), true
		displacementWidth = 32
	} else if indexReg == asm.NilRegister {
		modRM, p = register3bits(baseReg, registerSpecifierPositionModRMFieldRM)

		// Create ModR/M byte so that this instruction takes [R/M + displacement] operand if displacement !=0
		// and otherwise [R/M].
		withoutDisplacement := offset == 0 &&
			// If the target register is R13 or BP, we have to keep [R/M + displacement] even if the value
			// is zero since it's not [R/M] operand is not defined for these two registers.
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#32.2F64-bit_addressing
			baseReg != RegR13 && baseReg != RegBP
		if withoutDisplacement {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b00_000_000 // Specifying that operand is memory without displacement
			displacementWidth = 0
		} else if fitInSigned8bit(offset) {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b01_000_000 // Specifying that operand is memory + 8bit displacement.
			displacementWidth = 8
		} else {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b10_000_000 // Specifying that operand is memory + 32bit displacement.
			displacementWidth = 32
		}

		// For SP and R12 register, we have [SIB + displacement] if the const is non-zero, otherwise [SIP].
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#32.2F64-bit_addressing
		//
		// Thefore we emit the SIB byte before the const so that [SIB + displacement] ends up [register + displacement].
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#32.2F64-bit_addressing_2
		if baseReg == RegSP || baseReg == RegR12 {
			sbi, sbiExist = byte(0b00_100_100), true
		}
	} else {
		if indexReg == RegSP {
			err = errors.New("SP cannot be used for SIB index")
			return
		}

		modRM = 0b00_000_100 // Indicate that the memory location is specified by SIB.

		withoutDisplacement := offset == 0 &&
			// For R13 and BP, base registers cannot be encoded "without displacement" mod (i.e. 0b00 mod).
			baseReg != RegR13 && baseReg != RegBP
		if withoutDisplacement {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b00_000_000 // Specifying that operand is SIB without displacement
			displacementWidth = 0
		} else if fitInSigned8bit(offset) {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b01_000_000 // Specifying that operand is SIB + 8bit displacement.
			displacementWidth = 8
		} else {
			// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
			modRM |= 0b10_000_000 // Specifying that operand is SIB + 32bit displacement.
			displacementWidth = 32
		}

		var baseRegBits byte
		baseRegBits, p = register3bits(baseReg, registerSpecifierPositionModRMFieldRM)

		var indexRegBits byte
		var indexRegPrefix rexPrefix
		indexRegBits, indexRegPrefix = register3bits(indexReg, registerSpecifierPositionSIBIndex)
		p |= indexRegPrefix

		sbi, sbiExist = baseRegBits|(indexRegBits<<3), true
		switch scale {
		case 1:
			sbi |= 0b00_000_000
		case 2:
			sbi |= 0b01_000_000
		case 4:
			sbi |= 0b10_000_000
		case 8:
			sbi |= 0b11_000_000
		default:
			err = fmt.Errorf("scale in SIB must be one of 1, 2, 4, 8 but got %d", scale)
			return
		}

	}
	return
}

// getRegisterToRegisterModRM does XXXX
//
// TODO: srcOnModRMReg can be deleted after golang-asm removal. This is necessary to match our implementation
// with golang-asm, but in practice, there are equivalent opcodes to always have src on ModRM:reg without ambiguity.
func (n *nodeImpl) getRegisterToRegisterModRM(srcOnModRMReg bool) (rexPrefix, modRM byte, err error) {
	var reg3bits, rm3bits byte
	if srcOnModRMReg {
		reg3bits, rexPrefix = register3bits(n.srcReg,
			// Indicate that srcReg will be specified by ModRM:reg.
			registerSpecifierPositionModRMFieldReg)

		var dstRexPrefix byte
		rm3bits, dstRexPrefix = register3bits(n.dstReg,
			// Indicate that dstReg will be specified by ModRM:r/m.
			registerSpecifierPositionModRMFieldRM)
		rexPrefix |= dstRexPrefix
	} else {
		rm3bits, rexPrefix = register3bits(n.srcReg,
			// Indicate that srcReg will be specified by ModRM:r/m.
			registerSpecifierPositionModRMFieldRM)

		var dstRexPrefix byte
		reg3bits, dstRexPrefix = register3bits(n.dstReg,
			// Indicate that dstReg will be specified by ModRM:reg.
			registerSpecifierPositionModRMFieldReg)
		rexPrefix |= dstRexPrefix
	}

	// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
	modRM = 0b11_000_000 | // Specifying that dst operand is register.
		(reg3bits << 3) |
		rm3bits

	return
}

// RexPrefix represents REX prefix https://wiki.osdev.org/X86-64_Instruction_Encoding#REX_prefix
type rexPrefix = byte

// REX prefixes are independent of each other and can be combined with OR.
const (
	rexPrefixNone    rexPrefix = 0x0000_0000 // Indicates that the instruction doesn't need RexPrefix.
	rexPrefixDefault rexPrefix = 0b0100_0000
	rexPrefixW                 = 0b0000_1000 | rexPrefixDefault // REX.W
	rexPrefixR                 = 0b0000_0100 | rexPrefixDefault // REX.R
	rexPrefixX                 = 0b0000_0010 | rexPrefixDefault // REX.X
	rexPrefixB                 = 0b0000_0001 | rexPrefixDefault // REX.B
)

// registerSpecifierPosition represents the position in the instruction bytes where an operand register is placed.
type registerSpecifierPosition byte

const (
	registerSpecifierPositionModRMFieldReg registerSpecifierPosition = iota
	registerSpecifierPositionModRMFieldRM
	registerSpecifierPositionSIBIndex
)

var regInfo = [...]struct {
	bits    byte
	needRex bool
}{
	RegAX:  {bits: 0b000},
	RegCX:  {bits: 0b001},
	RegDX:  {bits: 0b010},
	RegBX:  {bits: 0b011},
	RegSP:  {bits: 0b100},
	RegBP:  {bits: 0b101},
	RegSI:  {bits: 0b110},
	RegDI:  {bits: 0b111},
	RegR8:  {bits: 0b000, needRex: true},
	RegR9:  {bits: 0b001, needRex: true},
	RegR10: {bits: 0b010, needRex: true},
	RegR11: {bits: 0b011, needRex: true},
	RegR12: {bits: 0b100, needRex: true},
	RegR13: {bits: 0b101, needRex: true},
	RegR14: {bits: 0b110, needRex: true},
	RegR15: {bits: 0b111, needRex: true},
	RegX0:  {bits: 0b000},
	RegX1:  {bits: 0b001},
	RegX2:  {bits: 0b010},
	RegX3:  {bits: 0b011},
	RegX4:  {bits: 0b100},
	RegX5:  {bits: 0b101},
	RegX6:  {bits: 0b110},
	RegX7:  {bits: 0b111},
	RegX8:  {bits: 0b000, needRex: true},
	RegX9:  {bits: 0b001, needRex: true},
	RegX10: {bits: 0b010, needRex: true},
	RegX11: {bits: 0b011, needRex: true},
	RegX12: {bits: 0b100, needRex: true},
	RegX13: {bits: 0b101, needRex: true},
	RegX14: {bits: 0b110, needRex: true},
	RegX15: {bits: 0b111, needRex: true},
}

func register3bits(
	reg asm.Register,
	registerSpecifierPosition registerSpecifierPosition,
) (bits byte, prefix rexPrefix) {
	info := regInfo[reg]
	bits = info.bits
	if info.needRex {
		// https://wiki.osdev.org/X86-64_Instruction_Encoding#REX_prefix
		switch registerSpecifierPosition {
		case registerSpecifierPositionModRMFieldReg:
			prefix = rexPrefixR
		case registerSpecifierPositionModRMFieldRM:
			prefix = rexPrefixB
		case registerSpecifierPositionSIBIndex:
			prefix = rexPrefixX
		}
	}
	return
}

func fitIn32bit(v int64) bool {
	return math.MinInt32 <= v && v <= math.MaxUint32
}

func fitInSigned8bit(v int64) bool {
	return math.MinInt8 <= v && v <= math.MaxInt8
}

func isVectorRegister(r asm.Register) bool {
	return RegX0 <= r && r <= RegX15
}
