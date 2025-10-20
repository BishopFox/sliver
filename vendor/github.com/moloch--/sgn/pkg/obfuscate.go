package sgn

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
)

// SGN ASM Label Definitions;
//-----------------------------
// {R} 	= RANDOM GENERAL PURPOSE REGISTER
// {K} 	= RANDOM BYTE OF DATA
// {L} 	= RANDOM ASM LABEL
// {G} 	= RANDOM GARBAGE ASSEMBLY

// GenerateGarbageAssembly generates random garbage instruction(s) assemblies
// based on the subject encoder architecture
func (encoder *Encoder) GenerateGarbageAssembly() string {
	if CoinFlip() {
		randomGarbageAssembly := GetRandomSafeAssembly()
		register := encoder.GetRandomRegister(encoder.architecture)
		randomGarbageAssembly = strings.ReplaceAll(randomGarbageAssembly, "{R}", register)
		randomGarbageAssembly = strings.ReplaceAll(randomGarbageAssembly, "{K}", fmt.Sprintf("0x%x", GetRandomByte()))
		randomGarbageAssembly = strings.ReplaceAll(randomGarbageAssembly, "{L}", RandomLabel())
		randomGarbageAssembly = strings.ReplaceAll(randomGarbageAssembly, "{G}", encoder.GenerateGarbageAssembly())
		return randomGarbageAssembly + ";"
	}
	return ";"
}

// GenerateGarbageInstructions generates random garbage instruction(s)
// with the specified architecture and returns the assembled bytes
func (encoder *Encoder) GenerateGarbageInstructions() ([]byte, error) {

	randomGarbageAssembly := encoder.GenerateGarbageAssembly()
	garbage, ok := encoder.Assemble(randomGarbageAssembly)
	if !ok {
		//fmt.Println(randomGarbageAssembly)
		return nil, errors.New("random garbage instruction assembly failed")
	}

	if CoinFlip() {
		garbageJmp, err := encoder.GenerateGarbageJump()
		if err != nil {
			return nil, err
		}
		if CoinFlip() {
			garbage = append(garbageJmp, garbage...)
		} else {
			garbage = append(garbage, garbageJmp...)
		}
	}

	if len(garbage) <= encoder.ObfuscationLimit {
		//fmt.Println(randomGarbageAssembly)
		return garbage, nil
	}

	return encoder.GenerateGarbageInstructions()
}

// GetRandomSafeAssembly return a safe garbage instruction assembly
func GetRandomSafeAssembly() string {

	newSafeGarbageInstructions := SafeGarbageInstructions
	// Add garbage confditional jumps for more possibility
	for _, jmp := range ConditionalJumpMnemonics {
		newSafeGarbageInstructions = append(newSafeGarbageInstructions, jmp+" {L};{G};{L}:")
		//newSafeGarbageInstructions = append(newSafeGarbageInstructions, jmp+" 2")
	}
	return newSafeGarbageInstructions[rand.Intn(len(SafeGarbageInstructions))]
}

// GetRandomUnsafeAssembly return a safe garbage instruction assembly
func (encoder *Encoder) GetRandomUnsafeAssembly(destReg string) string {

	// Random register size between 8-16-32-64
	randRegSize := int(math.Pow(2, float64(rand.Intn(3+(encoder.architecture/64))+3)))
	subReg := ""
	for _, i := range REGS[encoder.architecture] {
		if (encoder.architecture == 32 && i.Extended == destReg) || (encoder.architecture == 64 && i.Full == destReg) {
			switch randRegSize {
			case 8:
				subReg = i.Low
			case 16:
				subReg = i.High
			case 32:
				subReg = i.Extended
			case 64:
				subReg = i.Full
			}
		}
	}

	if subReg == "" {
		panic("invalid register selected")
	}

	// Add first unsafe garbage.
	newUnsafeMnemonic := encoder.GetRandomUnsafeMnemonic(randRegSize)
	// Generate a random operand value based on a random oprerand type of the selected instruction
	operand := encoder.GetRandomOperandValue(newUnsafeMnemonic.GetRandomMatchingOperandType(randRegSize))
	unsafeGarbageAssembly := fmt.Sprintf("%s %s,%s;", newUnsafeMnemonic.Mnemonic, subReg, operand)

	return unsafeGarbageAssembly
}

