package bofloader

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
)

type riscvRelocationSite struct {
	section int
	offset  uint64
}

type riscvRelocationPair struct {
	typeID uint32
	symbol uint32
	addend int64
	offset uint64
}

// prepareELFRISCV64Pairs resolves every LO12 relocation through the local
// label naming its companion HI20 relocation. Doing this once at parse time
// makes pairing independent of relocation-table order and avoids an O(n^2)
// scan over attacker-controlled relocation counts while loading.
func prepareELFRISCV64Pairs(object *objectFile) error {
	if object == nil {
		return fmt.Errorf("bofloader: prepare RISC-V pairs for nil object")
	}
	highs := make(map[riscvRelocationSite]objectRelocation)
	for _, relocation := range object.relocations {
		switch elf.R_RISCV(relocation.typeID) {
		case elf.R_RISCV_PCREL_HI20, elf.R_RISCV_GOT_HI20:
			if !relocation.hasAdd {
				return fmt.Errorf("bofloader: ELF/riscv64 HI20 relocation at %#x must use RELA encoding", relocation.offset)
			}
			if elf.R_RISCV(relocation.typeID) == elf.R_RISCV_GOT_HI20 && relocation.addend != 0 {
				return fmt.Errorf("bofloader: ELF/riscv64 GOT_HI20 relocation at %#x must have a zero addend, got %d", relocation.offset, relocation.addend)
			}
			site := riscvRelocationSite{section: relocation.section, offset: relocation.offset}
			if _, exists := highs[site]; exists {
				return fmt.Errorf("bofloader: ELF/riscv64 has multiple HI20 relocations in section %d at %#x", relocation.section, relocation.offset)
			}
			highs[site] = relocation
		}
	}

	pairs := make(map[riscvRelocationSite]riscvRelocationPair)
	for _, relocation := range object.relocations {
		typeID := elf.R_RISCV(relocation.typeID)
		if typeID != elf.R_RISCV_PCREL_LO12_I && typeID != elf.R_RISCV_PCREL_LO12_S {
			continue
		}
		if !relocation.hasAdd {
			return fmt.Errorf("bofloader: ELF/riscv64 LO12 relocation at %#x must use RELA encoding", relocation.offset)
		}
		if relocation.addend != 0 {
			return fmt.Errorf("bofloader: ELF/riscv64 LO12 relocation at %#x must have a zero addend, got %d", relocation.offset, relocation.addend)
		}
		label, ok := object.symbols[relocation.symbol]
		if !ok {
			return fmt.Errorf("bofloader: ELF/riscv64 LO12 relocation at %#x references missing pair label %d", relocation.offset, relocation.symbol)
		}
		if label.section < 0 || label.section != relocation.section {
			return fmt.Errorf("bofloader: ELF/riscv64 LO12 relocation at %#x pair label %q is not in its target section", relocation.offset, label.name)
		}
		highOffset := label.value
		high, ok := highs[riscvRelocationSite{section: label.section, offset: highOffset}]
		if !ok {
			return fmt.Errorf("bofloader: ELF/riscv64 LO12 relocation at %#x has no HI20 pair at %#x", relocation.offset, highOffset)
		}
		site := riscvRelocationSite{section: relocation.section, offset: relocation.offset}
		if _, exists := pairs[site]; exists {
			return fmt.Errorf("bofloader: ELF/riscv64 has multiple LO12 relocations in section %d at %#x", relocation.section, relocation.offset)
		}
		pairs[site] = riscvRelocationPair{
			typeID: high.typeID,
			symbol: high.symbol,
			addend: high.addend,
			offset: high.offset,
		}
	}
	object.riscvPairs = pairs
	return nil
}

