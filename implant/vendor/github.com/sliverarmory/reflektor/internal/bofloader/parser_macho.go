package bofloader

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	machoMagic32        = uint32(0xfeedface)
	machoMagic64        = uint32(0xfeedfacf)
	machoMagicFat       = uint32(0xcafebabe)
	machoMagicFat64     = uint32(0xcafebabf)
	machoCPUAMD64       = uint32(0x01000007)
	machoCPUARM64       = uint32(0x0100000c)
	machoCPUARM64E      = uint32(2)
	machoCPUSubtypeMask = uint32(0xff000000)
	machoTypeObject     = uint32(1)
	machoLoadSymtab     = uint32(0x2)
	machoLoadDysymtab   = uint32(0xb)
	machoLoadSegment64  = uint32(0x19)

	machoSectionTypeMask             = uint32(0xff)
	machoSectionRegular              = uint32(0)
	machoSectionZeroFill             = uint32(1)
	machoSectionCStringLiterals      = uint32(2)
	machoSectionFourByteLiterals     = uint32(3)
	machoSectionEightByteLiterals    = uint32(4)
	machoSectionLiteralPointers      = uint32(5)
	machoSectionCoalesced            = uint32(11)
	machoSectionGBZeroFill           = uint32(12)
	machoSectionSixteenByteLiterals  = uint32(14)
	machoSectionThreadLocalRegular   = uint32(17)
	machoSectionThreadLocalZeroFill  = uint32(18)
	machoSectionThreadLocalVariables = uint32(19)
	machoSectionThreadLocalPointers  = uint32(20)
	machoSectionThreadLocalInitFuncs = uint32(21)
	machoSectionAttrDebug            = uint32(0x02000000)
	machoSectionAttrSomeInstructions = uint32(0x00000400)
	machoSectionAttrPureInstructions = uint32(0x80000000)

	machoNStab     = uint8(0xe0)
	machoNTypeMask = uint8(0x0e)
	machoNUndef    = uint8(0x00)
	machoNAbs      = uint8(0x02)
	machoNIndirect = uint8(0x0a)
	machoNPrebound = uint8(0x0c)
	machoNSection  = uint8(0x0e)
	machoNWeakRef  = uint16(0x0040)

	machoSyntheticSectionSymbol = uint32(0x80000000)
)

type machoRawSection struct {
	name   string
	seg    string
	addr   uint64
	size   uint64
	offset uint32
	align  uint32
	reloff uint32
	nreloc uint32
	flags  uint32
}

type machoSymtabInfo struct {
	symoff  uint32
	nsyms   uint32
	stroff  uint32
	strsize uint32
}

type machoDysymtabInfo struct {
	data []byte
}

type machoPreflight struct {
	arch     string
	sections []machoRawSection
	symtab   machoSymtabInfo
	dysymtab []machoDysymtabInfo
}

func isMachOMagic(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	magic := binary.LittleEndian.Uint32(data[:4])
	switch magic {
	case machoMagic32, machoMagic64, machoMagicFat, machoMagicFat64,
		bitswap32(machoMagic32), bitswap32(machoMagic64), bitswap32(machoMagicFat), bitswap32(machoMagicFat64):
		return true
	default:
		return false
	}
}

