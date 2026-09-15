//go:build windows && (amd64 || arm64)

package bofloader

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type unwindRegistration struct {
	tables []uintptr
}

type unwindTable struct {
	address uintptr
	entries uint32
}

type unwindImageRange struct {
	name       string
	start      uint64
	end        uint64
	protection protection
}

type unwindFunctionRange struct {
	section string
	index   int
	begin   uint64
	end     uint64
}

//go:nocheckptr
func registerUnwindInfo(object *objectFile, region *memoryRegion) (*unwindRegistration, error) {
	tables, err := collectUnwindTables(object, region)
	if err != nil {
		return nil, fmt.Errorf("bofloader: validate Windows unwind information: %w", err)
	}
	if len(tables) == 0 {
		return nil, nil
	}

	registration := &unwindRegistration{tables: make([]uintptr, 0, len(tables))}
	for _, table := range tables {
		functionTable := (*windows.RUNTIME_FUNCTION)(unsafe.Pointer(table.address))
		if !windows.RtlAddFunctionTable(functionTable, table.entries, object.imageBase) {
			rollbackErr := registration.close()
			registrationError := errors.Join(
				fmt.Errorf("bofloader: RtlAddFunctionTable rejected %d entries at %#x", table.entries, table.address),
				rollbackErr,
			)
			if rollbackErr != nil {
				return registration, registrationError
			}
			return nil, registrationError
		}
		registration.tables = append(registration.tables, table.address)
	}
	return registration, nil
}

//go:nocheckptr
func (registration *unwindRegistration) close() error {
	if registration == nil || len(registration.tables) == 0 {
		return nil
	}
	for index := len(registration.tables) - 1; index >= 0; index-- {
		address := registration.tables[index]
		functionTable := (*windows.RUNTIME_FUNCTION)(unsafe.Pointer(address))
		if !windows.RtlDeleteFunctionTable(functionTable) {
			// Keep this and all earlier table addresses so a later Close call can
			// retry without unmapping memory still referenced by the OS unwinder.
			registration.tables = registration.tables[:index+1]
			return fmt.Errorf("RtlDeleteFunctionTable rejected table at %#x", address)
		}
	}
	registration.tables = nil
	return nil
}

func collectUnwindTables(object *objectFile, region *memoryRegion) ([]unwindTable, error) {
	if object == nil || region == nil || region.address == 0 {
		return nil, errors.New("nil or closed mapped object")
	}
	if object.imageBase == 0 || object.imageBase != region.address {
		return nil, errors.New("mapped object image base does not match its memory region")
	}
	entrySize := uint64(12)
	if runtime.GOARCH == "arm64" {
		entrySize = 8
	}

	imageSize := uint64(len(region.data))
	ranges, err := collectUnwindImageRanges(object, imageSize)
	if err != nil {
		return nil, err
	}
	tables := make([]unwindTable, 0, 1)
	functionRanges := make([]unwindFunctionRange, 0)
	for sectionIndex := range object.sections {
		section := &object.sections[sectionIndex]
		if section.name != ".pdata" && !strings.HasPrefix(section.name, ".pdata$") {
			continue
		}
		if !section.mapped || section.size == 0 {
			continue
		}
		if !unwindRangeContains(ranges, section.offset, section.size, protRead) {
			return nil, fmt.Errorf("section %q is not contained in a readable mapped section", section.name)
		}
		if section.address&3 != 0 {
			return nil, fmt.Errorf("section %q function table address %#x is not DWORD aligned", section.name, section.address)
		}
		if section.size%entrySize != 0 {
			return nil, fmt.Errorf("section %q size %d is not a multiple of %d", section.name, section.size, entrySize)
		}

		data := region.data[section.offset : section.offset+section.size]
		entries := len(data) / int(entrySize)
		for entries != 0 && allZero(data[(entries-1)*int(entrySize):entries*int(entrySize)]) {
			entries--
		}
		if entries == 0 {
			continue
		}
		validated, err := validateUnwindEntries(section.name, data[:entries*int(entrySize)], int(entrySize), region.data, ranges)
		if err != nil {
			return nil, err
		}
		functionRanges = append(functionRanges, validated...)
		tables = append(tables, unwindTable{address: section.address, entries: uint32(entries)})
	}
	if err := validateDistinctUnwindRanges(functionRanges); err != nil {
		return nil, err
	}
	return tables, nil
}

