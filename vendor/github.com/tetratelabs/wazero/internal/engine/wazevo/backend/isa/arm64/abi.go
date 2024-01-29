package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// References:
// * https://github.com/golang/go/blob/49d42128fd8594c172162961ead19ac95e247d24/src/cmd/compile/abi-internal.md#arm64-architecture
// * https://developer.arm.com/documentation/102374/0101/Procedure-Call-Standard

const xArgRetRegMax, vArgRetRegMax = x7, v7 // x0-x7 & v0-v7.

var regInfo = &regalloc.RegisterInfo{
	AllocatableRegisters: [regalloc.NumRegType][]regalloc.RealReg{
		// We don't allocate:
		// - x18: Reserved by the macOS: https://developer.apple.com/documentation/xcode/writing-arm64-code-for-apple-platforms#Respect-the-purpose-of-specific-CPU-registers
		// - x28: Reserved by Go runtime.
		// - x27(=tmpReg): because of the reason described on tmpReg.
		regalloc.RegTypeInt: {
			x8, x9, x10, x11, x12, x13, x14, x15,
			x16, x17, x19, x20, x21, x22, x23, x24, x25,
			x26, x29, x30,
			// These are the argument/return registers. Less preferred in the allocation.
			x7, x6, x5, x4, x3, x2, x1, x0,
		},
		regalloc.RegTypeFloat: {
			v8, v9, v10, v11, v12, v13, v14, v15, v16, v17, v18, v19,
			v20, v21, v22, v23, v24, v25, v26, v27, v28, v29, v30,
			// These are the argument/return registers. Less preferred in the allocation.
			v7, v6, v5, v4, v3, v2, v1, v0,
		},
	},
	CalleeSavedRegisters: [regalloc.RealRegsNumMax]bool{
		x19: true, x20: true, x21: true, x22: true, x23: true, x24: true, x25: true, x26: true, x28: true,
		v18: true, v19: true, v20: true, v21: true, v22: true, v23: true, v24: true, v25: true, v26: true,
		v27: true, v28: true, v29: true, v30: true, v31: true,
	},
	CallerSavedRegisters: [regalloc.RealRegsNumMax]bool{
		x0: true, x1: true, x2: true, x3: true, x4: true, x5: true, x6: true, x7: true, x8: true, x9: true, x10: true,
		x11: true, x12: true, x13: true, x14: true, x15: true, x16: true, x17: true, x29: true, x30: true,
		v0: true, v1: true, v2: true, v3: true, v4: true, v5: true, v6: true, v7: true, v8: true, v9: true, v10: true,
		v11: true, v12: true, v13: true, v14: true, v15: true, v16: true, v17: true,
	},
	RealRegToVReg: []regalloc.VReg{
		x0: x0VReg, x1: x1VReg, x2: x2VReg, x3: x3VReg, x4: x4VReg, x5: x5VReg, x6: x6VReg, x7: x7VReg, x8: x8VReg, x9: x9VReg, x10: x10VReg, x11: x11VReg, x12: x12VReg, x13: x13VReg, x14: x14VReg, x15: x15VReg, x16: x16VReg, x17: x17VReg, x18: x18VReg, x19: x19VReg, x20: x20VReg, x21: x21VReg, x22: x22VReg, x23: x23VReg, x24: x24VReg, x25: x25VReg, x26: x26VReg, x27: x27VReg, x28: x28VReg, x29: x29VReg, x30: x30VReg,
		v0: v0VReg, v1: v1VReg, v2: v2VReg, v3: v3VReg, v4: v4VReg, v5: v5VReg, v6: v6VReg, v7: v7VReg, v8: v8VReg, v9: v9VReg, v10: v10VReg, v11: v11VReg, v12: v12VReg, v13: v13VReg, v14: v14VReg, v15: v15VReg, v16: v16VReg, v17: v17VReg, v18: v18VReg, v19: v19VReg, v20: v20VReg, v21: v21VReg, v22: v22VReg, v23: v23VReg, v24: v24VReg, v25: v25VReg, v26: v26VReg, v27: v27VReg, v28: v28VReg, v29: v29VReg, v30: v30VReg, v31: v31VReg,
	},
	RealRegName: func(r regalloc.RealReg) string { return regNames[r] },
	RealRegType: func(r regalloc.RealReg) regalloc.RegType {
		if r < v0 {
			return regalloc.RegTypeInt
		}
		return regalloc.RegTypeFloat
	},
}

// abiImpl implements backend.FunctionABI.
type abiImpl struct {
	m                          *machine
	args, rets                 []backend.ABIArg
	argStackSize, retStackSize int64

	argRealRegs []regalloc.VReg
	retRealRegs []regalloc.VReg
}

