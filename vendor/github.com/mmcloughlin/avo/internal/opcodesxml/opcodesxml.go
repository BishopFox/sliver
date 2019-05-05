// Package opcodesxml is a reader for the Opcodes XML database.
package opcodesxml

import (
	"encoding/xml"
	"io"
	"os"
)

// Read reads Opcodes XML format.
func Read(r io.Reader) (*InstructionSet, error) {
	d := xml.NewDecoder(r)
	is := &InstructionSet{}
	if err := d.Decode(is); err != nil {
		return nil, err
	}
	return is, nil
}

// ReadFile reads the given Opcodes XML file.
func ReadFile(filename string) (*InstructionSet, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Read(f)
}

// InstructionSet is entire x86-64 instruction set.
type InstructionSet struct {
	Name         string        `xml:"name,attr"`
	Instructions []Instruction `xml:"Instruction"`
}

// Instruction represents one x86 mnemonic and its forms.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L7-L14
//
//	    """Instruction is defined by its mnemonic name (in Intel-style assembly).
//
//	    An instruction may have multiple forms, that mainly differ by operand types.
//
//	    :ivar name: instruction name in Intel-style assembly (PeachPy, NASM and YASM assemblers).
//	    :ivar summary: a summary description of the instruction name.
//	    :ivar forms: a list of :class:`InstructionForm` objects representing the instruction forms.
//	    """
//
type Instruction struct {
	Name    string `xml:"name,attr"`
	Summary string `xml:"summary,attr"`
	Forms   []Form `xml:"InstructionForm"`
}

// Form represents one possible collection of operands an instruction may take.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L29-L85
//
//	    """Instruction form is a combination of mnemonic name and operand types.
//
//	    An instruction form may have multiple possible encodings.
//
//	    :ivar name: instruction name in PeachPy, NASM and YASM assemblers.
//	    :ivar gas_name: instruction form name in GNU assembler (gas).
//	    :ivar go_name: instruction form name in Go/Plan 9 assembler (8a).
//
//	        None means instruction is not supported in Go/Plan 9 assembler.
//
//	    :ivar mmx_mode: MMX technology state required or forced by this instruction. Possible values are:
//
//	        "FPU"
//	            Instruction requires the MMX technology state to be clear.
//
//	        "MMX"
//	            Instruction causes transition to MMX technology state.
//
//	        None
//	            Instruction neither affects nor cares about the MMX technology state.
//
//	    :ivar xmm_mode: XMM registers state accessed by this instruction. Possible values are:
//
//	        "SSE"
//	            Instruction accesses XMM registers in legacy SSE mode.
//
//	        "AVX"
//	            Instruction accesses XMM registers in AVX mode.
//
//	        None
//	            Instruction does not affect XMM registers and does not change XMM registers access mode.
//
//	    :ivar cancelling_inputs: indicates that the instruction form has not dependency on the values of input operands
//	        when they refer to the same register. E.g. **VPXOR xmm1, xmm0, xmm0** does not depend on *xmm0*.
//
//	        Instruction forms with cancelling inputs have only two input operands, which have the same register type.
//
//	    :ivar nacl_version: indicates the earliest Pepper API version where validator supports this instruction.
//
//	        Possible values are integers >= 33 or None. Pepper 33 is the earliest version for which information on
//	        supported instructions is available; if instruction forms supported before Pepper 33 would have
//	        nacl_version == 33. None means instruction is either not yet supported by Native Client validator, or
//	        is forbidden in Native Client SFI model.
//
//	    :ivar nacl_zero_extends_outputs: indicates that Native Client validator recognizes that the instruction zeroes
//	        the upper 32 bits of the output registers.
//
//	        In x86-64 Native Client SFI model this means that the subsequent instruction can use registers written by
//	        this instruction for memory addressing.
//
//	    :ivar operands: a list of :class:`Operand` objects representing the instruction operands.
//	    :ivar implicit_inputs: a set of register names that are implicitly read by this instruction.
//	    :ivar implicit_outputs: a set of register names that are implicitly written by this instruction.
//	    :ivar isa_extensions: a list of :class:`ISAExtension` objects that represent the ISA extensions required to execute
//	        the instruction.
//	    :ivar encodings: a list of :class:`Encoding` objects representing the possible encodings for this instruction.
//	    """
//
type Form struct {
	GASName          string            `xml:"gas-name,attr"`
	GoName           string            `xml:"go-name,attr"`
	MMXMode          string            `xml:"mmx-mode,attr"`
	XMMMode          string            `xml:"xmm-mode,attr"`
	CancellingInputs bool              `xml:"cancelling-inputs,attr"`
	Operands         []Operand         `xml:"Operand"`
	ImplicitOperands []ImplicitOperand `xml:"ImplicitOperand"`
	ISA              []ISA             `xml:"ISA"`
	Encoding         Encoding          `xml:"Encoding"`
}

