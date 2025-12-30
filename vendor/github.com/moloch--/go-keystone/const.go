package keystone

type Arch = uint
type Mode = uint
type OptionType = uint
type OptionValue = uint
type Error = uint32

const (
	API_MAJOR = 0
	API_MINOR = 9
)

const (
	ARCH_ARM     Arch = 1
	ARCH_ARM64   Arch = 2
	ARCH_MIPS    Arch = 3
	ARCH_X86     Arch = 4
	ARCH_PPC     Arch = 5
	ARCH_SPARC   Arch = 6
	ARCH_SYSTEMZ Arch = 7
	ARCH_HEXAGON Arch = 8
	ARCH_EVM     Arch = 9
	ARCH_RISCV   Arch = 10
	ARCH_MAX     Arch = 11
)

const (
	MODE_LITTLE_ENDIAN Mode = 0
	MODE_BIG_ENDIAN    Mode = 1073741824
	MODE_ARM           Mode = 1
	MODE_THUMB         Mode = 16
	MODE_V8            Mode = 64
	MODE_MICRO         Mode = 16
	MODE_MIPS3         Mode = 32
	MODE_MIPS32R6      Mode = 64
	MODE_MIPS32        Mode = 4
	MODE_MIPS64        Mode = 8
	MODE_16            Mode = 2
	MODE_32            Mode = 4
	MODE_64            Mode = 8
	MODE_PPC32         Mode = 4
	MODE_PPC64         Mode = 8
	MODE_QPX           Mode = 16
	MODE_RISCV32       Mode = 4
	MODE_RISCV64       Mode = 8
	MODE_SPARC32       Mode = 4
	MODE_SPARC64       Mode = 8
	MODE_V9            Mode = 16
)

const (
	OPT_SYNTAX       OptionType = 1
	OPT_SYM_RESOLVER OptionType = 2
)

const (
	OPT_SYNTAX_INTEL   OptionValue = 1
	OPT_SYNTAX_ATT     OptionValue = 2
	OPT_SYNTAX_NASM    OptionValue = 4
	OPT_SYNTAX_MASM    OptionValue = 8
	OPT_SYNTAX_GAS     OptionValue = 16
	OPT_SYNTAX_RADIX16 OptionValue = 32
)

const (
	ERR_ASM_INVALID_OPERAND Error = 512
	ERR_ASM_MISSING_FEATURE Error = 513
	ERR_ASM_MNEMONIC_FAIL   Error = 514
)

const (
	ERR_ASM      Error = 128
	ERR_ASM_ARCH Error = 512
)

const (
	ERR_OK                        Error = 0
	ERR_NOMEM                     Error = 1
	ERR_ARCH                      Error = 2
	ERR_HANDLE                    Error = 3
	ERR_MODE                      Error = 4
	ERR_VERSION                   Error = 5
	ERR_OPT_INVALID               Error = 6
	ERR_ASM_EXPR_TOKEN            Error = 128
	ERR_ASM_DIRECTIVE_VALUE_RANGE Error = 129
	ERR_ASM_DIRECTIVE_ID          Error = 130
	ERR_ASM_DIRECTIVE_TOKEN       Error = 131
	ERR_ASM_DIRECTIVE_STR         Error = 132
	ERR_ASM_DIRECTIVE_COMMA       Error = 133
	ERR_ASM_DIRECTIVE_RELOC_NAME  Error = 134
	ERR_ASM_DIRECTIVE_RELOC_TOKEN Error = 135
	ERR_ASM_DIRECTIVE_FPOINT      Error = 136
	ERR_ASM_DIRECTIVE_UNKNOWN     Error = 137
	ERR_ASM_DIRECTIVE_EQU         Error = 138
	ERR_ASM_DIRECTIVE_INVALID     Error = 139
	ERR_ASM_VARIANT_INVALID       Error = 140
	ERR_ASM_EXPR_BRACKET          Error = 141
	ERR_ASM_SYMBOL_MODIFIER       Error = 142
	ERR_ASM_SYMBOL_REDEFINED      Error = 143
	ERR_ASM_SYMBOL_MISSING        Error = 144
	ERR_ASM_RPAREN                Error = 145
	ERR_ASM_STAT_TOKEN            Error = 146
	ERR_ASM_UNSUPPORTED           Error = 147
	ERR_ASM_MACRO_TOKEN           Error = 148
	ERR_ASM_MACRO_PAREN           Error = 149
	ERR_ASM_MACRO_EQU             Error = 150
	ERR_ASM_MACRO_ARGS            Error = 151
	ERR_ASM_MACRO_LEVELS_EXCEED   Error = 152
	ERR_ASM_MACRO_STR             Error = 153
	ERR_ASM_MACRO_INVALID         Error = 154
	ERR_ASM_ESC_BACKSLASH         Error = 155
	ERR_ASM_ESC_OCTAL             Error = 156
	ERR_ASM_ESC_SEQUENCE          Error = 157
	ERR_ASM_ESC_STR               Error = 158
	ERR_ASM_TOKEN_INVALID         Error = 159
	ERR_ASM_INSN_UNSUPPORTED      Error = 160
	ERR_ASM_FIXUP_INVALID         Error = 161
	ERR_ASM_LABEL_INVALID         Error = 162
	ERR_ASM_FRAGMENT_INVALID      Error = 163
	ERR_ASM_INVALIDOPERAND        Error = 512
	ERR_ASM_MISSINGFEATURE        Error = 513
	ERR_ASM_MNEMONICFAIL          Error = 514
)
