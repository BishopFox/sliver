package bofloader

import (
	"encoding/binary"
	"fmt"
)

const (
	coffAMD64Addr64   = 0x0001
	coffAMD64Addr32   = 0x0002
	coffAMD64Addr32NB = 0x0003
	coffAMD64Rel32    = 0x0004
	coffAMD64Rel32_1  = 0x0005
	coffAMD64Rel32_2  = 0x0006
	coffAMD64Rel32_3  = 0x0007
	coffAMD64Rel32_4  = 0x0008
	coffAMD64Rel32_5  = 0x0009
	coffAMD64Section  = 0x000a
	coffAMD64SecRel   = 0x000b

	coffI386Dir16   = 0x0001
	coffI386Rel16   = 0x0002
	coffI386Dir32   = 0x0006
	coffI386Dir32NB = 0x0007
	coffI386Section = 0x000a
	coffI386SecRel  = 0x000b
	coffI386Rel32   = 0x0014

	coffARM64Addr32        = 0x0001
	coffARM64Addr32NB      = 0x0002
	coffARM64Branch26      = 0x0003
	coffARM64PageBaseRel21 = 0x0004
	coffARM64Rel21         = 0x0005
	coffARM64PageOffset12A = 0x0006
	coffARM64PageOffset12L = 0x0007
	coffARM64SecRel        = 0x0008
	coffARM64SecRelLow12A  = 0x0009
	coffARM64SecRelHigh12A = 0x000a
	coffARM64SecRelLow12L  = 0x000b
	coffARM64Section       = 0x000d
	coffARM64Addr64        = 0x000e
	coffARM64Branch19      = 0x000f
	coffARM64Branch14      = 0x0010
	coffARM64Rel32         = 0x0011
)

func coffRelocationWidth(arch string, typeID uint32) (int, bool, error) {
	switch arch {
	case "amd64":
		switch typeID {
		case coffAMD64Addr64:
			return 8, false, nil
		case coffAMD64Addr32, coffAMD64Addr32NB,
			coffAMD64Rel32, coffAMD64Rel32_1, coffAMD64Rel32_2,
			coffAMD64Rel32_3, coffAMD64Rel32_4, coffAMD64Rel32_5,
			coffAMD64SecRel:
			return 4, false, nil
		case coffAMD64Section:
			return 2, false, nil
		}
	case "386":
		switch typeID {
		case coffI386Dir16, coffI386Rel16, coffI386Section:
			return 2, false, nil
		case coffI386Dir32, coffI386Dir32NB, coffI386SecRel, coffI386Rel32:
			return 4, false, nil
		}
	case "arm64":
		switch typeID {
		case coffARM64Addr64:
			return 8, false, nil
		case coffARM64Addr32, coffARM64Addr32NB, coffARM64Branch26,
			coffARM64PageBaseRel21, coffARM64Rel21, coffARM64PageOffset12A,
			coffARM64PageOffset12L, coffARM64SecRel, coffARM64SecRelLow12A,
			coffARM64SecRelHigh12A, coffARM64SecRelLow12L,
			coffARM64Branch19, coffARM64Branch14, coffARM64Rel32:
			return 4, false, nil
		case coffARM64Section:
			return 2, false, nil
		}
	default:
		return 0, false, fmt.Errorf("unsupported COFF architecture %q", arch)
	}
	return 0, false, fmt.Errorf("unsupported COFF/%s relocation", arch)
}

func applyCOFFRelocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	switch object.arch {
	case "amd64":
		return applyCOFFAMD64Relocation(object, relocation, location, place, linked)
	case "386":
		return applyCOFFI386Relocation(object, relocation, location, place, linked)
	case "arm64":
		return applyCOFFARM64Relocation(object, relocation, location, place, linked)
	default:
		return fmt.Errorf("unsupported COFF architecture %q", object.arch)
	}
}