// Operand describes an accepted operand type and the read/write action the instruction will perform.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L114-L338
//
//	    """An explicit instruction operand.
//
//	    :ivar type: the type of the instruction operand. Possible values are:
//
//	        "1"
//	            The constant value `1`.
//
//	        "3"
//	            The constant value `3`.
//
//	        "al"
//	            The al register.
//
//	        "ax"
//	            The ax register.
//
//	        "eax"
//	            The eax register.
//
//	        "rax"
//	            The rax register.
//
//	        "cl"
//	            The cl register.
//
//	        "xmm0"
//	            The xmm0 register.
//
//	        "rel8"
//	            An 8-bit signed offset relative to the address of instruction end.
//
//	        "rel32"
//	            A 32-bit signed offset relative to the address of instruction end.
//
//	        "imm4"
//	            A 4-bit immediate value.
//
//	        "imm8"
//	            An 8-bit immediate value.
//
//	        "imm16"
//	            A 16-bit immediate value.
//
//	        "imm32"
//	            A 32-bit immediate value.
//
//	        "imm64"
//	            A 64-bit immediate value.
//
//	        "r8"
//	            An 8-bit general-purpose register (al, bl, cl, dl, sil, dil, bpl, spl, r8b-r15b).
//
//	        "r16"
//	            A 16-bit general-purpose register (ax, bx, cx, dx, si, di, bp, sp, r8w-r15w).
//
//	        "r32"
//	            A 32-bit general-purpose register (eax, ebx, ecx, edx, esi, edi, ebp, esp, r8d-r15d).
//
//	        "r64"
//	            A 64-bit general-purpose register (rax, rbx, rcx, rdx, rsi, rdi, rbp, rsp, r8-r15).
//
//	        "mm"
//	            A 64-bit MMX SIMD register (mm0-mm7).
//
//	        "xmm"
//	            A 128-bit XMM SIMD register (xmm0-xmm31).
//
//	        "xmm{k}"
//	            A 128-bit XMM SIMD register (xmm0-xmm31), optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "xmm{k}{z}"
//	            A 128-bit XMM SIMD register (xmm0-xmm31), optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "ymm"
//	            A 256-bit YMM SIMD register (ymm0-ymm31).
//
//	        "ymm{k}"
//	            A 256-bit YMM SIMD register (ymm0-ymm31), optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "ymm{k}{z}"
//	            A 256-bit YMM SIMD register (ymm0-ymm31), optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "zmm"
//	            A 512-bit ZMM SIMD register (zmm0-zmm31).
//
//	        "zmm{k}"
//	            A 512-bit ZMM SIMD register (zmm0-zmm31), optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "zmm{k}{z}"
//	            A 512-bit ZMM SIMD register (zmm0-zmm31), optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "k"
//	            An AVX-512 mask register (k0-k7).
//
//	        "k{k}"
//	            An AVX-512 mask register (k0-k7), optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "m"
//	            A memory operand of any size.
//
//	        "m8"
//	            An 8-bit memory operand.
//
//	        "m16"
//	            A 16-bit memory operand.
//
//	        "m16{k}{z}"
//	            A 16-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m32"
//	            A 32-bit memory operand.
//
//	        "m32{k}"
//	            A 32-bit memory operand, optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "m32{k}{z}"
//	            A 32-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m64"
//	            A 64-bit memory operand.
//
//	        "m64{k}"
//	            A 64-bit memory operand, optionally merge-masked by an AVX-512 mask register (k1-k7).
//
//	        "m64{k}{z}"
//	            A 64-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m80"
//	            An 80-bit memory operand.
//
//	        "m128"
//	            A 128-bit memory operand.
//
//	        "m128{k}{z}"
//	            A 128-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m256"
//	            A 256-bit memory operand.
//
//	        "m256{k}{z}"
//	            A 256-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m512"
//	            A 512-bit memory operand.
//
//	        "m512{k}{z}"
//	            A 512-bit memory operand, optionally masked by an AVX-512 mask register (k1-k7).
//
//	        "m64/m32bcst"
//	            A 64-bit memory operand or a 32-bit memory operand broadcasted to 64 bits {1to2}.
//
//	        "m128/m32bcst"
//	            A 128-bit memory operand or a 32-bit memory operand broadcasted to 128 bits {1to4}.
//
//	        "m256/m32bcst"
//	            A 256-bit memory operand or a 32-bit memory operand broadcasted to 256 bits {1to8}.
//
//	        "m512/m32bcst"
//	            A 512-bit memory operand or a 32-bit memory operand broadcasted to 512 bits {1to16}.
//
//	        "m128/m64bcst"
//	            A 128-bit memory operand or a 64-bit memory operand broadcasted to 128 bits {1to2}.
//
//	        "m256/m64bcst"
//	            A 256-bit memory operand or a 64-bit memory operand broadcasted to 256 bits {1to4}.
//
//	        "m512/m64bcst"
//	            A 512-bit memory operand or a 64-bit memory operand broadcasted to 512 bits {1to8}.
//
//	        "vm32x"
//	            A vector of memory addresses using VSIB with 32-bit indices in XMM register.
//
//	        "vm32x{k}"
//	            A vector of memory addresses using VSIB with 32-bit indices in XMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "vm32y"
//	            A vector of memory addresses using VSIB with 32-bit indices in YMM register.
//
//	        "vm32y{k}"
//	            A vector of memory addresses using VSIB with 32-bit indices in YMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "vm32z"
//	            A vector of memory addresses using VSIB with 32-bit indices in ZMM register.
//
//	        "vm32z{k}"
//	            A vector of memory addresses using VSIB with 32-bit indices in ZMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "vm64x"
//	            A vector of memory addresses using VSIB with 64-bit indices in XMM register.
//
//	        "vm64x{k}"
//	            A vector of memory addresses using VSIB with 64-bit indices in XMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "vm64y"
//	            A vector of memory addresses using VSIB with 64-bit indices in YMM register.
//
//	        "vm64y{k}"
//	            A vector of memory addresses using VSIB with 64-bit indices in YMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "vm64z"
//	            A vector of memory addresses using VSIB with 64-bit indices in ZMM register.
//
//	        "vm64z{k}"
//	            A vector of memory addresses using VSIB with 64-bit indices in ZMM register merge-masked by an AVX-512 mask
//	            register (k1-k7).
//
//	        "{sae}"
//	            Suppress-all-exceptions modifier. This operand is optional and can be omitted.
//
//	        "{er}"
//	            Embedded rounding control. This operand is optional and can be omitted.
//
//	    :ivar is_input: indicates if the instruction reads the variable specified by this operand.
//	    :ivar is_output: indicates if the instruction writes the variable specified by this operand.
//	    :ivar extended_size: for immediate operands the size of the value in bytes after size-extension.
//
//	        The extended size affects which operand values can be encoded. E.g. a signed imm8 operand would normally \
//	        encode values in the [-128, 127] range. But if it is extended to 4 bytes, it can also encode values in \
//	        [2**32 - 128, 2**32 - 1] range.
//	    """
//
type Operand struct {
	Type   string `xml:"type,attr"`
	Input  bool   `xml:"input,attr"`
	Output bool   `xml:"output,attr"`
}

