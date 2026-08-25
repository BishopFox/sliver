package bofloader

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

const (
	// ELFv2 conventionally places .TOC. 0x8000 bytes beyond the start of the
	// GOT. This gives single-instruction TOC references a symmetric window.
	ppc64TOCBias = uint64(0x8000)

	ppc64NOP        = uint32(0x60000000)
	ppc64RestoreTOC = uint32(0xe8410018) // ld r2,24(r1)
)

func ppc64RelocationNeedsRestoreSlot(relocation objectRelocation, location []byte, linked linkedSymbol) bool {
	if elf.R_PPC64(relocation.typeID) != elf.R_PPC64_REL24 || linked.external == nil || len(location) < 4 {
		return false
	}
	word := binary.LittleEndian.Uint32(location)
	return word>>26 == 18 && word&2 == 0 && word&1 != 0
}

func applyELFPPC64LERelocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	if !relocation.hasAdd {
		return fmt.Errorf("PPC64 ELFv2 relocations require RELA encoding")
	}
	if ppc64InstructionRelocation(relocation.typeID) && relocation.offset&3 != 0 {
		return fmt.Errorf("PPC64 instruction relocation offset %#x is not 4-byte aligned", relocation.offset)
	}
	switch elf.R_PPC64(relocation.typeID) {
	case elf.R_PPC64_ADDR64:
		value, err := addSigned(directLinkedAddress(linked), relocation.addend)
		if err != nil {
			return err
		}
		putUint64(location, value)
		return nil
	case elf.R_PPC64_REL24:
		return applyPPC64REL24(relocation, location, place, linked)
	case elf.R_PPC64_REL16_HA:
		return applyPPC64InstructionHalf(location, directLinkedAddress(linked), relocation.addend, place, ppc64HalfRelativeHA)
	case elf.R_PPC64_REL16_LO:
		return applyPPC64InstructionHalf(location, directLinkedAddress(linked), relocation.addend, place, ppc64HalfRelativeLO)
	case elf.R_PPC64_TOC16_HA:
		return applyPPC64TOCHalf(object, relocation, location, linked, ppc64HalfTOCHA)
	case elf.R_PPC64_TOC16_LO:
		return applyPPC64TOCHalf(object, relocation, location, linked, ppc64HalfTOCLO)
	case elf.R_PPC64_TOC16_LO_DS:
		return applyPPC64TOCDS(object, relocation, location, linked)
	default:
		return fmt.Errorf("unsupported ELF/ppc64le relocation")
	}
}

func applyPPC64REL24(relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	if len(location) < 4 {
		return fmt.Errorf("PPC64 REL24 relocation requires a 4-byte branch instruction")
	}
	word := binary.LittleEndian.Uint32(location)
	if word>>26 != 18 {
		return ppc64InstructionClassError("relative branch", word)
	}
	if word&2 != 0 {
		return fmt.Errorf("PPC64 REL24 relocation cannot target an absolute branch instruction %#08x", word)
	}

	target := directLinkedAddress(linked)
	patchRestore := false
	if linked.external != nil {
		if relocation.addend != 0 {
			return fmt.Errorf("PPC64 external REL24 relocation must have a zero addend, got %d", relocation.addend)
		}
		if word&1 == 0 {
			return fmt.Errorf("PPC64 external REL24 tail branches are unsupported because they cannot restore the BOF TOC")
		}
		target = thunkLinkedAddress(linked)
		patchRestore = true
		if patchRestore {
			if len(location) < 8 {
				return fmt.Errorf("PPC64 external REL24 call is missing its 4-byte TOC restore slot")
			}
			if next := binary.LittleEndian.Uint32(location[4:8]); next != ppc64NOP {
				return fmt.Errorf("PPC64 external REL24 call must be followed by linker NOP %#08x, found %#08x", ppc64NOP, next)
			}
		}
	} else if linked.symbol.localEntry != 0 {
		var ok bool
		target, ok = checkedAddUint64(target, linked.symbol.localEntry)
		if !ok {
			return fmt.Errorf("PPC64 local entry address overflows")
		}
	}
	displacement, err := relativeValue(target, relocation.addend, place, 0)
	if err != nil {
		return err
	}
	if displacement&3 != 0 {
		return fmt.Errorf("PPC64 branch displacement %d is not 4-byte aligned", displacement)
	}
	if displacement < -(1<<25) || displacement > (1<<25)-4 {
		return fmt.Errorf("PPC64 branch displacement %d exceeds signed REL24 range", displacement)
	}
	patched := word&^uint32(0x03fffffc) | uint32(displacement)&0x03fffffc
	binary.LittleEndian.PutUint32(location, patched)
	if patchRestore {
		binary.LittleEndian.PutUint32(location[4:8], ppc64RestoreTOC)
	}
	return nil
}