// GetRandomUnsafeMnemonic returns a random unsafe instruction based on the encoder architecture and operand number/type
// Currently SGN only supports instructions with 2 parameter
func (encoder *Encoder) GetRandomUnsafeMnemonic(opRegSize int) *INSTRUCTION {

	// UnsafeInstructions contains instructions that manipulate certain registers/stack/memory
	var UnsafeInstructions []INSTRUCTION

	err := json.NewDecoder(strings.NewReader(INSTRUCTIONS)).Decode(&UnsafeInstructions)
	if err != nil {
		panic(err)
	}

	new := UnsafeInstructions[rand.Intn(len(UnsafeInstructions))]
	if ((new.V32 && encoder.GetArchitecture() == 32) || (new.V64 && encoder.GetArchitecture() == 64)) && len(new.Operands) == 2 {
		if include(new.Operands[0].Types, fmt.Sprintf("r/m%d", opRegSize)) || include(new.Operands[0].Types, fmt.Sprintf("r%d", opRegSize)) {
			// for _, ope := range new.Operands[1].Types {
			// 	// Mnemonic operand types include one of the unsupported types abot !
			// 	// because it may be the only combination for valid assembly
			// 	if !include(SupportedOperandTypes, ope) {
			// 		return encoder.GetRandomUnsafeMnemonic(opRegSize)
			// 	}
			// }
			return &new

		}
	}
	return encoder.GetRandomUnsafeMnemonic(opRegSize)

}

// GetRandomOperandValue generates a instruction parameter value based on given operand type
// Only some operand types are considered because SGN only uses 32-64 bit registers
func (encoder *Encoder) GetRandomOperandValue(operandType string) string {

	switch operandType {
	case "imm8":
		return fmt.Sprintf("0x%x", GetRandomByte()%127)
	case "imm16":
		return fmt.Sprintf("0x%x", rand.Intn(32767))
	case "imm32":
		return fmt.Sprintf("0x%x", rand.Int31n((2147483647)))
	case "imm64":
		return fmt.Sprintf("0x%x", GetRandomBytes(8))
	case "r8":
		return encoder.GetRandomRegister(8)
	case "r16":
		return encoder.GetRandomRegister(16)
	case "r32":
		return encoder.GetRandomRegister(32)
	case "r64":
		return encoder.GetRandomRegister(64)
	case "r/m8":
		if CoinFlip() {
			return encoder.GetRandomOperandValue("m8")
		}
		return encoder.GetRandomRegister(8)
	case "r/m16":
		if CoinFlip() {
			return encoder.GetRandomOperandValue("m16")
		}
		return encoder.GetRandomRegister(16)
	case "r/m32":
		if CoinFlip() {
			return encoder.GetRandomOperandValue("m32")
		}
		return encoder.GetRandomRegister(32)
	case "r/m64":
		if CoinFlip() {
			return encoder.GetRandomOperandValue("m64")
		}
		return encoder.GetRandomRegister(64)
	case "m":
		return encoder.GetRandomStackAddress()
	case "m8":
		return fmt.Sprintf("BYTE PTR %s", encoder.GetRandomStackAddress())
	case "m16":
		return fmt.Sprintf("WORD PTR %s", encoder.GetRandomStackAddress())
	case "m32":
		return fmt.Sprintf("DWORD PTR %s", encoder.GetRandomStackAddress())
	case "m64":
		return fmt.Sprintf("QWORD PTR %s", encoder.GetRandomStackAddress())
	case "RAX", "RCX", "RDX", "RBX", "RSP", "RBP", "RSI", "RDI", "EAX", "ECX", "EDX", "EBX", "ESP", "EBP", "ESI", "EDI", "AX", "CX", "DX", "BX", "SP", "BP", "SI", "DI", "AH", "AL", "CH", "CL", "DH", "DL", "BH", "BL", "SPL", "BPL", "SIL", "DIL":
		return operandType
	default:
		panic("unsupported instruction operand type: " + operandType)
	}
}

