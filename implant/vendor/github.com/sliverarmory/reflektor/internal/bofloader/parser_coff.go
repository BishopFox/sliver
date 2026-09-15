package bofloader

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
)

const (
	coffSectionLinkInfo           = 0x00000200
	coffSectionLinkRemove         = 0x00000800
	coffSectionRelocationOverflow = 0x01000000
	coffSectionAlignMask          = 0x00f00000
	coffStorageWeak               = 105
)

func parseCOFF(data []byte) (object *objectFile, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			object = nil
			err = fmt.Errorf("bofloader: malformed COFF object: %v", recovered)
		}
	}()

	if err := preflightCOFF(data); err != nil {
		return nil, err
	}
	file, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bofloader: parse COFF object: %w", err)
	}
	defer file.Close()
	if file.SizeOfOptionalHeader != 0 || file.OptionalHeader != nil {
		return nil, fmt.Errorf("bofloader: expected a relocatable COFF object, not a PE image")
	}

	arch := ""
	switch file.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		arch = "amd64"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		arch = "arm64"
	default:
		return nil, fmt.Errorf("bofloader: unsupported COFF machine 0x%x", file.Machine)
	}

	result := &objectFile{
		format:  "coff",
		arch:    arch,
		symbols: make(map[uint32]objectSymbol),
	}
	sectionMap := make([]int, len(file.Sections))
	sectionSizes := make([]uint64, len(file.Sections))
	for index := range sectionMap {
		sectionMap[index] = sectionDiscarded
		sectionSizes[index] = uint64(file.Sections[index].Size)
	}

	// Object-only section definitions keep the true uninitialized/COMDAT size
	// in an auxiliary record instead of the section header.
	for index := 0; index < len(file.COFFSymbols); {
		symbol := &file.COFFSymbols[index]
		if symbol.SectionNumber > 0 && int(symbol.SectionNumber) <= len(sectionSizes) && symbol.NumberOfAuxSymbols != 0 {
			if auxiliary, auxErr := file.COFFSymbolReadSectionDefAux(index); auxErr == nil {
				sectionIndex := int(symbol.SectionNumber) - 1
				if uint64(auxiliary.Size) > sectionSizes[sectionIndex] {
					sectionSizes[sectionIndex] = uint64(auxiliary.Size)
				}
			}
		}
		index += 1 + int(symbol.NumberOfAuxSymbols)
	}
	var mappedSectionBytes uint64
	for sourceIndex, section := range file.Sections {
		characteristics := section.Characteristics
		mapped := characteristics&(pe.IMAGE_SCN_CNT_CODE|pe.IMAGE_SCN_CNT_INITIALIZED_DATA|pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA|pe.IMAGE_SCN_MEM_READ|pe.IMAGE_SCN_MEM_WRITE|pe.IMAGE_SCN_MEM_EXECUTE) != 0
		if characteristics&(pe.IMAGE_SCN_MEM_DISCARDABLE|coffSectionLinkInfo|coffSectionLinkRemove) != 0 || !mapped {
			continue
		}
		var ok bool
		mappedSectionBytes, ok = checkedAddUint64(mappedSectionBytes, sectionSizes[sourceIndex])
		if !ok || mappedSectionBytes > maxImageSize {
			return nil, fmt.Errorf("bofloader: cumulative COFF mapped section size exceeds %d bytes", maxImageSize)
		}
	}

	for sourceIndex, section := range file.Sections {
		characteristics := section.Characteristics
		mapped := characteristics&(pe.IMAGE_SCN_CNT_CODE|pe.IMAGE_SCN_CNT_INITIALIZED_DATA|pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA|pe.IMAGE_SCN_MEM_READ|pe.IMAGE_SCN_MEM_WRITE|pe.IMAGE_SCN_MEM_EXECUTE) != 0
		if characteristics&(pe.IMAGE_SCN_MEM_DISCARDABLE|coffSectionLinkInfo|coffSectionLinkRemove) != 0 {
			mapped = false
		}
		if !mapped {
			continue
		}
		size := sectionSizes[sourceIndex]
		var contents []byte
		if section.Size != 0 && section.Offset != 0 {
			contents, err = section.Data()
			if err != nil {
				return nil, fmt.Errorf("bofloader: read COFF section %q: %w", section.Name, err)
			}
			if uint64(len(contents)) > size {
				size = uint64(len(contents))
			}
		}
		if size == 0 {
			// Keep a stable section index for symbols, while avoiding a mapping.
			sectionMap[sourceIndex] = len(result.sections)
			result.sections = append(result.sections, objectSection{name: section.Name, align: coffSectionAlignment(characteristics), mapped: false})
			continue
		}
		protection := protRead
		if characteristics&pe.IMAGE_SCN_MEM_WRITE != 0 || characteristics&pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA != 0 {
			protection |= protWrite
		}
		if characteristics&(pe.IMAGE_SCN_MEM_EXECUTE|pe.IMAGE_SCN_CNT_CODE) != 0 {
			protection |= protExec
		}
		sectionMap[sourceIndex] = len(result.sections)
		result.sections = append(result.sections, objectSection{
			name:       section.Name,
			data:       contents,
			size:       size,
			align:      coffSectionAlignment(characteristics),
			protection: protection,
			mapped:     true,
		})
	}

	primarySymbol := 0
	for rawIndex := 0; rawIndex < len(file.COFFSymbols); {
		raw := &file.COFFSymbols[rawIndex]
		if primarySymbol >= len(file.Symbols) {
			return nil, fmt.Errorf("bofloader: COFF primary symbol %d is missing", rawIndex)
		}
		name := file.Symbols[primarySymbol].Name
		primarySymbol++
		section := sectionUndefined
		switch {
		case raw.SectionNumber > 0 && int(raw.SectionNumber) <= len(sectionMap):
			section = sectionMap[int(raw.SectionNumber)-1]
		case raw.SectionNumber == -1:
			section = sectionAbsolute
		}
		result.symbols[uint32(rawIndex)] = objectSymbol{
			index:   uint32(rawIndex),
			name:    name,
			section: section,
			value:   uint64(raw.Value),
			weak:    raw.StorageClass == coffStorageWeak,
		}
		rawIndex += 1 + int(raw.NumberOfAuxSymbols)
	}
	if primarySymbol != len(file.Symbols) {
		return nil, fmt.Errorf("bofloader: COFF primary symbol count mismatch")
	}

	for sourceIndex, section := range file.Sections {
		targetSection := sectionMap[sourceIndex]
		if targetSection < 0 || targetSection >= len(result.sections) || !result.sections[targetSection].mapped {
			continue
		}
		for _, relocation := range section.Relocs {
			if _, ok := result.symbols[relocation.SymbolTableIndex]; !ok {
				return nil, fmt.Errorf("bofloader: COFF relocation in %q references invalid symbol %d", section.Name, relocation.SymbolTableIndex)
			}
			result.relocations = append(result.relocations, objectRelocation{
				section: targetSection,
				offset:  uint64(relocation.VirtualAddress),
				typeID:  uint32(relocation.Type),
				symbol:  relocation.SymbolTableIndex,
			})
		}
	}
	return result, nil
}

