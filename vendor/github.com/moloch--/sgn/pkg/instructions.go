package sgn

// ConditionalJumpMnemonics contains the conditional branching instruction mnemonics
var ConditionalJumpMnemonics = []string{
	"JAE",
	"JA",
	"JBE",
	"JB",
	"JC",
	"JE",
	"JGE",
	"JG",
	"JLE",
	"JL",
	"JNAE",
	"JNA",
	"JNBE",
	"JNB",
	"JNC",
	"JNE",
	"JNGE",
	"JNG",
	"JNLE",
	"JNL",
	"JNO",
	"JNP",
	"JNS",
	"JNZ",
	"JO",
	"JPE",
	"JPO",
	"JP",
	"JS",
	"JZ",
}

// SafeGarbageInstructions array containing safe garbage instructions
// that does not munipulate registers or stack (do not affect the overall execution of the program)
// !!! These instructions must not clobber registers or stack flags may be affected !!!
var SafeGarbageInstructions = []string{
	";", // no instruction (empty)
	"NOP",
	"CLD",
	"CLC",
	"CMC",
	"WAIT",
	"FNOP",
	"FXAM",
	"FTST",
	"JMP 2",
	"ROL {R},0",
	"ROR {R},0",
	"SHL {R},0",
	"SHR {R},0",
	"RCL {R},0",
	"RCR {R},0",
	"SAL {R},0",
	"SAR {R},0",
	"XOR {R},0",
	"SUB {R},0",
	"ADD {R},0",
	"AND {R},{R}",
	"OR {R},{R}",
	"BT {R},{R}",
	"CMP {R},{R}",
	"MOV {R},{R}",
	"XCHG {R},{R}",
	"TEST {R},{R}",
	"CMOVA {R},{R}",
	"CMOVB {R},{R}",
	"CMOVC {R},{R}",
	"CMOVE {R},{R}",
	"CMOVG {R},{R}",
	"CMOVL {R},{R}",
	"CMOVO {R},{R}",
	"CMOVP {R},{R}",
	"CMOVS {R},{R}",
	"CMOVZ {R},{R}",
	"CMOVAE {R},{R}",
	"CMOVGE {R},{R}",
	"CMOVLE {R},{R}",
	"CMOVNA {R},{R}",
	"CMOVNB {R},{R}",
	"CMOVNC {R},{R}",
	"CMOVNE {R},{R}",
	"CMOVNG {R},{R}",
	"CMOVNL {R},{R}",
	"CMOVNO {R},{R}",
	"CMOVNP {R},{R}",
	"CMOVNS {R},{R}",
	"CMOVNZ {R},{R}",
	"CMOVPE {R},{R}",
	"CMOVPO {R},{R}",
	"CMOVBE {R},{R}",
	"CMOVNAE {R},{R}",
	"CMOVNBE {R},{R}",
	"CMOVNLE {R},{R}",
	"CMOVNGE {R},{R}",
	// Recursion starts here...
	"JMP {L};{G};{L}:",
	"NOT {R};{G};NOT {R}",
	"NEG {R};{G};NEG {R}",
	"INC {R};{G};DEC {R}",
	"DEC {R};{G};INC {R}",
	// "PUSH {R};{G};POP {R}",
	// "BSWAP {R};{G};BSWAP {R}",
	"ADD {R},{K};{G};SUB {R},{K}",
	"SUB {R},{K};{G};ADD {R},{K}",
	"ROR {R},{K};{G};ROL {R},{K}",
	"ROL {R},{K};{G};ROR {R},{K}",
}

// SupportedOperandTypes contains all operand types supported by SGN
var SupportedOperandTypes = []string{
	"imm8",
	"imm16",
	"imm32",
	"imm64",
	"r8",
	"r16",
	"r32",
	"r64",
	"r/m8",
	"r/m16",
	"r/m32",
	"r/m64",
	"m",
	"m8",
	"m16",
	"m32",
	"m64",
	"RAX",
	"RCX",
	"RDX",
	"RBX",
	"RSP",
	"RBP",
	"RSI",
	"RDI",
	"EAX",
	"ECX",
	"EDX",
	"EBX",
	"ESP",
	"EBP",
	"ESI",
	"EDI",
	"AX",
	"CX",
	"DX",
	"BX",
	"SP",
	"BP",
	"SI",
	"DI",
	"AH",
	"AL",
	"CH",
	"CL",
	"DH",
	"DL",
	"BH",
	"BL",
	"SPL",
	"BPL",
	"SIL",
	"DIL",
}

// JMP 2 -> Jumps to next instruction
// func addGarbageJumpMnemonics() {
// 	// for _, i := range ConditionalJumpMnemonics {
// 	// 	GarbageMnemonics = append(GarbageMnemonics, i+" 2")
// 	// }

