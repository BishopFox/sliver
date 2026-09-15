package bofloader

import (
	"encoding/binary"
	"fmt"
)

const (
	machoX86RelocUnsigned   = uint32(0)
	machoX86RelocSigned     = uint32(1)
	machoX86RelocBranch     = uint32(2)
	machoX86RelocGOTLoad    = uint32(3)
	machoX86RelocGOT        = uint32(4)
	machoX86RelocSubtractor = uint32(5)
	machoX86RelocSigned1    = uint32(6)
	machoX86RelocSigned2    = uint32(7)
	machoX86RelocSigned4    = uint32(8)
	machoX86RelocTLV        = uint32(9)

	machoARM64RelocUnsigned          = uint32(0)
	machoARM64RelocSubtractor        = uint32(1)
	machoARM64RelocBranch26          = uint32(2)
	machoARM64RelocPage21            = uint32(3)
	machoARM64RelocPageOff12         = uint32(4)
	machoARM64RelocGOTLoadPage21     = uint32(5)
	machoARM64RelocGOTLoadPageOff12  = uint32(6)
	machoARM64RelocPointerToGOT      = uint32(7)
	machoARM64RelocTLVPLoadPage21    = uint32(8)
	machoARM64RelocTLVPLoadPageOff12 = uint32(9)
	machoARM64RelocAddend            = uint32(10)
	machoARM64RelocAuthenticatedPtr  = uint32(11)
)

func validateMachORelocationShape(arch string, typeID uint8, length uint8, pcrel, external bool) error {
	typeValue := uint32(typeID)
	switch arch {
	case "amd64":
		switch typeValue {
		case machoX86RelocUnsigned:
			if pcrel || length != 2 && length != 3 {
				return fmt.Errorf("X86_64_RELOC_UNSIGNED requires a 4- or 8-byte absolute field")
			}
		case machoX86RelocSigned, machoX86RelocBranch, machoX86RelocGOTLoad, machoX86RelocGOT,
			machoX86RelocSigned1, machoX86RelocSigned2, machoX86RelocSigned4:
			if !pcrel || length != 2 {
				return fmt.Errorf("x86_64 relocation type %d requires a 4-byte PC-relative field", typeID)
			}
		case machoX86RelocSubtractor:
			return fmt.Errorf("X86_64_RELOC_SUBTRACTOR pairs are unsupported")
		case machoX86RelocTLV:
			return fmt.Errorf("X86_64_RELOC_TLV is unsupported")
		default:
			return fmt.Errorf("unsupported x86_64 relocation type %d", typeID)
		}
	case "arm64":
		switch typeValue {
		case machoARM64RelocUnsigned:
			if pcrel || length != 3 {
				return fmt.Errorf("ARM64_RELOC_UNSIGNED requires an 8-byte absolute field")
			}
		case machoARM64RelocBranch26, machoARM64RelocPage21, machoARM64RelocGOTLoadPage21:
			if !pcrel || length != 2 {
				return fmt.Errorf("ARM64 relocation type %d requires a 4-byte PC-relative instruction", typeID)
			}
			if typeValue == machoARM64RelocBranch26 && !external {
				return fmt.Errorf("ARM64_RELOC_BRANCH26 requires an external symbol")
			}
		case machoARM64RelocPageOff12, machoARM64RelocGOTLoadPageOff12:
			if pcrel || length != 2 {
				return fmt.Errorf("ARM64 relocation type %d requires a 4-byte absolute instruction", typeID)
			}
		case machoARM64RelocPointerToGOT:
			if !pcrel || length != 2 {
				return fmt.Errorf("ARM64_RELOC_POINTER_TO_GOT requires a 4-byte PC-relative field; the ambiguous 8-byte absolute form is unsupported")
			}
		case machoARM64RelocAddend:
			if pcrel || external || length != 2 {
				return fmt.Errorf("ARM64_RELOC_ADDEND has invalid attributes")
			}
		case machoARM64RelocSubtractor:
			return fmt.Errorf("ARM64_RELOC_SUBTRACTOR pairs are unsupported")
		case machoARM64RelocTLVPLoadPage21, machoARM64RelocTLVPLoadPageOff12:
			return fmt.Errorf("ARM64 thread-local variable relocations are unsupported")
		case machoARM64RelocAuthenticatedPtr:
			return fmt.Errorf("ARM64 authenticated-pointer relocations are unsupported")
		default:
			return fmt.Errorf("unsupported ARM64 relocation type %d", typeID)
		}
	default:
		return fmt.Errorf("unsupported Mach-O architecture %q", arch)
	}
	return nil
}