func preflightCOFF(data []byte) error {
	const (
		coffHeaderSize        = 20
		coffSectionHeaderSize = 40
		coffSymbolSize        = 18
		coffRelocationSize    = 10
	)
	if len(data) < coffHeaderSize {
		return fmt.Errorf("bofloader: truncated COFF header")
	}
	if data[0] == 'M' && data[1] == 'Z' {
		return fmt.Errorf("bofloader: expected a relocatable COFF object, not a PE image")
	}
	sectionCount := uint64(binary.LittleEndian.Uint16(data[2:4]))
	symbolTableOffset := uint64(binary.LittleEndian.Uint32(data[8:12]))
	symbolCount := uint64(binary.LittleEndian.Uint32(data[12:16]))
	optionalHeaderSize := uint64(binary.LittleEndian.Uint16(data[16:18]))
	if optionalHeaderSize != 0 {
		return fmt.Errorf("bofloader: expected a relocatable COFF object, not a PE image")
	}
	if sectionCount > maxObjectSections {
		return fmt.Errorf("bofloader: COFF section count %d exceeds %d", sectionCount, maxObjectSections)
	}
	if symbolCount > maxObjectSymbols {
		return fmt.Errorf("bofloader: COFF symbol count %d exceeds %d", symbolCount, maxObjectSymbols)
	}
	sectionTableSize, ok := checkedMulUint64(sectionCount, coffSectionHeaderSize)
	if !ok || !parserRangeWithin(data, coffHeaderSize, sectionTableSize) {
		return fmt.Errorf("bofloader: COFF section header table is outside the object")
	}
	if symbolCount != 0 && symbolTableOffset == 0 {
		return fmt.Errorf("bofloader: COFF symbol table offset is zero with %d symbols", symbolCount)
	}
	symbolTableSize, mulOK := checkedMulUint64(symbolCount, coffSymbolSize)
	symbolTableEnd, addOK := checkedAddUint64(symbolTableOffset, symbolTableSize)
	if !mulOK || !addOK || symbolTableEnd > uint64(^uint32(0)) ||
		(symbolCount != 0 && !parserRangeWithin(data, symbolTableOffset, symbolTableSize)) {
		return fmt.Errorf("bofloader: COFF symbol table is outside the object")
	}

	var stringTable []byte
	if symbolTableOffset != 0 {
		if !parserRangeWithin(data, symbolTableEnd, 4) {
			return fmt.Errorf("bofloader: COFF string-table length is outside the object")
		}
		stringTableSize := uint64(binary.LittleEndian.Uint32(data[symbolTableEnd:]))
		if stringTableSize < 4 || stringTableSize > maxObjectNameBytes || !parserRangeWithin(data, symbolTableEnd, stringTableSize) {
			return fmt.Errorf("bofloader: COFF string table is outside parser limits")
		}
		stringTable = data[symbolTableEnd : symbolTableEnd+stringTableSize]
	}

	var mappedBytes, relocationCount uint64
	sectionNameCache := make(map[uint64]uint64)
	var resolvedSectionNameBytes uint64
	for index := uint64(0); index < sectionCount; index++ {
		offset := uint64(coffHeaderSize) + index*coffSectionHeaderSize
		header := data[offset : offset+coffSectionHeaderSize]
		nameLength, nameErr := coffNameLength(header[:8], stringTable, sectionNameCache, true)
		if nameErr != nil {
			return fmt.Errorf("bofloader: COFF section %d name: %w", index, nameErr)
		}
		resolvedSectionNameBytes, ok = checkedAddUint64(resolvedSectionNameBytes, nameLength)
		if !ok || resolvedSectionNameBytes > maxObjectNameBytes {
			return fmt.Errorf("bofloader: cumulative COFF section-name bytes exceed %d", maxObjectNameBytes)
		}
		rawSize := uint64(binary.LittleEndian.Uint32(header[16:20]))
		rawOffset := uint64(binary.LittleEndian.Uint32(header[20:24]))
		relocationOffset := uint64(binary.LittleEndian.Uint32(header[24:28]))
		relocations := uint64(binary.LittleEndian.Uint16(header[32:34]))
		characteristics := binary.LittleEndian.Uint32(header[36:40])
		// IMAGE_SCN_LNK_NRELOC_OVFL stores the actual count in a sentinel
		// relocation record. debug/pe does not implement that encoding and would
		// silently expose only the uint16 header count, so reject it before
		// pe.NewFile allocates or parses the section relocation slice.
		if characteristics&coffSectionRelocationOverflow != 0 {
			return fmt.Errorf("bofloader: COFF section %d uses unsupported extended relocation count", index)
		}
		mapped := characteristics&(pe.IMAGE_SCN_CNT_CODE|pe.IMAGE_SCN_CNT_INITIALIZED_DATA|pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA|pe.IMAGE_SCN_MEM_READ|pe.IMAGE_SCN_MEM_WRITE|pe.IMAGE_SCN_MEM_EXECUTE) != 0
		if characteristics&(pe.IMAGE_SCN_MEM_DISCARDABLE|coffSectionLinkInfo|coffSectionLinkRemove) == 0 && mapped {
			mappedBytes, ok = checkedAddUint64(mappedBytes, rawSize)
			if !ok || mappedBytes > maxImageSize {
				return fmt.Errorf("bofloader: cumulative COFF mapped section size exceeds %d bytes", maxImageSize)
			}
			if rawSize != 0 && rawOffset != 0 && !parserRangeWithin(data, rawOffset, rawSize) {
				return fmt.Errorf("bofloader: COFF section %d data is outside the object", index)
			}
		}
		relocationCount, ok = checkedAddUint64(relocationCount, relocations)
		if !ok || relocationCount > maxObjectRelocations {
			return fmt.Errorf("bofloader: COFF relocation count exceeds %d", maxObjectRelocations)
		}
		if relocations != 0 {
			relocationSize, mulOK := checkedMulUint64(relocations, coffRelocationSize)
			if relocationOffset == 0 || !mulOK || !parserRangeWithin(data, relocationOffset, relocationSize) {
				return fmt.Errorf("bofloader: COFF section %d relocations are outside the object", index)
			}
		}
	}

	if symbolCount != 0 {
		symbolNameCache := make(map[uint64]uint64)
		var resolvedSymbolNameBytes uint64
		for index := uint64(0); index < symbolCount; {
			recordOffset := symbolTableOffset + index*coffSymbolSize
			record := data[recordOffset : recordOffset+coffSymbolSize]
			auxiliaryCount := uint64(record[17])
			next, chainOK := checkedAddUint64(index, 1+auxiliaryCount)
			if !chainOK || next > symbolCount {
				return fmt.Errorf("bofloader: COFF symbol %d auxiliary chain exceeds %d records", index, symbolCount)
			}
			nameLength, nameErr := coffNameLength(record[:8], stringTable, symbolNameCache, false)
			if nameErr != nil {
				return fmt.Errorf("bofloader: COFF symbol %d name: %w", index, nameErr)
			}
			resolvedSymbolNameBytes, ok = checkedAddUint64(resolvedSymbolNameBytes, nameLength)
			if !ok || resolvedSymbolNameBytes > maxObjectNameBytes {
				return fmt.Errorf("bofloader: cumulative COFF symbol-name bytes exceed %d", maxObjectNameBytes)
			}
			index = next
		}
	}
	return nil
}