func (m *machine) getOrCreateABIImpl(sig *ssa.Signature) *abiImpl {
	if int(sig.ID) >= len(m.abis) {
		m.abis = append(m.abis, make([]abiImpl, int(sig.ID)+1)...)
	}

	abi := &m.abis[sig.ID]
	if abi.m != nil {
		return abi
	}

	abi.m = m
	abi.init(sig)
	return abi
}

// int initializes the abiImpl for the given signature.
func (a *abiImpl) init(sig *ssa.Signature) {
	if len(a.rets) < len(sig.Results) {
		a.rets = make([]backend.ABIArg, len(sig.Results))
	}
	a.rets = a.rets[:len(sig.Results)]
	a.retStackSize = a.setABIArgs(a.rets, sig.Results)
	if argsNum := len(sig.Params); len(a.args) < argsNum {
		a.args = make([]backend.ABIArg, argsNum)
	}
	a.args = a.args[:len(sig.Params)]
	a.argStackSize = a.setABIArgs(a.args, sig.Params)

	// Gather the real registers usages in arg/return.
	a.retRealRegs = a.retRealRegs[:0]
	for i := range a.rets {
		r := &a.rets[i]
		if r.Kind == backend.ABIArgKindReg {
			a.retRealRegs = append(a.retRealRegs, r.Reg)
		}
	}
	a.argRealRegs = a.argRealRegs[:0]
	for i := range a.args {
		arg := &a.args[i]
		if arg.Kind == backend.ABIArgKindReg {
			reg := arg.Reg
			a.argRealRegs = append(a.argRealRegs, reg)
		}
	}
}

// setABIArgs sets the ABI arguments in the given slice. This assumes that len(s) >= len(types)
// where if len(s) > len(types), the last elements of s is for the multi-return slot.
func (a *abiImpl) setABIArgs(s []backend.ABIArg, types []ssa.Type) (stackSize int64) {
	var stackOffset int64
	nextX, nextV := x0, v0
	for i, typ := range types {
		arg := &s[i]
		arg.Index = i
		arg.Type = typ
		if typ.IsInt() {
			if nextX > xArgRetRegMax {
				arg.Kind = backend.ABIArgKindStack
				const slotSize = 8 // Align 8 bytes.
				arg.Offset = stackOffset
				stackOffset += slotSize
			} else {
				arg.Kind = backend.ABIArgKindReg
				arg.Reg = regalloc.FromRealReg(nextX, regalloc.RegTypeInt)
				nextX++
			}
		} else {
			if nextV > vArgRetRegMax {
				arg.Kind = backend.ABIArgKindStack
				slotSize := int64(8)   // Align at least 8 bytes.
				if typ.Bits() == 128 { // Vector.
					slotSize = 16
				}
				arg.Offset = stackOffset
				stackOffset += slotSize
			} else {
				arg.Kind = backend.ABIArgKindReg
				arg.Reg = regalloc.FromRealReg(nextV, regalloc.RegTypeFloat)
				nextV++
			}
		}
	}
	return stackOffset
}

// CalleeGenFunctionArgsToVRegs implements backend.FunctionABI.
func (a *abiImpl) CalleeGenFunctionArgsToVRegs(args []ssa.Value) {
	for i, ssaArg := range args {
		if !ssaArg.Valid() {
			continue
		}
		reg := a.m.compiler.VRegOf(ssaArg)
		arg := &a.args[i]
		if arg.Kind == backend.ABIArgKindReg {
			a.m.InsertMove(reg, arg.Reg, arg.Type)
		} else {
			// TODO: we could use pair load if there's consecutive loads for the same type.
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
			//          |      arg 0      |    <-|
			//          |   ReturnAddress |      |
			//          +-----------------+      |
			//          |   ...........   |      |
			//          |   spill slot M  |      |   argStackOffset: is unknown at this point of compilation.
			//          |   ............  |      |
			//          |   spill slot 2  |      |
			//          |   spill slot 1  |      |
			//          |   clobbered 0   |      |
			//          |   clobbered 1   |      |
			//          |   ...........   |      |
			//          |   clobbered N   |      |
			//   SP---> +-----------------+    <-+
			//             (low address)

			m := a.m
			bits := arg.Type.Bits()
			// At this point of compilation, we don't yet know how much space exist below the return address.
			// So we instruct the address mode to add the `argStackOffset` to the offset at the later phase of compilation.
			amode := addressMode{imm: arg.Offset, rn: spVReg, kind: addressModeKindArgStackSpace}
			load := m.allocateInstr()
			switch arg.Type {
			case ssa.TypeI32, ssa.TypeI64:
				load.asULoad(operandNR(reg), amode, bits)
			case ssa.TypeF32, ssa.TypeF64, ssa.TypeV128:
				load.asFpuLoad(operandNR(reg), amode, bits)
			default:
				panic("BUG")
			}
			m.insert(load)
			a.m.unresolvedAddressModes = append(a.m.unresolvedAddressModes, load)
		}
	}
}