func collectUnwindImageRanges(object *objectFile, imageSize uint64) ([]unwindImageRange, error) {
	ranges := make([]unwindImageRange, 0, len(object.sections))
	base := uint64(object.imageBase)
	for index := range object.sections {
		section := &object.sections[index]
		if !section.mapped || section.size == 0 {
			continue
		}
		if section.offset > imageSize || section.size > imageSize-section.offset {
			return nil, fmt.Errorf("section %q is outside the mapped image", section.name)
		}
		expectedAddress, ok := checkedAddUint64(base, section.offset)
		if !ok || expectedAddress != uint64(section.address) {
			return nil, fmt.Errorf("section %q address does not match its mapped-image offset", section.name)
		}
		ranges = append(ranges, unwindImageRange{
			name:       section.name,
			start:      section.offset,
			end:        section.offset + section.size,
			protection: section.protection,
		})
	}
	return ranges, nil
}

func validateUnwindEntries(sectionName string, data []byte, entrySize int, image []byte, ranges []unwindImageRange) ([]unwindFunctionRange, error) {
	imageSize := uint64(len(image))
	validated := make([]unwindFunctionRange, 0, len(data)/entrySize)
	var previousBegin uint32
	for offset := 0; offset < len(data); offset += entrySize {
		entry := data[offset : offset+entrySize]
		if allZero(entry) {
			return nil, fmt.Errorf("section %q has an empty unwind entry at index %d", sectionName, offset/entrySize)
		}
		begin := binary.LittleEndian.Uint32(entry[0:4])
		if uint64(begin) >= imageSize {
			return nil, fmt.Errorf("section %q entry %d begin RVA %#x exceeds image size %#x", sectionName, offset/entrySize, begin, imageSize)
		}
		if offset != 0 && begin < previousBegin {
			return nil, fmt.Errorf("section %q unwind entries are not sorted by begin RVA", sectionName)
		}
		previousBegin = begin
		entryIndex := offset / entrySize
		if entrySize == 8 && begin&3 != 0 {
			return nil, fmt.Errorf("section %q entry %d has unaligned ARM64 begin RVA %#x", sectionName, entryIndex, begin)
		}

		if entrySize == 12 {
			end := binary.LittleEndian.Uint32(entry[4:8])
			if end <= begin || !unwindRangeContains(ranges, uint64(begin), uint64(end-begin), protExec) {
				return nil, fmt.Errorf("section %q entry %d has invalid executable range [%#x, %#x)", sectionName, entryIndex, begin, end)
			}
			unwindRVA := binary.LittleEndian.Uint32(entry[8:12])
			if unwindRVA&3 != 0 {
				return nil, fmt.Errorf("section %q entry %d has unaligned x64 unwind RVA %#x", sectionName, entryIndex, unwindRVA)
			}
			if unwindRVA == 0 || !unwindRangeContains(ranges, uint64(unwindRVA), 4, protRead) {
				return nil, fmt.Errorf("section %q entry %d has invalid readable unwind RVA %#x", sectionName, entryIndex, unwindRVA)
			}
			if err := validateX64UnwindInfo(sectionName, entryIndex, uint64(unwindRVA), uint64(end-begin), image, ranges, make(map[uint64]struct{})); err != nil {
				return nil, err
			}
			validated = append(validated, unwindFunctionRange{section: sectionName, index: entryIndex, begin: uint64(begin), end: uint64(end)})
			continue
		}

		// ARM64 uses either packed unwind data (low bits non-zero) or an RVA
		// to an xdata record (low bits zero).
		unwindData := binary.LittleEndian.Uint32(entry[4:8])
		flag := unwindData & 3
		if flag == 3 {
			return nil, fmt.Errorf("section %q entry %d uses reserved ARM64 packed-unwind flag 3", sectionName, entryIndex)
		}
		var functionSize uint64
		if flag != 0 {
			functionSize = uint64((unwindData>>2)&0x7ff) * 4
			if functionSize == 0 {
				return nil, fmt.Errorf("section %q entry %d has a zero-length ARM64 packed function", sectionName, entryIndex)
			}
		} else {
			var err error
			functionSize, err = validateARM64XData(sectionName, entryIndex, uint64(unwindData), image, ranges)
			if err != nil {
				return nil, err
			}
		}
		if !unwindRangeContains(ranges, uint64(begin), functionSize, protExec) {
			end, _ := checkedAddUint64(uint64(begin), functionSize)
			return nil, fmt.Errorf("section %q entry %d has invalid executable range [%#x, %#x)", sectionName, entryIndex, begin, end)
		}
		validated = append(validated, unwindFunctionRange{section: sectionName, index: entryIndex, begin: uint64(begin), end: uint64(begin) + functionSize})
	}
	return validated, nil
}

