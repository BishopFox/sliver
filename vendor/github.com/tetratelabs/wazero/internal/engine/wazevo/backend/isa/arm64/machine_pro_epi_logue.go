package arm64

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

// PostRegAlloc implements backend.Machine.
func (m *machine) PostRegAlloc() {
	m.setupPrologue()
	m.postRegAlloc()
}

// setupPrologue initializes the prologue of the function.
func (m *machine) setupPrologue() {
	ectx := m.executableContext

	cur := ectx.RootInstr
	prevInitInst := cur.next

	//
	//                   (high address)                    (high address)
	//         SP----> +-----------------+               +------------------+ <----+
	//                 |     .......     |               |     .......      |      |
	//                 |      ret Y      |               |      ret Y       |      |
	//                 |     .......     |               |     .......      |      |
	//                 |      ret 0      |               |      ret 0       |      |
	//                 |      arg X      |               |      arg X       |      |  size_of_arg_ret.
	//                 |     .......     |     ====>     |     .......      |      |
	//                 |      arg 1      |               |      arg 1       |      |
	//                 |      arg 0      |               |      arg 0       | <----+
	//                 |-----------------|               |  size_of_arg_ret |
	//                                                   |  return address  |
	//                                                   +------------------+ <---- SP
	//                    (low address)                     (low address)

	// Saves the return address (lr) and the size_of_arg_ret below the SP.
	// size_of_arg_ret is used for stack unwinding.
	cur = m.createReturnAddrAndSizeOfArgRetSlot(cur)

	if !m.stackBoundsCheckDisabled {
		cur = m.insertStackBoundsCheck(m.requiredStackSize(), cur)
	}

	// Decrement SP if spillSlotSize > 0.
	if m.spillSlotSize == 0 && len(m.spillSlots) != 0 {
		panic(fmt.Sprintf("BUG: spillSlotSize=%d, spillSlots=%v\n", m.spillSlotSize, m.spillSlots))
	}

	if regs := m.clobberedRegs; len(regs) > 0 {
		//
		//            (high address)                  (high address)
		//          +-----------------+             +-----------------+
		//          |     .......     |             |     .......     |
		//          |      ret Y      |             |      ret Y      |
		//          |     .......     |             |     .......     |
		//          |      ret 0      |             |      ret 0      |
		//          |      arg X      |             |      arg X      |
		//          |     .......     |             |     .......     |
		//          |      arg 1      |             |      arg 1      |
		//          |      arg 0      |             |      arg 0      |
		//          | size_of_arg_ret |             | size_of_arg_ret |
		//          |   ReturnAddress |             |  ReturnAddress  |
		//  SP----> +-----------------+    ====>    +-----------------+
		//             (low address)                |   clobbered M   |
		//                                          |   ............  |
		//                                          |   clobbered 0   |
		//                                          +-----------------+ <----- SP
		//                                             (low address)
		//
		_amode := addressModePreOrPostIndex(spVReg,
			-16,  // stack pointer must be 16-byte aligned.
			true, // Decrement before store.
		)
		for _, vr := range regs {
			// TODO: pair stores to reduce the number of instructions.
			store := m.allocateInstr()
			store.asStore(operandNR(vr), _amode, regTypeToRegisterSizeInBits(vr.RegType()))
			cur = linkInstr(cur, store)
		}
	}

	if size := m.spillSlotSize; size > 0 {
		// Check if size is 16-byte aligned.
		if size&0xf != 0 {
			panic(fmt.Errorf("BUG: spill slot size %d is not 16-byte aligned", size))
		}

		cur = m.addsAddOrSubStackPointer(cur, spVReg, size, false)

		// At this point, the stack looks like:
		//
		//            (high address)
		//          +------------------+
		//          |     .......      |
		//          |      ret Y       |
		//          |     .......      |
		//          |      ret 0       |
		//          |      arg X       |
		//          |     .......      |
		//          |      arg 1       |
		//          |      arg 0       |
		//          |  size_of_arg_ret |
		//          |   ReturnAddress  |
		//          +------------------+
		//          |    clobbered M   |
		//          |   ............   |
		//          |    clobbered 0   |
		//          |   spill slot N   |
		//          |   ............   |
		//          |   spill slot 2   |
		//          |   spill slot 0   |
		//  SP----> +------------------+
		//             (low address)
	}

	// We push the frame size into the stack to make it possible to unwind stack:
	//
	//
	//            (high address)                  (high address)
	//         +-----------------+                +-----------------+
	//         |     .......     |                |     .......     |
	//         |      ret Y      |                |      ret Y      |
	//         |     .......     |                |     .......     |
	//         |      ret 0      |                |      ret 0      |
	//         |      arg X      |                |      arg X      |
	//         |     .......     |                |     .......     |
	//         |      arg 1      |                |      arg 1      |
	//         |      arg 0      |                |      arg 0      |
	//         | size_of_arg_ret |                | size_of_arg_ret |
	//         |  ReturnAddress  |                |  ReturnAddress  |
	//         +-----------------+      ==>       +-----------------+ <----+
	//         |   clobbered  M  |                |   clobbered  M  |      |
	//         |   ............  |                |   ............  |      |
	//         |   clobbered  2  |                |   clobbered  2  |      |
	//         |   clobbered  1  |                |   clobbered  1  |      | frame size
	//         |   clobbered  0  |                |   clobbered  0  |      |
	//         |   spill slot N  |                |   spill slot N  |      |
	//         |   ............  |                |   ............  |      |
	//         |   spill slot 0  |                |   spill slot 0  | <----+
	// SP--->  +-----------------+                |     xxxxxx      |  ;; unused space to make it 16-byte aligned.
	//                                            |   frame_size    |
	//                                            +-----------------+ <---- SP
	//            (low address)
	//
	cur = m.createFrameSizeSlot(cur, m.frameSize())

	linkInstr(cur, prevInitInst)
}