// ImplicitOperand represents the implicit action of an instruction on a register.
type ImplicitOperand struct {
	ID     string `xml:"id,attr"`
	Input  bool   `xml:"input,attr"`
	Output bool   `xml:"output,attr"`
}

// ISA is the name of an instruction set extension an instruction form belongs to.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L430-L487
//
//	    """An extension to x86-64 instruction set.
//
//	    :ivar name: name of the ISA extension. Possible values are:
//
//	        - "RDTSC"           := The `RDTSC` instruction.
//	        - "RDTSCP"          := The `RDTSCP` instruction.
//	        - "CPUID"           := The `CPUID` instruction.
//	        - "FEMMS"           := The `FEMMS` instruction.
//	        - "MOVBE"           := The `MOVBE` instruction.
//	        - "POPCNT"          := The `POPCNT` instruction.
//	        - "LZCNT"           := The `LZCNT` instruction.
//	        - "PCLMULQDQ"       := The `PCLMULQDQ` instruction.
//	        - "RDRAND"          := The `RDRAND` instruction.
//	        - "RDSEED"          := The `RDSEED` instruction.
//	        - "CLFLUSH"         := The `CLFLUSH` instruction.
//	        - "CLFLUSHOPT"      := The `CLFLUSHOPT` instruction.
//	        - "CLWB"            := The `CLWB` instruction.
//	        - "CLZERO"          := The `CLZERO` instruction.
//	        - "PREFETCH"        := The `PREFETCH` instruction (3dnow! Prefetch).
//	        - "PREFETCHW"       := The `PREFETCHW` instruction (3dnow! Prefetch/Intel PRFCHW).
//	        - "PREFETCHWT1"     := The `PREFETCHWT1` instruction.
//	        - "MONITOR"         := The `MONITOR` and `MWAIT` instructions.
//	        - "MONITORX"        := The `MONITORX` and `MWAITX` instructions.
//	        - "CMOV"            := Conditional MOVe instructions.
//	        - "MMX"             := MultiMedia eXtension.
//	        - "MMX+"            := AMD MMX+ extension / Integer SSE (Intel).
//	        - "3dnow!"          := AMD 3dnow! extension.
//	        - "3dnow+!"         := AMD 3dnow!+ extension.
//	        - "SSE"             := Streaming SIMD Extension.
//	        - "SSE2"            := Streaming SIMD Extension 2.
//	        - "SSE3"            := Streaming SIMD Extension 3.
//	        - "SSSE3"           := Supplemental Streaming SIMD Extension 3.
//	        - "SSE4.1"          := Streaming SIMD Extension 4.1.
//	        - "SSE4.2"          := Streaming SIMD Extension 4.2.
//	        - "SSE4A"           := Streaming SIMD Extension 4a.
//	        - "AVX"             := Advanced Vector eXtension.
//	        - "AVX2"            := Advanced Vector eXtension 2.
//	        - "AVX512F"         := AVX-512 Foundation instructions.
//	        - "AVX512BW"        := AVX-512 Byte and Word instructions.
//	        - "AVX512DQ"        := AVX-512 Doubleword and Quadword instructions.
//	        - "AVX512VL"        := AVX-512 Vector Length extension (EVEX-encoded XMM/YMM operations).
//	        - "AVX512PF"        := AVX-512 Prefetch instructions.
//	        - "AVX512ER"        := AVX-512 Exponential and Reciprocal instructions.
//	        - "AVX512CD"        := AVX-512 Conflict Detection instructions.
//	        - "AVX512VBMI"      := AVX-512 Vector Bit Manipulation instructions.
//	        - "AVX512IFMA"      := AVX-512 Integer 52-bit Multiply-Accumulate instructions.
//	        - "AVX512VPOPCNTDQ" := AVX-512 Vector Population Count instructions.
//	        - "XOP"             := eXtended OPerations extension.
//	        - "F16C"            := Half-Precision (F16) Conversion instructions.
//	        - "FMA3"            := Fused Multiply-Add instructions (3-operand).
//	        - "FMA4"            := Fused Multiply-Add instructions (4-operand).
//	        - "BMI"             := Bit Manipulation Instructions.
//	        - "BMI2"            := Bit Manipulation Instructions 2.
//	        - "TBM"             := Trailing Bit Manipulation instructions.
//	        - "ADX"             := The `ADCX` and `ADOX` instructions.
//	        - "AES"             := `AES` instruction set.
//	        - "SHA"             := `SHA` instruction set.
//	    """
//
type ISA struct {
	ID string `xml:"id,attr"`
}

