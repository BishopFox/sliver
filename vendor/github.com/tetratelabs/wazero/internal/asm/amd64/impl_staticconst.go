package amd64

import (
	"fmt"
	"math"

	"github.com/tetratelabs/wazero/internal/asm"
)

// defaultMaxDisplacementForConstantPool is the maximum displacement allowed for literal move instructions which access
// the constant pool. This is set as 2 ^30 conservatively while the actual limit is 2^31 since we actually allow this
// limit plus max(length(c) for c in the pool) so we must ensure that limit is less than 2^31.
const defaultMaxDisplacementForConstantPool = 1 << 30

func (a *AssemblerImpl) maybeFlushConstants(buf asm.Buffer, isEndOfFunction bool) {
	if a.pool.Empty() {
		return
	}

	if isEndOfFunction ||
		// If the distance between (the first use in binary) and (end of constant pool) can be larger
		// than MaxDisplacementForConstantPool, we have to emit the constant pool now, otherwise
		// a const might be unreachable by a literal move whose maximum offset is +- 2^31.
		((a.pool.PoolSizeInBytes+buf.Len())-int(a.pool.FirstUseOffsetInBinary)) >= a.MaxDisplacementForConstantPool {

		if !isEndOfFunction {
			// Adds the jump instruction to skip the constants if this is not the end of function.
			//
			// TODO: consider NOP padding for this jump, though this rarely happens as most functions should be
			// small enough to fit all consts after the end of function.
			if a.pool.PoolSizeInBytes >= math.MaxInt8-2 {
				// long (near-relative) jump: https://www.felixcloutier.com/x86/jmp
				buf.AppendByte(0xe9)
				buf.AppendUint32(uint32(a.pool.PoolSizeInBytes))
			} else {
				// short jump: https://www.felixcloutier.com/x86/jmp
				buf.AppendByte(0xeb)
				buf.AppendByte(byte(a.pool.PoolSizeInBytes))
			}
		}

		for _, c := range a.pool.Consts {
			c.SetOffsetInBinary(uint64(buf.Len()))
			buf.AppendBytes(c.Raw)
		}

		a.pool.Reset()
	}
}

func (a *AssemblerImpl) encodeRegisterToStaticConst(buf asm.Buffer, n *nodeImpl) (err error) {
	var opc []byte
	var rex byte
	switch n.instruction {
	case CMPL:
		opc, rex = []byte{0x3b}, rexPrefixNone
	case CMPQ:
		opc, rex = []byte{0x3b}, rexPrefixW
	default:
		return errorEncodingUnsupported(n)
	}
	return a.encodeStaticConstImpl(buf, n, opc, rex, 0)
}

var staticConstToRegisterOpcodes = [...]struct {
	opcode, vopcode                   []byte
	mandatoryPrefix, vmandatoryPrefix byte
	rex                               rexPrefix
}{
	// https://www.felixcloutier.com/x86/movdqu:vmovdqu8:vmovdqu16:vmovdqu32:vmovdqu64
	MOVDQU: {mandatoryPrefix: 0xf3, opcode: []byte{0x0f, 0x6f}},
	// https://www.felixcloutier.com/x86/lea
	LEAQ: {opcode: []byte{0x8d}, rex: rexPrefixW},
	// https://www.felixcloutier.com/x86/movupd
	MOVUPD: {mandatoryPrefix: 0x66, opcode: []byte{0x0f, 0x10}},
	// https://www.felixcloutier.com/x86/mov
	MOVL: {opcode: []byte{0x8b}, vopcode: []byte{0x0f, 0x6e}, vmandatoryPrefix: 0x66},
	MOVQ: {opcode: []byte{0x8b}, rex: rexPrefixW, vopcode: []byte{0x0f, 0x7e}, vmandatoryPrefix: 0xf3},
	// https://www.felixcloutier.com/x86/ucomisd
	UCOMISD: {opcode: []byte{0x0f, 0x2e}, mandatoryPrefix: 0x66},
	// https://www.felixcloutier.com/x86/ucomiss
	UCOMISS: {opcode: []byte{0x0f, 0x2e}},
	// https://www.felixcloutier.com/x86/subss
	SUBSS: {opcode: []byte{0x0f, 0x5c}, mandatoryPrefix: 0xf3},
	// https://www.felixcloutier.com/x86/subsd
	SUBSD: {opcode: []byte{0x0f, 0x5c}, mandatoryPrefix: 0xf2},
	// https://www.felixcloutier.com/x86/cmp
	CMPL: {opcode: []byte{0x39}},
	CMPQ: {opcode: []byte{0x39}, rex: rexPrefixW},
	// https://www.felixcloutier.com/x86/add
	ADDL: {opcode: []byte{0x03}},
	ADDQ: {opcode: []byte{0x03}, rex: rexPrefixW},
}

