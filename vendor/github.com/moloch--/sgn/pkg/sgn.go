package sgn

import (
	"errors"
	"fmt"
	"math/rand"

	keystone "github.com/moloch--/go-keystone"
)

// REG structure for registers
type REG struct {
	Full     string
	Extended string
	High     string
	Low      string
	Arch     int
}

// INSTRUCTION contains instruction information
// Intel syntax mandates "When two operands are present in an arithmetic or logical instruction, the right operand is the source and the left
// operand is the destination." for our case first operand will allways will be considered destination operand
type INSTRUCTION struct {
	Mnemonic string `json:"Mnemonic"`
	V64      bool   `json:"V64"`
	V32      bool   `json:"V32"`
	Operands []struct {
		Types []string `json:"Types"`
	} `json:"Operands"`
}

// Initialize the register values
func init() {

	// Setup x86 GP the register values
	REGS = make(map[int][]REG)
	REGS[32] = append(REGS[32], REG{Extended: "EAX", High: "AX", Low: "AL", Arch: 32})
	REGS[32] = append(REGS[32], REG{Extended: "EBX", High: "BX", Low: "BL", Arch: 32})
	REGS[32] = append(REGS[32], REG{Extended: "ECX", High: "CX", Low: "CL", Arch: 32})
	REGS[32] = append(REGS[32], REG{Extended: "EDX", High: "DX", Low: "DL", Arch: 32})
	// since there is no way to access 1 byte use above instead
	REGS[32] = append(REGS[32], REG{Extended: "ESI", High: "SI", Low: "AL", Arch: 32})
	REGS[32] = append(REGS[32], REG{Extended: "EDI", High: "DI", Low: "BL", Arch: 32})
	// Setup x64 GP the register values
	REGS[64] = append(REGS[64], REG{Full: "RAX", Extended: "EAX", High: "AX", Low: "AL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "RBX", Extended: "EBX", High: "BX", Low: "BL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "RCX", Extended: "ECX", High: "CX", Low: "CL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "RDX", Extended: "EDX", High: "DX", Low: "DL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "RSI", Extended: "ESI", High: "SI", Low: "SIL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "RDI", Extended: "EDI", High: "DX", Low: "DIL", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R8", Extended: "R8D", High: "R8W", Low: "R8B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R9", Extended: "R9D", High: "R9W", Low: "R9B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R10", Extended: "R10D", High: "R10W", Low: "R10B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R11", Extended: "R11D", High: "R11W", Low: "R11B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R12", Extended: "R12D", High: "R12W", Low: "R12B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R13", Extended: "R13D", High: "R13W", Low: "R13B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R14", Extended: "R14D", High: "R14W", Low: "R14B", Arch: 64})
	REGS[64] = append(REGS[64], REG{Full: "R15", Extended: "R15D", High: "R15W", Low: "R15B", Arch: 64})

	// Set the decoder stubs
	STUB = make(map[int]string)
	STUB[32] = X86_DECODER_STUB
	STUB[64] = X64_DECODER_STUB

	// Set safe register prefix/suffix
	SafeRegisterPrefix = make(map[int]([]byte))
	SafeRegisterSuffix = make(map[int]([]byte))
	SafeRegisterPrefix[32] = X86_REG_SAVE_PREFIX
	SafeRegisterPrefix[64] = X64_REG_SAVE_PREFIX

	SafeRegisterSuffix[32] = X86_REG_SAVE_SUFFIX
	SafeRegisterSuffix[64] = X64_REG_SAVE_SUFFIX
}

// SafeRegisterPrefix contains the instructions for saving registers to stack
var SafeRegisterPrefix map[int]([]byte)

// SafeRegisterSuffix contains the instructions for restoring registers from stack
var SafeRegisterSuffix map[int]([]byte)

