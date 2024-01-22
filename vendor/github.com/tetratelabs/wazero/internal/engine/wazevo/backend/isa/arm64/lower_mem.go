package arm64

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend/regalloc"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

type (
	// addressMode represents an ARM64 addressing mode.
	//
	// https://developer.arm.com/documentation/102374/0101/Loads-and-stores---addressing
	// TODO: use the bit-packed layout like operand struct.
	addressMode struct {
		kind   addressModeKind
		rn, rm regalloc.VReg
		extOp  extendOp
		imm    int64
	}

	// addressModeKind represents the kind of ARM64 addressing mode.
	addressModeKind byte
)

const (
	// addressModeKindRegExtended takes a base register and an index register. The index register is sign/zero-extended,
	// and then scaled by bits(type)/8.
	//
	// e.g.
	// 	- ldrh w1, [x2, w3, SXTW #1] ;; sign-extended and scaled by 2 (== LSL #1)
	// 	- strh w1, [x2, w3, UXTW #1] ;; zero-extended and scaled by 2 (== LSL #1)
	// 	- ldr w1, [x2, w3, SXTW #2] ;; sign-extended and scaled by 4 (== LSL #2)
	// 	- str x1, [x2, w3, UXTW #3] ;; zero-extended and scaled by 8 (== LSL #3)
	//
	// See the following pages:
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRH--register---Load-Register-Halfword--register--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDR--register---Load-Register--register--
	addressModeKindRegScaledExtended addressModeKind = iota

	// addressModeKindRegScaled is the same as addressModeKindRegScaledExtended, but without extension factor.
	addressModeKindRegScaled

	// addressModeKindRegScaled is the same as addressModeKindRegScaledExtended, but without scale factor.
	addressModeKindRegExtended

	// addressModeKindRegReg takes a base register and an index register. The index register is not either scaled or extended.
	addressModeKindRegReg

	// addressModeKindRegSignedImm9 takes a base register and a 9-bit "signed" immediate offset (-256 to 255).
	// The immediate will be sign-extended, and be added to the base register.
	// This is a.k.a. "unscaled" since the immediate is not scaled.
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDUR--Load-Register--unscaled--
	addressModeKindRegSignedImm9

	// addressModeKindRegUnsignedImm12 takes a base register and a 12-bit "unsigned" immediate offset.  scaled by
	// the size of the type. In other words, the actual offset will be imm12 * bits(type)/8.
	// See "Unsigned offset" in the following pages:
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRB--immediate---Load-Register-Byte--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRH--immediate---Load-Register-Halfword--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDR--immediate---Load-Register--immediate--
	addressModeKindRegUnsignedImm12

	// addressModePostIndex takes a base register and a 9-bit "signed" immediate offset.
	// After the load/store, the base register will be updated by the offset.
	//
	// Note that when this is used for pair load/store, the offset will be 7-bit "signed" immediate offset.
	//
	// See "Post-index" in the following pages for examples:
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRB--immediate---Load-Register-Byte--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRH--immediate---Load-Register-Halfword--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDR--immediate---Load-Register--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDP--Load-Pair-of-Registers-
	addressModeKindPostIndex

	// addressModePostIndex takes a base register and a 9-bit "signed" immediate offset.
	// Before the load/store, the base register will be updated by the offset.
	//
	// Note that when this is used for pair load/store, the offset will be 7-bit "signed" immediate offset.
	//
	// See "Pre-index" in the following pages for examples:
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRB--immediate---Load-Register-Byte--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDRH--immediate---Load-Register-Halfword--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDR--immediate---Load-Register--immediate--
	// https://developer.arm.com/documentation/ddi0596/2021-12/Base-Instructions/LDP--Load-Pair-of-Registers-
	addressModeKindPreIndex

	// addressModeKindArgStackSpace is used to resolve the address of the argument stack space
	// exiting right above the stack pointer. Since we don't know the exact stack space needed for a function
	// at a compilation phase, this is used as a placeholder and further lowered to a real addressing mode like above.
	addressModeKindArgStackSpace

	// addressModeKindResultStackSpace is used to resolve the address of the result stack space
	// exiting right above the stack pointer. Since we don't know the exact stack space needed for a function
	// at a compilation phase, this is used as a placeholder and further lowered to a real addressing mode like above.
	addressModeKindResultStackSpace
)

