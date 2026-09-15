package bofloader

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"unsafe"
)

const (
	maxObjectSize = 64 << 20
	maxImageSize  = 256 << 20

	// Parser metadata is attacker-controlled and can reuse the same on-disk
	// ranges from many sections. Keep its in-memory expansion bounded
	// independently of the raw object and mapped-image limits.
	maxObjectSections    = maxObjectSize / (4 << 10)
	maxObjectSymbols     = maxObjectSize / 64
	maxObjectRelocations = maxObjectSize / 64
	maxObjectNameSize    = 64 << 10
	maxObjectNameBytes   = maxObjectSize

	sectionUndefined = -1
	sectionAbsolute  = -2
	sectionCommon    = -3
	sectionDiscarded = -4
)

var ErrClosed = errors.New("bofloader: object is closed")

// Output is one Beacon output record emitted by a BOF.
type Output struct {
	Type int
	Data []byte
}

type Loader struct {
	mu      sync.RWMutex
	region  *memoryRegion
	unwind  *unwindRegistration
	entry   uintptr
	closing bool
	closed  bool
}

type protection uint8

const (
	protRead protection = 1 << iota
	protWrite
	protExec
)

type objectSection struct {
	name       string
	data       []byte
	size       uint64
	align      uint64
	protection protection
	mapped     bool
	offset     uint64
	address    uintptr
}

type objectSymbol struct {
	index      uint32
	name       string
	section    int
	value      uint64
	size       uint64
	localEntry uint64
	weak       bool
}

type objectRelocation struct {
	section int
	offset  uint64
	typeID  uint32
	symbol  uint32
	addend  int64
	hasAdd  bool
	width   uint8
}

type objectFile struct {
	format      string
	arch        string
	sections    []objectSection
	symbols     map[uint32]objectSymbol
	relocations []objectRelocation
	riscvPairs  map[riscvRelocationSite]riscvRelocationPair
	ppc64TOC    uintptr
	imageBase   uintptr
}

type externalSymbol struct {
	name      string
	target    uintptr
	got       uintptr
	thunk     uintptr
	gotOffset uint64
	thunkOff  uint64
}

func Load(data []byte) (*Loader, error) {
	return LoadWithOptions(data, LoadOptions{})
}