func parseMachO(data []byte) (object *objectFile, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			object = nil
			err = fmt.Errorf("bofloader: malformed Mach-O object: %v", recovered)
		}
	}()

	preflight, err := preflightMachO(data)
	if err != nil {
		return nil, err
	}
	result := &objectFile{
		format:  "macho",
		arch:    preflight.arch,
		symbols: make(map[uint32]objectSymbol),
	}
	sectionMap := make([]int, len(preflight.sections))
	for index := range sectionMap {
		sectionMap[index] = sectionDiscarded
	}
	var mappedSectionBytes uint64
	for sourceIndex, raw := range preflight.sections {
		mapped, protection, mapErr := machoSectionMapping(raw)
		if mapErr != nil {
			return nil, mapErr
		}
		if !mapped {
			continue
		}
		alignment := uint64(1) << raw.align
		var contents []byte
		sectionType := raw.flags & machoSectionTypeMask
		if sectionType != machoSectionZeroFill && sectionType != machoSectionGBZeroFill && raw.size != 0 {
			contents = data[uint64(raw.offset) : uint64(raw.offset)+raw.size]
		}
		var ok bool
		mappedSectionBytes, ok = checkedAddUint64(mappedSectionBytes, raw.size)
		if !ok || mappedSectionBytes > maxImageSize {
			return nil, fmt.Errorf("bofloader: cumulative Mach-O mapped section size exceeds %d bytes", maxImageSize)
		}
		sectionMap[sourceIndex] = len(result.sections)
		result.sections = append(result.sections, objectSection{
			name:       raw.seg + "," + raw.name,
			data:       contents,
			size:       raw.size,
			align:      alignment,
			protection: protection,
			mapped:     raw.size != 0,
		})
	}

	stringsData := data[uint64(preflight.symtab.stroff) : uint64(preflight.symtab.stroff)+uint64(preflight.symtab.strsize)]
	commonOffset := uint64(0)
	commonAlignment := uint64(1)
	commonSymbols := make([]uint32, 0)
	for index := uint32(0); index < preflight.symtab.nsyms; index++ {
		offset := uint64(preflight.symtab.symoff) + uint64(index)*16
		raw := data[offset : offset+16]
		nameOffset := binary.LittleEndian.Uint32(raw[0:4])
		name := machoSymbolName(stringsData, nameOffset)
		typeID := raw[4]
		sectionOrdinal := raw[5]
		description := binary.LittleEndian.Uint16(raw[6:8])
		value := binary.LittleEndian.Uint64(raw[8:16])
		section := sectionUndefined

		if typeID&machoNStab != 0 {
			section = sectionDiscarded
		} else {
			switch typeID & machoNTypeMask {
			case machoNUndef:
				if value != 0 {
					alignmentExponent := uint((description >> 8) & 0x0f)
					alignment := uint64(1) << alignmentExponent
					commonOffset = alignUp(commonOffset, alignment)
					if value > maxImageSize-commonOffset {
						return nil, fmt.Errorf("bofloader: Mach-O common symbols exceed %d bytes", maxImageSize)
					}
					if alignment > commonAlignment {
						commonAlignment = alignment
					}
					value, commonOffset = commonOffset, commonOffset+value
					section = sectionCommon
					commonSymbols = append(commonSymbols, index)
				}
			case machoNAbs:
				section = sectionAbsolute
			case machoNSection:
				if sectionOrdinal == 0 || int(sectionOrdinal) > len(sectionMap) {
					return nil, fmt.Errorf("bofloader: Mach-O symbol %q references invalid section %d", name, sectionOrdinal)
				}
				rawSection := preflight.sections[int(sectionOrdinal)-1]
				section = sectionMap[int(sectionOrdinal)-1]
				if value < rawSection.addr {
					return nil, fmt.Errorf("bofloader: Mach-O symbol %q value %#x precedes section address %#x", name, value, rawSection.addr)
				}
				value -= rawSection.addr
				if value > rawSection.size {
					return nil, fmt.Errorf("bofloader: Mach-O symbol %q value %#x exceeds section %q size %#x", name, value, rawSection.name, rawSection.size)
				}
			case machoNIndirect, machoNPrebound:
				return nil, fmt.Errorf("bofloader: Mach-O symbol %q uses unsupported type %#x", name, typeID&machoNTypeMask)
			default:
				return nil, fmt.Errorf("bofloader: Mach-O symbol %q uses unknown type %#x", name, typeID&machoNTypeMask)
			}
		}
		result.symbols[index] = objectSymbol{
			index:   index,
			name:    name,
			section: section,
			value:   value,
			weak:    description&machoNWeakRef != 0,
		}
	}
	if commonOffset != 0 {
		if mappedSectionBytes > maxImageSize || commonOffset > maxImageSize-mappedSectionBytes {
			return nil, fmt.Errorf("bofloader: cumulative Mach-O mapped section size exceeds %d bytes", maxImageSize)
		}
		commonSection := len(result.sections)
		result.sections = append(result.sections, objectSection{
			name:       "__DATA,__common",
			size:       commonOffset,
			align:      commonAlignment,
			protection: protRead | protWrite,
			mapped:     true,
		})
		for _, index := range commonSymbols {
			symbol := result.symbols[index]
			symbol.section = commonSection
			result.symbols[index] = symbol
		}
	}

	sectionSymbols := make(map[int]uint32)
	for sourceIndex, rawSection := range preflight.sections {
		targetSection := sectionMap[sourceIndex]
		if targetSection < 0 || targetSection >= len(result.sections) || !result.sections[targetSection].mapped {
			continue
		}
		relocations, relocErr := decodeMachORelocations(data, result, preflight, rawSection, targetSection, sectionMap, sectionSymbols)
		if relocErr != nil {
			return nil, relocErr
		}
		result.relocations = append(result.relocations, relocations...)
	}
	return result, nil
}