func (a addressMode) format(dstSizeBits byte) (ret string) {
	base := formatVRegSized(a.rn, 64)
	if rn := a.rn; rn.RegType() != regalloc.RegTypeInt {
		panic("invalid base register type: " + a.rn.RegType().String())
	} else if rn.IsRealReg() && v0 <= a.rn.RealReg() && a.rn.RealReg() <= v30 {
		panic("BUG: likely a bug in reg alloc or reset behavior")
	}

	switch a.kind {
	case addressModeKindRegScaledExtended:
		amount := a.sizeInBitsToShiftAmount(dstSizeBits)
		ret = fmt.Sprintf("[%s, %s, %s #%#x]", base, formatVRegSized(a.rm, a.indexRegBits()), a.extOp, amount)
	case addressModeKindRegScaled:
		amount := a.sizeInBitsToShiftAmount(dstSizeBits)
		ret = fmt.Sprintf("[%s, %s, lsl #%#x]", base, formatVRegSized(a.rm, a.indexRegBits()), amount)
	case addressModeKindRegExtended:
		ret = fmt.Sprintf("[%s, %s, %s]", base, formatVRegSized(a.rm, a.indexRegBits()), a.extOp)
	case addressModeKindRegReg:
		ret = fmt.Sprintf("[%s, %s]", base, formatVRegSized(a.rm, a.indexRegBits()))
	case addressModeKindRegSignedImm9:
		if a.imm != 0 {
			ret = fmt.Sprintf("[%s, #%#x]", base, a.imm)
		} else {
			ret = fmt.Sprintf("[%s]", base)
		}
	case addressModeKindRegUnsignedImm12:
		if a.imm != 0 {
			ret = fmt.Sprintf("[%s, #%#x]", base, a.imm)
		} else {
			ret = fmt.Sprintf("[%s]", base)
		}
	case addressModeKindPostIndex:
		ret = fmt.Sprintf("[%s], #%#x", base, a.imm)
	case addressModeKindPreIndex:
		ret = fmt.Sprintf("[%s, #%#x]!", base, a.imm)
	case addressModeKindArgStackSpace:
		ret = fmt.Sprintf("[#arg_space, #%#x]", a.imm)
	case addressModeKindResultStackSpace:
		ret = fmt.Sprintf("[#ret_space, #%#x]", a.imm)
	}
	return
}

func addressModePreOrPostIndex(rn regalloc.VReg, imm int64, preIndex bool) addressMode {
	if !offsetFitsInAddressModeKindRegSignedImm9(imm) {
		panic(fmt.Sprintf("BUG: offset %#x does not fit in addressModeKindRegSignedImm9", imm))
	}
	if preIndex {
		return addressMode{kind: addressModeKindPreIndex, rn: rn, imm: imm}
	} else {
		return addressMode{kind: addressModeKindPostIndex, rn: rn, imm: imm}
	}
}

func offsetFitsInAddressModeKindRegUnsignedImm12(dstSizeInBits byte, offset int64) bool {
	divisor := int64(dstSizeInBits) / 8
	return 0 < offset && offset%divisor == 0 && offset/divisor < 4096
}

func offsetFitsInAddressModeKindRegSignedImm9(offset int64) bool {
	return -256 <= offset && offset <= 255
}

func (a addressMode) indexRegBits() byte {
	bits := a.extOp.srcBits()
	if bits != 32 && bits != 64 {
		panic("invalid index register for address mode. it must be either 32 or 64 bits")
	}
	return bits
}

func (a addressMode) sizeInBitsToShiftAmount(sizeInBits byte) (lsl byte) {
	switch sizeInBits {
	case 8:
		lsl = 0
	case 16:
		lsl = 1
	case 32:
		lsl = 2
	case 64:
		lsl = 3
	}
	return
}

func extLoadSignSize(op ssa.Opcode) (size byte, signed bool) {
	switch op {
	case ssa.OpcodeUload8:
		size, signed = 8, false
	case ssa.OpcodeUload16:
		size, signed = 16, false
	case ssa.OpcodeUload32:
		size, signed = 32, false
	case ssa.OpcodeSload8:
		size, signed = 8, true
	case ssa.OpcodeSload16:
		size, signed = 16, true
	case ssa.OpcodeSload32:
		size, signed = 32, true
	default:
		panic("BUG")
	}
	return
}

