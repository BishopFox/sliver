package bofloader

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

type linkedSymbol struct {
	symbol   objectSymbol
	address  uint64
	external *externalSymbol
	got      uintptr
}

func applyRelocations(object *objectFile, region *memoryRegion, externals map[uint32]externalSymbol) error {
	if object == nil {
		return errors.New("bofloader: relocate nil object")
	}
	if region == nil {
		return errors.New("bofloader: relocate into nil memory region")
	}

	for _, relocation := range object.relocations {
		width, noop, err := relocationWidth(object.format, object.arch, relocation)
		if err != nil {
			return relocationError(object, relocation, err)
		}
		if noop {
			continue
		}
		if relocation.section < 0 || relocation.section >= len(object.sections) {
			return relocationError(object, relocation, errors.New("target section index is invalid"))
		}
		section := &object.sections[relocation.section]
		if !section.mapped {
			return relocationError(object, relocation, fmt.Errorf("target section %q is not mapped", section.name))
		}
		if relocation.offset > section.size || uint64(width) > section.size-relocation.offset {
			return relocationError(object, relocation, fmt.Errorf("write of %d bytes exceeds target section size %d", width, section.size))
		}
		imageOffset, ok := checkedAddUint64(section.offset, relocation.offset)
		if !ok || imageOffset > uint64(len(region.data)) || uint64(width) > uint64(len(region.data))-imageOffset {
			return relocationError(object, relocation, errors.New("write exceeds mapped image"))
		}
		place, ok := checkedAddUint64(uint64(section.address), relocation.offset)
		if !ok {
			return relocationError(object, relocation, errors.New("relocation address overflow"))
		}
		location := region.data[imageOffset : imageOffset+uint64(width)]

		linked, err := linkRelocationSymbol(object, relocation, externals)
		if err != nil {
			return relocationError(object, relocation, err)
		}
		if object.format == "elf" && object.arch == "ppc64le" && ppc64RelocationNeedsRestoreSlot(relocation, location, linked) {
			const callSpan = uint64(8)
			if relocation.offset > section.size || callSpan > section.size-relocation.offset {
				return relocationError(object, relocation, fmt.Errorf("external ELFv2 call and TOC restore slot exceed target section size %d", section.size))
			}
			if imageOffset > uint64(len(region.data)) || callSpan > uint64(len(region.data))-imageOffset {
				return relocationError(object, relocation, errors.New("external ELFv2 call and TOC restore slot exceed mapped image"))
			}
			location = region.data[imageOffset : imageOffset+callSpan]
		}
		switch object.format {
		case "coff":
			err = applyCOFFRelocation(object, relocation, location, place, linked)
		case "elf":
			err = applyELFRelocation(object, relocation, location, place, linked, externals)
		case "macho":
			err = applyMachORelocation(object, relocation, location, place, linked)
		default:
			err = fmt.Errorf("unsupported object format %q", object.format)
		}
		if err != nil {
			return relocationError(object, relocation, err)
		}
	}
	return nil
}

func relocationError(object *objectFile, relocation objectRelocation, err error) error {
	section := fmt.Sprintf("section %d", relocation.section)
	if object != nil && relocation.section >= 0 && relocation.section < len(object.sections) {
		section = fmt.Sprintf("section %q", object.sections[relocation.section].name)
	}
	format, arch := "object", "unknown"
	if object != nil {
		format, arch = object.format, object.arch
	}
	return fmt.Errorf("bofloader: %s/%s relocation type %d in %s at %#x: %w", format, arch, relocation.typeID, section, relocation.offset, err)
}