func preflightMachO(data []byte) (*machoPreflight, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("bofloader: truncated Mach-O magic")
	}
	magic := binary.LittleEndian.Uint32(data[:4])
	if magic != machoMagic64 {
		switch magic {
		case machoMagic32:
			return nil, fmt.Errorf("bofloader: 32-bit Mach-O BOFs are unsupported")
		case machoMagicFat, machoMagicFat64, bitswap32(machoMagicFat), bitswap32(machoMagicFat64):
			return nil, fmt.Errorf("bofloader: universal Mach-O images are unsupported; provide one thin MH_OBJECT slice")
		case bitswap32(machoMagic32), bitswap32(machoMagic64):
			return nil, fmt.Errorf("bofloader: only little-endian Mach-O objects are supported")
		default:
			return nil, fmt.Errorf("bofloader: invalid Mach-O magic %#x", magic)
		}
	}
	if len(data) < 32 {
		return nil, fmt.Errorf("bofloader: truncated Mach-O header")
	}
	cpu := binary.LittleEndian.Uint32(data[4:8])
	subcpu := binary.LittleEndian.Uint32(data[8:12])
	fileType := binary.LittleEndian.Uint32(data[12:16])
	commandCount := binary.LittleEndian.Uint32(data[16:20])
	commandBytes := binary.LittleEndian.Uint32(data[20:24])
	if fileType != machoTypeObject {
		return nil, fmt.Errorf("bofloader: expected a Mach-O MH_OBJECT file, got type %#x", fileType)
	}
	arch := ""
	switch cpu {
	case machoCPUAMD64:
		arch = "amd64"
	case machoCPUARM64:
		if subcpu&^machoCPUSubtypeMask == machoCPUARM64E {
			return nil, fmt.Errorf("bofloader: arm64e Mach-O objects are unsupported")
		}
		arch = "arm64"
	default:
		return nil, fmt.Errorf("bofloader: unsupported Mach-O CPU %#x", cpu)
	}
	if commandCount > maxObjectSections {
		return nil, fmt.Errorf("bofloader: Mach-O load command count %d exceeds %d", commandCount, maxObjectSections)
	}
	if !parserRangeWithin(data, 32, uint64(commandBytes)) {
		return nil, fmt.Errorf("bofloader: Mach-O load commands extend outside the object")
	}

	result := &machoPreflight{arch: arch}
	cursor := uint64(32)
	commandsEnd := cursor + uint64(commandBytes)
	haveSymtab := false
	var cumulativeRelocations uint64
	for commandIndex := uint32(0); commandIndex < commandCount; commandIndex++ {
		if cursor > commandsEnd || commandsEnd-cursor < 8 {
			return nil, fmt.Errorf("bofloader: Mach-O load command %d is truncated", commandIndex)
		}
		command := binary.LittleEndian.Uint32(data[cursor : cursor+4])
		commandSize := uint64(binary.LittleEndian.Uint32(data[cursor+4 : cursor+8]))
		if commandSize < 8 || commandSize&7 != 0 || commandSize > commandsEnd-cursor {
			return nil, fmt.Errorf("bofloader: Mach-O load command %d has invalid size %d", commandIndex, commandSize)
		}
		commandData := data[cursor : cursor+commandSize]
		switch command {
		case machoLoadSegment64:
			if commandSize < 72 {
				return nil, fmt.Errorf("bofloader: Mach-O LC_SEGMENT_64 command is truncated")
			}
			fileOffset := binary.LittleEndian.Uint64(commandData[40:48])
			fileSize := binary.LittleEndian.Uint64(commandData[48:56])
			if fileSize != 0 && !parserRangeWithin(data, fileOffset, fileSize) {
				return nil, fmt.Errorf("bofloader: Mach-O LC_SEGMENT_64 file range extends outside the object")
			}
			sectionCount := uint64(binary.LittleEndian.Uint32(commandData[64:68]))
			sectionBytes, ok := checkedMulUint64(sectionCount, 80)
			if !ok || sectionCount > maxObjectSections || 72+sectionBytes != commandSize {
				return nil, fmt.Errorf("bofloader: Mach-O LC_SEGMENT_64 has invalid section count %d", sectionCount)
			}
			if uint64(len(result.sections))+sectionCount > maxObjectSections {
				return nil, fmt.Errorf("bofloader: Mach-O section count exceeds %d", maxObjectSections)
			}
			for sectionIndex := uint64(0); sectionIndex < sectionCount; sectionIndex++ {
				offset := 72 + sectionIndex*80
				raw := commandData[offset : offset+80]
				section := machoRawSection{
					name:   fixedMachOName(raw[0:16]),
					seg:    fixedMachOName(raw[16:32]),
					addr:   binary.LittleEndian.Uint64(raw[32:40]),
					size:   binary.LittleEndian.Uint64(raw[40:48]),
					offset: binary.LittleEndian.Uint32(raw[48:52]),
					align:  binary.LittleEndian.Uint32(raw[52:56]),
					reloff: binary.LittleEndian.Uint32(raw[56:60]),
					nreloc: binary.LittleEndian.Uint32(raw[60:64]),
					flags:  binary.LittleEndian.Uint32(raw[64:68]),
				}
				if section.align >= 64 || uint64(1)<<section.align > maxImageSize {
					return nil, fmt.Errorf("bofloader: Mach-O section %q has unsupported alignment exponent %d", section.name, section.align)
				}
				if _, ok := checkedAddUint64(section.addr, section.size); !ok {
					return nil, fmt.Errorf("bofloader: Mach-O section %q address range overflows", section.name)
				}
				mapped, _, mapErr := machoSectionMapping(section)
				if mapErr != nil {
					return nil, mapErr
				}
				if mapped && section.size > maxImageSize {
					return nil, fmt.Errorf("bofloader: Mach-O section %q size %d exceeds %d", section.name, section.size, maxImageSize)
				}
				sectionType := section.flags & machoSectionTypeMask
				if sectionType != machoSectionZeroFill && sectionType != machoSectionGBZeroFill && section.size != 0 &&
					!parserRangeWithin(data, uint64(section.offset), section.size) {
					return nil, fmt.Errorf("bofloader: Mach-O section %q data extends outside the object", section.name)
				}
				relocationBytes, ok := checkedMulUint64(uint64(section.nreloc), 8)
				if !ok || !parserRangeWithin(data, uint64(section.reloff), relocationBytes) {
					return nil, fmt.Errorf("bofloader: Mach-O section %q relocations extend outside the object", section.name)
				}
				cumulativeRelocations, ok = checkedAddUint64(cumulativeRelocations, uint64(section.nreloc))
				if !ok || cumulativeRelocations > maxObjectRelocations {
					return nil, fmt.Errorf("bofloader: Mach-O relocation count exceeds %d", maxObjectRelocations)
				}
				result.sections = append(result.sections, section)
			}
		case machoLoadSymtab:
			if commandSize != 24 || haveSymtab {
				return nil, fmt.Errorf("bofloader: Mach-O object has an invalid or duplicate LC_SYMTAB command")
			}
			haveSymtab = true
			result.symtab = machoSymtabInfo{
				symoff:  binary.LittleEndian.Uint32(commandData[8:12]),
				nsyms:   binary.LittleEndian.Uint32(commandData[12:16]),
				stroff:  binary.LittleEndian.Uint32(commandData[16:20]),
				strsize: binary.LittleEndian.Uint32(commandData[20:24]),
			}
		case machoLoadDysymtab:
			if commandSize != 80 {
				return nil, fmt.Errorf("bofloader: Mach-O LC_DYSYMTAB command has invalid size %d", commandSize)
			}
			result.dysymtab = append(result.dysymtab, machoDysymtabInfo{data: commandData})
		}
		cursor += commandSize
	}
	if cursor != commandsEnd {
		return nil, fmt.Errorf("bofloader: Mach-O load commands leave %d unparsed bytes", commandsEnd-cursor)
	}
	if !haveSymtab {
		return nil, fmt.Errorf("bofloader: Mach-O object has no LC_SYMTAB command")
	}
	if result.symtab.nsyms > maxObjectSymbols {
		return nil, fmt.Errorf("bofloader: Mach-O symbol count %d exceeds %d", result.symtab.nsyms, maxObjectSymbols)
	}
	symbolBytes, ok := checkedMulUint64(uint64(result.symtab.nsyms), 16)
	if !ok || !parserRangeWithin(data, uint64(result.symtab.symoff), symbolBytes) {
		return nil, fmt.Errorf("bofloader: Mach-O symbol table extends outside the object")
	}
	if result.symtab.strsize > maxObjectNameBytes || !parserRangeWithin(data, uint64(result.symtab.stroff), uint64(result.symtab.strsize)) {
		return nil, fmt.Errorf("bofloader: Mach-O string table is invalid or exceeds %d bytes", maxObjectNameBytes)
	}
	stringsData := data[uint64(result.symtab.stroff) : uint64(result.symtab.stroff)+uint64(result.symtab.strsize)]
	var cumulativeNames uint64
	for index := uint32(0); index < result.symtab.nsyms; index++ {
		offset := uint64(result.symtab.symoff) + uint64(index)*16
		nameOffset := binary.LittleEndian.Uint32(data[offset : offset+4])
		if nameOffset >= uint32(len(stringsData)) {
			return nil, fmt.Errorf("bofloader: Mach-O symbol %d has invalid string offset %d", index, nameOffset)
		}
		terminator := bytes.IndexByte(stringsData[nameOffset:], 0)
		if terminator < 0 {
			return nil, fmt.Errorf("bofloader: Mach-O symbol %d name is not NUL-terminated", index)
		}
		nameBytes := uint64(terminator)
		if nameBytes > maxObjectNameSize {
			return nil, fmt.Errorf("bofloader: Mach-O symbol %d name exceeds %d bytes", index, maxObjectNameSize)
		}
		cumulativeNames, ok = checkedAddUint64(cumulativeNames, nameBytes)
		if !ok || cumulativeNames > maxObjectNameBytes {
			return nil, fmt.Errorf("bofloader: cumulative Mach-O symbol names exceed %d bytes", maxObjectNameBytes)
		}
	}
	for _, dysymtab := range result.dysymtab {
		if err := preflightMachODysymtab(data, dysymtab.data, result.symtab.nsyms); err != nil {
			return nil, err
		}
	}
	for _, section := range result.sections {
		mapped, _, mapErr := machoSectionMapping(section)
		if mapErr != nil {
			return nil, mapErr
		}
		if !mapped {
			continue
		}
		for index := uint32(0); index < section.nreloc; index++ {
			offset := uint64(section.reloff) + uint64(index)*8
			address := binary.LittleEndian.Uint32(data[offset : offset+4])
			info := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			if address&(1<<31) != 0 {
				return nil, fmt.Errorf("bofloader: Mach-O section %q uses unsupported scattered relocation %d", section.name, index)
			}
			length := uint8((info >> 25) & 0x3)
			width := uint64(1) << length
			if uint64(address) > section.size || width > section.size-uint64(address) {
				return nil, fmt.Errorf("bofloader: Mach-O relocation %d in section %q exceeds section size %#x", index, section.name, section.size)
			}
			typeID := uint8(info >> 28)
			external := info&(1<<27) != 0
			value := info & 0x00ffffff
			if external && value >= result.symtab.nsyms {
				return nil, fmt.Errorf("bofloader: Mach-O relocation %d in section %q references invalid symbol %d", index, section.name, value)
			}
			if !external && !(arch == "arm64" && uint32(typeID) == machoARM64RelocAddend) && (value == 0 || uint64(value) > uint64(len(result.sections))) {
				return nil, fmt.Errorf("bofloader: Mach-O relocation %d in section %q references invalid section %d", index, section.name, value)
			}
			if err := validateMachORelocationShape(arch, typeID, length, info&(1<<24) != 0, external); err != nil {
				return nil, fmt.Errorf("bofloader: Mach-O relocation %d in section %q: %w", index, section.name, err)
			}
		}
	}
	return result, nil
}