func (m *machine) lowerExtLoad(op ssa.Opcode, ptr ssa.Value, offset uint32, ret regalloc.VReg) {
	size, signed := extLoadSignSize(op)
	amode := m.lowerToAddressMode(ptr, offset, size)
	load := m.allocateInstr()
	if signed {
		load.asSLoad(operandNR(ret), amode, size)
	} else {
		load.asULoad(operandNR(ret), amode, size)
	}
	m.insert(load)
}

func (m *machine) lowerLoad(ptr ssa.Value, offset uint32, typ ssa.Type, ret ssa.Value) {
	amode := m.lowerToAddressMode(ptr, offset, typ.Bits())

	dst := m.compiler.VRegOf(ret)
	load := m.allocateInstr()
	switch typ {
	case ssa.TypeI32, ssa.TypeI64:
		load.asULoad(operandNR(dst), amode, typ.Bits())
	case ssa.TypeF32, ssa.TypeF64:
		load.asFpuLoad(operandNR(dst), amode, typ.Bits())
	case ssa.TypeV128:
		load.asFpuLoad(operandNR(dst), amode, 128)
	default:
		panic("TODO")
	}
	m.insert(load)
}

func (m *machine) lowerLoadSplat(ptr ssa.Value, offset uint32, lane ssa.VecLane, ret ssa.Value) {
	var opSize byte
	switch lane {
	case ssa.VecLaneI8x16:
		opSize = 8
	case ssa.VecLaneI16x8:
		opSize = 16
	case ssa.VecLaneI32x4:
		opSize = 32
	case ssa.VecLaneI64x2:
		opSize = 64
	}
	amode := m.lowerToAddressMode(ptr, offset, opSize)
	rd := operandNR(m.compiler.VRegOf(ret))
	m.lowerLoadSplatFromAddressMode(rd, amode, lane)
}

// lowerLoadSplatFromAddressMode is extracted from lowerLoadSplat for testing.
func (m *machine) lowerLoadSplatFromAddressMode(rd operand, amode addressMode, lane ssa.VecLane) {
	tmpReg := operandNR(m.compiler.AllocateVReg(ssa.TypeI64))

	// vecLoad1R has offset address mode (base+imm) only for post index, so the only addressing mode
	// we can use here is "no-offset" register addressing mode, i.e. `addressModeKindRegReg`.
	switch amode.kind {
	case addressModeKindRegReg:
		add := m.allocateInstr()
		add.asALU(aluOpAdd, tmpReg, operandNR(amode.rn), operandNR(amode.rm), true)
		m.insert(add)
	case addressModeKindRegSignedImm9:
		add := m.allocateInstr()
		add.asALU(aluOpAdd, tmpReg, operandNR(amode.rn), operandImm12(uint16(amode.imm), 0), true)
		m.insert(add)
	case addressModeKindRegUnsignedImm12:
		if amode.imm != 0 {
			offsetReg := m.compiler.AllocateVReg(ssa.TypeI64)
			m.load64bitConst(amode.imm, offsetReg)
			add := m.allocateInstr()
			m.insert(add)
			add.asALU(aluOpAdd, tmpReg, operandNR(amode.rn), operandNR(offsetReg), true)
		} else {
			tmpReg = operandNR(amode.rn)
		}
	default:
		panic("unsupported address mode for LoadSplat")
	}

	arr := ssaLaneToArrangement(lane)

	ld1r := m.allocateInstr()
	ld1r.asVecLoad1R(rd, tmpReg, arr)
	m.insert(ld1r)
}

func (m *machine) lowerStore(si *ssa.Instruction) {
	// TODO: merge consecutive stores into a single pair store instruction.
	value, ptr, offset, storeSizeInBits := si.StoreData()
	amode := m.lowerToAddressMode(ptr, offset, storeSizeInBits)

	valueOp := m.getOperand_NR(m.compiler.ValueDefinition(value), extModeNone)
	store := m.allocateInstr()
	store.asStore(valueOp, amode, storeSizeInBits)
	m.insert(store)
}

// lowerToAddressMode converts a pointer to an addressMode that can be used as an operand for load/store instructions.
func (m *machine) lowerToAddressMode(ptr ssa.Value, offsetBase uint32, size byte) (amode addressMode) {
	// TODO: currently the instruction selection logic doesn't support addressModeKindRegScaledExtended and
	// addressModeKindRegScaled since collectAddends doesn't take ssa.OpcodeIshl into account. This should be fixed
	// to support more efficient address resolution.

	a32s, a64s, offset := m.collectAddends(ptr)
	offset += int64(offsetBase)
	return m.lowerToAddressModeFromAddends(a32s, a64s, size, offset)
}