func linkRelocationSymbol(object *objectFile, relocation objectRelocation, externals map[uint32]externalSymbol) (linkedSymbol, error) {
	symbol, ok := object.symbols[relocation.symbol]
	if !ok {
		// ELF symbol table entry zero is the undefined symbol with value zero.
		if object.format == "elf" && relocation.symbol == 0 {
			return linkedSymbol{symbol: objectSymbol{index: 0, section: sectionAbsolute}}, nil
		}
		return linkedSymbol{}, fmt.Errorf("symbol index %d is missing", relocation.symbol)
	}
	linked := linkedSymbol{symbol: symbol}
	switch symbol.section {
	case sectionUndefined:
		external, found := externals[symbol.index]
		if !found {
			return linkedSymbol{}, fmt.Errorf("external symbol %q has no resolved linkage", symbol.name)
		}
		linked.external = &external
		linked.address = uint64(external.target)
		linked.got = external.got
		return linked, nil
	case sectionAbsolute:
		linked.address = symbol.value
	case sectionCommon:
		return linkedSymbol{}, fmt.Errorf("common symbol %q was not assigned to a mapped section", symbol.name)
	case sectionDiscarded:
		return linkedSymbol{}, fmt.Errorf("symbol %q belongs to a discarded section", symbol.name)
	default:
		if symbol.section < 0 || symbol.section >= len(object.sections) {
			return linkedSymbol{}, fmt.Errorf("symbol %q has invalid section %d", symbol.name, symbol.section)
		}
		section := object.sections[symbol.section]
		if !section.mapped {
			return linkedSymbol{}, fmt.Errorf("symbol %q references unmapped section %q", symbol.name, section.name)
		}
		if symbol.value > section.size {
			return linkedSymbol{}, fmt.Errorf("symbol %q value %#x exceeds section %q size %#x", symbol.name, symbol.value, section.name, section.size)
		}
		address, ok := checkedAddUint64(uint64(section.address), symbol.value)
		if !ok {
			return linkedSymbol{}, fmt.Errorf("symbol %q address overflow", symbol.name)
		}
		linked.address = address
	}
	if linkage, found := externals[symbol.index]; found {
		linked.got = linkage.got
	}
	return linked, nil
}

func relocationWidth(format, arch string, relocation objectRelocation) (width int, noop bool, err error) {
	if format != "macho" && relocation.typeID == 0 {
		return 0, true, nil
	}
	switch format {
	case "coff":
		return coffRelocationWidth(arch, relocation.typeID)
	case "elf":
		return elfRelocationWidth(arch, relocation.typeID)
	case "macho":
		if relocation.width != 1 && relocation.width != 2 && relocation.width != 4 && relocation.width != 8 {
			return 0, false, fmt.Errorf("invalid Mach-O relocation width %d", relocation.width)
		}
		return int(relocation.width), false, nil
	default:
		return 0, false, fmt.Errorf("unsupported object format %q", format)
	}
}

func directLinkedAddress(linked linkedSymbol) uint64 {
	return linked.address
}

func thunkLinkedAddress(linked linkedSymbol) uint64 {
	if linked.external != nil {
		return uint64(linked.external.thunk)
	}
	return linked.address
}

func gotLinkedAddress(linked linkedSymbol) (uint64, error) {
	if linked.got == 0 {
		return 0, fmt.Errorf("symbol %q requires a GOT entry, but none was allocated", linked.symbol.name)
	}
	return uint64(linked.got), nil
}

func coffLinkedAddress(linked linkedSymbol, branch bool) uint64 {
	if linked.external == nil {
		return linked.address
	}
	if coffImportPointerSymbol(linked.symbol.name) {
		return uint64(linked.external.got)
	}
	if branch {
		return uint64(linked.external.thunk)
	}
	return linked.address
}

func coffImportPointerSymbol(name string) bool {
	return strings.HasPrefix(name, "__imp_")
}

func gotBaseAddress(externals map[uint32]externalSymbol) (uint64, error) {
	base := uint64(math.MaxUint64)
	for _, external := range externals {
		address := uint64(external.got)
		if address != 0 && address < base {
			base = address
		}
	}
	if base == math.MaxUint64 {
		return 0, errors.New("relocation requires a GOT, but the object has no GOT entries")
	}
	return base, nil
}

func relocationAddend(location []byte, explicit bool, addend int64) (int64, error) {
	if explicit {
		return addend, nil
	}
	switch len(location) {
	case 2:
		return int64(int16(binary.LittleEndian.Uint16(location))), nil
	case 4:
		return int64(int32(binary.LittleEndian.Uint32(location))), nil
	case 8:
		return int64(binary.LittleEndian.Uint64(location)), nil
	default:
		return 0, fmt.Errorf("cannot read addend from %d-byte relocation", len(location))
	}
}