// X86_REG_SAVE_PREFIX instructions for saving registers to stack
var X86_REG_SAVE_PREFIX = []byte{0x60, 0x9c} // PUSHAD, PUSHFD
// X86_REG_SAVE_SUFFIX instructions for saving registers to stack
var X86_REG_SAVE_SUFFIX = []byte{0x9d, 0x61} // POPFD, POPAD

// X64_REG_SAVE_PREFIX instructions for saving registers to stack
var X64_REG_SAVE_PREFIX = []byte{
	0x50, 0x53, 0x51, 0x52, // PUSH RAX,RBX,RCX,RDX
	0x56, 0x57, 0x55, 0x54, // PUSH RSI,RDI,RBP,RSP
	0x41, 0x50, 0x41, 0x51, // PUSH R8,R9
	0x41, 0x52, 0x41, 0x53, // PUSH R10,R11
	0x41, 0x54, 0x41, 0x55, // PUSH R12,R13
	0x41, 0x56, 0x41, 0x57, // PUSH R14,R15
}

// X64_REG_SAVE_SUFFIX instructions for saving registers to stack
var X64_REG_SAVE_SUFFIX = []byte{
	0x41, 0x5f, 0x41, 0x5e, // POP R15,R14
	0x41, 0x5d, 0x41, 0x5c, // POP R13,R12
	0x41, 0x5b, 0x41, 0x5a, // POP R11,R10
	0x41, 0x59, 0x41, 0x58, // POP R9,R8
	0x5c, 0x5d, 0x5f, 0x5e, // POP RSP,RBP,RDI,RSI
	0x5a, 0x59, 0x5b, 0x58, // POP RDX,RCX,RBX,RAX
}

// REGS contains 32/64 bit registers
var REGS map[int][]REG

// GetRandomRegister returns a random register name based on given size and architecture
func (encoder Encoder) GetRandomRegister(size int) string {
	switch size {
	case 8:
		return REGS[encoder.architecture][rand.Intn(len(REGS[encoder.architecture]))].Low
	case 16:
		return REGS[encoder.architecture][rand.Intn(len(REGS[encoder.architecture]))].High
	case 32:
		return REGS[encoder.architecture][rand.Intn(len(REGS[encoder.architecture]))].Extended
	case 64:
		return REGS[encoder.architecture][rand.Intn(len(REGS[encoder.architecture]))].Full
	default:
		panic("invalid register size")
	}

}

// GetRandomStackAddress returns a stack address assembly referance based on the encoder architecture
// Ex: [esp+10] (address range is 1 byte)
func (encoder Encoder) GetRandomStackAddress() string {
	if CoinFlip() {
		return fmt.Sprintf("[%s+0x%x]", encoder.GetStackPointer(), GetRandomByte())
	}
	return fmt.Sprintf("[%s-0x%x]", encoder.GetStackPointer(), GetRandomByte())
}

// GetStackPointer returns the stack pointer register string based on the encoder architecture
func (encoder Encoder) GetStackPointer() string {
	switch encoder.architecture {
	case 32:
		return "ESP"
	case 64:
		return "RSP"
	default:
		panic("invalid architecture")
	}

}

// GetBasePointer returns the base pointer register string based on the encoder architecture
func (encoder Encoder) GetBasePointer() string {
	switch encoder.architecture {
	case 32:
		return "EBP"
	case 64:
		return "RBP"
	default:
		panic("invalid architecture")
	}

}

// GetSafeRandomRegister returns a random register among all (registers-excluded parameters) based on given size
func (encoder Encoder) GetSafeRandomRegister(size int, excludes ...string) (string, error) {
	regs := []REG{}
	for _, r := range REGS[encoder.architecture] {
		for _, x := range excludes {
			if r.Extended != x && r.Full != x && r.High != x && r.Low != x {
				regs = append(regs, r)
			}
		}
	}

	r := regs[rand.Intn(len(regs))]
	switch size {
	case 8:
		return r.Low, nil
	case 16:
		return r.High, nil
	case 32:
		return r.Extended, nil
	case 64:
		return r.Full, nil
	default:
		return "", errors.New("invalid register size")
	}
}