func validateX64UnwindInfo(sectionName string, entryIndex int, unwindRVA, functionSize uint64, image []byte, ranges []unwindImageRange, visited map[uint64]struct{}) error {
	if len(visited) >= 64 {
		return fmt.Errorf("section %q entry %d x64 unwind chain exceeds 64 records", sectionName, entryIndex)
	}
	if _, ok := visited[unwindRVA]; ok {
		return fmt.Errorf("section %q entry %d x64 unwind chain contains a cycle at RVA %#x", sectionName, entryIndex, unwindRVA)
	}
	visited[unwindRVA] = struct{}{}
	defer delete(visited, unwindRVA)

	header := image[unwindRVA : unwindRVA+4]
	version := header[0] & 7
	flags := header[0] >> 3
	if version != 1 {
		return fmt.Errorf("section %q entry %d uses unsupported x64 unwind version %d", sectionName, entryIndex, version)
	}
	if uint64(header[1]) > functionSize {
		return fmt.Errorf("section %q entry %d x64 prolog size %d exceeds function size %d", sectionName, entryIndex, header[1], functionSize)
	}
	const (
		x64UnwindFlagException = uint8(1)
		x64UnwindFlagTerminate = uint8(2)
		x64UnwindFlagChain     = uint8(4)
		x64UnwindKnownFlags    = x64UnwindFlagException | x64UnwindFlagTerminate | x64UnwindFlagChain
	)
	if flags&^x64UnwindKnownFlags != 0 {
		return fmt.Errorf("section %q entry %d uses unsupported x64 unwind flags %#x", sectionName, entryIndex, flags)
	}
	if flags&x64UnwindFlagChain != 0 && flags&(x64UnwindFlagException|x64UnwindFlagTerminate) != 0 {
		return fmt.Errorf("section %q entry %d combines chained and handler x64 unwind flags", sectionName, entryIndex)
	}

	codeCount := uint64(header[2])
	paddedCodes := (codeCount + 1) &^ uint64(1)
	fixedSize := uint64(4) + paddedCodes*2
	trailerSize := uint64(0)
	if flags&(x64UnwindFlagException|x64UnwindFlagTerminate) != 0 {
		trailerSize = 4
	} else if flags&x64UnwindFlagChain != 0 {
		trailerSize = 12
	}
	recordSize := fixedSize + trailerSize
	if !unwindRangeContains(ranges, unwindRVA, recordSize, protRead) {
		return fmt.Errorf("section %q entry %d x64 unwind record of %d bytes crosses a readable-section boundary", sectionName, entryIndex, recordSize)
	}
	codes := image[unwindRVA+4 : unwindRVA+4+codeCount*2]
	if err := validateX64UnwindCodes(codes, header[1]); err != nil {
		return fmt.Errorf("section %q entry %d has invalid x64 unwind codes: %w", sectionName, entryIndex, err)
	}

	trailer := unwindRVA + fixedSize
	if flags&(x64UnwindFlagException|x64UnwindFlagTerminate) != 0 {
		handlerRVA := uint64(binary.LittleEndian.Uint32(image[trailer : trailer+4]))
		if handlerRVA == 0 || !unwindRangeContains(ranges, handlerRVA, 1, protExec) {
			return fmt.Errorf("section %q entry %d has invalid executable x64 exception-handler RVA %#x", sectionName, entryIndex, handlerRVA)
		}
	} else if flags&x64UnwindFlagChain != 0 {
		chainedBegin := binary.LittleEndian.Uint32(image[trailer : trailer+4])
		chainedEnd := binary.LittleEndian.Uint32(image[trailer+4 : trailer+8])
		chainedUnwind := binary.LittleEndian.Uint32(image[trailer+8 : trailer+12])
		if chainedEnd <= chainedBegin || !unwindRangeContains(ranges, uint64(chainedBegin), uint64(chainedEnd-chainedBegin), protExec) {
			return fmt.Errorf("section %q entry %d has invalid chained executable range [%#x, %#x)", sectionName, entryIndex, chainedBegin, chainedEnd)
		}
		if chainedUnwind == 0 || chainedUnwind&3 != 0 || !unwindRangeContains(ranges, uint64(chainedUnwind), 4, protRead) {
			return fmt.Errorf("section %q entry %d has invalid chained x64 unwind RVA %#x", sectionName, entryIndex, chainedUnwind)
		}
		if err := validateX64UnwindInfo(sectionName, entryIndex, uint64(chainedUnwind), uint64(chainedEnd-chainedBegin), image, ranges, visited); err != nil {
			return err
		}
	}
	return nil
}

