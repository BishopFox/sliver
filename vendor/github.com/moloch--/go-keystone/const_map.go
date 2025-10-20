package keystone

import (
	"sort"
	"strings"
)

var archM = map[string]Arch{
	"arm":     ARCH_ARM,
	"arm64":   ARCH_ARM64,
	"mips":    ARCH_MIPS,
	"x86":     ARCH_X86,
	"ppc":     ARCH_PPC,
	"sparc":   ARCH_SPARC,
	"systemz": ARCH_SYSTEMZ,
	"hexagon": ARCH_HEXAGON,
	"evm":     ARCH_EVM,
	"riscv":   ARCH_RISCV,
	"max":     ARCH_MAX,
}

var modeM = map[string]Mode{
	"le":       MODE_LITTLE_ENDIAN,
	"be":       MODE_BIG_ENDIAN,
	"arm":      MODE_ARM,
	"thumb":    MODE_THUMB,
	"v8":       MODE_V8,
	"micro":    MODE_MICRO,
	"mips3":    MODE_MIPS3,
	"mips32r6": MODE_MIPS32R6,
	"mips32":   MODE_MIPS32,
	"mips64":   MODE_MIPS64,
	"16":       MODE_16,
	"32":       MODE_32,
	"64":       MODE_64,
	"ppc32":    MODE_PPC32,
	"ppc64":    MODE_PPC64,
	"qpx":      MODE_QPX,
	"riscv32":  MODE_RISCV32,
	"riscv64":  MODE_RISCV64,
	"sparc32":  MODE_SPARC32,
	"sparc64":  MODE_SPARC64,
	"v9":       MODE_V9,
}

var syntaxM = map[string]OptionValue{
	"intel":   OPT_SYNTAX_INTEL,
	"att":     OPT_SYNTAX_ATT,
	"nasm":    OPT_SYNTAX_NASM,
	"masm":    OPT_SYNTAX_MASM,
	"gas":     OPT_SYNTAX_GAS,
	"radix16": OPT_SYNTAX_RADIX16,
}

// StringToArch is used to convert string to arch.
func StringToArch(arch string) Arch {
	return archM[strings.ToLower(arch)]
}

// StringToMode is used to convert string to mode.
func StringToMode(mode string) Mode {
	return modeM[strings.ToLower(mode)]
}

// StringToSyntax is used to convert string to syntax.
func StringToSyntax(syntax string) OptionValue {
	return syntaxM[strings.ToLower(syntax)]
}

// ArchOptions returns the list of supported architecture keywords.
func ArchOptions() []string {
	keys := make([]string, 0, len(archM))
	for key := range archM {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ModeOptions returns the list of supported mode keywords.
func ModeOptions() []string {
	keys := make([]string, 0, len(modeM))
	for key := range modeM {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// SyntaxOptions returns the list of supported syntax keywords.
func SyntaxOptions() []string {
	keys := make([]string, 0, len(syntaxM))
	for key := range syntaxM {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