func preflightMachODysymtab(data, command []byte, symbolCount uint32) error {
	ranges := []struct {
		name       string
		indexField int
		countField int
	}{
		{"local symbols", 8, 12},
		{"external symbols", 16, 20},
		{"undefined symbols", 24, 28},
	}
	for _, item := range ranges {
		index := binary.LittleEndian.Uint32(command[item.indexField : item.indexField+4])
		count := binary.LittleEndian.Uint32(command[item.countField : item.countField+4])
		if index > symbolCount || count > symbolCount-index {
			return fmt.Errorf("bofloader: Mach-O LC_DYSYMTAB %s range exceeds the symbol table", item.name)
		}
	}
	tables := []struct {
		name        string
		offsetField int
		countField  int
		entrySize   uint64
		limit       uint64
	}{
		{"table of contents", 32, 36, 8, maxObjectSymbols},
		{"module table", 40, 44, 56, maxObjectSymbols},
		{"external references", 48, 52, 4, maxObjectSymbols},
		{"indirect symbols", 56, 60, 4, maxObjectSymbols},
		{"external relocations", 64, 68, 8, maxObjectRelocations},
		{"local relocations", 72, 76, 8, maxObjectRelocations},
	}
	for _, table := range tables {
		offset := uint64(binary.LittleEndian.Uint32(command[table.offsetField : table.offsetField+4]))
		count := uint64(binary.LittleEndian.Uint32(command[table.countField : table.countField+4]))
		if count > table.limit {
			return fmt.Errorf("bofloader: Mach-O LC_DYSYMTAB %s count %d exceeds %d", table.name, count, table.limit)
		}
		size, ok := checkedMulUint64(count, table.entrySize)
		if !ok || !parserRangeWithin(data, offset, size) {
			return fmt.Errorf("bofloader: Mach-O LC_DYSYMTAB %s extends outside the object", table.name)
		}
	}
	return nil
}