func LoadWithOptions(data []byte, options LoadOptions) (*Loader, error) {
	if len(data) == 0 {
		return nil, errors.New("bofloader: empty object image")
	}
	if len(data) > maxObjectSize {
		return nil, fmt.Errorf("bofloader: object image is %d bytes; maximum is %d", len(data), maxObjectSize)
	}

	object, err := parseObject(data)
	if err != nil {
		return nil, err
	}
	if err := validateHost(object); err != nil {
		return nil, err
	}

	entryNames, err := selectedEntryNames(options)
	if err != nil {
		return nil, err
	}
	entrySymbol := objectSymbol{}
	for _, name := range entryNames {
		symbol, ok, findErr := findDefinedSymbolDefinition(object, name)
		if findErr != nil {
			return nil, findErr
		}
		if ok {
			entrySymbol = symbol
			break
		}
	}
	if entrySymbol.name == "" {
		if options.EntryPoint != "" {
			return nil, fmt.Errorf("bofloader: entry symbol %q not found", options.EntryPoint)
		}
		return nil, errors.New("bofloader: entry symbol not found (expected go or coffee)")
	}
	referenced := referencedLinkageSymbols(object)
	imports := objectImports(object, referenced)
	if options.ValidateImports != nil {
		if err := options.ValidateImports(append([]Import(nil), imports...)); err != nil {
			return nil, fmt.Errorf("bofloader: validate imports: %w", err)
		}
	}
	pageSize := uint64(systemPageSize())
	thunkStride := importThunkStride(object.arch)
	gotStride := uint64(pointerSize())
	thunkSize := alignUp(uint64(len(referenced))*thunkStride, pageSize)
	gotSize := alignUp(uint64(len(referenced))*gotStride, pageSize)
	if thunkSize == 0 {
		thunkSize = pageSize
	}
	if gotSize == 0 {
		gotSize = pageSize
	}

	offset := thunkSize + gotSize
	for i := range object.sections {
		section := &object.sections[i]
		if !section.mapped || section.size == 0 {
			continue
		}
		if section.size > maxImageSize {
			return nil, fmt.Errorf("bofloader: section %q size %d exceeds %d", section.name, section.size, maxImageSize)
		}
		alignment, err := mappedSectionAlignment(*section, pageSize)
		if err != nil {
			return nil, err
		}
		offset = alignUp(offset, alignment)
		if offset > maxImageSize || alignUp(section.size, pageSize) > maxImageSize-offset {
			return nil, fmt.Errorf("bofloader: mapped object exceeds %d bytes", maxImageSize)
		}
		section.offset = offset
		offset += alignUp(section.size, pageSize)
	}
	totalSize := alignUp(offset, pageSize)
	if totalSize == 0 || totalSize > maxImageSize {
		return nil, fmt.Errorf("bofloader: invalid mapped object size %d", totalSize)
	}

	region, err := allocateMemory(int(totalSize))
	if err != nil {
		return nil, fmt.Errorf("bofloader: allocate mapped object: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = region.close()
		}
	}()

	base := region.base()
	object.imageBase = base
	if object.arch == "ppc64le" {
		gotBase, ok := checkedAddUint64(uint64(base), thunkSize)
		if !ok || gotBase > ^uint64(0)-ppc64TOCBias {
			return nil, errors.New("bofloader: PPC64 ELFv2 TOC address overflows")
		}
		object.ppc64TOC = uintptr(gotBase + ppc64TOCBias)
		if uint64(object.ppc64TOC) != gotBase+ppc64TOCBias {
			return nil, errors.New("bofloader: PPC64 ELFv2 TOC address exceeds pointer size")
		}
	}
	for i := range object.sections {
		section := &object.sections[i]
		if !section.mapped || section.size == 0 {
			continue
		}
		section.address = base + uintptr(section.offset)
		if uint64(len(section.data)) > section.size {
			return nil, fmt.Errorf("bofloader: section %q data exceeds mapped size", section.name)
		}
		copy(region.data[section.offset:section.offset+uint64(len(section.data))], section.data)
	}
	entry, err := definedSymbolAddress(object, entrySymbol)
	if err != nil {
		return nil, err
	}

	externals, err := resolveExternalSymbols(object, referenced, region, thunkSize, gotSize, imports, options)
	if err != nil {
		return nil, err
	}
	if err := applyRelocations(object, region, externals); err != nil {
		return nil, err
	}

	if err := region.protect(0, int(thunkSize), protRead|protExec); err != nil {
		return nil, fmt.Errorf("bofloader: protect import thunks: %w", err)
	}
	if err := region.protect(int(thunkSize), int(gotSize), protRead); err != nil {
		return nil, fmt.Errorf("bofloader: protect import table: %w", err)
	}
	for i := range object.sections {
		section := &object.sections[i]
		if !section.mapped || section.size == 0 {
			continue
		}
		length := int(alignUp(section.size, pageSize))
		if err := region.protect(int(section.offset), length, section.protection); err != nil {
			return nil, fmt.Errorf("bofloader: protect section %q: %w", section.name, err)
		}
		if section.protection&protExec != 0 {
			if err := region.flushInstructionCache(int(section.offset), length); err != nil {
				return nil, fmt.Errorf("bofloader: flush section %q instruction cache: %w", section.name, err)
			}
		}
	}
	if err := region.flushInstructionCache(0, int(thunkSize)); err != nil {
		return nil, fmt.Errorf("bofloader: flush import thunk instruction cache: %w", err)
	}

	unwind, err := registerUnwindInfo(object, region)
	if err != nil {
		if unwind != nil {
			// At least one OS unwind-table entry could not be removed. Keep
			// its backing memory mapped rather than leave Windows with a
			// dangling function-table pointer.
			cleanup = false
		}
		return nil, err
	}

	cleanup = false
	return &Loader{region: region, unwind: unwind, entry: entry}, nil
}

func parseObject(data []byte) (*objectFile, error) {
	if len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
		return parseELF(data)
	}
	if isMachOMagic(data) {
		return parseMachO(data)
	}
	return parseCOFF(data)
}

func validateHost(object *objectFile) error {
	if object == nil {
		return errors.New("bofloader: nil parsed object")
	}
	if object.arch != runtime.GOARCH {
		return fmt.Errorf("bofloader: object architecture %s does not match host %s", object.arch, runtime.GOARCH)
	}
	if err := validateRuntimeVariant(); err != nil {
		return err
	}
	switch object.format {
	case "coff":
		if runtime.GOOS != "windows" {
			return fmt.Errorf("bofloader: COFF BOFs require a Windows host; use an ELF relocatable BOF on %s", runtime.GOOS)
		}
	case "elf":
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "freebsd" {
			return fmt.Errorf("bofloader: ELF BOFs are unsupported on %s", runtime.GOOS)
		}
	case "macho":
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("bofloader: Mach-O BOFs require a Darwin host; use a native relocatable object on %s", runtime.GOOS)
		}
	default:
		return fmt.Errorf("bofloader: unsupported object format %q", object.format)
	}
	return nil
}