func validateELFPPC64LERelocations(object *objectFile) error {
	if object == nil {
		return fmt.Errorf("bofloader: validate PPC64 relocations for nil object")
	}
	type restoreSlot struct {
		section int
		start   uint64
		end     uint64
		owner   int
	}
	type relocationSpan struct {
		section int
		start   uint64
		end     uint64
		owner   int
	}
	restores := make([]restoreSlot, 0)
	spans := make([]relocationSpan, 0, len(object.relocations))
	for index, relocation := range object.relocations {
		if ppc64InstructionRelocation(relocation.typeID) && relocation.offset&3 != 0 {
			return relocationError(object, relocation, fmt.Errorf("instruction relocation offset is not 4-byte aligned"))
		}
		width, noop, widthErr := elfRelocationWidth("ppc64le", relocation.typeID)
		if widthErr != nil || noop {
			continue
		}
		end, ok := checkedAddUint64(relocation.offset, uint64(width))
		if !ok {
			return relocationError(object, relocation, fmt.Errorf("relocation write range overflows"))
		}
		spans = append(spans, relocationSpan{section: relocation.section, start: relocation.offset, end: end, owner: index})
	}
	for index, relocation := range object.relocations {
		if elf.R_PPC64(relocation.typeID) != elf.R_PPC64_REL24 {
			continue
		}
		symbol, ok := object.symbols[relocation.symbol]
		if !ok || symbol.section != sectionUndefined {
			continue
		}
		if !relocation.hasAdd || relocation.addend != 0 {
			return relocationError(object, relocation, fmt.Errorf("external PPC64 REL24 relocation must use RELA with a zero addend"))
		}
		if relocation.section < 0 || relocation.section >= len(object.sections) {
			return relocationError(object, relocation, fmt.Errorf("target section index is invalid"))
		}
		section := object.sections[relocation.section]
		if relocation.offset > section.size || 8 > section.size-relocation.offset ||
			relocation.offset > uint64(len(section.data)) || 8 > uint64(len(section.data))-relocation.offset {
			return relocationError(object, relocation, fmt.Errorf("external ELFv2 call is missing its 4-byte TOC restore slot"))
		}
		word := binary.LittleEndian.Uint32(section.data[relocation.offset : relocation.offset+4])
		if word>>26 != 18 || word&2 != 0 {
			return relocationError(object, relocation, ppc64InstructionClassError("relative branch", word))
		}
		if word&1 == 0 {
			return relocationError(object, relocation, fmt.Errorf("external REL24 tail branches are unsupported because they cannot restore the BOF TOC"))
		}
		if next := binary.LittleEndian.Uint32(section.data[relocation.offset+4 : relocation.offset+8]); next != ppc64NOP {
			return relocationError(object, relocation, fmt.Errorf("external REL24 call must be followed by linker NOP %#08x, found %#08x", ppc64NOP, next))
		}
		restores = append(restores, restoreSlot{section: relocation.section, start: relocation.offset + 4, end: relocation.offset + 8, owner: index})
	}
	sort.Slice(spans, func(left, right int) bool {
		if spans[left].section != spans[right].section {
			return spans[left].section < spans[right].section
		}
		if spans[left].start != spans[right].start {
			return spans[left].start < spans[right].start
		}
		return spans[left].end < spans[right].end
	})
	sort.Slice(restores, func(left, right int) bool {
		if restores[left].section != restores[right].section {
			return restores[left].section < restores[right].section
		}
		return restores[left].start < restores[right].start
	})

	spanIndex := 0
	activeSection := -1
	var longest, secondLongest relocationSpan
	longestSet, secondLongestSet := false, false
	for _, restore := range restores {
		if restore.section != activeSection {
			activeSection = restore.section
			longestSet, secondLongestSet = false, false
			for spanIndex < len(spans) && spans[spanIndex].section < activeSection {
				spanIndex++
			}
		}
		for spanIndex < len(spans) && spans[spanIndex].section == activeSection && spans[spanIndex].start < restore.end {
			span := spans[spanIndex]
			spanIndex++
			if !longestSet || span.end > longest.end {
				secondLongest, secondLongestSet = longest, longestSet
				longest, longestSet = span, true
			} else if !secondLongestSet || span.end > secondLongest.end {
				secondLongest, secondLongestSet = span, true
			}
		}
		candidate, candidateSet := longest, longestSet
		if candidateSet && candidate.owner == restore.owner {
			candidate, candidateSet = secondLongest, secondLongestSet
		}
		if candidateSet && candidate.end > restore.start {
			relocation := object.relocations[candidate.owner]
			return relocationError(object, relocation, fmt.Errorf("relocation write overlaps PPC64 external-call TOC restore slot at %#x", restore.start))
		}
	}
	return nil
}

func ppc64InstructionRelocation(typeID uint32) bool {
	switch elf.R_PPC64(typeID) {
	case elf.R_PPC64_REL24, elf.R_PPC64_REL16_HA, elf.R_PPC64_REL16_LO,
		elf.R_PPC64_TOC16_HA, elf.R_PPC64_TOC16_LO, elf.R_PPC64_TOC16_LO_DS:
		return true
	default:
		return false
	}
}