func (m *machine) createReturnAddrAndSizeOfArgRetSlot(cur *instruction) *instruction {
	// First we decrement the stack pointer to point the arg0 slot.
	var sizeOfArgRetReg regalloc.VReg
	s := int64(m.currentABI.AlignedArgResultStackSlotSize())
	if s > 0 {
		cur = m.lowerConstantI64AndInsert(cur, tmpRegVReg, s)
		sizeOfArgRetReg = tmpRegVReg

		subSp := m.allocateInstr()
		subSp.asALU(aluOpSub, operandNR(spVReg), operandNR(spVReg), operandNR(sizeOfArgRetReg), true)
		cur = linkInstr(cur, subSp)
	} else {
		sizeOfArgRetReg = xzrVReg
	}

	// Saves the return address (lr) and the size_of_arg_ret below the SP.
	// size_of_arg_ret is used for stack unwinding.
	pstr := m.allocateInstr()
	amode := addressModePreOrPostIndex(spVReg, -16, true /* decrement before store */)
	pstr.asStorePair64(lrVReg, sizeOfArgRetReg, amode)
	cur = linkInstr(cur, pstr)
	return cur
}

func (m *machine) createFrameSizeSlot(cur *instruction, s int64) *instruction {
	var frameSizeReg regalloc.VReg
	if s > 0 {
		cur = m.lowerConstantI64AndInsert(cur, tmpRegVReg, s)
		frameSizeReg = tmpRegVReg
	} else {
		frameSizeReg = xzrVReg
	}
	_amode := addressModePreOrPostIndex(spVReg,
		-16,  // stack pointer must be 16-byte aligned.
		true, // Decrement before store.
	)
	store := m.allocateInstr()
	store.asStore(operandNR(frameSizeReg), _amode, 64)
	cur = linkInstr(cur, store)
	return cur
}