func decodeMachORelocations(data []byte, object *objectFile, preflight *machoPreflight, rawSection machoRawSection, targetSection int, sectionMap []int, sectionSymbols map[int]uint32) ([]objectRelocation, error) {
	result := make([]objectRelocation, 0, rawSection.nreloc)
	var pendingAddend *int64
	var pendingOffset uint64
	for index := uint32(0); index < rawSection.nreloc; index++ {
		relocationOffset := uint64(rawSection.reloff) + uint64(index)*8
		address := binary.LittleEndian.Uint32(data[relocationOffset : relocationOffset+4])
		info := binary.LittleEndian.Uint32(data[relocationOffset+4 : relocationOffset+8])
		typeID := uint32(info >> 28)
		length := uint8((info >> 25) & 0x3)
		width := uint8(1) << length
		external := info&(1<<27) != 0
		value := info & 0x00ffffff
		if object.arch == "arm64" && typeID == machoARM64RelocAddend {
			if pendingAddend != nil {
				return nil, fmt.Errorf("bofloader: Mach-O ARM64 ADDEND relocation in %q is not followed by a target relocation", rawSection.name)
			}
			addend := signExtend(uint64(value), 24)
			pendingAddend = &addend
			pendingOffset = uint64(address)
			continue
		}

		symbolIndex := value
		if !external {
			rawTarget := int(value) - 1
			if rawTarget < 0 || rawTarget >= len(sectionMap) || sectionMap[rawTarget] < 0 {
				return nil, fmt.Errorf("bofloader: Mach-O relocation in %q references discarded section %d", rawSection.name, value)
			}
			if object.arch == "arm64" && (typeID == machoARM64RelocPage21 || typeID == machoARM64RelocPageOff12 ||
				typeID == machoARM64RelocGOTLoadPage21 || typeID == machoARM64RelocGOTLoadPageOff12 || typeID == machoARM64RelocPointerToGOT) {
				return nil, fmt.Errorf("bofloader: Mach-O ARM64 relocation type %d in %q requires an external symbol; section-ordinal form is unsupported", typeID, rawSection.name)
			}
			var ok bool
			symbolIndex, ok = sectionSymbols[rawTarget]
			if !ok {
				symbolIndex = machoSyntheticSectionSymbol | uint32(rawTarget)
				if _, collision := object.symbols[symbolIndex]; collision {
					return nil, fmt.Errorf("bofloader: Mach-O synthetic section symbol index collision")
				}
				object.symbols[symbolIndex] = objectSymbol{
					index:   symbolIndex,
					name:    preflight.sections[rawTarget].seg + "," + preflight.sections[rawTarget].name,
					section: sectionMap[rawTarget],
				}
				sectionSymbols[rawTarget] = symbolIndex
			}
		}
		relocation := objectRelocation{
			section: targetSection,
			offset:  uint64(address),
			typeID:  typeID,
			symbol:  symbolIndex,
			width:   width,
		}
		var zeroLocation [8]byte
		location := zeroLocation[:width]
		sectionType := rawSection.flags & machoSectionTypeMask
		if sectionType != machoSectionZeroFill && sectionType != machoSectionGBZeroFill {
			start := uint64(rawSection.offset) + relocation.offset
			location = data[start : start+uint64(width)]
		}
		if object.arch == "arm64" {
			if validateErr := validateMachOARM64EmbeddedAddend(typeID, location); validateErr != nil {
				return nil, fmt.Errorf("bofloader: Mach-O relocation in %q at %#x: %w", rawSection.name, relocation.offset, validateErr)
			}
		}
		if pendingAddend != nil {
			if pendingOffset != relocation.offset {
				return nil, fmt.Errorf("bofloader: Mach-O ARM64 ADDEND relocation at %#x is paired with relocation at %#x", pendingOffset, relocation.offset)
			}
			if typeID != machoARM64RelocBranch26 && typeID != machoARM64RelocPage21 && typeID != machoARM64RelocPageOff12 {
				return nil, fmt.Errorf("bofloader: Mach-O ARM64 ADDEND relocation cannot precede relocation type %d", typeID)
			}
			if !external {
				return nil, fmt.Errorf("bofloader: Mach-O ARM64 ADDEND paired with a section-ordinal relocation is unsupported")
			}
			relocation.hasAdd = true
			relocation.addend = *pendingAddend
			pendingAddend = nil
		}
		if !external {
			addend, addErr := normalizeMachOLocalAddend(object.arch, typeID, location, rawSection, preflight.sections[int(value)-1], relocation.offset)
			if addErr != nil {
				return nil, fmt.Errorf("bofloader: normalize Mach-O relocation in %q at %#x: %w", rawSection.name, relocation.offset, addErr)
			}
			if !relocation.hasAdd {
				relocation.hasAdd = true
				relocation.addend = addend
			}
		}
		if object.arch == "arm64" && external {
			if symbol, ok := object.symbols[symbolIndex]; ok && symbol.section == sectionUndefined {
				callbackName := normalizeImportedSymbol(symbol.name)
				if callbackName == "BeaconPrintf" || callbackName == "BeaconFormatPrintf" {
					return nil, fmt.Errorf("bofloader: Mach-O arm64 BOFs cannot import variadic callback %q because Apple's arm64 variadic ABI is unsupported; use BeaconOutput", symbol.name)
				}
			}
		}
		result = append(result, relocation)
	}
	if pendingAddend != nil {
		return nil, fmt.Errorf("bofloader: Mach-O ARM64 ADDEND relocation in %q is missing its target relocation", rawSection.name)
	}
	return result, nil
}