// lowerToAddressModeFromAddends creates an addressMode from a list of addends collected by collectAddends.
// During the construction, this might emit additional instructions.
//
// Extracted as a separate function for easy testing.
func (m *machine) lowerToAddressModeFromAddends(a32s *queue[addend32], a64s *queue[regalloc.VReg], size byte, offset int64) (amode addressMode) {
	switch a64sExist, a32sExist := !a64s.empty(), !a32s.empty(); {
	case a64sExist && a32sExist:
		var base regalloc.VReg
		base = a64s.dequeue()
		var a32 addend32
		a32 = a32s.dequeue()
		amode = addressMode{kind: addressModeKindRegExtended, rn: base, rm: a32.r, extOp: a32.ext}
	case a64sExist && offsetFitsInAddressModeKindRegUnsignedImm12(size, offset):
		var base regalloc.VReg
		base = a64s.dequeue()
		amode = addressMode{kind: addressModeKindRegUnsignedImm12, rn: base, imm: offset}
		offset = 0
	case a64sExist && offsetFitsInAddressModeKindRegSignedImm9(offset):
		var base regalloc.VReg
		base = a64s.dequeue()
		amode = addressMode{kind: addressModeKindRegSignedImm9, rn: base, imm: offset}
		offset = 0
	case a64sExist:
		var base regalloc.VReg
		base = a64s.dequeue()
		if !a64s.empty() {
			index := a64s.dequeue()
			amode = addressMode{kind: addressModeKindRegReg, rn: base, rm: index, extOp: extendOpUXTX /* indicates index reg is 64-bit */}
		} else {
			amode = addressMode{kind: addressModeKindRegUnsignedImm12, rn: base, imm: 0}
		}
	case a32sExist:
		base32 := a32s.dequeue()

		// First we need 64-bit base.
		base := m.compiler.AllocateVReg(ssa.TypeI64)
		baseExt := m.allocateInstr()
		var signed bool
		if base32.ext == extendOpSXTW {
			signed = true
		}
		baseExt.asExtend(base, base32.r, 32, 64, signed)
		m.insert(baseExt)

		if !a32s.empty() {
			index := a32s.dequeue()
			amode = addressMode{kind: addressModeKindRegExtended, rn: base, rm: index.r, extOp: index.ext}
		} else {
			amode = addressMode{kind: addressModeKindRegUnsignedImm12, rn: base, imm: 0}
		}
	default: // Only static offsets.
		tmpReg := m.compiler.AllocateVReg(ssa.TypeI64)
		m.lowerConstantI64(tmpReg, offset)
		amode = addressMode{kind: addressModeKindRegUnsignedImm12, rn: tmpReg, imm: 0}
		offset = 0
	}

	baseReg := amode.rn
	if offset > 0 {
		baseReg = m.addConstToReg64(baseReg, offset) // baseReg += offset
	}

	for !a64s.empty() {
		a64 := a64s.dequeue()
		baseReg = m.addReg64ToReg64(baseReg, a64) // baseReg += a64
	}

	for !a32s.empty() {
		a32 := a32s.dequeue()
		baseReg = m.addRegToReg64Ext(baseReg, a32.r, a32.ext) // baseReg += (a32 extended to 64-bit)
	}
	amode.rn = baseReg
	return
}

var addendsMatchOpcodes = [4]ssa.Opcode{ssa.OpcodeUExtend, ssa.OpcodeSExtend, ssa.OpcodeIadd, ssa.OpcodeIconst}