// postRegAlloc does multiple things while walking through the instructions:
// 1. Removes the redundant copy instruction.
// 2. Inserts the epilogue.
func (m *machine) postRegAlloc() {
	ectx := m.executableContext
	for cur := ectx.RootInstr; cur != nil; cur = cur.next {
		switch cur.kind {
		case ret:
			m.setupEpilogueAfter(cur.prev)
		case loadConstBlockArg:
			lc := cur
			next := lc.next
			m.executableContext.PendingInstructions = m.executableContext.PendingInstructions[:0]
			m.lowerLoadConstantBlockArgAfterRegAlloc(lc)
			for _, instr := range m.executableContext.PendingInstructions {
				cur = linkInstr(cur, instr)
			}
			linkInstr(cur, next)
			m.executableContext.PendingInstructions = m.executableContext.PendingInstructions[:0]
		default:
			// Removes the redundant copy instruction.
			if cur.IsCopy() && cur.rn.realReg() == cur.rd.realReg() {
				prev, next := cur.prev, cur.next
				// Remove the copy instruction.
				prev.next = next
				if next != nil {
					next.prev = prev
				}
			}
		}
	}
}

func (m *machine) setupEpilogueAfter(cur *instruction) {
	prevNext := cur.next

	// We've stored the frame size in the prologue, and now that we are about to return from this function, we won't need it anymore.
	cur = m.addsAddOrSubStackPointer(cur, spVReg, 16, true)

	if s := m.spillSlotSize; s > 0 {
		// Adjust SP to the original value:
		//
		//            (high address)                        (high address)
		//          +-----------------+                  +-----------------+
		//          |     .......     |                  |     .......     |
		//          |      ret Y      |                  |      ret Y      |
		//          |     .......     |                  |     .......     |
		//          |      ret 0      |                  |      ret 0      |
		//          |      arg X      |                  |      arg X      |
		//          |     .......     |                  |     .......     |
		//          |      arg 1      |                  |      arg 1      |
		//          |      arg 0      |                  |      arg 0      |
		//          |      xxxxx      |                  |      xxxxx      |
		//          |   ReturnAddress |                  |   ReturnAddress |
		//          +-----------------+      ====>       +-----------------+
		//          |    clobbered M  |                  |    clobbered M  |
		//          |   ............  |                  |   ............  |
		//          |    clobbered 1  |                  |    clobbered 1  |
		//          |    clobbered 0  |                  |    clobbered 0  |
		//          |   spill slot N  |                  +-----------------+ <---- SP
		//          |   ............  |
		//          |   spill slot 0  |
		//   SP---> +-----------------+
		//             (low address)
		//
		cur = m.addsAddOrSubStackPointer(cur, spVReg, s, true)
	}

	// First we need to restore the clobbered registers.
	if len(m.clobberedRegs) > 0 {
		//            (high address)
		//          +-----------------+                      +-----------------+
		//          |     .......     |                      |     .......     |
		//          |      ret Y      |                      |      ret Y      |
		//          |     .......     |                      |     .......     |
		//          |      ret 0      |                      |      ret 0      |
		//          |      arg X      |                      |      arg X      |
		//          |     .......     |                      |     .......     |
		//          |      arg 1      |                      |      arg 1      |
		//          |      arg 0      |                      |      arg 0      |
		//          |      xxxxx      |                      |      xxxxx      |
		//          |   ReturnAddress |                      |   ReturnAddress |
		//          +-----------------+      ========>       +-----------------+ <---- SP
		//          |   clobbered M   |
		//          |   ...........   |
		//          |   clobbered 1   |
		//          |   clobbered 0   |
		//   SP---> +-----------------+
		//             (low address)

		l := len(m.clobberedRegs) - 1
		for i := range m.clobberedRegs {
			vr := m.clobberedRegs[l-i] // reverse order to restore.
			load := m.allocateInstr()
			amode := addressModePreOrPostIndex(spVReg,
				16,    // stack pointer must be 16-byte aligned.
				false, // Increment after store.
			)
			// TODO: pair loads to reduce the number of instructions.
			switch regTypeToRegisterSizeInBits(vr.RegType()) {
			case 64: // save int reg.
				load.asULoad(operandNR(vr), amode, 64)
			case 128: // save vector reg.
				load.asFpuLoad(operandNR(vr), amode, 128)
			}
			cur = linkInstr(cur, load)
		}
	}

	// Reload the return address (lr).
	//
	//            +-----------------+          +-----------------+
	//            |     .......     |          |     .......     |
	//            |      ret Y      |          |      ret Y      |
	//            |     .......     |          |     .......     |
	//            |      ret 0      |          |      ret 0      |
	//            |      arg X      |          |      arg X      |
	//            |     .......     |   ===>   |     .......     |
	//            |      arg 1      |          |      arg 1      |
	//            |      arg 0      |          |      arg 0      |
	//            |      xxxxx      |          +-----------------+ <---- SP
	//            |  ReturnAddress  |
	//    SP----> +-----------------+

	ldr := m.allocateInstr()
	ldr.asULoad(operandNR(lrVReg),
		addressModePreOrPostIndex(spVReg, 16 /* stack pointer must be 16-byte aligned. */, false /* increment after loads */), 64)
	cur = linkInstr(cur, ldr)

	if s := int64(m.currentABI.AlignedArgResultStackSlotSize()); s > 0 {
		cur = m.addsAddOrSubStackPointer(cur, spVReg, s, true)
	}

	linkInstr(cur, prevNext)
}