func applyCOFFAMD64Relocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
	if err != nil {
		return err
	}
	switch relocation.typeID {
	case coffAMD64Addr64:
		value, err := addSigned(coffLinkedAddress(linked, false), addend)
		if err != nil {
			return err
		}
		putUint64(location, value)
		return nil
	case coffAMD64Addr32:
		value, err := addSigned(coffLinkedAddress(linked, false), addend)
		if err != nil {
			return err
		}
		return putUint32(location, value)
	case coffAMD64Addr32NB:
		value, err := relativeValue(coffLinkedAddress(linked, false), addend, uint64(object.imageBase), 0)
		if err != nil {
			return err
		}
		if value < 0 {
			return fmt.Errorf("image-relative value %d is negative", value)
		}
		return putUint32(location, uint64(value))
	case coffAMD64Rel32, coffAMD64Rel32_1, coffAMD64Rel32_2,
		coffAMD64Rel32_3, coffAMD64Rel32_4, coffAMD64Rel32_5:
		bias := uint64(4 + relocation.typeID - coffAMD64Rel32)
		value, err := relativeValue(coffAMD64RelativeAddress(object, relocation, linked), addend, place, bias)
		if err != nil {
			return err
		}
		return putInt32(location, value)
	case coffAMD64Section:
		return applyCOFFSectionRelocation(location, addend, linked)
	case coffAMD64SecRel:
		return applyCOFFSecRelRelocation(location, addend, linked)
	default:
		return fmt.Errorf("unsupported COFF/amd64 relocation")
	}
}

func applyCOFFI386Relocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
	if err != nil {
		return err
	}
	switch relocation.typeID {
	case coffI386Dir16:
		value, err := addSigned(coffLinkedAddress(linked, false), addend)
		if err != nil {
			return err
		}
		return putUint16(location, value)
	case coffI386Rel16:
		value, err := relativeValue(coffLinkedAddress(linked, true), addend, place, 2)
		if err != nil {
			return err
		}
		return putInt16(location, value)
	case coffI386Dir32:
		value, err := addSigned(coffLinkedAddress(linked, false), addend)
		if err != nil {
			return err
		}
		return putUint32(location, value)
	case coffI386Dir32NB:
		value, err := relativeValue(coffLinkedAddress(linked, false), addend, uint64(object.imageBase), 0)
		if err != nil {
			return err
		}
		if value < 0 {
			return fmt.Errorf("image-relative value %d is negative", value)
		}
		return putUint32(location, uint64(value))
	case coffI386Section:
		return applyCOFFSectionRelocation(location, addend, linked)
	case coffI386SecRel:
		return applyCOFFSecRelRelocation(location, addend, linked)
	case coffI386Rel32:
		value, err := relativeValue32(coffLinkedAddress(linked, false), addend, place, 4)
		if err != nil {
			return err
		}
		return putUint32(location, uint64(value))
	default:
		return fmt.Errorf("unsupported COFF/386 relocation")
	}
}