// GetRandomMatchingOperandType randomly selects a operand type for subject instruction
func (ins *INSTRUCTION) GetRandomMatchingOperandType(srcRegSize int) string {
	if len(ins.Operands) != 2 {
		panic(errors.New("instruction operand index out of range"))
	}
	if len(ins.Operands[0].Types) == 0 || len(ins.Operands[1].Types) == 0 {
		panic(errors.New("instruction operand has no type"))
	}
	if len(ins.Operands[0].Types) != len(ins.Operands[1].Types) {
		panic(errors.New("unsupported instruction operand types"))
	}

	index := []int{}

	for i, j := range ins.Operands[0].Types {
		if j == fmt.Sprintf("r/m%d", srcRegSize) || j == fmt.Sprintf("r%d", srcRegSize) {
			index = append(index, i)
		}
	}

	return ins.Operands[1].Types[index[rand.Intn(len(index))]]
}

func include(arr []string, str string) bool {
	for _, i := range arr {
		if i == str {
			return true
		}
	}
	return false
}

// CalculateAverageGarbageInstructionSize calculate the avarage size of generated random garbage instructions
func (encoder *Encoder) CalculateAverageGarbageInstructionSize() (float64, error) {

	var average float64 = 0
	for i := 0; i < 100; i++ {
		randomGarbageAssembly := encoder.GenerateGarbageAssembly()
		garbage, ok := encoder.Assemble(randomGarbageAssembly)
		if !ok {
			return 0, errors.New("random garbage instruction assembly failed")
		}
		average += float64(len(garbage))
	}
	average = average / 100
	return average, nil
}

func (encoder *Encoder) debugAssembly(str string) string {
	for _, i := range strings.Split(str, ";") {
		_, ok := encoder.Assemble(i)
		if !ok {
			return i
		}
	}
	return ""
}

// GetRandomFunctionAssembly generates a function frame assembly with garbage instructions inside
func (encoder *Encoder) GetRandomFunctionAssembly() string {

	bp := ""
	sp := ""

	switch encoder.architecture {
	case 32:
		bp = "EBP"
		sp = "ESP"
	case 64:
		bp = "RBP"
		sp = "RSP"
	default:
		panic(errors.New("invalid architecture selected"))
	}

	prologue := fmt.Sprintf("PUSH %s;", bp)
	prologue += fmt.Sprintf("MOV %s,%s;", bp, sp)
	prologue += fmt.Sprintf("SUB %s,0x%x;", sp, GetRandomByte())

	// Fill the function body with garbage instructions
	garbage := encoder.GenerateGarbageAssembly()

	epilogue := fmt.Sprintf("MOV %s,%s;", sp, bp)
	epilogue += fmt.Sprintf("POP %s;", bp)

	return prologue + garbage + epilogue
}

// GenerateGarbageJump generates a JMP instruction over random bytes
func (encoder Encoder) GenerateGarbageJump() ([]byte, error) {
	GetRandomBytes := GetRandomBytes(encoder.ObfuscationLimit / 10)
	garbageJmp, err := encoder.AddJmpOver(GetRandomBytes)
	if err != nil {
		return nil, err
	}
	return garbageJmp, nil
}

// RandomLabel generates a random assembly label
func RandomLabel() string {
	numbers := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 5)
	for i := range b {
		b[i] = numbers[rand.Intn(len(numbers))]
	}
	return string(b)
}
