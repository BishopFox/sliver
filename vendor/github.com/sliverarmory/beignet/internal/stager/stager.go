package stager

import (
	"debug/macho"
	"fmt"
)

// LoaderText returns the embedded loader image bytes for the requested Mach-O
// CPU architecture and the entry function offset relative to the start of the
// image.
func LoaderText(cpu macho.Cpu) ([]byte, uint64, error) {
	switch cpu {
	case macho.CpuArm64:
		text, entryOff := loaderTextDarwinArm64()
		return text, entryOff, nil
	case macho.CpuAmd64:
		text, entryOff := loaderTextDarwinAMD64()
		return text, entryOff, nil
	default:
		return nil, 0, fmt.Errorf("stager: unsupported cpu %s", cpu)
	}
}