// CalleeGenVRegsToFunctionReturns implements backend.FunctionABI.
func (a *abiImpl) CalleeGenVRegsToFunctionReturns(rets []ssa.Value) {
	l := len(rets) - 1
	for i := range rets {
		// Reverse order in order to avoid overwriting the stack returns existing in the return registers.
		ret := rets[l-i]
		r := &a.rets[l-i]
		reg := a.m.compiler.VRegOf(ret)
		if def := a.m.compiler.ValueDefinition(ret); def.IsFromInstr() {
			// Constant instructions are inlined.
			if inst := def.Instr; inst.Constant() {
				a.m.InsertLoadConstant(inst, reg)
			}
		}
		if r.Kind == backend.ABIArgKindReg {
			a.m.InsertMove(r.Reg, reg, ret.Type())
		} else {
			// TODO: we could use pair store if there's consecutive stores for the same type.
			//
			//            (high address)
			//          +-----------------+
			//          |     .......     |
			//          |      ret Y      |
			//          |     .......     |
			//          |      ret 0      |    <-+
			//          |      arg X      |      |
			//          |     .......     |      |
			//          |      arg 1      |      |
			//          |      arg 0      |      |
			//          |   ReturnAddress |      |
			//          +-----------------+      |
			//          |   ...........   |      |
			//          |   spill slot M  |      |   retStackOffset: is unknown at this point of compilation.
			//          |   ............  |      |
			//          |   spill slot 2  |      |
			//          |   spill slot 1  |      |
			//          |   clobbered 0   |      |
			//          |   clobbered 1   |      |
			//          |   ...........   |      |
			//          |   clobbered N   |      |
			//   SP---> +-----------------+    <-+
			//             (low address)

			bits := r.Type.Bits()

			// At this point of compilation, we don't yet know how much space exist below the return address.
			// So we instruct the address mode to add the `retStackOffset` to the offset at the later phase of compilation.
			amode := addressMode{imm: r.Offset, rn: spVReg, kind: addressModeKindResultStackSpace}
			store := a.m.allocateInstr()
			store.asStore(operandNR(reg), amode, bits)
			a.m.insert(store)
			a.m.unresolvedAddressModes = append(a.m.unresolvedAddressModes, store)
		}
	}
}

// callerGenVRegToFunctionArg is the opposite of GenFunctionArgToVReg, which is used to generate the
// caller side of the function call.
func (a *abiImpl) callerGenVRegToFunctionArg(argIndex int, reg regalloc.VReg, def *backend.SSAValueDefinition, slotBegin int64) {
	arg := &a.args[argIndex]
	if def != nil && def.IsFromInstr() {
		// Constant instructions are inlined.
		if inst := def.Instr; inst.Constant() {
			a.m.InsertLoadConstant(inst, reg)
		}
	}
	if arg.Kind == backend.ABIArgKindReg {
		a.m.InsertMove(arg.Reg, reg, arg.Type)
	} else {
		// TODO: we could use pair store if there's consecutive stores for the same type.
		//
		// Note that at this point, stack pointer is already adjusted.
		bits := arg.Type.Bits()
		amode := a.m.resolveAddressModeForOffset(arg.Offset-slotBegin, bits, spVReg, false)
		store := a.m.allocateInstr()
		store.asStore(operandNR(reg), amode, bits)
		a.m.insert(store)
	}
}

func (a *abiImpl) callerGenFunctionReturnVReg(retIndex int, reg regalloc.VReg, slotBegin int64) {
	r := &a.rets[retIndex]
	if r.Kind == backend.ABIArgKindReg {
		a.m.InsertMove(reg, r.Reg, r.Type)
	} else {
		// TODO: we could use pair load if there's consecutive loads for the same type.
		amode := a.m.resolveAddressModeForOffset(a.argStackSize+r.Offset-slotBegin, r.Type.Bits(), spVReg, false)
		ldr := a.m.allocateInstr()
		switch r.Type {
		case ssa.TypeI32, ssa.TypeI64:
			ldr.asULoad(operandNR(reg), amode, r.Type.Bits())
		case ssa.TypeF32, ssa.TypeF64, ssa.TypeV128:
			ldr.asFpuLoad(operandNR(reg), amode, r.Type.Bits())
		default:
			panic("BUG")
		}
		a.m.insert(ldr)
	}
}