// Assemble assembes the given instructions
// and return a byte array with a boolean value indicating wether the operation is successful or not
func (encoder Encoder) Assemble(asm string) ([]byte, bool) {
	var mode keystone.Mode
	switch encoder.architecture {
	case 32:
		mode = keystone.MODE_32
	case 64:
		mode = keystone.MODE_64
	default:
		return nil, false
	}

	ks, err := keystone.NewEngine(keystone.ARCH_X86, mode)
	if err != nil {
		return nil, false
	}
	defer ks.Close()

	//err = ks.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	//err = ks.Option(keystone.OPT_SYNTAX, keystone.KS_OPT_SYNTAX_NASM)
	err = ks.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	if err != nil {
		return nil, false
	}
	//log.Println(asm)
	bin, err := ks.Assemble(asm, 0)
	return bin, err == nil
}

// GetAssemblySize assembes the given  instructions and returns the total instruction size
// if assembly fails return value is -1
func (encoder Encoder) GetAssemblySize(asm string) int {
	var mode keystone.Mode
	switch encoder.architecture {
	case 32:
		mode = keystone.MODE_32
	case 64:
		mode = keystone.MODE_64
	default:
		return -1
	}

	ks, err := keystone.NewEngine(keystone.ARCH_X86, mode)
	if err != nil {
		return -1
	}
	defer ks.Close()

	//err = ks.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	//err = ks.Option(keystone.OPT_SYNTAX, keystone.KS_OPT_SYNTAX_NASM)
	err = ks.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	if err != nil {
		return -1
	}
	//log.Println(asm)
	bin, err := ks.Assemble(asm, 0)
	if err != nil {
		return -1
	}
	return len(bin)
}

// GenerateIPToStack function generates instructions series that pushes the instruction pointer to stack
func (encoder Encoder) GenerateIPToStack() []byte {

	callBin, ok := encoder.Assemble("call 5")
	if !ok {
		panic("call 5 assembly failed")
	}
	return callBin
}

// AddCallOver function adds a call instruction over the end of the given payload
// address of the payload will be pushed to the stack and execution will continiou after the end of payload
func (encoder Encoder) AddCallOver(payload []byte) ([]byte, error) {

	// Perform a shport call over the payload
	call := fmt.Sprintf("call 0x%x", len(payload)+5)
	callBin, ok := encoder.Assemble(call)
	if !ok {
		return nil, errors.New("call-over assembly failed")
	}
	payload = append(callBin, payload...)

	return payload, nil
}

// AddJmpOver function adds a jmp instruction over the end of the given payload
// execution will continiou after the end of payload
func (encoder Encoder) AddJmpOver(payload []byte) ([]byte, error) {
	// JMP 2 -> Jumps to next instruction
	// Perform a short call over the payload
	jmp := fmt.Sprintf("jmp 0x%x", len(payload)+2)
	jmpBin, ok := encoder.Assemble(jmp)
	if !ok {
		return nil, errors.New("jmp-over assembly failed")
	}
	payload = append(jmpBin, payload...)

	return payload, nil
}

// AddCondJmpOver function adds a jmp instruction over the end of the given payload
// execution will continiou after the end of payload
func (encoder Encoder) AddCondJmpOver(payload []byte) ([]byte, error) {
	// JZ 2 -> Jumps to next instruction
	// Perform a short call over the payload

	randomConditional := ConditionalJumpMnemonics[rand.Intn(len(ConditionalJumpMnemonics))]

	jmp := fmt.Sprintf("%s 0x%x", randomConditional, len(payload)+2)
	jmpBin, ok := encoder.Assemble(jmp)
	if !ok {
		return nil, errors.New("conditional call-over assembly failed")
	}
	payload = append(jmpBin, payload...)

	return payload, nil
}