func normalizeMachOLocalAddend(arch string, typeID uint32, location []byte, source, target machoRawSection, offset uint64) (int64, error) {
	switch arch {
	case "amd64":
		switch typeID {
		case machoX86RelocUnsigned:
			var value uint64
			switch len(location) {
			case 4:
				value = uint64(binary.LittleEndian.Uint32(location))
			case 8:
				value = binary.LittleEndian.Uint64(location)
			default:
				return 0, fmt.Errorf("UNSIGNED relocation has invalid width %d", len(location))
			}
			if value < target.addr || value-target.addr > target.size {
				return 0, fmt.Errorf("absolute value %#x does not reference target section %#x..%#x", value, target.addr, target.addr+target.size)
			}
			return int64(value - target.addr), nil
		case machoX86RelocSigned, machoX86RelocBranch, machoX86RelocSigned1, machoX86RelocSigned2, machoX86RelocSigned4:
			encoded := int64(int32(binary.LittleEndian.Uint32(location)))
			bias := uint64(4)
			switch typeID {
			case machoX86RelocSigned1:
				bias = 5
			case machoX86RelocSigned2:
				bias = 6
			case machoX86RelocSigned4:
				bias = 8
			}
			place, ok := checkedAddUint64(source.addr, offset)
			if !ok {
				return 0, fmt.Errorf("source address overflow")
			}
			reference, ok := checkedAddUint64(place, bias)
			if !ok {
				return 0, fmt.Errorf("source reference overflow")
			}
			absolute, err := addSigned(reference, encoded)
			if err != nil {
				return 0, err
			}
			if absolute < target.addr || absolute-target.addr > target.size {
				return 0, fmt.Errorf("PC-relative target %#x does not reference target section %#x..%#x", absolute, target.addr, target.addr+target.size)
			}
			return int64(absolute - target.addr), nil
		default:
			return 0, fmt.Errorf("section-ordinal relocation type %d is unsupported", typeID)
		}
	case "arm64":
		switch typeID {
		case machoARM64RelocUnsigned:
			value := binary.LittleEndian.Uint64(location)
			if value < target.addr || value-target.addr > target.size {
				return 0, fmt.Errorf("absolute value %#x does not reference target section %#x..%#x", value, target.addr, target.addr+target.size)
			}
			return int64(value - target.addr), nil
		case machoARM64RelocBranch26:
			word := binary.LittleEndian.Uint32(location)
			delta := signExtend(uint64(word&0x03ffffff), 26) << 2
			place, ok := checkedAddUint64(source.addr, offset)
			if !ok {
				return 0, fmt.Errorf("source address overflow")
			}
			absolute, err := addSigned(place, delta)
			if err != nil {
				return 0, err
			}
			if absolute < target.addr || absolute-target.addr > target.size {
				return 0, fmt.Errorf("branch target %#x does not reference target section %#x..%#x", absolute, target.addr, target.addr+target.size)
			}
			return int64(absolute - target.addr), nil
		default:
			return 0, fmt.Errorf("section-ordinal relocation type %d is unsupported", typeID)
		}
	default:
		return 0, fmt.Errorf("unsupported Mach-O architecture %q", arch)
	}
}