func (m *machine) resolveAddressModeForOffsetAndInsert(cur *instruction, offset int64, dstBits byte, rn regalloc.VReg, allowTmpRegUse bool) (*instruction, addressMode) {
	m.pendingInstructions = m.pendingInstructions[:0]
	mode := m.resolveAddressModeForOffset(offset, dstBits, rn, allowTmpRegUse)
	for _, instr := range m.pendingInstructions {
		cur = linkInstr(cur, instr)
	}
	return cur, mode
}

func (m *machine) resolveAddressModeForOffset(offset int64, dstBits byte, rn regalloc.VReg, allowTmpRegUse bool) addressMode {
	if rn.RegType() != regalloc.RegTypeInt {
		panic("BUG: rn should be a pointer: " + formatVRegSized(rn, 64))
	}
	var amode addressMode
	if offsetFitsInAddressModeKindRegUnsignedImm12(dstBits, offset) {
		amode = addressMode{kind: addressModeKindRegUnsignedImm12, rn: rn, imm: offset}
	} else if offsetFitsInAddressModeKindRegSignedImm9(offset) {
		amode = addressMode{kind: addressModeKindRegSignedImm9, rn: rn, imm: offset}
	} else {
		var indexReg regalloc.VReg
		if allowTmpRegUse {
			m.lowerConstantI64(tmpRegVReg, offset)
			indexReg = tmpRegVReg
		} else {
			indexReg = m.compiler.AllocateVReg(ssa.TypeI64)
			m.lowerConstantI64(indexReg, offset)
		}
		amode = addressMode{kind: addressModeKindRegReg, rn: rn, rm: indexReg, extOp: extendOpUXTX /* indicates index rm is 64-bit */}
	}
	return amode
}

func (a *abiImpl) alignedArgResultStackSlotSize() int64 {
	stackSlotSize := a.retStackSize + a.argStackSize
	// Align stackSlotSize to 16 bytes.
	stackSlotSize = (stackSlotSize + 15) &^ 15
	return stackSlotSize
}

func (m *machine) lowerCall(si *ssa.Instruction) {
	isDirectCall := si.Opcode() == ssa.OpcodeCall
	var indirectCalleePtr ssa.Value
	var directCallee ssa.FuncRef
	var sigID ssa.SignatureID
	var args []ssa.Value
	if isDirectCall {
		directCallee, sigID, args = si.CallData()
	} else {
		indirectCalleePtr, sigID, args = si.CallIndirectData()
	}
	calleeABI := m.getOrCreateABIImpl(m.compiler.SSABuilder().ResolveSignature(sigID))

	stackSlotSize := calleeABI.alignedArgResultStackSlotSize()
	if m.maxRequiredStackSizeForCalls < stackSlotSize+16 {
		m.maxRequiredStackSizeForCalls = stackSlotSize + 16 // return address frame.
	}

	for i, arg := range args {
		reg := m.compiler.VRegOf(arg)
		def := m.compiler.ValueDefinition(arg)
		calleeABI.callerGenVRegToFunctionArg(i, reg, def, stackSlotSize)
	}

	if isDirectCall {
		call := m.allocateInstr()
		call.asCall(directCallee, calleeABI)
		m.insert(call)
	} else {
		ptr := m.compiler.VRegOf(indirectCalleePtr)
		callInd := m.allocateInstr()
		callInd.asCallIndirect(ptr, calleeABI)
		m.insert(callInd)
	}

	var index int
	r1, rs := si.Returns()
	if r1.Valid() {
		calleeABI.callerGenFunctionReturnVReg(0, m.compiler.VRegOf(r1), stackSlotSize)
		index++
	}

	for _, r := range rs {
		calleeABI.callerGenFunctionReturnVReg(index, m.compiler.VRegOf(r), stackSlotSize)
		index++
	}
}

func (m *machine) insertAddOrSubStackPointer(rd regalloc.VReg, diff int64, add bool) {
	if imm12Operand, ok := asImm12Operand(uint64(diff)); ok {
		alu := m.allocateInstr()
		var ao aluOp
		if add {
			ao = aluOpAdd
		} else {
			ao = aluOpSub
		}
		alu.asALU(ao, operandNR(rd), operandNR(spVReg), imm12Operand, true)
		m.insert(alu)
	} else {
		m.lowerConstantI64(tmpRegVReg, diff)
		alu := m.allocateInstr()
		var ao aluOp
		if add {
			ao = aluOpAdd
		} else {
			ao = aluOpSub
		}
		alu.asALU(ao, operandNR(rd), operandNR(spVReg), operandNR(tmpRegVReg), true)
		m.insert(alu)
	}
}