// Encoding describes the instruction form binary representation.
type Encoding struct {
	REX  *REX  `xml:"REX"`
	VEX  *VEX  `xml:"VEX"`
	EVEX *EVEX `xml:"EVEX"`
}

// REX specifies the REX encoding of an instruction form.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L541-L574
//
//	    """REX prefix.
//
//	    Encoding may have only one REX prefix and if present, it immediately precedes the opcode.
//
//	    :ivar is_mandatory: indicates whether the REX prefix must be encoded even if no extended registers are used.
//
//	        REX is mandatory for most 64-bit instructions (encoded with REX.W = 1) and instructions that operate on the \
//	        extended set of 8-bit registers (to indicate access to dil/sil/bpl/spl as opposed to ah/bh/ch/dh which use the \
//	        same ModR/M).
//
//	    :ivar W: the REX.W bit. Possible values are 0, 1, and None.
//
//	        None indicates that the bit is ignored.
//
//	    :ivar R: the REX.R bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If R is a reference to an instruction operand, the operand is of register type and REX.R bit specifies the \
//	        high bit (bit 3) of the register number.
//
//	    :ivar B: the REX.B bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If R is a reference to an instruction operand, the operand can be of register or memory type. If the operand \
//	        is of register type, the REX.R bit specifies the high bit (bit 3) of the register number, and the REX.X bit is \
//	        ignored. If the operand is of memory type, the REX.R bit specifies the high bit (bit 3) of the base register \
//	        number, and the X instance variable refers to the same operand.
//
//	    :ivar X: the REX.X bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If X is a reference to an instruction operand, the operand is of memory type and the REX.X bit specifies the \
//	        high bit (bit 3) of the index register number, and the B instance variable refers to the same operand.
//	    """
//
type REX struct {
	Mandatory bool   `xml:"mandatory,attr"`
	W         uint   `xml:"W,attr"`
	R         string `xml:"R,attr"`
	X         string `xml:"X,attr"`
	B         string `xml:"B,attr"`
}

