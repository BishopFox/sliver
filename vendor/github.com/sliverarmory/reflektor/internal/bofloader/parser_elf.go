package bofloader

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	armELFEABIMask  = 0xff000000
	armELFEABI5     = 0x05000000
	armELFFloatSoft = 0x00000200
	armELFFloatHard = 0x00000400

	riscvELFRVC            = 0x00000001
	riscvELFFloatABIMask   = 0x00000006
	riscvELFFloatABIDouble = 0x00000004
	riscvELFRVE            = 0x00000008
	riscvELFTSO            = 0x00000010
	riscvELFKnownFlags     = riscvELFRVC | riscvELFFloatABIMask | riscvELFRVE | riscvELFTSO

	ppc64ELFABI          = 0x00000003
	ppc64ELFABI2         = 0x00000002
	ppc64LocalEntryMask  = 0xe0
	ppc64ReservedSTOther = 0x1c
)

func parseELF(data []byte) (object *objectFile, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			object = nil
			err = fmt.Errorf("bofloader: malformed ELF object: %v", recovered)
		}
	}()

	if err := preflightELFHeader(data); err != nil {
		return nil, err
	}
	file, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bofloader: parse ELF object: %w", err)
	}
	defer file.Close()
	if file.Type != elf.ET_REL {
		return nil, fmt.Errorf("bofloader: expected an ELF relocatable object, got %s", file.Type)
	}
	if file.Data != elf.ELFDATA2LSB {
		return nil, fmt.Errorf("bofloader: only little-endian ELF objects are supported")
	}

	arch := ""
	switch {
	case file.Machine == elf.EM_386 && file.Class == elf.ELFCLASS32:
		arch = "386"
	case file.Machine == elf.EM_ARM && file.Class == elf.ELFCLASS32:
		flags := binary.LittleEndian.Uint32(data[36:40])
		if flags&armELFEABIMask != armELFEABI5 || flags&(armELFFloatSoft|armELFFloatHard) != armELFFloatHard {
			return nil, fmt.Errorf("bofloader: ELF/arm BOFs require EABI5 hard-float flags, got %#08x", flags)
		}
		arch = "arm"
	case file.Machine == elf.EM_X86_64 && file.Class == elf.ELFCLASS64:
		arch = "amd64"
	case file.Machine == elf.EM_AARCH64 && file.Class == elf.ELFCLASS64:
		arch = "arm64"
	case file.Machine == elf.EM_RISCV && file.Class == elf.ELFCLASS64:
		flags := binary.LittleEndian.Uint32(data[48:52])
		if flags&riscvELFFloatABIMask != riscvELFFloatABIDouble {
			return nil, fmt.Errorf("bofloader: ELF/riscv64 BOFs require the LP64D double-float ABI, got flags %#08x", flags)
		}
		if flags&riscvELFRVE != 0 {
			return nil, fmt.Errorf("bofloader: ELF/riscv64 BOFs cannot use the RV32E register ABI, got flags %#08x", flags)
		}
		if flags&riscvELFTSO != 0 {
			return nil, fmt.Errorf("bofloader: ELF/riscv64 BOFs requiring RVTSO are unsupported, got flags %#08x", flags)
		}
		if unknown := flags &^ riscvELFKnownFlags; unknown != 0 {
			return nil, fmt.Errorf("bofloader: ELF/riscv64 BOF has unknown flags %#08x", unknown)
		}
		arch = "riscv64"
	case file.Machine == elf.EM_PPC64 && file.Class == elf.ELFCLASS64:
		flags := binary.LittleEndian.Uint32(data[48:52])
		if flags&ppc64ELFABI != ppc64ELFABI2 || flags&^uint32(ppc64ELFABI) != 0 {
			return nil, fmt.Errorf("bofloader: ELF/ppc64le BOFs require the ELFv2 ABI flags, got %#08x", flags)
		}
		arch = "ppc64le"
	default:
		return nil, fmt.Errorf("bofloader: unsupported ELF machine/class %s/%s", file.Machine, file.Class)
	}
	mappedSectionBytes, err := preflightELFSections(file, data)
	if err != nil {
		return nil, err
	}

	result := &objectFile{
		format:  "elf",
		arch:    arch,
		symbols: make(map[uint32]objectSymbol),
	}
	sectionMap := make([]int, len(file.Sections))
	for index := range sectionMap {
		sectionMap[index] = sectionDiscarded
	}
	for sourceIndex, section := range file.Sections {
		if section.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		if section.Flags&elf.SHF_TLS != 0 {
			return nil, fmt.Errorf("bofloader: ELF TLS section %q is unsupported", section.Name)
		}
		alignment := section.Addralign
		if alignment == 0 {
			alignment = 1
		}
		if alignment&(alignment-1) != 0 {
			return nil, fmt.Errorf("bofloader: ELF section %q has invalid alignment %d", section.Name, alignment)
		}
		if alignment > maxImageSize || section.Size > maxImageSize {
			return nil, fmt.Errorf("bofloader: ELF section %q exceeds loader limits", section.Name)
		}
		var contents []byte
		if section.Type != elf.SHT_NOBITS && section.Size != 0 {
			contents, err = section.Data()
			if err != nil {
				return nil, fmt.Errorf("bofloader: read ELF section %q: %w", section.Name, err)
			}
			if uint64(len(contents)) > section.Size {
				return nil, fmt.Errorf("bofloader: ELF section %q data exceeds declared size", section.Name)
			}
		}
		protection := protRead
		if section.Flags&elf.SHF_WRITE != 0 {
			protection |= protWrite
		}
		if section.Flags&elf.SHF_EXECINSTR != 0 {
			protection |= protExec
		}
		sectionMap[sourceIndex] = len(result.sections)
		result.sections = append(result.sections, objectSection{
			name:       section.Name,
			data:       contents,
			size:       section.Size,
			align:      alignment,
			protection: protection,
			mapped:     section.Size != 0,
		})
	}

	symbols, err := file.Symbols()
	if err != nil {
		return nil, fmt.Errorf("bofloader: read ELF symbols: %w", err)
	}
	commonOffset := uint64(0)
	commonAlignment := uint64(1)
	commonSymbols := make([]uint32, 0)
	for position, raw := range symbols {
		index := uint32(position + 1) // debug/elf omits ELF symbol table entry zero.
		var section int
		value := raw.Value
		switch raw.Section {
		case elf.SHN_UNDEF:
			section = sectionUndefined
		case elf.SHN_ABS:
			section = sectionAbsolute
		case elf.SHN_COMMON:
			alignment := raw.Value
			if alignment == 0 {
				alignment = 1
			}
			if alignment&(alignment-1) != 0 {
				return nil, fmt.Errorf("bofloader: ELF common symbol %q has invalid alignment %d", raw.Name, alignment)
			}
			if alignment > maxImageSize {
				return nil, fmt.Errorf("bofloader: ELF common symbol %q alignment exceeds loader limits", raw.Name)
			}
			if alignment > commonAlignment {
				commonAlignment = alignment
			}
			commonOffset = alignUp(commonOffset, alignment)
			value = commonOffset
			if raw.Size > maxImageSize-commonOffset {
				return nil, fmt.Errorf("bofloader: ELF common symbols exceed %d bytes", maxImageSize)
			}
			commonOffset += raw.Size
			section = sectionCommon
			commonSymbols = append(commonSymbols, index)
		default:
			rawSection := int(raw.Section)
			if rawSection < 0 || rawSection >= len(sectionMap) {
				return nil, fmt.Errorf("bofloader: ELF symbol %q references invalid section %d", raw.Name, rawSection)
			}
			section = sectionMap[rawSection]
		}
		localEntry := uint64(0)
		if arch == "ppc64le" {
			if raw.Other&ppc64ReservedSTOther != 0 {
				return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q has reserved st_other bits %#02x", raw.Name, raw.Other&ppc64ReservedSTOther)
			}
			encoding := (raw.Other & ppc64LocalEntryMask) >> 5
			switch encoding {
			case 1:
				return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q uses unsupported local-entry encoding 1 (NOTOC caller-save-r2 semantics)", raw.Name)
			case 7:
				return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q uses reserved local-entry encoding 7", raw.Name)
			}
			localEntry = ppc64LocalEntryOffset(raw.Other)
			if localEntry != 0 {
				if raw.Section == elf.SHN_UNDEF || elf.ST_TYPE(raw.Info) != elf.STT_FUNC {
					return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q has local-entry metadata but is not a defined function", raw.Name)
				}
				if raw.Size != 0 && localEntry >= raw.Size {
					return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q local entry offset %#x exceeds function size %#x", raw.Name, localEntry, raw.Size)
				}
				if section >= 0 {
					sectionSize := result.sections[section].size
					if value > sectionSize || localEntry > sectionSize-value {
						return nil, fmt.Errorf("bofloader: ELF/ppc64le symbol %q local entry exceeds section %q", raw.Name, result.sections[section].name)
					}
				}
			}
		}
		result.symbols[index] = objectSymbol{
			index:      index,
			name:       raw.Name,
			section:    section,
			value:      value,
			size:       raw.Size,
			localEntry: localEntry,
			weak:       elf.ST_BIND(raw.Info) == elf.STB_WEAK,
		}
	}
	if commonOffset != 0 {
		if mappedSectionBytes > maxImageSize || commonOffset > maxImageSize-mappedSectionBytes {
			return nil, fmt.Errorf("bofloader: cumulative ELF mapped section size exceeds %d bytes", maxImageSize)
		}
		commonSection := len(result.sections)
		result.sections = append(result.sections, objectSection{
			name:       ".common",
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

	symbolTableIndex := -1
	for index, section := range file.Sections {
		if section.Type == elf.SHT_SYMTAB {
			symbolTableIndex = index
			break
		}
	}
	for _, section := range file.Sections {
		if section.Type != elf.SHT_REL && section.Type != elf.SHT_RELA {
			continue
		}
		if symbolTableIndex < 0 || int(section.Link) != symbolTableIndex {
			return nil, fmt.Errorf("bofloader: ELF relocation section %q uses an unsupported symbol table", section.Name)
		}
		targetRaw := int(section.Info)
		if targetRaw < 0 || targetRaw >= len(sectionMap) || sectionMap[targetRaw] < 0 {
			continue
		}
		targetSection := sectionMap[targetRaw]
		if !result.sections[targetSection].mapped {
			continue
		}
		if (arch == "riscv64" || arch == "ppc64le") && section.Type != elf.SHT_RELA {
			return nil, fmt.Errorf("bofloader: ELF/%s relocation section %q must use RELA encoding", arch, section.Name)
		}
		contents, dataErr := section.Data()
		if dataErr != nil {
			return nil, fmt.Errorf("bofloader: read ELF relocation section %q: %w", section.Name, dataErr)
		}
		relocations, decodeErr := decodeELFRelocations(file, section, contents, targetSection)
		if decodeErr != nil {
			return nil, fmt.Errorf("bofloader: decode ELF relocation section %q: %w", section.Name, decodeErr)
		}
		for _, relocation := range relocations {
			if relocation.symbol != 0 {
				if _, ok := result.symbols[relocation.symbol]; !ok {
					return nil, fmt.Errorf("bofloader: ELF relocation references invalid symbol %d", relocation.symbol)
				}
			}
			result.relocations = append(result.relocations, relocation)
		}
	}
	if arch == "riscv64" {
		if err := prepareELFRISCV64Pairs(result); err != nil {
			return nil, err
		}
	}
	if arch == "ppc64le" {
		if err := validateELFPPC64LERelocations(result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func ppc64LocalEntryOffset(other byte) uint64 {
	encoding := (other & ppc64LocalEntryMask) >> 5
	return (uint64(1) << encoding) &^ uint64(3)
}

func preflightELFHeader(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("bofloader: truncated ELF identification")
	}
	var headerSize, programCountField, sectionOffsetField, sectionEntrySizeField, sectionCountField, sectionNameIndexField int
	var expectedSectionEntrySize uint64
	switch elf.Class(data[elf.EI_CLASS]) {
	case elf.ELFCLASS32:
		headerSize = 52
		programCountField = 44
		sectionOffsetField = 32
		sectionEntrySizeField = 46
		sectionCountField = 48
		sectionNameIndexField = 50
		expectedSectionEntrySize = 40
	case elf.ELFCLASS64:
		headerSize = 64
		programCountField = 56
		sectionOffsetField = 40
		sectionEntrySizeField = 58
		sectionCountField = 60
		sectionNameIndexField = 62
		expectedSectionEntrySize = 64
	default:
		return nil // debug/elf reports the authoritative class error.
	}
	if len(data) < headerSize {
		return fmt.Errorf("bofloader: truncated ELF header")
	}
	if elf.Data(data[elf.EI_DATA]) != elf.ELFDATA2LSB {
		return fmt.Errorf("bofloader: only little-endian ELF objects are supported")
	}
	if elf.Type(binary.LittleEndian.Uint16(data[16:18])) != elf.ET_REL {
		return fmt.Errorf("bofloader: expected an ELF relocatable object")
	}
	if programCount := binary.LittleEndian.Uint16(data[programCountField:]); programCount != 0 {
		return fmt.Errorf("bofloader: ELF relocatable object has %d program headers", programCount)
	}
	sectionOffset := uint64(binary.LittleEndian.Uint32(data[sectionOffsetField:]))
	if headerSize == 64 {
		sectionOffset = binary.LittleEndian.Uint64(data[sectionOffsetField:])
	}
	sectionEntrySize := uint64(binary.LittleEndian.Uint16(data[sectionEntrySizeField:]))
	sectionCount := uint64(binary.LittleEndian.Uint16(data[sectionCountField:]))
	sectionNameIndex := uint64(binary.LittleEndian.Uint16(data[sectionNameIndexField:]))
	if sectionCount == 0 && sectionOffset != 0 {
		if sectionEntrySize < expectedSectionEntrySize || !parserRangeWithin(data, sectionOffset, expectedSectionEntrySize) {
			return fmt.Errorf("bofloader: ELF extended section count is outside the object")
		}
		sectionSizeField := sectionOffset + 20
		if headerSize == 64 {
			sectionSizeField = sectionOffset + 32
			sectionCount = binary.LittleEndian.Uint64(data[sectionSizeField:])
		} else {
			sectionCount = uint64(binary.LittleEndian.Uint32(data[sectionSizeField:]))
		}
	}
	if sectionCount > maxObjectSections {
		return fmt.Errorf("bofloader: ELF section count %d exceeds %d", sectionCount, maxObjectSections)
	}
	if sectionCount == 0 {
		return nil
	}
	if sectionEntrySize < expectedSectionEntrySize {
		return fmt.Errorf("bofloader: ELF section header size %d is smaller than %d", sectionEntrySize, expectedSectionEntrySize)
	}
	tableSize, ok := checkedMulUint64(sectionCount, sectionEntrySize)
	if !ok || !parserRangeWithin(data, sectionOffset, tableSize) {
		return fmt.Errorf("bofloader: ELF section header table is outside the object")
	}
	if sectionNameIndex == uint64(elf.SHN_XINDEX) {
		sectionZero := sectionOffset
		if headerSize == 64 {
			sectionNameIndex = uint64(binary.LittleEndian.Uint32(data[sectionZero+40:]))
		} else {
			sectionNameIndex = uint64(binary.LittleEndian.Uint32(data[sectionZero+24:]))
		}
	}
	if sectionNameIndex >= sectionCount {
		return fmt.Errorf("bofloader: ELF section-name string table index %d exceeds section count %d", sectionNameIndex, sectionCount)
	}

	// debug/elf may decompress the section-name string table while constructing
	// File, before parseELF can inspect the parsed sections. BOFs do not need
	// compressed sections, so reject their raw headers before elf.NewFile can
	// expand attacker-controlled compressed data.
	for index := uint64(0); index < sectionCount; index++ {
		headerOffset := sectionOffset + index*sectionEntrySize
		_, sectionType, flags, rawOffset, rawSize, _ := rawELFSection(data, headerOffset, headerSize == 64)
		if flags&uint64(elf.SHF_COMPRESSED) != 0 {
			return fmt.Errorf("bofloader: compressed ELF sections are unsupported")
		}
		if elf.SectionType(sectionType) != elf.SHT_NOBITS && !parserRangeWithin(data, rawOffset, rawSize) {
			return fmt.Errorf("bofloader: ELF section %d data is outside the object", index)
		}
	}

	var sectionNames []byte
	if sectionNameIndex != 0 {
		nameHeader := sectionOffset + sectionNameIndex*sectionEntrySize
		_, sectionType, _, rawOffset, rawSize, _ := rawELFSection(data, nameHeader, headerSize == 64)
		if elf.SectionType(sectionType) != elf.SHT_STRTAB || !parserRangeWithin(data, rawOffset, rawSize) {
			return fmt.Errorf("bofloader: ELF section-name string table is invalid")
		}
		if rawSize > maxObjectNameBytes {
			return fmt.Errorf("bofloader: ELF section-name string table exceeds %d bytes", maxObjectNameBytes)
		}
		sectionNames = data[rawOffset : rawOffset+rawSize]
	}
	nameCache := make(map[uint64]uint64)
	var resolvedNameBytes uint64
	for index := uint64(0); index < sectionCount; index++ {
		headerOffset := sectionOffset + index*sectionEntrySize
		nameOffset, _, _, _, _, _ := rawELFSection(data, headerOffset, headerSize == 64)
		if len(sectionNames) == 0 {
			if nameOffset != 0 {
				return fmt.Errorf("bofloader: ELF section %d has a name without a section-name string table", index)
			}
			continue
		}
		length, nameErr := parserStringLength(sectionNames, uint64(nameOffset), 0, nameCache)
		if nameErr != nil {
			return fmt.Errorf("bofloader: ELF section %d name: %w", index, nameErr)
		}
		resolvedNameBytes, ok = checkedAddUint64(resolvedNameBytes, length)
		if !ok || resolvedNameBytes > maxObjectNameBytes {
			return fmt.Errorf("bofloader: cumulative ELF section-name bytes exceed %d", maxObjectNameBytes)
		}
		name := sectionNames[uint64(nameOffset) : uint64(nameOffset)+length]
		if bytes.HasPrefix(name, []byte(".zdebug")) {
			return fmt.Errorf("bofloader: compressed ELF sections are unsupported")
		}
	}
	return nil
}

func preflightELFSections(file *elf.File, data []byte) (uint64, error) {
	if len(file.Sections) > maxObjectSections {
		return 0, fmt.Errorf("bofloader: ELF section count %d exceeds %d", len(file.Sections), maxObjectSections)
	}
	var mappedBytes, symbolCount, relocationCount uint64
	var symbolTable *elf.Section
	for _, section := range file.Sections {
		// debug/elf also recognizes the legacy .zdebug ZLIB convention lazily
		// from section names. Reject it before any parser-owned Section.Data call
		// can expand compressed metadata.
		if strings.HasPrefix(section.Name, ".zdebug") {
			return 0, fmt.Errorf("bofloader: compressed ELF sections are unsupported")
		}
		if section.Flags&elf.SHF_ALLOC != 0 {
			var ok bool
			mappedBytes, ok = checkedAddUint64(mappedBytes, section.Size)
			if !ok || mappedBytes > maxImageSize {
				return 0, fmt.Errorf("bofloader: cumulative ELF mapped section size exceeds %d bytes", maxImageSize)
			}
		}

		var entrySize uint64
		switch section.Type {
		case elf.SHT_SYMTAB:
			if symbolTable != nil {
				return 0, fmt.Errorf("bofloader: multiple ELF symbol tables are unsupported")
			}
			symbolTable = section
			if file.Class == elf.ELFCLASS64 {
				entrySize = uint64(binary.Size(elf.Sym64{}))
			} else {
				entrySize = uint64(binary.Size(elf.Sym32{}))
			}
			if section.Size%entrySize != 0 {
				return 0, fmt.Errorf("bofloader: ELF symbol section %q size %d is not a multiple of %d", section.Name, section.Size, entrySize)
			}
			count := section.Size / entrySize
			var ok bool
			symbolCount, ok = checkedAddUint64(symbolCount, count)
			if !ok || symbolCount > maxObjectSymbols {
				return 0, fmt.Errorf("bofloader: ELF symbol count exceeds %d", maxObjectSymbols)
			}
		case elf.SHT_REL, elf.SHT_RELA:
			switch {
			case file.Class == elf.ELFCLASS64 && section.Type == elf.SHT_RELA:
				entrySize = uint64(binary.Size(elf.Rela64{}))
			case file.Class == elf.ELFCLASS64:
				entrySize = uint64(binary.Size(elf.Rel64{}))
			case section.Type == elf.SHT_RELA:
				entrySize = uint64(binary.Size(elf.Rela32{}))
			default:
				entrySize = uint64(binary.Size(elf.Rel32{}))
			}
			if section.Size%entrySize != 0 {
				return 0, fmt.Errorf("bofloader: ELF relocation section %q size %d is not a multiple of %d", section.Name, section.Size, entrySize)
			}
			count := section.Size / entrySize
			var ok bool
			relocationCount, ok = checkedAddUint64(relocationCount, count)
			if !ok || relocationCount > maxObjectRelocations {
				return 0, fmt.Errorf("bofloader: ELF relocation count exceeds %d", maxObjectRelocations)
			}
		}
	}
	if symbolTable == nil {
		return 0, fmt.Errorf("bofloader: ELF object has no symbol table")
	}
	expectedSymbolSize := uint64(binary.Size(elf.Sym32{}))
	if file.Class == elf.ELFCLASS64 {
		expectedSymbolSize = uint64(binary.Size(elf.Sym64{}))
	}
	if symbolTable.Entsize != expectedSymbolSize || symbolTable.Size%expectedSymbolSize != 0 {
		return 0, fmt.Errorf("bofloader: ELF symbol table has invalid entry size")
	}
	if int(symbolTable.Link) <= 0 || int(symbolTable.Link) >= len(file.Sections) {
		return 0, fmt.Errorf("bofloader: ELF symbol table has invalid string-table link %d", symbolTable.Link)
	}
	stringTable := file.Sections[symbolTable.Link]
	if stringTable.Type != elf.SHT_STRTAB || stringTable.Size > maxObjectNameBytes ||
		!parserRangeWithin(data, stringTable.Offset, stringTable.FileSize) ||
		!parserRangeWithin(data, symbolTable.Offset, symbolTable.FileSize) {
		return 0, fmt.Errorf("bofloader: ELF symbol or string table is outside parser limits")
	}
	symbolData := data[symbolTable.Offset : symbolTable.Offset+symbolTable.FileSize]
	stringData := data[stringTable.Offset : stringTable.Offset+stringTable.FileSize]
	nameCache := make(map[uint64]uint64)
	var resolvedSymbolNameBytes uint64
	for offset := uint64(0); offset < uint64(len(symbolData)); offset += expectedSymbolSize {
		nameOffset := uint64(binary.LittleEndian.Uint32(symbolData[offset:]))
		length, nameErr := parserStringLength(stringData, nameOffset, 0, nameCache)
		if nameErr != nil {
			return 0, fmt.Errorf("bofloader: ELF symbol name: %w", nameErr)
		}
		var ok bool
		resolvedSymbolNameBytes, ok = checkedAddUint64(resolvedSymbolNameBytes, length)
		if !ok || resolvedSymbolNameBytes > maxObjectNameBytes {
			return 0, fmt.Errorf("bofloader: cumulative ELF symbol-name bytes exceed %d", maxObjectNameBytes)
		}
	}
	return mappedBytes, nil
}

func rawELFSection(data []byte, offset uint64, is64 bool) (name uint32, sectionType uint32, flags, rawOffset, rawSize uint64, link uint32) {
	name = binary.LittleEndian.Uint32(data[offset:])
	sectionType = binary.LittleEndian.Uint32(data[offset+4:])
	if is64 {
		flags = binary.LittleEndian.Uint64(data[offset+8:])
		rawOffset = binary.LittleEndian.Uint64(data[offset+24:])
		rawSize = binary.LittleEndian.Uint64(data[offset+32:])
		link = binary.LittleEndian.Uint32(data[offset+40:])
		return
	}
	flags = uint64(binary.LittleEndian.Uint32(data[offset+8:]))
	rawOffset = uint64(binary.LittleEndian.Uint32(data[offset+16:]))
	rawSize = uint64(binary.LittleEndian.Uint32(data[offset+20:]))
	link = binary.LittleEndian.Uint32(data[offset+24:])
	return
}

func parserStringLength(table []byte, offset, minimum uint64, cache map[uint64]uint64) (uint64, error) {
	if offset < minimum || offset >= uint64(len(table)) {
		return 0, fmt.Errorf("string offset %d is outside a %d-byte table", offset, len(table))
	}
	if length, ok := cache[offset]; ok {
		return length, nil
	}
	remaining := table[offset:]
	scan := len(remaining)
	if scan > maxObjectNameSize+1 {
		scan = maxObjectNameSize + 1
	}
	terminator := bytes.IndexByte(remaining[:scan], 0)
	if terminator < 0 {
		if len(remaining) > maxObjectNameSize {
			return 0, fmt.Errorf("name at offset %d exceeds %d bytes", offset, maxObjectNameSize)
		}
		return 0, fmt.Errorf("name at offset %d is not NUL-terminated", offset)
	}
	length := uint64(terminator)
	cache[offset] = length
	return length, nil
}

func decodeELFRelocations(file *elf.File, section *elf.Section, contents []byte, targetSection int) ([]objectRelocation, error) {
	reader := bytes.NewReader(contents)
	result := make([]objectRelocation, 0)
	for reader.Len() != 0 {
		var relocation objectRelocation
		relocation.section = targetSection
		switch {
		case file.Class == elf.ELFCLASS64 && section.Type == elf.SHT_RELA:
			var raw elf.Rela64
			if err := binary.Read(reader, file.ByteOrder, &raw); err != nil {
				return nil, err
			}
			relocation.offset = raw.Off
			relocation.typeID = elf.R_TYPE64(raw.Info)
			relocation.symbol = elf.R_SYM64(raw.Info)
			relocation.addend = raw.Addend
			relocation.hasAdd = true
		case file.Class == elf.ELFCLASS64 && section.Type == elf.SHT_REL:
			var raw elf.Rel64
			if err := binary.Read(reader, file.ByteOrder, &raw); err != nil {
				return nil, err
			}
			relocation.offset = raw.Off
			relocation.typeID = elf.R_TYPE64(raw.Info)
			relocation.symbol = elf.R_SYM64(raw.Info)
		case file.Class == elf.ELFCLASS32 && section.Type == elf.SHT_RELA:
			var raw elf.Rela32
			if err := binary.Read(reader, file.ByteOrder, &raw); err != nil {
				return nil, err
			}
			relocation.offset = uint64(raw.Off)
			relocation.typeID = elf.R_TYPE32(raw.Info)
			relocation.symbol = elf.R_SYM32(raw.Info)
			relocation.addend = int64(raw.Addend)
			relocation.hasAdd = true
		case file.Class == elf.ELFCLASS32 && section.Type == elf.SHT_REL:
			var raw elf.Rel32
			if err := binary.Read(reader, file.ByteOrder, &raw); err != nil {
				return nil, err
			}
			relocation.offset = uint64(raw.Off)
			relocation.typeID = elf.R_TYPE32(raw.Info)
			relocation.symbol = elf.R_SYM32(raw.Info)
		default:
			return nil, fmt.Errorf("unsupported ELF relocation encoding")
		}
		result = append(result, relocation)
	}
	return result, nil
}