func validateX64UnwindCodes(codes []byte, prologSize byte) error {
	previousOffset := uint16(256)
	for slot := 0; slot < len(codes)/2; {
		codeOffset := codes[slot*2]
		operation := codes[slot*2+1] & 0x0f
		operationInfo := codes[slot*2+1] >> 4
		if codeOffset > prologSize {
			return fmt.Errorf("code offset %d exceeds prolog size %d", codeOffset, prologSize)
		}
		if uint16(codeOffset) > previousOffset {
			return fmt.Errorf("code offsets are not sorted in descending order")
		}
		previousOffset = uint16(codeOffset)

		slots := 1
		switch operation {
		case 0, 2: // UWOP_PUSH_NONVOL, UWOP_ALLOC_SMALL
		case 1: // UWOP_ALLOC_LARGE
			switch operationInfo {
			case 0:
				slots = 2
			case 1:
				slots = 3
			default:
				return fmt.Errorf("UWOP_ALLOC_LARGE has reserved operation info %d", operationInfo)
			}
		case 3: // UWOP_SET_FPREG
			if operationInfo != 0 {
				return fmt.Errorf("UWOP_SET_FPREG has reserved operation info %d", operationInfo)
			}
		case 4, 8: // UWOP_SAVE_NONVOL, UWOP_SAVE_XMM128
			slots = 2
		case 5, 9: // UWOP_SAVE_NONVOL_FAR, UWOP_SAVE_XMM128_FAR
			slots = 3
		case 10: // UWOP_PUSH_MACHFRAME
			if operationInfo > 1 {
				return fmt.Errorf("UWOP_PUSH_MACHFRAME has reserved operation info %d", operationInfo)
			}
		default:
			return fmt.Errorf("unsupported unwind operation %d", operation)
		}
		if slot+slots > len(codes)/2 {
			return fmt.Errorf("unwind operation %d at slot %d requires %d slots", operation, slot, slots)
		}
		slot += slots
	}
	return nil
}