// VEX specifies the VEX encoding of an instruction form.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L606-L691
//
//	    """VEX or XOP prefix.
//
//	    VEX and XOP prefixes use the same format and differ only by leading byte.
//	    The `type` property helps to differentiate between the two prefix types.
//
//	    Encoding may have only one VEX prefix and if present, it immediately precedes the opcode, and no other prefix is \
//	    allowed.
//
//	    :ivar type: the type of the leading byte for VEX encoding. Possible values are:
//
//	        "VEX"
//	            The VEX prefix (0xC4 or 0xC5) is used.
//
//	        "XOP"
//	            The XOP prefix (0x8F) is used.
//
//	    :ivar mmmmm: the VEX m-mmmm (implied leading opcode bytes) field. In AMD documentation this field is called map_select. Possible values are:
//
//	        0b00001
//	            Implies 0x0F leading opcode byte.
//
//	        0b00010
//	            Implies 0x0F 0x38 leading opcode bytes.
//
//	        0b00011
//	            Implies 0x0F 0x3A leading opcode bytes.
//
//	        0b01000
//	            This value does not have opcode byte interpretation. Only XOP instructions use this value.
//
//	        0b01001
//	            This value does not have opcode byte interpretation. Only XOP and TBM instructions use this value.
//
//	        0b01010
//	            This value does not have opcode byte interpretation. Only TBM instructions use this value.
//
//	        Only VEX prefix with m-mmmm equal to 0b00001 could be encoded in two bytes.
//
//	    :ivar pp: the VEX pp (implied legacy prefix) field. Possible values are:
//
//	        0b00
//	            No implied prefix.
//
//	        0b01
//	            Implied 0x66 prefix.
//
//	        0b10
//	            Implied 0xF3 prefix.
//
//	        0b11
//	            Implied 0xF2 prefix.
//
//	    :ivar W: the VEX.W bit. Possible values are 0, 1, and None.
//
//	        None indicates that the bit is ignored.
//
//	    :ivar L: the VEX.L bit. Possible values are 0, 1, and None.
//
//	        None indicates that the bit is ignored.
//
//	    :ivar R: the VEX.R bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If R is a reference to an instruction operand, the operand is of register type and VEX.R bit specifies the \
//	        high bit (bit 3) of the register number.
//
//	    :ivar B: the VEX.B bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If R is a reference to an instruction operand, the operand can be of register or memory type. If the operand is \
//	        of register type, the VEX.R bit specifies the high bit (bit 3) of the register number, and the VEX.X bit is \
//	        ignored. If the operand is of memory type, the VEX.R bit specifies the high bit (bit 3) of the base register \
//	        number, and the X instance variable refers to the same operand.
//
//	    :ivar X: the VEX.X bit. Possible values are 0, 1, None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If X is a reference to an instruction operand, the operand is of memory type and the VEX.X bit specifies the \
//	        high bit (bit 3) of the index register number, and the B instance variable refers to the same operand.
//
//	    :ivar vvvv: the VEX vvvv field. Possible values are 0b0000 or a reference to one of the instruction operands.
//
//	        The value 0b0000 indicates that this field is not used. \
//	        If vvvv is a reference to an instruction operand, the operand is of register type and VEX.vvvv field specifies \
//	        its number.
//	    """
//
type VEX struct {
	Type string `xml:"type,attr"`
	W    *uint  `xml:"W,attr"`
	L    uint   `xml:"L,attr"`
	M5   string `xml:"m-mmmm,attr"`
	PP   string `xml:"pp,attr"`
	R    string `xml:"R,attr"`
	X    string `xml:"X,attr"`
	B    string `xml:"B,attr"`
	V4   string `xml:"vvvv,attr"`
}