func referencedLinkageSymbols(object *objectFile) []uint32 {
	seen := make(map[uint32]struct{})
	result := make([]uint32, 0)
	for _, relocation := range object.relocations {
		if relocation.section < 0 || relocation.section >= len(object.sections) || !object.sections[relocation.section].mapped {
			continue
		}
		symbol, ok := object.symbols[relocation.symbol]
		needsDefinedGOT := object.format == "elf" && elfRelocationNeedsGOT(object.arch, relocation.typeID) ||
			object.format == "macho" && machoRelocationNeedsGOT(object.arch, relocation.typeID)
		if !ok || (symbol.section != sectionUndefined && !needsDefinedGOT) {
			continue
		}
		if _, ok := seen[symbol.index]; ok {
			continue
		}
		seen[symbol.index] = struct{}{}
		result = append(result, symbol.index)
	}
	return result
}

func importThunkStride(arch string) uint64 {
	if arch == "ppc64le" {
		// Keep each ELFv2 linkage stub on a 32-byte boundary. The current stub
		// occupies five instructions; the larger stride also matches a common
		// PPC64 instruction-cache line and leaves room for future ABI sequences.
		return 32
	}
	return 16
}

func resolveExternalSymbols(object *objectFile, referenced []uint32, region *memoryRegion, thunkSize, gotSize uint64, imports []Import, options LoadOptions) (map[uint32]externalSymbol, error) {
	resolved := make(map[uint32]externalSymbol, len(referenced))
	importsByName := make(map[string]Import, len(imports))
	for _, imported := range imports {
		importsByName[imported.Name] = imported
	}
	targetsByName := make(map[string]uintptr, len(imports))
	for position, index := range referenced {
		symbol := object.symbols[index]
		target := uintptr(0)
		var err error
		if object.format == "elf" && symbol.name == "_GLOBAL_OFFSET_TABLE_" {
			target = region.base() + uintptr(thunkSize)
		} else if object.format == "elf" && object.arch == "ppc64le" && symbol.name == ".TOC." {
			if object.ppc64TOC == 0 {
				return nil, errors.New("bofloader: unexpected synthetic ELF .TOC. symbol")
			}
			target = object.ppc64TOC
		} else if symbol.section == sectionUndefined {
			if cached, ok := targetsByName[symbol.name]; ok {
				target = cached
			} else {
				if imported, ok := importsByName[symbol.name]; ok {
					symbol.weak = imported.Weak
				}
				target, err = resolveImportedSymbol(symbol, options)
				if err != nil {
					return nil, fmt.Errorf("bofloader: resolve external symbol %q: %w", symbol.name, err)
				}
				targetsByName[symbol.name] = target
			}
		} else {
			linked, linkErr := linkRelocationSymbol(object, objectRelocation{symbol: index}, nil)
			if linkErr != nil {
				return nil, fmt.Errorf("bofloader: resolve defined GOT symbol %q: %w", symbol.name, linkErr)
			}
			target = uintptr(linked.address)
			if uint64(target) != linked.address {
				return nil, fmt.Errorf("bofloader: defined GOT symbol %q address %#x exceeds pointer size", symbol.name, linked.address)
			}
		}
		thunkStride := importThunkStride(object.arch)
		thunkOffset := uint64(position) * thunkStride
		gotOffset := thunkSize + uint64(position)*uint64(pointerSize())
		if thunkOffset+thunkStride > thunkSize || gotOffset+uint64(pointerSize()) > thunkSize+gotSize {
			return nil, errors.New("bofloader: import linkage table overflow")
		}
		ext := externalSymbol{
			name:      symbol.name,
			target:    target,
			got:       region.base() + uintptr(gotOffset),
			thunk:     region.base() + uintptr(thunkOffset),
			gotOffset: gotOffset,
			thunkOff:  thunkOffset,
		}
		writePointer(region.data[gotOffset:gotOffset+uint64(pointerSize())], target)
		if err := writeThunk(region.data[thunkOffset:thunkOffset+thunkStride], target, ext.thunk, ext.got, object.ppc64TOC); err != nil {
			return nil, fmt.Errorf("bofloader: create thunk for %q: %w", symbol.name, err)
		}
		resolved[index] = ext
	}
	return resolved, nil
}

func findDefinedSymbol(object *objectFile, name string) (uintptr, bool, error) {
	symbol, ok, err := findDefinedSymbolDefinition(object, name)
	if err != nil || !ok {
		return 0, ok, err
	}
	address, err := definedSymbolAddress(object, symbol)
	if err != nil {
		return 0, false, err
	}
	return address, true, nil
}