func machoSectionMapping(section machoRawSection) (bool, protection, error) {
	if section.flags&machoSectionAttrDebug != 0 || section.seg == "__DWARF" ||
		(section.seg == "__LD" && section.name == "__compact_unwind") || section.name == "__eh_frame" {
		return false, 0, nil
	}
	sectionType := section.flags & machoSectionTypeMask
	if sectionType >= machoSectionThreadLocalRegular && sectionType <= machoSectionThreadLocalInitFuncs {
		return false, 0, fmt.Errorf("bofloader: Mach-O TLS section %q is unsupported", section.name)
	}
	switch sectionType {
	case machoSectionRegular, machoSectionZeroFill, machoSectionCStringLiterals,
		machoSectionFourByteLiterals, machoSectionEightByteLiterals, machoSectionLiteralPointers,
		machoSectionCoalesced, machoSectionGBZeroFill, machoSectionSixteenByteLiterals:
	default:
		return false, 0, fmt.Errorf("bofloader: Mach-O section %q uses unsupported type %d", section.name, sectionType)
	}
	if section.flags&(machoSectionAttrPureInstructions|machoSectionAttrSomeInstructions) != 0 {
		return true, protRead | protExec, nil
	}
	if section.seg == "__TEXT" || section.seg == "__DATA_CONST" {
		return true, protRead, nil
	}
	return true, protRead | protWrite, nil
}

func machoSymbolName(stringsData []byte, offset uint32) string {
	nameData := stringsData[offset:]
	end := bytes.IndexByte(nameData, 0)
	name := string(nameData[:end])
	return name
}

func fixedMachOName(data []byte) string {
	if end := bytes.IndexByte(data, 0); end >= 0 {
		data = data[:end]
	}
	return string(data)
}

func bitswap32(value uint32) uint32 {
	return value>>24 | value>>8&0x0000ff00 | value<<8&0x00ff0000 | value<<24
}