// EVEX specifies the EVEX encoding of an instruction form.
//
// Reference: https://github.com/Maratyszcza/Opcodes/blob/6e2b0cd9f1403ecaf164dea7019dd54db5aea252/opcodes/x86_64.py#L731-L845
//
//	    """EVEX prefix.
//
//	    Encoding may have only one EVEX prefix and if present, it immediately precedes the opcode, and no other prefix is \
//	    allowed.
//
//	    :ivar mm: the EVEX mm (compressed legacy escape) field. Identical to two low bits of VEX.m-mmmm field. Possible \
//	    values are:
//
//	        0b01
//	            Implies 0x0F leading opcode byte.
//
//	        0b10
//	            Implies 0x0F 0x38 leading opcode bytes.
//
//	        0b11
//	            Implies 0x0F 0x3A leading opcode bytes.
//
//	    :ivar pp: the EVEX pp (compressed legacy prefix) field. Possible values are:
//
//	        0b00
//	            No implied prefix.
//
//	        0b01
//	            Implied 0x66 prefix.
//
//	        0b10
//	            Implied 0xF3 prefix.
//
//	        0b11
//	            Implied 0xF2 prefix.
//
//	    :ivar W: the EVEX.W bit. Possible values are 0, 1, and None.
//
//	        None indicates that the bit is ignored.
//
//	    :ivar LL: the EVEX.L'L bits. Specify either vector length for the operation, or explicit rounding control \
//	    (in which case operation is 512 bits wide). Possible values:
//
//	        None
//	            Indicates that the EVEX.L'L field is ignored.
//
//	        0b00
//	            128-bits wide operation.
//
//	        0b01
//	            256-bits wide operation.
//
//	        0b10
//	            512-bits wide operation.
//
//	        Reference to the last instruction operand
//	            EVEX.L'L are interpreted as rounding control and set to the value specified by the operand. If the rounding
//	            control operand is omitted, EVEX.L'L is set to 0b10 (embedded rounding control is only supported for 512-bit
//	            wide operations).
//
//	    :ivar RR: the EVEX.R'R bits. Possible values are None, or a reference to an register-type instruction operand.
//
//	        None indicates that the field is ignored.
//	        The R' bit specifies bit 4 of the register number and the R bit specifies bit 3 of the register number.
//
//	    :ivar B: the EVEX.B bit. Possible values are None, or a reference to one of the instruction operands.
//
//	        None indicates that this bit is ignored. \
//	        If R is a reference to an instruction operand, the operand can be of register or memory type. If the operand is\
//	        of register type, the EVEX.R bit specifies the high bit (bit 3) of the register number, and the EVEX.X bit is \
//	        ignored. If the operand is of memory type, the EVEX.R bit specifies the high bit (bit 3) of the base register \
//	        number, and the X instance variable refers to the same operand.
//
//	    :ivar X: the EVEX.X bit. Possible values are None, or a reference to one of the instruction operands.
//
//	        The value None indicates that this bit is ignored. \
//	        If X is a reference to an instruction operand, the operand is of memory type and the EVEX.X bit specifies the \
//	        high bit (bit 3) of the index register number, and the B instance variable refers to the same operand.
//
//	    :ivar vvvv: the EVEX vvvv field. Possible values are 0b0000 or a reference to one of the instruction operands.
//
//	        The value 0b0000 indicates that this field is not used. \
//	        If vvvv is a reference to an instruction operand, the operand is of register type and EVEX.vvvv field \
//	        specifies the register number.
//
//	    :ivar V: the EVEX V field. Possible values are 0, or a reference to one of the instruction operands.
//
//	        The value 0 indicates that this field is not used (EVEX.vvvv is not used or encodes a general-purpose register).
//
//	    :ivar b: the EVEX b (broadcast/rounding control/suppress all exceptions context) bit. Possible values are 0 or a \
//	    reference to one of the instruction operands.
//
//	        The value 0 indicates that this field is not used. \
//	        If b is a reference to an instruction operand, the operand can be a memory operand with optional broadcasting, \
//	        an optional rounding specification, or an optional Suppress-all-exceptions specification. \
//	        If b is a reference to a memory operand, EVEX.b encodes whether broadcasting is used to the operand. \
//	        If b is a reference to a optional rounding control specification, EVEX.b encodes whether explicit rounding \
//	        control is used. \
//	        If b is a reference to a suppress-all-exceptions specification, EVEX.b encodes whether suppress-all-exceptions \
//	        is enabled.
//
//	    :ivar aaa: the EVEX aaa (embedded opmask register specifier) field. Possible values are 0 or a reference to one of \
//	    the instruction operands.
//
//	        The value 0 indicates that this field is not used. \
//	        If aaa is a reference to an instruction operand, the operand supports register mask, and EVEX.aaa encodes the \
//	        mask register.
//
//	    :ivar z: the EVEX z bit. Possible values are None, 0 or a reference to one of the instruction operands.
//
//	        None indicates that the bit is ignored. \
//	        The value 0 indicates that the bit is not used. \
//	        If z is a reference to an instruction operand, the operand supports zero-masking with register mask, and \
//	        EVEX.z indicates whether zero-masking is used.
//
//	    :ivar disp8xN: the N value used for encoding compressed 8-bit displacement. Possible values are powers of 2 in \
//	    [1, 64] range or None.
//
//	        None indicates that this instruction form does not use displacement (the form has no memory operands).
//	    """
//
type EVEX struct {
	M2      string `xml:"mm,attr"`
	PP      string `xml:"pp,attr"`
	W       *uint  `xml:"W,attr"`
	LL      string `xml:"LL,attr"`
	V4      string `xml:"vvvv,attr"`
	V       string `xml:"V,attr"`
	RR      string `xml:"RR,attr"`
	B       string `xml:"B,attr"`
	X       string `xml:"X,attr"`
	Bsml    string `xml:"b,attr"`
	A3      string `xml:"aaa,attr"`
	Z       string `xml:"Z,attr"`
	Disp8xN string `xml:"disp8xN,attr"`
}