func coffNameLength(encoded, stringTable []byte, cache map[uint64]uint64, section bool) (uint64, error) {
	if section && len(encoded) != 0 && encoded[0] == '/' {
		offset, err := parseCOFFDecimalOffset(encoded[1:])
		if err != nil {
			return 0, err
		}
		return parserStringLength(stringTable, offset, 4, cache)
	}
	if !section && len(encoded) >= 8 && encoded[0] == 0 && encoded[1] == 0 && encoded[2] == 0 && encoded[3] == 0 {
		offset := uint64(binary.LittleEndian.Uint32(encoded[4:8]))
		if offset == 0 {
			return 0, nil
		}
		return parserStringLength(stringTable, offset, 4, cache)
	}
	if terminator := bytes.IndexByte(encoded, 0); terminator >= 0 {
		return uint64(terminator), nil
	}
	return uint64(len(encoded)), nil
}

func parseCOFFDecimalOffset(encoded []byte) (uint64, error) {
	var value uint64
	digits := 0
	for _, character := range encoded {
		if character == 0 {
			break
		}
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("invalid long-name offset %q", encoded)
		}
		if value > (^uint64(0)-uint64(character-'0'))/10 {
			return 0, fmt.Errorf("long-name offset overflows")
		}
		value = value*10 + uint64(character-'0')
		digits++
	}
	if digits == 0 {
		return 0, fmt.Errorf("empty long-name offset")
	}
	return value, nil
}

func parserRangeWithin(data []byte, offset, size uint64) bool {
	return offset <= uint64(len(data)) && size <= uint64(len(data))-offset
}

func checkedMulUint64(left, right uint64) (uint64, bool) {
	if left != 0 && right > ^uint64(0)/left {
		return 0, false
	}
	return left * right, true
}

func coffSectionAlignment(characteristics uint32) uint64 {
	encoded := (characteristics & coffSectionAlignMask) >> 20
	if encoded >= 1 && encoded <= 14 {
		return uint64(1) << (encoded - 1)
	}
	return 16
}
