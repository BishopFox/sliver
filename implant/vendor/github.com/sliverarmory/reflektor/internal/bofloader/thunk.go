package bofloader

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
)

func writePointer(destination []byte, value uintptr) {
	switch pointerSize() {
	case 4:
		binary.LittleEndian.PutUint32(destination, uint32(value))
	case 8:
		binary.LittleEndian.PutUint64(destination, uint64(value))
	default:
		panic("bofloader: unsupported pointer size")
	}
}

func writeThunk(destination []byte, target, thunk, got, toc uintptr) error {
	if len(destination) < 16 {
		return errors.New("import thunk requires a 16-byte destination")
	}
	if runtime.GOARCH != "ppc64le" {
		clear(destination[:16])
	}
	switch runtime.GOARCH {
	case "386":
		// MOV EAX, imm32; JMP EAX.
		destination[0] = 0xb8
		binary.LittleEndian.PutUint32(destination[1:5], uint32(target))
		destination[5] = 0xff
		destination[6] = 0xe0
	case "amd64":
		// JMP QWORD PTR [RIP+0], followed by the absolute destination.
		copy(destination[:6], []byte{0xff, 0x25, 0x00, 0x00, 0x00, 0x00})
		binary.LittleEndian.PutUint64(destination[6:14], uint64(target))
	case "arm":
		writeARMThunk(destination, uint32(target))
	case "arm64":
		// LDR X16, #8; BR X16; followed by the absolute destination.
		binary.LittleEndian.PutUint32(destination[0:4], 0x58000050)
		binary.LittleEndian.PutUint32(destination[4:8], 0xd61f0200)
		binary.LittleEndian.PutUint64(destination[8:16], uint64(target))
	case "riscv64":
		return writeRISCV64Thunk(destination, thunk, got)
	case "ppc64le":
		return writePPC64LEThunk(destination, got, toc)
	default:
		return fmt.Errorf("import thunks are unsupported on %s", runtime.GOARCH)
	}
	return nil
}

func writePPC64LEThunk(destination []byte, got, toc uintptr) error {
	if len(destination) < 32 {
		return errors.New("PPC64 ELFv2 import thunk requires a 32-byte destination")
	}
	displacement, err := signedDifference(uint64(got), uint64(toc))
	if err != nil {
		return fmt.Errorf("PPC64 ELFv2 GOT-relative import thunk: %w", err)
	}
	high, low, err := ppc64SplitAddress(displacement, true)
	if err != nil {
		return fmt.Errorf("PPC64 ELFv2 GOT displacement %d: %w", displacement, err)
	}
	clear(destination[:32])
	// Save the BOF's TOC in its caller-provided save slot, load the external
	// global entry through the loader GOT, and enter it with r12 set as ELFv2
	// requires. The call-site relocation replaces the following linker NOP with
	// ld r2,24(r1), restoring the BOF TOC after the callback returns.
	binary.LittleEndian.PutUint32(destination[0:4], 0xf8410018)                             // std r2,24(r1)
	binary.LittleEndian.PutUint32(destination[4:8], 0x3d820000|uint32(uint16(high)))        // addis r12,r2,ha(d)
	binary.LittleEndian.PutUint32(destination[8:12], 0xe98c0000|uint32(uint16(low))&0xfffc) // ld r12,lo(d)(r12)
	binary.LittleEndian.PutUint32(destination[12:16], 0x7d8903a6)                           // mtctr r12
	binary.LittleEndian.PutUint32(destination[16:20], 0x4e800420)                           // bctr
	return nil
}

func writeRISCV64Thunk(destination []byte, thunk, got uintptr) error {
	if len(destination) < 16 {
		return errors.New("RISC-V import thunk requires a 16-byte destination")
	}
	high, low, err := riscvPCRelativeParts(uint64(got), 0, uint64(thunk))
	if err != nil {
		return fmt.Errorf("RISC-V GOT-relative import thunk: %w", err)
	}
	// AUIPC t0, hi20(GOT-thunk); LD t0, lo12(t0); JALR zero, 0(t0); NOP.
	// The GOT slot contains the unrestricted 64-bit callback address, while
	// the PC-relative instruction pair only spans loader-owned nearby memory.
	binary.LittleEndian.PutUint32(destination[0:4], 0x00000297|((uint32(high)&0xfffff)<<12))
	binary.LittleEndian.PutUint32(destination[4:8], 0x0002b283|((uint32(low)&0xfff)<<20))
	binary.LittleEndian.PutUint32(destination[8:12], 0x00028067)
	binary.LittleEndian.PutUint32(destination[12:16], 0x00000013)
	return nil
}

func writeARMThunk(destination []byte, target uint32) {
	// LDR IP, [PC]; BX IP; followed by the absolute destination. BX preserves
	// ARM/Thumb interworking when the resolved function has its low bit set.
	binary.LittleEndian.PutUint32(destination[0:4], 0xe59fc000)
	binary.LittleEndian.PutUint32(destination[4:8], 0xe12fff1c)
	binary.LittleEndian.PutUint32(destination[8:12], target)
}