func validateARM64XData(sectionName string, entryIndex int, xdataRVA uint64, image []byte, ranges []unwindImageRange) (uint64, error) {
	if xdataRVA == 0 || !unwindRangeContains(ranges, xdataRVA, 4, protRead) {
		return 0, fmt.Errorf("section %q entry %d has invalid readable xdata RVA %#x", sectionName, entryIndex, xdataRVA)
	}
	header := binary.LittleEndian.Uint32(image[xdataRVA : xdataRVA+4])
	functionSize := uint64(header&0x3ffff) * 4
	if functionSize == 0 {
		return 0, fmt.Errorf("section %q entry %d has a zero-length ARM64 xdata function", sectionName, entryIndex)
	}
	if version := (header >> 18) & 3; version != 0 {
		return 0, fmt.Errorf("section %q entry %d uses unsupported ARM64 xdata version %d", sectionName, entryIndex, version)
	}

	recordSize := uint64(4)
	epilogField := uint64((header >> 22) & 0x1f)
	unwindWords := uint64((header >> 27) & 0x1f)
	if header>>22 == 0 {
		if !unwindRangeContains(ranges, xdataRVA, 8, protRead) {
			return 0, fmt.Errorf("section %q entry %d has a truncated ARM64 extended xdata header", sectionName, entryIndex)
		}
		extended := binary.LittleEndian.Uint32(image[xdataRVA+4 : xdataRVA+8])
		if extended>>24 != 0 {
			return 0, fmt.Errorf("section %q entry %d uses non-zero reserved ARM64 extended-xdata bits", sectionName, entryIndex)
		}
		recordSize = 8
		epilogField = uint64(extended & 0xffff)
		unwindWords = uint64((extended >> 16) & 0xff)
	}
	if header&(1<<21) == 0 {
		recordSize += epilogField * 4
	}
	recordSize += unwindWords * 4
	if header&(1<<20) != 0 {
		recordSize += 4
	}
	if !unwindRangeContains(ranges, xdataRVA, recordSize, protRead) {
		return 0, fmt.Errorf("section %q entry %d ARM64 xdata record of %d bytes crosses a readable-section boundary", sectionName, entryIndex, recordSize)
	}
	headerSize := uint64(4)
	if header>>22 == 0 {
		headerSize = 8
	}
	codeBytes := unwindWords * 4
	if header&(1<<21) != 0 {
		if codeBytes == 0 || epilogField >= codeBytes {
			return 0, fmt.Errorf("section %q entry %d has ARM64 packed epilog index %d outside %d unwind-code bytes", sectionName, entryIndex, epilogField, codeBytes)
		}
	} else {
		var previousStart uint64
		for scopeIndex := uint64(0); scopeIndex < epilogField; scopeIndex++ {
			offset := xdataRVA + headerSize + scopeIndex*4
			scope := binary.LittleEndian.Uint32(image[offset : offset+4])
			start := uint64(scope & 0x3ffff)
			reserved := (scope >> 18) & 0xf
			codeIndex := uint64(scope >> 22)
			if reserved != 0 {
				return 0, fmt.Errorf("section %q entry %d ARM64 epilog scope %d has non-zero reserved bits", sectionName, entryIndex, scopeIndex)
			}
			if start*4 >= functionSize {
				return 0, fmt.Errorf("section %q entry %d ARM64 epilog scope %d starts outside the function", sectionName, entryIndex, scopeIndex)
			}
			if scopeIndex != 0 && start <= previousStart {
				return 0, fmt.Errorf("section %q entry %d ARM64 epilog scopes are not sorted by start offset", sectionName, entryIndex)
			}
			previousStart = start
			if codeBytes == 0 || codeIndex >= codeBytes {
				return 0, fmt.Errorf("section %q entry %d ARM64 epilog scope %d code index %d exceeds %d unwind-code bytes", sectionName, entryIndex, scopeIndex, codeIndex, codeBytes)
			}
		}
	}
	if header&(1<<20) != 0 {
		handlerOffset := xdataRVA + recordSize - 4
		handlerRVA := uint64(binary.LittleEndian.Uint32(image[handlerOffset : handlerOffset+4]))
		if handlerRVA == 0 || handlerRVA&3 != 0 || !unwindRangeContains(ranges, handlerRVA, 4, protExec) {
			return 0, fmt.Errorf("section %q entry %d has invalid executable ARM64 exception-handler RVA %#x", sectionName, entryIndex, handlerRVA)
		}
	}
	return functionSize, nil
}

func unwindRangeContains(ranges []unwindImageRange, start, size uint64, required protection) bool {
	if size == 0 {
		return false
	}
	end, ok := checkedAddUint64(start, size)
	if !ok {
		return false
	}
	for _, candidate := range ranges {
		if candidate.protection&required == required && start >= candidate.start && end <= candidate.end {
			return true
		}
	}
	return false
}

func validateDistinctUnwindRanges(ranges []unwindFunctionRange) error {
	sort.Slice(ranges, func(left, right int) bool {
		if ranges[left].begin != ranges[right].begin {
			return ranges[left].begin < ranges[right].begin
		}
		return ranges[left].end < ranges[right].end
	})
	for index := 1; index < len(ranges); index++ {
		previous := ranges[index-1]
		current := ranges[index]
		if current.begin < previous.end {
			return fmt.Errorf(
				"unwind function ranges overlap: section %q entry %d [%#x, %#x) and section %q entry %d [%#x, %#x)",
				previous.section, previous.index, previous.begin, previous.end,
				current.section, current.index, current.begin, current.end,
			)
		}
	}
	return nil
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}