func applyELFRISCV64Relocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol, externals map[uint32]externalSymbol) error {
	if !relocation.hasAdd {
		return fmt.Errorf("RISC-V relocations require RELA encoding")
	}
	switch elf.R_RISCV(relocation.typeID) {
	case elf.R_RISCV_64:
		value, err := addSigned(directLinkedAddress(linked), relocation.addend)
		if err != nil {
			return err
		}
		putUint64(location, value)
		return nil
	case elf.R_RISCV_CALL, elf.R_RISCV_CALL_PLT:
		if linked.external != nil && relocation.addend != 0 {
			return fmt.Errorf("RISC-V external CALL relocation must have a zero addend, got %d", relocation.addend)
		}
		return applyRISCVCall(location, thunkLinkedAddress(linked), place, relocation.addend)
	case elf.R_RISCV_PCREL_HI20:
		return applyRISCVHigh20(location, directLinkedAddress(linked), place, relocation.addend)
	case elf.R_RISCV_GOT_HI20:
		if relocation.addend != 0 {
			return fmt.Errorf("RISC-V GOT_HI20 relocation must have a zero addend, got %d", relocation.addend)
		}
		got, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		if got&7 != 0 {
			return fmt.Errorf("RISC-V GOT entry address %#x is not 8-byte aligned", got)
		}
		return applyRISCVHigh20(location, got, place, relocation.addend)
	case elf.R_RISCV_PCREL_LO12_I, elf.R_RISCV_PCREL_LO12_S:
		return applyRISCVLow12(object, relocation, location, externals)
	default:
		return fmt.Errorf("unsupported ELF/riscv64 relocation")
	}
}