func machoRelocationNeedsGOT(arch string, typeID uint32) bool {
	switch arch {
	case "amd64":
		return typeID == machoX86RelocGOT || typeID == machoX86RelocGOTLoad
	case "arm64":
		return typeID == machoARM64RelocGOTLoadPage21 ||
			typeID == machoARM64RelocGOTLoadPageOff12 || typeID == machoARM64RelocPointerToGOT
	default:
		return false
	}
}

func applyMachORelocation(object *objectFile, relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	switch object.arch {
	case "amd64":
		return applyMachOX86Relocation(relocation, location, place, linked)
	case "arm64":
		return applyMachOARM64Relocation(relocation, location, place, linked)
	default:
		return fmt.Errorf("unsupported Mach-O architecture %q", object.arch)
	}
}

func applyMachOX86Relocation(relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
	if err != nil {
		return err
	}
	switch relocation.typeID {
	case machoX86RelocUnsigned:
		value, err := addSigned(directLinkedAddress(linked), addend)
		if err != nil {
			return err
		}
		switch len(location) {
		case 4:
			return putUint32(location, value)
		case 8:
			putUint64(location, value)
			return nil
		default:
			return fmt.Errorf("X86_64_RELOC_UNSIGNED has invalid width %d", len(location))
		}
	case machoX86RelocSigned, machoX86RelocSigned1, machoX86RelocSigned2, machoX86RelocSigned4:
		bias := uint64(4)
		pcrelOffset := int64(0)
		switch relocation.typeID {
		case machoX86RelocSigned1:
			bias = 5
			pcrelOffset = 1
		case machoX86RelocSigned2:
			bias = 6
			pcrelOffset = 2
		case machoX86RelocSigned4:
			bias = 8
			pcrelOffset = 4
		}
		// Mach-O's SIGNED_1/2/4 external forms encode the ordinary addend
		// in the field. The suffix contributes to both the normalized addend
		// and the relocation's PC bias, so account for it on the addend side
		// before applying the biased PC-relative expression. Local relocations
		// have already been normalized to an explicit section-relative addend.
		if !relocation.hasAdd {
			addend += pcrelOffset
		}
		value, err := relativeValue(directLinkedAddress(linked), addend, place, bias)
		if err != nil {
			return err
		}
		return putInt32(location, value)
	case machoX86RelocBranch:
		value, err := relativeValue(thunkLinkedAddress(linked), addend, place, 4)
		if err != nil {
			return err
		}
		return putInt32(location, value)
	case machoX86RelocGOT, machoX86RelocGOTLoad:
		target, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		value, err := relativeValue(target, addend, place, 4)
		if err != nil {
			return err
		}
		return putInt32(location, value)
	default:
		return fmt.Errorf("unsupported x86_64 relocation type %d", relocation.typeID)
	}
}