// saveRequiredRegs is the set of registers that must be saved/restored during growing stack when there's insufficient
// stack space left. Basically this is the combination of CalleeSavedRegisters plus argument registers execpt for x0,
// which always points to the execution context whenever the native code is entered from Go.
var saveRequiredRegs = []regalloc.VReg{
	x1VReg, x2VReg, x3VReg, x4VReg, x5VReg, x6VReg, x7VReg,
	x19VReg, x20VReg, x21VReg, x22VReg, x23VReg, x24VReg, x25VReg, x26VReg, x28VReg, lrVReg,
	v0VReg, v1VReg, v2VReg, v3VReg, v4VReg, v5VReg, v6VReg, v7VReg,
	v18VReg, v19VReg, v20VReg, v21VReg, v22VReg, v23VReg, v24VReg, v25VReg, v26VReg, v27VReg, v28VReg, v29VReg, v30VReg, v31VReg,
}

// insertStackBoundsCheck will insert the instructions after `cur` to check the
// stack bounds, and if there's no sufficient spaces required for the function,
// exit the execution and try growing it in Go world.
//
// TODO: we should be able to share the instructions across all the functions to reduce the size of compiled executable.
func (m *machine) insertStackBoundsCheck(requiredStackSize int64, cur *instruction) *instruction {
	if requiredStackSize%16 != 0 {
		panic("BUG")
	}

	if immm12op, ok := asImm12Operand(uint64(requiredStackSize)); ok {
		// sub tmp, sp, #requiredStackSize
		sub := m.allocateInstr()
		sub.asALU(aluOpSub, operandNR(tmpRegVReg), operandNR(spVReg), immm12op, true)
		cur = linkInstr(cur, sub)
	} else {
		// This case, we first load the requiredStackSize into the temporary register,
		cur = m.lowerConstantI64AndInsert(cur, tmpRegVReg, requiredStackSize)
		// Then subtract it.
		sub := m.allocateInstr()
		sub.asALU(aluOpSub, operandNR(tmpRegVReg), operandNR(spVReg), operandNR(tmpRegVReg), true)
		cur = linkInstr(cur, sub)
	}

	tmp2 := x11VReg // Caller save, so it is safe to use it here in the prologue.

	// ldr tmp2, [executionContext #StackBottomPtr]
	ldr := m.allocateInstr()
	ldr.asULoad(operandNR(tmp2), addressMode{
		kind: addressModeKindRegUnsignedImm12,
		rn:   x0VReg, // execution context is always the first argument.
		imm:  wazevoapi.ExecutionContextOffsetStackBottomPtr.I64(),
	}, 64)
	cur = linkInstr(cur, ldr)

	// subs xzr, tmp, tmp2
	subs := m.allocateInstr()
	subs.asALU(aluOpSubS, operandNR(xzrVReg), operandNR(tmpRegVReg), operandNR(tmp2), true)
	cur = linkInstr(cur, subs)

	// b.ge #imm
	cbr := m.allocateInstr()
	cbr.asCondBr(ge.asCond(), labelInvalid, false /* ignored */)
	cur = linkInstr(cur, cbr)

	// Set the required stack size and set it to the exec context.
	{
		// First load the requiredStackSize into the temporary register,
		cur = m.lowerConstantI64AndInsert(cur, tmpRegVReg, requiredStackSize)
		setRequiredStackSize := m.allocateInstr()
		setRequiredStackSize.asStore(operandNR(tmpRegVReg),
			addressMode{
				kind: addressModeKindRegUnsignedImm12,
				// Execution context is always the first argument.
				rn: x0VReg, imm: wazevoapi.ExecutionContextOffsetStackGrowRequiredSize.I64(),
			}, 64)

		cur = linkInstr(cur, setRequiredStackSize)
	}

	ldrAddress := m.allocateInstr()
	ldrAddress.asULoad(operandNR(tmpRegVReg), addressMode{
		kind: addressModeKindRegUnsignedImm12,
		rn:   x0VReg, // execution context is always the first argument
		imm:  wazevoapi.ExecutionContextOffsetStackGrowCallTrampolineAddress.I64(),
	}, 64)
	cur = linkInstr(cur, ldrAddress)

	// Then jumps to the stack grow call sequence's address, meaning
	// transferring the control to the code compiled by CompileStackGrowCallSequence.
	bl := m.allocateInstr()
	bl.asCallIndirect(tmpRegVReg, nil)
	cur = linkInstr(cur, bl)

	// Now that we know the entire code, we can finalize how many bytes
	// we have to skip when the stack size is sufficient.
	var cbrOffset int64
	for _cur := cbr; ; _cur = _cur.next {
		cbrOffset += _cur.size()
		if _cur == cur {
			break
		}
	}
	cbr.condBrOffsetResolve(cbrOffset)
	return cur
}