func findDefinedSymbolDefinition(object *objectFile, name string) (objectSymbol, bool, error) {
	indices := make([]uint32, 0, 1)
	for index, symbol := range object.symbols {
		if symbol.name == name && symbol.section != sectionUndefined {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		return objectSymbol{}, false, nil
	}
	sort.Slice(indices, func(left, right int) bool { return indices[left] < indices[right] })
	if len(indices) != 1 {
		return objectSymbol{}, false, fmt.Errorf("bofloader: entry symbol %q has multiple definitions at symbol indices %v", name, indices)
	}

	symbol := object.symbols[indices[0]]
	if symbol.section < 0 || symbol.section >= len(object.sections) {
		return objectSymbol{}, false, fmt.Errorf("bofloader: entry symbol %q references invalid section %d", name, symbol.section)
	}
	section := object.sections[symbol.section]
	if !section.mapped || section.protection&protExec == 0 {
		return objectSymbol{}, false, fmt.Errorf("bofloader: entry symbol %q is not in a mapped executable section", name)
	}
	if symbol.value >= section.size {
		return objectSymbol{}, false, fmt.Errorf("bofloader: entry symbol %q value %#x exceeds section %q size %#x", name, symbol.value, section.name, section.size)
	}
	return symbol, true, nil
}

func definedSymbolAddress(object *objectFile, symbol objectSymbol) (uintptr, error) {
	section := object.sections[symbol.section]
	address, ok := checkedAddUint64(uint64(section.address), symbol.value)
	if !ok || uint64(uintptr(address)) != address || address == 0 {
		return 0, fmt.Errorf("bofloader: entry symbol %q address overflows the host pointer size", symbol.name)
	}
	return uintptr(address), nil
}

func mappedSectionAlignment(section objectSection, pageSize uint64) (uint64, error) {
	alignment := section.align
	if alignment == 0 {
		alignment = 1
	}
	if alignment > maxImageSize || alignment&(alignment-1) != 0 {
		return 0, fmt.Errorf("bofloader: section %q has invalid alignment %d", section.name, alignment)
	}
	if pageSize == 0 || pageSize&(pageSize-1) != 0 {
		return 0, fmt.Errorf("bofloader: invalid system page size %d", pageSize)
	}
	if alignment > pageSize {
		return 0, fmt.Errorf("bofloader: section %q alignment %d exceeds guaranteed allocation alignment %d", section.name, alignment, pageSize)
	}
	return pageSize, nil
}

func (loader *Loader) Execute(args []byte) ([]Output, error) {
	loader.mu.RLock()
	defer loader.mu.RUnlock()
	if loader.closing || loader.closed || loader.region == nil || loader.entry == 0 {
		return nil, ErrClosed
	}
	if len(args) > maxCallbackData {
		return nil, fmt.Errorf("bofloader: argument buffer is %d bytes; maximum is %d", len(args), maxCallbackData)
	}

	executionLock.Lock()
	defer executionLock.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	context := newExecutionContext()
	activeExecution.Store(context)
	defer activeExecution.Store(nil)

	var address uintptr
	if len(args) != 0 {
		address = uintptr(unsafe.Pointer(unsafe.SliceData(args)))
	}
	invokeEntry(loader.entry, address, int32(len(args)))
	runtime.KeepAlive(args)

	outputs, err := context.result()
	if err != nil {
		return outputs, fmt.Errorf("bofloader: execute BOF: %w", err)
	}
	return outputs, nil
}

func (loader *Loader) Close() error {
	loader.mu.Lock()
	defer loader.mu.Unlock()
	if loader.closed {
		return nil
	}
	loader.closing = true
	if loader.unwind != nil {
		if err := loader.unwind.close(); err != nil {
			return fmt.Errorf("bofloader: unregister unwind information: %w", err)
		}
		loader.unwind = nil
	}
	if loader.region == nil {
		loader.entry = 0
		loader.closed = true
		return nil
	}
	if err := loader.region.close(); err != nil {
		return fmt.Errorf("bofloader: release mapped object: %w", err)
	}
	loader.region = nil
	loader.entry = 0
	loader.closed = true
	return nil
}

func alignUp(value, alignment uint64) uint64 {
	if alignment <= 1 {
		return value
	}
	return (value + alignment - 1) &^ (alignment - 1)
}

func pointerSize() int {
	return int(unsafe.Sizeof(uintptr(0)))
}

func systemPageSize() int {
	return memoryPageSize()
}