// 	for _, i := range ConditionalJumpMnemonics {
// 		GarbageMnemonics = append(GarbageMnemonics, i+" {L};{G};{L}:")
// 	}
// }

/*
  !! BLACKLISTED INSTRUCTIONS !!

  * XCHG
  * UD1
  * CMPXCHG
  * CMPXCHG8B

*/

// INSTRUCTIONS contains the ENTIRE x86/x64 instruction set
const INSTRUCTIONS string = `


[
  {
    "Mnemonic": "AAD",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "AAM",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ADC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ADCX",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ADD",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ADOX",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "AND",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BLSI",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BLSMSK",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BLSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BSF",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BSWAP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "imm8",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BTC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "imm8",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BTR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "imm8",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "BTS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "imm8",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CALL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CLFLUSH",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CLFLUSHOPT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CLWB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVAE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVBE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVGE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVLE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNAE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNBE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNGE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNLE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNO",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVNZ",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVO",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMOVPE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CMPS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      },
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "CRC32",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r32",
          "r32",
          "r32",
          "r32",
          "r64",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m8",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "DEC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r16",
          "r32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "DIV",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ENTER",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm16"
        ]
      },
      {
        "Types": [
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "FSTSW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AX"
        ]
      }
    ]
  },
  {
    "Mnemonic": "FNSTSW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AX"
        ]
      }
    ]
  },
  {
    "Mnemonic": "IDIV",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "IMUL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "INC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r16",
          "r32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "INS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32"
        ]
      },
      {
        "Types": [
          "DX",
          "DX",
          "DX"
        ]
      }
    ]
  },
  {
    "Mnemonic": "INVLPG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m"
        ]
      }
    ]
  },
  {
    "Mnemonic": "JMP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "LDMXCSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "VLDMXCSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "LEA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "m",
          "m",
          "m"
        ]
      }
    ]
  },
  {
    "Mnemonic": "LMSW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "LODS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "LZCNT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOV",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm64",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOVBE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "m16",
          "m32",
          "m64"
        ]
      },
      {
        "Types": [
          "m16",
          "m32",
          "m64",
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOVNTI",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m32",
          "m64"
        ]
      },
      {
        "Types": [
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOVS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      },
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOVSX",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MOVZX",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "MUL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "NEG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "NOP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "NOT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "OR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "OUT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm8",
          "imm8",
          "imm8",
          "DX",
          "DX",
          "DX"
        ]
      },
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "AL",
          "AX",
          "EAX"
        ]
      }
    ]
  },
  {
    "Mnemonic": "OUTS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "DX",
          "DX",
          "DX"
        ]
      },
      {
        "Types": [
          "m8",
          "m16",
          "m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "POP",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "POPCNT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHT0",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHT1",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHT2",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHNTA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PREFETCHWT1",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "PUSH",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64",
          "r16",
          "r32",
          "r64",
          "imm8",
          "imm16",
          "imm32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "RCL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "RCR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ROL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "ROR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "RDRAND",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "RDSEED",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "RET",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm16",
          "imm16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SAL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SAR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SHL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SHR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "CL",
          "CL",
          "imm8",
          "imm8",
          "CL",
          "imm8",
          "CL",
          "CL",
          "imm8",
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SBB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SCAS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETAE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETBE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETGE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETLE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNA",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNAE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNBE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNC",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNG",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNGE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNL",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SETNLE",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SGDT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SIDT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SLDT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SMSW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "STMXCSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "VSTMXCSR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m32"
        ]
      }
    ]
  },
  {
    "Mnemonic": "STOS",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8",
          "m16",
          "m32",
          "m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "STR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "SUB",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "TEST",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "TZCNT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "VERR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "VERW",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m16"
        ]
      }
    ]
  },
  {
    "Mnemonic": "XABORT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "imm8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "XADD",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      },
      {
        "Types": [
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      }
    ]
  },
  {
    "Mnemonic": "XLAT",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "m8"
        ]
      }
    ]
  },
  {
    "Mnemonic": "XOR",
    "V64": true,
    "V32": true,
    "Operands": [
      {
        "Types": [
          "AL",
          "AX",
          "EAX",
          "RAX",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m16",
          "r/m32",
          "r/m64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64"
        ]
      },
      {
        "Types": [
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm16",
          "imm32",
          "imm32",
          "imm8",
          "imm8",
          "imm8",
          "r8",
          "r8",
          "r16",
          "r32",
          "r64",
          "r/m8",
          "r/m8",
          "r/m16",
          "r/m32",
          "r/m64"
        ]
      }
    ]
  }
]



`