func (a *AssemblerImpl) encodeStaticConstToRegister(buf asm.Buffer, n *nodeImpl) (err error) {
	var opc []byte
	var rex, mandatoryPrefix byte
	info := staticConstToRegisterOpcodes[n.instruction]
	switch n.instruction {
	case MOVL, MOVQ:
		if isVectorRegister(n.dstReg) {
			opc, mandatoryPrefix = info.vopcode, info.vmandatoryPrefix
			break
		}
		fallthrough
	default:
		opc, rex, mandatoryPrefix = info.opcode, info.rex, info.mandatoryPrefix
	}
	return a.encodeStaticConstImpl(buf, n, opc, rex, mandatoryPrefix)
}

// encodeStaticConstImpl encodes an instruction where mod:r/m points to the memory location of the static constant n.staticConst,
// and the other operand is the register given at n.srcReg or n.dstReg.
func (a *AssemblerImpl) encodeStaticConstImpl(buf asm.Buffer, n *nodeImpl, opcode []byte, rex rexPrefix, mandatoryPrefix byte) error {
	a.pool.AddConst(n.staticConst, uint64(buf.Len()))

	var reg asm.Register
	if n.dstReg != asm.NilRegister {
		reg = n.dstReg
	} else {
		reg = n.srcReg
	}

	reg3Bits, rexPrefix := register3bits(reg, registerSpecifierPositionModRMFieldReg)
	rexPrefix |= rex

	base := buf.Len()
	code := buf.Append(len(opcode) + 7)[:0]

	if mandatoryPrefix != 0 {
		code = append(code, mandatoryPrefix)
	}

	if rexPrefix != rexPrefixNone {
		code = append(code, rexPrefix)
	}

	code = append(code, opcode...)

	// https://wiki.osdev.org/X86-64_Instruction_Encoding#32.2F64-bit_addressing
	modRM := 0b00_000_101 | // Indicate "[RIP + 32bit displacement]" encoding.
		(reg3Bits << 3) // Place the reg on ModRM:reg.
	code = append(code, modRM)

	// Preserve 4 bytes for displacement which will be filled after we finalize the location.
	code = append(code, 0, 0, 0, 0)

	if !n.staticConstReferrersAdded {
		a.staticConstReferrers = append(a.staticConstReferrers, staticConstReferrer{n: n, instLen: len(code)})
		n.staticConstReferrersAdded = true
	}

	buf.Truncate(base + len(code))
	return nil
}

// CompileStaticConstToRegister implements Assembler.CompileStaticConstToRegister.
func (a *AssemblerImpl) CompileStaticConstToRegister(instruction asm.Instruction, c *asm.StaticConst, dstReg asm.Register) (err error) {
	if len(c.Raw)%2 != 0 {
		err = fmt.Errorf("the length of a static constant must be even but was %d", len(c.Raw))
		return
	}

	n := a.newNode(instruction, operandTypesStaticConstToRegister)
	n.dstReg = dstReg
	n.staticConst = c
	return
}

// CompileRegisterToStaticConst implements Assembler.CompileRegisterToStaticConst.
func (a *AssemblerImpl) CompileRegisterToStaticConst(instruction asm.Instruction, srcReg asm.Register, c *asm.StaticConst) (err error) {
	if len(c.Raw)%2 != 0 {
		err = fmt.Errorf("the length of a static constant must be even but was %d", len(c.Raw))
		return
	}

	n := a.newNode(instruction, operandTypesRegisterToStaticConst)
	n.srcReg = srcReg
	n.staticConst = c
	return
}