func applyRISCVCall(location []byte, target, place uint64, addend int64) error {
	if len(location) != 8 {
		return fmt.Errorf("RISC-V CALL relocation requires an 8-byte instruction pair")
	}
	auipc := binary.LittleEndian.Uint32(location[0:4])
	jalr := binary.LittleEndian.Uint32(location[4:8])
	baseRegister, err := riscvAUIPCRegister(auipc)
	if err != nil {
		return err
	}
	if jalr&0x707f != 0x0067 {
		return riscvInstructionClassError("JALR", jalr)
	}
	if source := (jalr >> 15) & 0x1f; source != baseRegister {
		return fmt.Errorf("RISC-V CALL JALR base register x%d does not match AUIPC destination x%d", source, baseRegister)
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta&1 != 0 {
		return fmt.Errorf("RISC-V call displacement %d is not 2-byte aligned", delta)
	}
	high, low, err := riscvSplitPCRelative(delta)
	if err != nil {
		return fmt.Errorf("RISC-V call displacement %d: %w", delta, err)
	}
	auipc = (auipc & 0x00000fff) | ((uint32(high) & 0xfffff) << 12)
	jalr = (jalr & 0x000fffff) | ((uint32(low) & 0xfff) << 20)
	binary.LittleEndian.PutUint32(location[0:4], auipc)
	binary.LittleEndian.PutUint32(location[4:8], jalr)
	return nil
}

func applyRISCVHigh20(location []byte, target, place uint64, addend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if _, err := riscvAUIPCRegister(word); err != nil {
		return err
	}
	high, _, err := riscvPCRelativeParts(target, addend, place)
	if err != nil {
		return err
	}
	word = (word & 0x00000fff) | ((uint32(high) & 0xfffff) << 12)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyRISCVLow12(object *objectFile, relocation objectRelocation, location []byte, externals map[uint32]externalSymbol) error {
	if object == nil {
		return fmt.Errorf("RISC-V LO12 relocation has no object")
	}
	pair, ok := object.riscvPairs[riscvRelocationSite{section: relocation.section, offset: relocation.offset}]
	if !ok {
		return fmt.Errorf("RISC-V LO12 relocation has no validated HI20 pair")
	}
	if relocation.section < 0 || relocation.section >= len(object.sections) {
		return fmt.Errorf("RISC-V LO12 relocation target section is invalid")
	}
	section := object.sections[relocation.section]
	if pair.offset > uint64(len(section.data)) || 4 > uint64(len(section.data))-pair.offset {
		return fmt.Errorf("RISC-V HI20 pair at %#x is outside initialized section data", pair.offset)
	}
	highWord := binary.LittleEndian.Uint32(section.data[pair.offset : pair.offset+4])
	baseRegister, err := riscvAUIPCRegister(highWord)
	if err != nil {
		return fmt.Errorf("RISC-V HI20 pair at %#x: %w", pair.offset, err)
	}
	paired, err := linkRelocationSymbol(object, objectRelocation{symbol: pair.symbol}, externals)
	if err != nil {
		return fmt.Errorf("RISC-V HI20 pair symbol: %w", err)
	}
	var target uint64
	switch elf.R_RISCV(pair.typeID) {
	case elf.R_RISCV_PCREL_HI20:
		target = directLinkedAddress(paired)
	case elf.R_RISCV_GOT_HI20:
		if pair.addend != 0 {
			return fmt.Errorf("RISC-V GOT_HI20 pair must have a zero addend, got %d", pair.addend)
		}
		target, err = gotLinkedAddress(paired)
		if err != nil {
			return err
		}
		if target&7 != 0 {
			return fmt.Errorf("RISC-V GOT entry address %#x is not 8-byte aligned", target)
		}
	default:
		return fmt.Errorf("RISC-V LO12 relocation has unsupported HI20 pair type %d", pair.typeID)
	}
	highPlace, ok := checkedAddUint64(uint64(section.address), pair.offset)
	if !ok {
		return fmt.Errorf("RISC-V HI20 pair address overflows")
	}
	_, low, err := riscvPCRelativeParts(target, pair.addend, highPlace)
	if err != nil {
		return err
	}
	word := binary.LittleEndian.Uint32(location)
	switch elf.R_RISCV(relocation.typeID) {
	case elf.R_RISCV_PCREL_LO12_I:
		if !riscvITypeOpcode(word & 0x7f) {
			return riscvInstructionClassError("I-type immediate", word)
		}
		if source := (word >> 15) & 0x1f; source != baseRegister {
			return fmt.Errorf("RISC-V LO12_I base register x%d does not match AUIPC destination x%d", source, baseRegister)
		}
		word = (word & 0x000fffff) | ((uint32(low) & 0xfff) << 20)
	case elf.R_RISCV_PCREL_LO12_S:
		if !riscvSTypeOpcode(word & 0x7f) {
			return riscvInstructionClassError("S-type immediate", word)
		}
		if source := (word >> 15) & 0x1f; source != baseRegister {
			return fmt.Errorf("RISC-V LO12_S base register x%d does not match AUIPC destination x%d", source, baseRegister)
		}
		immediate := uint32(low) & 0xfff
		word = (word &^ uint32(0xfe000f80)) | ((immediate & 0xfe0) << 20) | ((immediate & 0x1f) << 7)
	default:
		return fmt.Errorf("unsupported RISC-V low relocation")
	}
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func riscvPCRelativeParts(target uint64, addend int64, place uint64) (int64, int64, error) {
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return 0, 0, err
	}
	return riscvSplitPCRelative(delta)
}

func riscvSplitPCRelative(delta int64) (int64, int64, error) {
	// Start with an arithmetic floor division, then move values whose low
	// twelve bits would be negative into the canonical signed LO12 range.
	high := delta >> 12
	low := delta - (high << 12)
	if low >= 1<<11 {
		high++
		low -= 1 << 12
	}
	if high < -(1<<19) || high > (1<<19)-1 {
		return 0, 0, fmt.Errorf("displacement exceeds signed AUIPC/LO12 range")
	}
	return high, low, nil
}

func riscvAUIPCRegister(word uint32) (uint32, error) {
	if word&0x7f != 0x17 {
		return 0, riscvInstructionClassError("AUIPC", word)
	}
	register := (word >> 7) & 0x1f
	if register == 0 {
		return 0, fmt.Errorf("RISC-V AUIPC destination register cannot be x0")
	}
	return register, nil
}

func riscvITypeOpcode(opcode uint32) bool {
	switch opcode {
	case 0x03, 0x07, 0x13, 0x1b, 0x67:
		return true
	default:
		return false
	}
}

func riscvSTypeOpcode(opcode uint32) bool {
	return opcode == 0x23 || opcode == 0x27
}

func riscvInstructionClassError(expected string, word uint32) error {
	return fmt.Errorf("RISC-V relocation expected %s instruction, found %#08x", expected, word)
}