func (m *machine) collectAddends(ptr ssa.Value) (addends32 *queue[addend32], addends64 *queue[regalloc.VReg], offset int64) {
	m.addendsWorkQueue.reset()
	m.addends32.reset()
	m.addends64.reset()
	m.addendsWorkQueue.enqueue(ptr)

	for !m.addendsWorkQueue.empty() {
		v := m.addendsWorkQueue.dequeue()

		def := m.compiler.ValueDefinition(v)
		switch op := m.compiler.MatchInstrOneOf(def, addendsMatchOpcodes[:]); op {
		case ssa.OpcodeIadd:
			// If the addend is an add, we recursively collect its operands.
			x, y := def.Instr.Arg2()
			m.addendsWorkQueue.enqueue(x)
			m.addendsWorkQueue.enqueue(y)
			def.Instr.MarkLowered()
		case ssa.OpcodeIconst:
			// If the addend is constant, we just statically merge it into the offset.
			ic := def.Instr
			u64 := ic.ConstantVal()
			if ic.Return().Type().Bits() == 32 {
				offset += int64(int32(u64)) // sign-extend.
			} else {
				offset += int64(u64)
			}
			def.Instr.MarkLowered()
		case ssa.OpcodeUExtend, ssa.OpcodeSExtend:
			switch input := def.Instr.Arg(); input.Type().Bits() {
			case 64:
				// If the input is already 64-bit, this extend is a no-op. TODO: shouldn't this be optimized out at much earlier stage? no?
				m.addends64.enqueue(m.getOperand_NR(m.compiler.ValueDefinition(input), extModeNone).nr())
				def.Instr.MarkLowered()
				continue
			case 32:
				var ext extendOp
				if op == ssa.OpcodeUExtend {
					ext = extendOpUXTW
				} else {
					ext = extendOpSXTW
				}

				inputDef := m.compiler.ValueDefinition(input)
				constInst := inputDef.IsFromInstr() && inputDef.Instr.Constant()
				switch {
				case constInst && ext == extendOpUXTW:
					// Zero-extension of a 32-bit constant can be merged into the offset.
					offset += int64(uint32(inputDef.Instr.ConstantVal()))
				case constInst && ext == extendOpSXTW:
					// Sign-extension of a 32-bit constant can be merged into the offset.
					offset += int64(int32(inputDef.Instr.ConstantVal())) // sign-extend!
				default:
					m.addends32.enqueue(addend32{r: m.getOperand_NR(inputDef, extModeNone).nr(), ext: ext})
				}
				def.Instr.MarkLowered()
				continue
			}
			// If this is the extension smaller than 32 bits, this cannot be merged into addressing mode since
			// arm64 requires index registers must be at least 32 bits (extension modes can only be applied in 32 bits).
			// fallthrough
			panic("TODO: add tests")
		default:
			// If the addend is not one of them, we simply use it as-is (without merging!), optionally zero-extending it.
			m.addends64.enqueue(m.getOperand_NR(def, extModeZeroExtend64 /* optional zero ext */).nr())
		}
	}
	return &m.addends32, &m.addends64, offset
}

func (m *machine) addConstToReg64(r regalloc.VReg, c int64) (rd regalloc.VReg) {
	rd = m.compiler.AllocateVReg(ssa.TypeI64)
	alu := m.allocateInstr()
	if imm12Op, ok := asImm12Operand(uint64(c)); ok {
		alu.asALU(aluOpAdd, operandNR(rd), operandNR(r), imm12Op, true)
	} else if imm12Op, ok = asImm12Operand(uint64(-c)); ok {
		alu.asALU(aluOpSub, operandNR(rd), operandNR(r), imm12Op, true)
	} else {
		tmp := m.compiler.AllocateVReg(ssa.TypeI64)
		m.load64bitConst(c, tmp)
		alu.asALU(aluOpAdd, operandNR(rd), operandNR(r), operandNR(tmp), true)
	}
	m.insert(alu)
	return
}

func (m *machine) addReg64ToReg64(rn, rm regalloc.VReg) (rd regalloc.VReg) {
	rd = m.compiler.AllocateVReg(ssa.TypeI64)
	alu := m.allocateInstr()
	alu.asALU(aluOpAdd, operandNR(rd), operandNR(rn), operandNR(rm), true)
	m.insert(alu)
	return
}

func (m *machine) addRegToReg64Ext(rn, rm regalloc.VReg, ext extendOp) (rd regalloc.VReg) {
	rd = m.compiler.AllocateVReg(ssa.TypeI64)
	alu := m.allocateInstr()
	alu.asALU(aluOpAdd, operandNR(rd), operandNR(rn), operandER(rm, ext, 64), true)
	m.insert(alu)
	return
}

// queue is the resettable queue where the underlying slice is reused.
type queue[T any] struct {
	index int
	data  []T
}

func (q *queue[T]) enqueue(v T) {
	q.data = append(q.data, v)
}

func (q *queue[T]) dequeue() (ret T) {
	ret = q.data[q.index]
	q.index++
	return
}

func (q *queue[T]) empty() bool {
	return q.index >= len(q.data)
}

func (q *queue[T]) reset() {
	q.index = 0
	q.data = q.data[:0]
}