func applyMachOARM64Relocation(relocation objectRelocation, location []byte, place uint64, linked linkedSymbol) error {
	// Mach-O instruction relocations carry addends in an explicit preceding
	// ARM64_RELOC_ADDEND record. Their instruction/address field must be zero;
	// the parser enforces that invariant. Keep explicit mode here as a second
	// line of defense so shared ELF/COFF helpers never infer an embedded addend.
	explicitAddend := relocation.addend
	explicit := relocation.hasAdd
	if relocation.typeID != machoARM64RelocUnsigned {
		explicit = true
	}
	switch relocation.typeID {
	case machoARM64RelocUnsigned:
		addend, err := relocationAddend(location, relocation.hasAdd, relocation.addend)
		if err != nil {
			return err
		}
		value, err := addSigned(directLinkedAddress(linked), addend)
		if err != nil {
			return err
		}
		putUint64(location, value)
		return nil
	case machoARM64RelocBranch26:
		return applyARM64Branch26(location, thunkLinkedAddress(linked), place, explicit, explicitAddend)
	case machoARM64RelocPage21:
		return applyARM64ADRP(location, directLinkedAddress(linked), place, explicit, explicitAddend, true)
	case machoARM64RelocPageOff12:
		relocation.hasAdd = explicit
		relocation.addend = explicitAddend
		return applyMachOARM64PageOff12(location, directLinkedAddress(linked), relocation)
	case machoARM64RelocGOTLoadPage21:
		target, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		return applyARM64ADRP(location, target, place, explicit, explicitAddend, true)
	case machoARM64RelocGOTLoadPageOff12:
		target, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		relocation.hasAdd = explicit
		relocation.addend = explicitAddend
		return applyMachOARM64PageOff12(location, target, relocation)
	case machoARM64RelocPointerToGOT:
		target, err := gotLinkedAddress(linked)
		if err != nil {
			return err
		}
		addend, err := relocationAddend(location, explicit, explicitAddend)
		if err != nil {
			return err
		}
		if len(location) != 4 {
			return fmt.Errorf("ARM64_RELOC_POINTER_TO_GOT has invalid width %d", len(location))
		}
		value, err := relativeValue(target, addend, place, 0)
		if err != nil {
			return err
		}
		return putInt32(location, value)
	default:
		return fmt.Errorf("unsupported ARM64 relocation type %d", relocation.typeID)
	}
}

func validateMachOARM64EmbeddedAddend(typeID uint32, location []byte) error {
	switch typeID {
	case machoARM64RelocBranch26, machoARM64RelocPage21, machoARM64RelocGOTLoadPage21,
		machoARM64RelocPageOff12, machoARM64RelocGOTLoadPageOff12, machoARM64RelocPointerToGOT:
	default:
		return nil
	}
	if len(location) != 4 {
		return fmt.Errorf("ARM64 Mach-O instruction relocation has invalid width %d", len(location))
	}
	word := binary.LittleEndian.Uint32(location)
	var nonzero bool
	switch typeID {
	case machoARM64RelocBranch26:
		nonzero = word&0x03ffffff != 0
	case machoARM64RelocPage21, machoARM64RelocGOTLoadPage21:
		nonzero = word&(0x3<<29|0x7ffff<<5) != 0
	case machoARM64RelocPageOff12, machoARM64RelocGOTLoadPageOff12:
		nonzero = word&(0xfff<<10) != 0
	case machoARM64RelocPointerToGOT:
		nonzero = word != 0
	}
	if nonzero {
		return fmt.Errorf("ARM64 Mach-O relocation type %d has a non-zero embedded addend; use ARM64_RELOC_ADDEND", typeID)
	}
	return nil
}

func applyMachOARM64PageOff12(location []byte, target uint64, relocation objectRelocation) error {
	word := binary.LittleEndian.Uint32(location)
	if word&0x7f000000 == 0x11000000 {
		return applyARM64AddLO12(location, target, relocation.hasAdd, relocation.addend)
	}
	if word&0x3b000000 == 0x39000000 {
		return applyARM64LoadStoreLO12(location, target, arm64LoadStoreScale(word), relocation.hasAdd, relocation.addend)
	}
	return arm64InstructionClassError("ADD or unsigned-immediate load/store", word)
}