type ppc64InstructionHalf uint8

const (
	ppc64HalfRelativeHA ppc64InstructionHalf = iota
	ppc64HalfRelativeLO
	ppc64HalfTOCHA
	ppc64HalfTOCLO
)

func applyPPC64InstructionHalf(location []byte, target uint64, addend int64, place uint64, kind ppc64InstructionHalf) error {
	if len(location) != 4 {
		return fmt.Errorf("PPC64 halfword relocation requires a 4-byte instruction")
	}
	word := binary.LittleEndian.Uint32(location)
	wantOpcode := uint32(15) // addis
	if kind == ppc64HalfRelativeLO {
		wantOpcode = 14 // addi
	}
	if word>>26 != wantOpcode {
		return ppc64InstructionClassError(ppc64OpcodeName(wantOpcode), word)
	}
	value, err := relativeValue(target, addend, place, 0)
	if err != nil {
		return err
	}
	high, low, err := ppc64SplitAddress(value, false)
	if err != nil {
		return err
	}
	immediate := low
	if kind == ppc64HalfRelativeHA {
		immediate = high
	}
	binary.LittleEndian.PutUint32(location, word&0xffff0000|uint32(uint16(immediate)))
	return nil
}

func applyPPC64TOCHalf(object *objectFile, relocation objectRelocation, location []byte, linked linkedSymbol, kind ppc64InstructionHalf) error {
	if object == nil || object.ppc64TOC == 0 {
		return fmt.Errorf("PPC64 TOC relocation has no ELFv2 TOC base")
	}
	if len(location) != 4 {
		return fmt.Errorf("PPC64 TOC halfword relocation requires a 4-byte instruction")
	}
	word := binary.LittleEndian.Uint32(location)
	wantOpcode := uint32(15) // addis
	if kind == ppc64HalfTOCLO {
		wantOpcode = 14 // addi
	}
	if word>>26 != wantOpcode {
		return ppc64InstructionClassError(ppc64OpcodeName(wantOpcode), word)
	}
	value, err := addSigned(directLinkedAddress(linked), relocation.addend)
	if err != nil {
		return err
	}
	displacement, err := signedDifference(value, uint64(object.ppc64TOC))
	if err != nil {
		return err
	}
	high, low, err := ppc64SplitAddress(displacement, false)
	if err != nil {
		return fmt.Errorf("PPC64 TOC displacement %d: %w", displacement, err)
	}
	immediate := low
	if kind == ppc64HalfTOCHA {
		immediate = high
	}
	binary.LittleEndian.PutUint32(location, word&0xffff0000|uint32(uint16(immediate)))
	return nil
}

func applyPPC64TOCDS(object *objectFile, relocation objectRelocation, location []byte, linked linkedSymbol) error {
	if object == nil || object.ppc64TOC == 0 {
		return fmt.Errorf("PPC64 TOC relocation has no ELFv2 TOC base")
	}
	if len(location) != 4 {
		return fmt.Errorf("PPC64 TOC DS relocation requires a 4-byte instruction")
	}
	word := binary.LittleEndian.Uint32(location)
	if opcode := word >> 26; opcode != 58 && opcode != 62 {
		return ppc64InstructionClassError("DS-form load/store", word)
	}
	value, err := addSigned(directLinkedAddress(linked), relocation.addend)
	if err != nil {
		return err
	}
	displacement, err := signedDifference(value, uint64(object.ppc64TOC))
	if err != nil {
		return err
	}
	_, low, err := ppc64SplitAddress(displacement, true)
	if err != nil {
		return fmt.Errorf("PPC64 TOC displacement %d: %w", displacement, err)
	}
	binary.LittleEndian.PutUint32(location, word&0xffff0003|uint32(uint16(low))&0xfffc)
	return nil
}

func ppc64SplitAddress(value int64, requireDSAlignment bool) (high int16, low int16, err error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, 0, fmt.Errorf("value is outside the signed 32-bit HA/LO range")
	}
	if requireDSAlignment && value&3 != 0 {
		return 0, 0, fmt.Errorf("value is not 4-byte aligned for a DS-form instruction")
	}
	highValue := (value + 0x8000) >> 16
	if highValue < math.MinInt16 || highValue > math.MaxInt16 {
		return 0, 0, fmt.Errorf("high-adjusted value %d overflows a signed 16-bit immediate", highValue)
	}
	return int16(highValue), int16(value), nil
}

func ppc64OpcodeName(opcode uint32) string {
	if opcode == 15 {
		return "addis"
	}
	return "addi"
}

func ppc64InstructionClassError(expected string, word uint32) error {
	return fmt.Errorf("PPC64 relocation expected %s instruction, found %#08x", expected, word)
}