func checkedAddUint64(left, right uint64) (uint64, bool) {
	if left > math.MaxUint64-right {
		return 0, false
	}
	return left + right, true
}

func addSigned(base uint64, addend int64) (uint64, error) {
	if addend >= 0 {
		value, ok := checkedAddUint64(base, uint64(addend))
		if !ok {
			return 0, errors.New("address plus addend overflows 64 bits")
		}
		return value, nil
	}
	magnitude := uint64(-(addend + 1)) + 1
	if magnitude > base {
		return 0, errors.New("address plus addend is negative")
	}
	return base - magnitude, nil
}

func relativeValue(target uint64, addend int64, place uint64, bias uint64) (int64, error) {
	value, err := addSigned(target, addend)
	if err != nil {
		return 0, err
	}
	reference, ok := checkedAddUint64(place, bias)
	if !ok {
		return 0, errors.New("relocation place plus bias overflows 64 bits")
	}
	return signedDifference(value, reference)
}

func relativeValue32(target uint64, addend int64, place uint64, bias uint64) (uint32, error) {
	if target > math.MaxUint32 {
		return 0, fmt.Errorf("relocation target %#x exceeds 32-bit address space", target)
	}
	if place > math.MaxUint32 {
		return 0, fmt.Errorf("relocation place %#x exceeds 32-bit address space", place)
	}
	if bias > math.MaxUint32 {
		return 0, fmt.Errorf("relocation bias %#x exceeds 32 bits", bias)
	}
	// On 32-bit x86, S + A - P is computed modulo 2^32. The resulting bit
	// pattern remains valid even when the mathematical difference falls outside
	// the signed int32 range.
	return uint32(target) + uint32(addend) - uint32(place) - uint32(bias), nil
}

func signedDifference(left, right uint64) (int64, error) {
	if left >= right {
		difference := left - right
		if difference > math.MaxInt64 {
			return 0, errors.New("positive relative offset overflows 64 bits")
		}
		return int64(difference), nil
	}
	difference := right - left
	if difference > uint64(math.MaxInt64)+1 {
		return 0, errors.New("negative relative offset overflows 64 bits")
	}
	if difference == uint64(math.MaxInt64)+1 {
		return math.MinInt64, nil
	}
	return -int64(difference), nil
}

func absoluteSignedValue(target uint64, addend int64) (int64, error) {
	value, err := addSigned(target, addend)
	if err != nil {
		return 0, err
	}
	if value > math.MaxInt64 {
		return 0, errors.New("absolute value exceeds signed 64-bit range")
	}
	return int64(value), nil
}

func putUint16(location []byte, value uint64) error {
	if value > math.MaxUint16 {
		return fmt.Errorf("value %#x overflows 16-bit unsigned relocation", value)
	}
	binary.LittleEndian.PutUint16(location, uint16(value))
	return nil
}

func putInt16(location []byte, value int64) error {
	if value < math.MinInt16 || value > math.MaxInt16 {
		return fmt.Errorf("value %d overflows 16-bit signed relocation", value)
	}
	binary.LittleEndian.PutUint16(location, uint16(int16(value)))
	return nil
}

func putUint32(location []byte, value uint64) error {
	if value > math.MaxUint32 {
		return fmt.Errorf("value %#x overflows 32-bit unsigned relocation", value)
	}
	binary.LittleEndian.PutUint32(location, uint32(value))
	return nil
}

func putInt32(location []byte, value int64) error {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return fmt.Errorf("value %d overflows 32-bit signed relocation", value)
	}
	binary.LittleEndian.PutUint32(location, uint32(int32(value)))
	return nil
}

func putUint64(location []byte, value uint64) {
	binary.LittleEndian.PutUint64(location, value)
}

func putInt64(location []byte, value int64) {
	binary.LittleEndian.PutUint64(location, uint64(value))
}