// CompileStackGrowCallSequence implements backend.Machine.
func (m *machine) CompileStackGrowCallSequence() []byte {
	ectx := m.executableContext

	cur := m.allocateInstr()
	cur.asNop0()
	ectx.RootInstr = cur

	// Save the callee saved and argument registers.
	cur = m.saveRegistersInExecutionContext(cur, saveRequiredRegs)

	// Save the current stack pointer.
	cur = m.saveCurrentStackPointer(cur, x0VReg)

	// Set the exit status on the execution context.
	cur = m.setExitCode(cur, x0VReg, wazevoapi.ExitCodeGrowStack)

	// Exit the execution.
	cur = m.storeReturnAddressAndExit(cur)

	// After the exit, restore the saved registers.
	cur = m.restoreRegistersInExecutionContext(cur, saveRequiredRegs)

	// Then goes back the original address of this stack grow call.
	ret := m.allocateInstr()
	ret.asRet()
	linkInstr(cur, ret)

	m.encode(ectx.RootInstr)
	return m.compiler.Buf()
}

func (m *machine) addsAddOrSubStackPointer(cur *instruction, rd regalloc.VReg, diff int64, add bool) *instruction {
	ectx := m.executableContext

	ectx.PendingInstructions = ectx.PendingInstructions[:0]
	m.insertAddOrSubStackPointer(rd, diff, add)
	for _, inserted := range ectx.PendingInstructions {
		cur = linkInstr(cur, inserted)
	}
	return cur
}