func applyCOFFARM64Relocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	switch relocation.typeID {
	case coffARM64Addr64:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		value, err := addSigned(coffLinkedAddress(linked, false), addend)
		if err != nil {
			return err
		}
		putUint64(location, value)
		return nil
	case coffARM64Addr32, coffARM64Addr32NB, coffARM64SecRel, coffARM64Rel32:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		switch relocation.typeID {
		case coffARM64Addr32:
			value, err := addSigned(coffLinkedAddress(linked, false), addend)
			if err != nil {
				return err
			}
			return putUint32(location, value)
		case coffARM64Addr32NB:
			value, err := relativeValue(coffLinkedAddress(linked, false), addend, uint64(object.imageBase), 0)
			if err != nil {
				return err
			}
			if value < 0 {
				return fmt.Errorf("image-relative value %d is negative", value)
			}
			return putUint32(location, uint64(value))
		case coffARM64SecRel:
			return applyCOFFSecRelRelocation(location, addend, linked)
		case coffARM64Rel32:
			value, err := relativeValue(coffLinkedAddress(linked, false), addend, place, 4)
			if err != nil {
				return err
			}
			return putInt32(location, value)
		}
	case coffARM64Section:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		return applyCOFFSectionRelocation(location, addend, linked)
	case coffARM64Branch26:
		return applyARM64Branch26(location, coffLinkedAddress(linked, true), place, relocation.hasAdd, relocation.addend)
	case coffARM64Branch19:
		return applyARM64Branch19(location, coffLinkedAddress(linked, true), place, relocation.hasAdd, relocation.addend)
	case coffARM64Branch14:
		return applyARM64Branch14(location, coffLinkedAddress(linked, true), place, relocation.hasAdd, relocation.addend)
	case coffARM64PageBaseRel21:
		// COFF stores the implicit ADRP addend as an unscaled byte offset.
		// The shared helper's implicit path treats it as a page count, so decode
		// the COFF addend before calling it. This matches link.exe/lld and
		// is required for section-symbol references such as .rdata+0x269.
		addend := relocation.addend
		if !relocation.hasAdd {
			addend = decodeARM64ADRImmediate(binary.LittleEndian.Uint32(location))
		}
		return applyARM64ADRP(location, coffLinkedAddress(linked, false), place, true, addend, true)
	case coffARM64Rel21:
		return applyARM64ADR(location, coffLinkedAddress(linked, false), place, relocation.hasAdd, relocation.addend)
	case coffARM64PageOffset12A:
		return applyARM64AddLO12(location, coffLinkedAddress(linked, false), relocation.hasAdd, relocation.addend)
	case coffARM64PageOffset12L:
		word := binary.LittleEndian.Uint32(location)
		return applyARM64LoadStoreLO12(location, coffLinkedAddress(linked, false), arm64LoadStoreScale(word), relocation.hasAdd, relocation.addend)
	case coffARM64SecRelLow12A, coffARM64SecRelHigh12A, coffARM64SecRelLow12L:
		if linked.external != nil || linked.symbol.section < 0 {
			return fmt.Errorf("section-relative relocation cannot target external symbol %q", linked.symbol.name)
		}
		target := linked.symbol.value
		switch relocation.typeID {
		case coffARM64SecRelLow12A:
			return applyARM64AddLO12(location, target, relocation.hasAdd, relocation.addend)
		case coffARM64SecRelHigh12A:
			return applyARM64AddHigh12(location, target, relocation.hasAdd, relocation.addend)
		case coffARM64SecRelLow12L:
			word := binary.LittleEndian.Uint32(location)
			return applyARM64LoadStoreLO12(location, target, arm64LoadStoreScale(word), relocation.hasAdd, relocation.addend)
		}
	}
	return fmt.Errorf("unsupported COFF/arm64 relocation")
}

func coffAMD64RelativeAddress(object *objectFile, relocation objectRelocation, linked linkedSymbol) uint64 {
	if linked.external == nil || coffImportPointerSymbol(linked.symbol.name) {
		return coffLinkedAddress(linked, false)
	}
	if coffAMD64RelativeBranch(object, relocation) {
		return coffLinkedAddress(linked, true)
	}
	return coffLinkedAddress(linked, false)
}

func coffAMD64RelativeBranch(object *objectFile, relocation objectRelocation) bool {
	// Undefined COFF symbols do not reliably retain function/data type
	// metadata. Inspect relocation sites in executable sections instead. If the
	// bytes are unavailable, resolving the symbol directly is safer than
	// treating data as executable thunk bytes.
	if object == nil || relocation.section < 0 || relocation.section >= len(object.sections) {
		return false
	}
	section := object.sections[relocation.section]
	if section.protection&protExec == 0 || relocation.offset == 0 || relocation.offset > uint64(len(section.data)) {
		return false
	}
	offset := int(relocation.offset)
	opcode := section.data[offset-1]
	if opcode == 0xe8 || opcode == 0xe9 {
		return true
	}
	return opcode >= 0x80 && opcode <= 0x8f && offset >= 2 && section.data[offset-2] == 0x0f
}

func applyCOFFSectionRelocation(location []byte, addend int64, linked linkedSymbol) error {
	if linked.external != nil || linked.symbol.section < 0 {
		return fmt.Errorf("section-index relocation cannot target external symbol %q", linked.symbol.name)
	}
	value := uint64(linked.symbol.section + 1)
	adjusted, err := addSigned(value, addend)
	if err != nil {
		return err
	}
	return putUint16(location, adjusted)
}

func applyCOFFSecRelRelocation(location []byte, addend int64, linked linkedSymbol) error {
	if linked.external != nil || linked.symbol.section < 0 {
		return fmt.Errorf("section-relative relocation cannot target external symbol %q", linked.symbol.name)
	}
	value, err := addSigned(linked.symbol.value, addend)
	if err != nil {
		return err
	}
	return putUint32(location, value)
}
