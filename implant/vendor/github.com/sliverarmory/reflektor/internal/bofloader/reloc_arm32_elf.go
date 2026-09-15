package bofloader

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"math"
)

func applyELFARMRelocation(relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	typeID := elf.R_ARM(relocation.typeID)
	switch typeID {
	case elf.R_ARM_ABS32:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		value, err := addSigned(directLinkedAddress(linked), addend)
		if err != nil {
			return err
		}
		return putUint32(location, value)
	case elf.R_ARM_REL32:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		value, err := relativeValue32(directLinkedAddress(linked), addend, place, 0)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(location, value)
		return nil
	case elf.R_ARM_GOT_PREL:
		got, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		if got&3 != 0 {
			return fmt.Errorf("GOT entry address %#x is not 4-byte aligned", got)
		}
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		value, err := relativeValue32(got, addend, place, 0)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(location, value)
		return nil
	case elf.R_ARM_PREL31:
		return applyARMPrel31(location, directLinkedAddress(linked), place, relocation.hasAdd, relocation.addend)
	case elf.R_ARM_CALL:
		word := binary.LittleEndian.Uint32(location)
		if word&0xff000000 != 0xeb000000 {
			return armInstructionClassError("unconditional BL", word)
		}
		return applyARMBranch24(location, thunkLinkedAddress(linked), place, relocation.hasAdd, relocation.addend)
	case elf.R_ARM_PC24, elf.R_ARM_PLT32, elf.R_ARM_JUMP24:
		return applyARMBranch24(location, thunkLinkedAddress(linked), place, relocation.hasAdd, relocation.addend)
	default:
		return fmt.Errorf("unsupported ELF/arm relocation")
	}
}

func applyARMBranch24(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	// Bits 27:25 identify the Arm B/BL immediate class. Condition 0xf is
	// reserved for BLX, whose state-switching encoding is deliberately not
	// accepted by the ARM-state BOF backend.
	if word&0x0e000000 != 0x0a000000 || word>>28 == 0xf {
		return armInstructionClassError("B/BL", word)
	}
	if target > math.MaxUint32 {
		return fmt.Errorf("branch target %#x exceeds 32-bit address space", target)
	}
	if place > math.MaxUint32 {
		return fmt.Errorf("branch place %#x exceeds 32-bit address space", place)
	}
	if target&1 != 0 {
		return fmt.Errorf("branch target %#x enters Thumb state, which is unsupported", target)
	}
	addend := explicitAddend
	if !explicit {
		addend = signExtend(uint64(word&0x00ffffff), 24) << 2
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta&3 != 0 {
		return fmt.Errorf("branch target displacement %d is not 4-byte aligned", delta)
	}
	if delta < -(1<<25) || delta > (1<<25)-4 {
		return fmt.Errorf("branch target displacement %d exceeds signed 26-bit range", delta)
	}
	word = (word &^ 0x00ffffff) | (uint32(delta>>2) & 0x00ffffff)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARMPrel31(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	if target > math.MaxUint32 {
		return fmt.Errorf("PREL31 target %#x exceeds 32-bit address space", target)
	}
	if place > math.MaxUint32 {
		return fmt.Errorf("PREL31 place %#x exceeds 32-bit address space", place)
	}
	word := binary.LittleEndian.Uint32(location)
	addend := explicitAddend
	if !explicit {
		addend = signExtend(uint64(word&0x7fffffff), 31)
	}
	value, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if value < -(1<<30) || value > (1<<30)-1 {
		return fmt.Errorf("PREL31 displacement %d exceeds signed 31-bit range", value)
	}
	word = (word & 0x80000000) | (uint32(value) & 0x7fffffff)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func armInstructionClassError(expected string, word uint32) error {
	return fmt.Errorf("ARM relocation expected %s instruction, found %#08x", expected, word)
}
