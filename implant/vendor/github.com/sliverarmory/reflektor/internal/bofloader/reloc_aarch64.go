package bofloader

import (
	"encoding/binary"
	"fmt"
)

const (
	arm64Imm12Mask = uint32(0x003ffc00)
	arm64ADRMask   = uint32(0x60ffffe0)
)

func applyARM64Branch26(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x7c000000 != 0x14000000 {
		return arm64InstructionClassError("B/BL", word)
	}
	addend := explicitAddend
	if !explicit {
		addend = signExtend(uint64(word&0x03ffffff), 26) << 2
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta&3 != 0 {
		return fmt.Errorf("branch target displacement %d is not 4-byte aligned", delta)
	}
	if delta < -(1<<27) || delta > (1<<27)-4 {
		return fmt.Errorf("branch target displacement %d exceeds signed 28-bit range", delta)
	}
	word = (word &^ 0x03ffffff) | (uint32(delta>>2) & 0x03ffffff)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64Branch19(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0xff000000 != 0x54000000 && word&0x7e000000 != 0x34000000 {
		return arm64InstructionClassError("B.cond/CBZ/CBNZ", word)
	}
	return applyARM64Immediate19(location, word, target, place, explicit, explicitAddend, "conditional branch")
}

func applyARM64Literal19(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x3b000000 != 0x18000000 {
		return arm64InstructionClassError("literal load", word)
	}
	return applyARM64Immediate19(location, word, target, place, explicit, explicitAddend, "literal load")
}

func applyARM64Immediate19(location []byte, word uint32, target, place uint64, explicit bool, explicitAddend int64, description string) error {
	addend := explicitAddend
	if !explicit {
		addend = signExtend(uint64((word>>5)&0x7ffff), 19) << 2
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta&3 != 0 {
		return fmt.Errorf("%s target displacement %d is not 4-byte aligned", description, delta)
	}
	if delta < -(1<<20) || delta > (1<<20)-4 {
		return fmt.Errorf("%s target displacement %d exceeds signed 21-bit range", description, delta)
	}
	word = (word &^ (0x7ffff << 5)) | ((uint32(delta>>2) & 0x7ffff) << 5)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64Branch14(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x7e000000 != 0x36000000 {
		return arm64InstructionClassError("TBZ/TBNZ", word)
	}
	addend := explicitAddend
	if !explicit {
		addend = signExtend(uint64((word>>5)&0x3fff), 14) << 2
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta&3 != 0 {
		return fmt.Errorf("test branch target displacement %d is not 4-byte aligned", delta)
	}
	if delta < -(1<<15) || delta > (1<<15)-4 {
		return fmt.Errorf("test branch target displacement %d exceeds signed 16-bit range", delta)
	}
	word = (word &^ (0x3fff << 5)) | ((uint32(delta>>2) & 0x3fff) << 5)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64ADR(location []byte, target, place uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x9f000000 != 0x10000000 {
		return arm64InstructionClassError("ADR", word)
	}
	addend := explicitAddend
	if !explicit {
		addend = decodeARM64ADRImmediate(word)
	}
	delta, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	if delta < -(1<<20) || delta > (1<<20)-1 {
		return fmt.Errorf("ADR displacement %d exceeds signed 21-bit range", delta)
	}
	word = encodeARM64ADRImmediate(word, delta)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64ADRP(location []byte, target, place uint64, explicit bool, explicitAddend int64, checkRange bool) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x9f000000 != 0x90000000 {
		return arm64InstructionClassError("ADRP", word)
	}
	addend := explicitAddend
	if !explicit {
		decoded := decodeARM64ADRImmediate(word)
		if decoded < -(1<<20) || decoded > (1<<20)-1 {
			return fmt.Errorf("invalid ADRP addend %d", decoded)
		}
		addend = decoded << 12
	}
	targetWithAddend, err := addSigned(target, addend)
	if err != nil {
		return err
	}
	targetPage := targetWithAddend &^ 0xfff
	placePage := place &^ 0xfff
	delta, err := signedDifference(targetPage, placePage)
	if err != nil {
		return err
	}
	if delta&0xfff != 0 {
		return fmt.Errorf("ADRP page displacement %d is not page aligned", delta)
	}
	pages := delta >> 12
	if checkRange && (pages < -(1<<20) || pages > (1<<20)-1) {
		return fmt.Errorf("ADRP page displacement %d exceeds signed 21-bit range", pages)
	}
	word = encodeARM64ADRImmediate(word, pages)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64AddLO12(location []byte, target uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x7f000000 != 0x11000000 {
		return arm64InstructionClassError("ADD (immediate)", word)
	}
	addend := explicitAddend
	if !explicit {
		addend = int64((word >> 10) & 0xfff)
		if word&(1<<22) != 0 {
			addend <<= 12
		}
	}
	if word&(1<<22) != 0 {
		return fmt.Errorf("low-12 ADD relocation targets an instruction with LSL #12")
	}
	value, err := addSigned(target, addend)
	if err != nil {
		return err
	}
	word = (word &^ arm64Imm12Mask) | (uint32(value&0xfff) << 10)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64AddHigh12(location []byte, target uint64, explicit bool, explicitAddend int64) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x7f000000 != 0x11000000 {
		return arm64InstructionClassError("ADD (immediate)", word)
	}
	addend := explicitAddend
	if !explicit {
		addend = int64((word>>10)&0xfff) << 12
	}
	value, err := addSigned(target, addend)
	if err != nil {
		return err
	}
	word = (word &^ arm64Imm12Mask) | (uint32((value>>12)&0xfff) << 10) | (1 << 22)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func applyARM64LoadStoreLO12(location []byte, target uint64, scale uint, explicit bool, explicitAddend int64) error {
	if scale > 4 {
		return fmt.Errorf("invalid load/store scale %d", scale)
	}
	word := binary.LittleEndian.Uint32(location)
	if word&0x3b000000 != 0x39000000 {
		return arm64InstructionClassError("unsigned-immediate load/store", word)
	}
	if actualScale := arm64LoadStoreScale(word); actualScale != scale {
		return fmt.Errorf("ARM64 load/store relocation scale %d does not match instruction scale %d", scale, actualScale)
	}
	addend := explicitAddend
	if !explicit {
		addend = int64((word>>10)&0xfff) << scale
	}
	value, err := addSigned(target, addend)
	if err != nil {
		return err
	}
	pageOffset := value & 0xfff
	alignment := uint64(1) << scale
	if pageOffset&(alignment-1) != 0 {
		return fmt.Errorf("load/store page offset %#x is not aligned to %d bytes", pageOffset, alignment)
	}
	word = (word &^ arm64Imm12Mask) | (uint32(pageOffset>>scale) << 10)
	binary.LittleEndian.PutUint32(location, word)
	return nil
}

func arm64InstructionClassError(expected string, word uint32) error {
	return fmt.Errorf("ARM64 relocation expected %s instruction, found %#08x", expected, word)
}

func decodeARM64ADRImmediate(word uint32) int64 {
	immlo := uint64((word >> 29) & 0x3)
	immhi := uint64((word >> 5) & 0x7ffff)
	return signExtend((immhi<<2)|immlo, 21)
}

func encodeARM64ADRImmediate(word uint32, immediate int64) uint32 {
	value := uint32(immediate) & 0x1fffff
	immlo := (value & 0x3) << 29
	immhi := ((value >> 2) & 0x7ffff) << 5
	return (word &^ arm64ADRMask) | immlo | immhi
}

func arm64LoadStoreScale(word uint32) uint {
	// Integer and scalar FP/SIMD unsigned-immediate loads encode log2(size)
	// in bits 31:30. The Q-register form uses the vector bit plus opc=3.
	if word&(1<<26) != 0 && (word>>22)&0x2 != 0 {
		return 4
	}
	return uint((word >> 30) & 0x3)
}

func signExtend(value uint64, bits uint) int64 {
	shift := 64 - bits
	return int64(value<<shift) >> shift
}
