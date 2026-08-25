//go:build (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) || (freebsd && (amd64 || arm64))

// SPDX-License-Identifier: MIT
//
// This loader is adapted from Reflektor's memmod Linux backend. It is kept in
// a separate leaf package so native-only callers do not link the Go c-shared
// static TLS provider. See ../../../memmod/COPYING for license details.
package linuxmem

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/sliverarmory/reflektor/native/internal/rejection"
	"golang.org/x/sys/unix"
)

type linuxDynAPI struct {
	dlopen        uintptr
	dlsym         uintptr
	dlvsym        uintptr
	dlclose       uintptr
	dlerror       uintptr
	defaultHandle uintptr
}

var (
	linuxAPIOnce sync.Once
	linuxAPI     linuxDynAPI
	linuxAPIErr  error
)

const (
	rtldNow    = 0x2
	rtldGlobal = 0x100

	// ELF dynamic tags used for runtime initialization hooks.
	dynTagNull        = 0
	dynTagInit        = 12
	dynTagFini        = 13
	dynTagInitArray   = 25
	dynTagFiniArray   = 26
	dynTagInitArraySz = 27
	dynTagFiniArraySz = 28
	dynTagPreinitArr  = 32
	dynTagPreinitSz   = 33

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

	ppc64ELFABI  = 0x00000003
	ppc64ELFABI2 = 0x00000002
)

type Module struct {
	mu                sync.RWMutex
	mapping           []byte
	loadBias          uintptr
	symbols           map[string]uintptr
	finalizers        []uintptr
	dynamicAPI        *linuxDynAPI
	ownedDlopen       []uintptr
	closeDlopenHandle func(*linuxDynAPI, uintptr) error
	closed            bool
}

type mappedELF struct {
	mapping   []byte
	loadBias  uintptr
	progs     []*elf.Prog
	tlsOffset int64
	hasTLS    bool
}

type dynamicInitInfo struct {
	init        uint64
	fini        uint64
	initArray   uint64
	initArraySz uint64
	finiArray   uint64
	finiArraySz uint64
	preinitArr  uint64
	preinitSz   uint64
}

type runtimeELFModule struct {
	path  string
	base  uintptr
	score int
}

type symbolResolver struct {
	api           *linuxDynAPI
	modules       []runtimeELFModule
	resolved      map[string]uintptr
	misses        map[string]error
	opened        map[string]uintptr
	ownedDlopen   []uintptr
	openLibrary   func(*linuxDynAPI, string) (uintptr, error)
	closeLibrary  func(*linuxDynAPI, uintptr) error
	resolveSymbol func(elf.Symbol) (uintptr, error)
}

const maxExportArguments = 3

func LoadLibrary(data []byte) (*Module, error) {
	if len(data) == 0 {
		return nil, errors.New("empty ELF image")
	}

	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid ELF image: %w", err)
	}
	defer f.Close()

	// This check is intentionally duplicated behind native.LoadLibrary's
	// format-independent preflight. It keeps the Linux backend safe if its
	// loading seam is ever reused inside the native package.
	if f.Section(".go.buildinfo") != nil {
		return nil, rejection.ErrGoSharedLibraryUnsupported
	}
	if err := validateELFImage(data, f); err != nil {
		return nil, err
	}
	initInfo, err := parseDynamicInitInfo(f)
	if err != nil {
		return nil, err
	}

	mapped, err := mapELFImage(data, f)
	if err != nil {
		return nil, err
	}
	cleanup := true
	defer func() {
		if cleanup && len(mapped.mapping) != 0 {
			_ = unix.Munmap(mapped.mapping)
		}
	}()

	resolver := newSymbolResolver()
	defer resolver.closeOwnedLibraries()
	if err := resolver.primeDependencies(f); err != nil {
		return nil, err
	}
	if err := applyDynamicRelocations(mapped, f, resolver); err != nil {
		return nil, err
	}
	if err := flushMappedInstructionCache(mapped.mapping); err != nil {
		return nil, err
	}

	if err := applySegmentProtections(mapped); err != nil {
		return nil, err
	}
	initializers, err := collectELFInitializers(mapped, f.Class, initInfo)
	if err != nil {
		return nil, err
	}
	finalizers, err := collectELFFinalizers(mapped, f.Class, initInfo)
	if err != nil {
		return nil, err
	}
	for _, initializer := range initializers {
		_ = callExportFunction(initializer, 0, 0, 0)
	}

	module := &Module{
		mapping:           mapped.mapping,
		loadBias:          mapped.loadBias,
		symbols:           buildExportedSymbolTable(f, mapped.loadBias),
		finalizers:        finalizers,
		dynamicAPI:        resolver.api,
		ownedDlopen:       resolver.takeOwnedLibraries(),
		closeDlopenHandle: resolver.closeLibrary,
	}
	cleanup = false
	return module, nil
}

func (module *Module) Free() {
	module.mu.Lock()
	if module.closed {
		module.mu.Unlock()
		return
	}
	module.closed = true
	finalizers := module.finalizers
	mapping := module.mapping
	dynamicAPI := module.dynamicAPI
	ownedDlopen := module.ownedDlopen
	closeDlopenHandle := module.closeDlopenHandle
	module.finalizers = nil
	module.mapping = nil
	module.symbols = nil
	module.loadBias = 0
	module.dynamicAPI = nil
	module.ownedDlopen = nil
	module.closeDlopenHandle = nil
	module.mu.Unlock()

	for _, finalizer := range finalizers {
		_ = callExportFunction(finalizer)
	}
	if len(mapping) != 0 {
		_ = unix.Munmap(mapping)
	}
	closeDlopenHandles(dynamicAPI, closeDlopenHandle, ownedDlopen)
}

func (module *Module) CallExport(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("export name cannot be empty")
	}

	candidates := []string{name}
	if strings.HasPrefix(name, "_") {
		candidates = append(candidates, strings.TrimPrefix(name, "_"))
	} else {
		candidates = append(candidates, "_"+name)
	}

	var (
		addr uintptr
		err  error
	)
	for _, candidate := range candidates {
		addr, err = module.ProcAddressByName(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("resolve export %q: %w", name, err)
	}

	_ = callExportFunction(addr)
	return nil
}

// CallExportWithArgs resolves and calls an exported native C/Rust function
// with up to three machine-word arguments and returns the value from the
// platform's primary return register.
//
//go:uintptrescapes
func (module *Module) CallExportWithArgs(name string, args ...uintptr) (uintptr, error) {
	if len(args) > maxExportArguments {
		return 0, fmt.Errorf("export call has %d arguments; maximum is %d", len(args), maxExportArguments)
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("export name cannot be empty")
	}

	candidates := []string{name}
	if strings.HasPrefix(name, "_") {
		candidates = append(candidates, strings.TrimPrefix(name, "_"))
	} else {
		candidates = append(candidates, "_"+name)
	}

	var (
		addr uintptr
		err  error
	)
	for _, candidate := range candidates {
		addr, err = module.ProcAddressByName(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, fmt.Errorf("resolve export %q: %w", name, err)
	}

	result := callExportFunction(addr, args...)
	runtime.KeepAlive(args)
	return result, nil
}

func (module *Module) ProcAddressByName(name string) (uintptr, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("export name cannot be empty")
	}

	module.mu.RLock()
	defer module.mu.RUnlock()

	if module.closed {
		return 0, errors.New("library is closed")
	}
	if len(module.mapping) == 0 {
		return 0, errors.New("library image is empty")
	}
	if module.symbols == nil {
		return 0, errors.New("symbol table is empty")
	}

	if addr, ok := module.symbols[name]; ok && addr != 0 {
		return addr, nil
	}
	return 0, fmt.Errorf("symbol %q not found", name)
}

func (module *Module) ProcAddressByOrdinal(ordinal uint16) (uintptr, error) {
	_ = ordinal
	return 0, errors.New("ProcAddressByOrdinal is not supported on linux; use ProcAddressByName")
}

func mapELFImage(raw []byte, f *elf.File) (mappedELF, error) {
	pageSize := uint64(unix.Getpagesize())
	if pageSize == 0 {
		return mappedELF{}, errors.New("invalid page size")
	}

	var (
		minVAddr uint64 = ^uint64(0)
		maxVAddr uint64
		progs    []*elf.Prog
	)

	for _, p := range f.Progs {
		if p.Type != elf.PT_LOAD || p.Memsz == 0 {
			continue
		}
		segStart := alignDown64(p.Vaddr, pageSize)
		segEnd := alignUp64(p.Vaddr+p.Memsz, pageSize)
		if segEnd <= segStart {
			return mappedELF{}, fmt.Errorf("invalid PT_LOAD range vaddr=%#x memsz=%#x", p.Vaddr, p.Memsz)
		}
		if segStart < minVAddr {
			minVAddr = segStart
		}
		if segEnd > maxVAddr {
			maxVAddr = segEnd
		}
		progs = append(progs, p)
	}
	if len(progs) == 0 || minVAddr == ^uint64(0) || maxVAddr <= minVAddr {
		return mappedELF{}, errors.New("ELF image has no loadable segments")
	}

	mapSize := maxVAddr - minVAddr
	if mapSize == 0 {
		return mappedELF{}, errors.New("ELF image mapping size is zero")
	}
	mapLen, err := u64ToInt(mapSize)
	if err != nil {
		return mappedELF{}, err
	}

	mapping, err := unix.Mmap(-1, 0, mapLen, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		return mappedELF{}, fmt.Errorf("mmap ELF image: %w", err)
	}
	if len(mapping) == 0 {
		return mappedELF{}, errors.New("mmap ELF image returned empty mapping")
	}

	loadBias := uintptr(unsafe.Pointer(&mapping[0])) - uintptr(minVAddr)
	for _, p := range progs {
		if p.Filesz == 0 {
			continue
		}
		if p.Off > uint64(len(raw)) || p.Filesz > uint64(len(raw))-p.Off {
			_ = unix.Munmap(mapping)
			return mappedELF{}, fmt.Errorf("segment file range out of bounds off=%#x filesz=%#x", p.Off, p.Filesz)
		}
		dstLen, err := u64ToInt(p.Filesz)
		if err != nil {
			_ = unix.Munmap(mapping)
			return mappedELF{}, err
		}
		dst := unsafe.Slice((*byte)(unsafe.Pointer(loadBias+uintptr(p.Vaddr))), dstLen)
		src := raw[p.Off : p.Off+p.Filesz]
		copy(dst, src)
	}

	return mappedELF{
		mapping:  mapping,
		loadBias: loadBias,
		progs:    progs,
	}, nil
}

func applyDynamicRelocations(mapped mappedELF, f *elf.File, resolver *symbolResolver) error {
	if f.Class != elf.ELFCLASS32 && f.Class != elf.ELFCLASS64 {
		return fmt.Errorf("unsupported ELF class: %s", f.Class)
	}
	if f.Data != elf.ELFDATA2LSB {
		return fmt.Errorf("unsupported ELF endianness: %s", f.Data)
	}

	dynSyms, err := f.DynamicSymbols()
	if err != nil {
		return fmt.Errorf("read dynamic symbol table: %w", err)
	}

	for _, sec := range relocationSections(f) {
		data, err := sec.Data()
		if err != nil {
			return fmt.Errorf("read relocation section %s: %w", sec.Name, err)
		}
		if len(data) == 0 {
			continue
		}

		switch sec.Type {
		case elf.SHT_RELA:
			if err := applyRELASection(data, f, mapped, dynSyms, resolver, sec.Name); err != nil {
				return err
			}
		case elf.SHT_REL:
			if err := applyRELSection(data, f, mapped, dynSyms, resolver, sec.Name); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported relocation section type %s in %s", sec.Type, sec.Name)
		}
	}

	return nil
}

func relocationSections(f *elf.File) []*elf.Section {
	names := []string{
		".rela.dyn",
		".rela.plt",
		".rela.plt.sec",
		".rel.dyn",
		".rel.plt",
		".rel.plt.sec",
	}
	out := make([]*elf.Section, 0, len(names))
	for _, name := range names {
		if sec := f.Section(name); sec != nil {
			out = append(out, sec)
		}
	}
	return out
}

func applyRELASection(data []byte, f *elf.File, mapped mappedELF, dynSyms []elf.Symbol, resolver *symbolResolver, sectionName string) error {
	switch f.Class {
	case elf.ELFCLASS64:
		const ent = 24
		if len(data)%ent != 0 {
			return fmt.Errorf("malformed %s: size %d is not a multiple of %d", sectionName, len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			off := binary.LittleEndian.Uint64(data[i : i+8])
			info := binary.LittleEndian.Uint64(data[i+8 : i+16])
			addend := int64(binary.LittleEndian.Uint64(data[i+16 : i+24]))
			if err := applyOneRelocation(f.Machine, f.Class, mapped, dynSyms, resolver, uint32(elf.R_SYM64(info)), uint32(elf.R_TYPE64(info)), off, addend, true); err != nil {
				return fmt.Errorf("%s[%d]: %w", sectionName, i/ent, err)
			}
		}
	case elf.ELFCLASS32:
		const ent = 12
		if len(data)%ent != 0 {
			return fmt.Errorf("malformed %s: size %d is not a multiple of %d", sectionName, len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			off := uint64(binary.LittleEndian.Uint32(data[i : i+4]))
			info := binary.LittleEndian.Uint32(data[i+4 : i+8])
			addend := int64(int32(binary.LittleEndian.Uint32(data[i+8 : i+12])))
			if err := applyOneRelocation(f.Machine, f.Class, mapped, dynSyms, resolver, elf.R_SYM32(info), elf.R_TYPE32(info), off, addend, true); err != nil {
				return fmt.Errorf("%s[%d]: %w", sectionName, i/ent, err)
			}
		}
	default:
		return fmt.Errorf("unsupported ELF class in %s: %s", sectionName, f.Class)
	}
	return nil
}

func applyRELSection(data []byte, f *elf.File, mapped mappedELF, dynSyms []elf.Symbol, resolver *symbolResolver, sectionName string) error {
	switch f.Class {
	case elf.ELFCLASS64:
		const ent = 16
		if len(data)%ent != 0 {
			return fmt.Errorf("malformed %s: size %d is not a multiple of %d", sectionName, len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			off := binary.LittleEndian.Uint64(data[i : i+8])
			info := binary.LittleEndian.Uint64(data[i+8 : i+16])
			if err := applyOneRelocation(f.Machine, f.Class, mapped, dynSyms, resolver, uint32(elf.R_SYM64(info)), uint32(elf.R_TYPE64(info)), off, 0, false); err != nil {
				return fmt.Errorf("%s[%d]: %w", sectionName, i/ent, err)
			}
		}
	case elf.ELFCLASS32:
		const ent = 8
		if len(data)%ent != 0 {
			return fmt.Errorf("malformed %s: size %d is not a multiple of %d", sectionName, len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			off := uint64(binary.LittleEndian.Uint32(data[i : i+4]))
			info := binary.LittleEndian.Uint32(data[i+4 : i+8])
			if err := applyOneRelocation(f.Machine, f.Class, mapped, dynSyms, resolver, elf.R_SYM32(info), elf.R_TYPE32(info), off, 0, false); err != nil {
				return fmt.Errorf("%s[%d]: %w", sectionName, i/ent, err)
			}
		}
	default:
		return fmt.Errorf("unsupported ELF class in %s: %s", sectionName, f.Class)
	}
	return nil
}

func applyOneRelocation(machine elf.Machine, class elf.Class, mapped mappedELF, dynSyms []elf.Symbol, resolver *symbolResolver, symIndex uint32, relocType uint32, offset uint64, addend int64, hasAddend bool) error {
	place := mapped.loadBias + uintptr(offset)

	wordSize := 8
	if class == elf.ELFCLASS32 {
		wordSize = 4
	}
	if !mappedAddressInRange(mapped.mapping, place, wordSize) {
		return fmt.Errorf("relocation target %#x out of mapped image", offset)
	}

	if !hasAddend {
		switch class {
		case elf.ELFCLASS64:
			addend = int64(readU64(place))
		case elf.ELFCLASS32:
			addend = int64(int32(readU32(place)))
		default:
			return fmt.Errorf("unsupported ELF class: %s", class)
		}
	}

	var symValue uintptr
	if symIndex != 0 {
		resolved, err := resolveRelocationSymbol(symIndex, dynSyms, mapped.loadBias, resolver)
		if err != nil {
			return err
		}
		symValue = resolved
	}

	switch machine {
	case elf.EM_X86_64:
		return applyX8664Reloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	case elf.EM_386:
		return apply386Reloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	case elf.EM_AARCH64:
		return applyAArch64Reloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	case elf.EM_ARM:
		return applyARMReloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	case elf.EM_RISCV:
		return applyRISCV64Reloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	case elf.EM_PPC64:
		return applyPPC64LEReloc(relocType, place, mapped.loadBias, symValue, addend, mapped.tlsOffset, mapped.hasTLS)
	default:
		return fmt.Errorf("unsupported machine for relocation: %s", machine)
	}
}

func applyX8664Reloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_X86_64(relocType) {
	case elf.R_X86_64_NONE:
		return nil
	case elf.R_X86_64_RELATIVE:
		writeU64(place, uint64(int64(loadBias)+addend))
		return nil
	case elf.R_X86_64_TPOFF64:
		if !hasTLS {
			return errors.New("x86_64 static TLS relocation has no reserved host TLS slot")
		}
		writeU64(place, uint64(tlsOffset+addend))
		return nil
	case elf.R_X86_64_JMP_SLOT, elf.R_X86_64_GLOB_DAT, elf.R_X86_64_64:
		writeU64(place, uint64(int64(symValue)+addend))
		return nil
	case elf.R_X86_64_32:
		v := int64(symValue) + addend
		if v < 0 || v > 0xffffffff {
			return fmt.Errorf("x86_64 32 relocation overflow: value=%d", v)
		}
		writeU32(place, uint32(v))
		return nil
	case elf.R_X86_64_32S:
		v := int64(symValue) + addend
		if v < -0x80000000 || v > 0x7fffffff {
			return fmt.Errorf("x86_64 32S relocation overflow: value=%d", v)
		}
		writeU32(place, uint32(int32(v)))
		return nil
	case elf.R_X86_64_PC32:
		v := int64(symValue) + addend - int64(place)
		if v < -0x80000000 || v > 0x7fffffff {
			return fmt.Errorf("x86_64 PC32 relocation overflow: value=%d", v)
		}
		writeU32(place, uint32(int32(v)))
		return nil
	default:
		return fmt.Errorf("unsupported x86_64 relocation type: %d", relocType)
	}
}

func apply386Reloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_386(relocType) {
	case elf.R_386_NONE:
		return nil
	case elf.R_386_RELATIVE:
		writeU32(place, uint32(int64(loadBias)+addend))
		return nil
	case elf.R_386_TLS_TPOFF:
		if !hasTLS {
			return errors.New("386 static TLS relocation has no reserved host TLS slot")
		}
		writeU32(place, uint32(tlsOffset+addend))
		return nil
	case elf.R_386_JMP_SLOT, elf.R_386_GLOB_DAT:
		writeU32(place, uint32(symValue))
		return nil
	case elf.R_386_32, elf.R_386_32PLT:
		writeU32(place, uint32(int64(symValue)+addend))
		return nil
	case elf.R_386_PC32:
		v := int64(symValue) + addend - int64(place)
		if v < -0x80000000 || v > 0x7fffffff {
			return fmt.Errorf("386 PC32 relocation overflow: value=%d", v)
		}
		writeU32(place, uint32(int32(v)))
		return nil
	default:
		return fmt.Errorf("unsupported 386 relocation type: %d", relocType)
	}
}

func applyAArch64Reloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_AARCH64(relocType) {
	case elf.R_AARCH64_NONE:
		return nil
	case elf.R_AARCH64_RELATIVE:
		writeU64(place, uint64(int64(loadBias)+addend))
		return nil
	case elf.R_AARCH64_TLS_TPREL64:
		if !hasTLS {
			return errors.New("arm64 static TLS relocation has no reserved host TLS slot")
		}
		writeU64(place, uint64(tlsOffset+addend))
		return nil
	case elf.R_AARCH64_JUMP_SLOT, elf.R_AARCH64_GLOB_DAT, elf.R_AARCH64_ABS64:
		writeU64(place, uint64(int64(symValue)+addend))
		return nil
	default:
		return fmt.Errorf("unsupported aarch64 relocation type: %d", relocType)
	}
}

func applyARMReloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_ARM(relocType) {
	case elf.R_ARM_NONE:
		return nil
	case elf.R_ARM_RELATIVE:
		writeU32(place, uint32(int64(loadBias)+addend))
		return nil
	case elf.R_ARM_TLS_TPOFF32:
		if !hasTLS {
			return errors.New("arm static TLS relocation has no reserved host TLS slot")
		}
		writeU32(place, uint32(tlsOffset+addend))
		return nil
	case elf.R_ARM_JUMP_SLOT, elf.R_ARM_GLOB_DAT:
		writeU32(place, uint32(symValue))
		return nil
	case elf.R_ARM_ABS32:
		writeU32(place, uint32(int64(symValue)+addend))
		return nil
	case elf.R_ARM_REL32:
		writeU32(place, uint32(int64(symValue)+addend-int64(place)))
		return nil
	default:
		return fmt.Errorf("unsupported arm relocation type: %d", relocType)
	}
}

func applyRISCV64Reloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_RISCV(relocType) {
	case elf.R_RISCV_NONE:
		return nil
	case elf.R_RISCV_RELATIVE:
		writeU64(place, uint64(int64(loadBias)+addend))
		return nil
	case elf.R_RISCV_TLS_TPREL64:
		if !hasTLS {
			return errors.New("riscv64 static TLS relocation has no reserved host TLS slot")
		}
		writeU64(place, uint64(tlsOffset+addend))
		return nil
	case elf.R_RISCV_JUMP_SLOT:
		writeU64(place, uint64(symValue))
		return nil
	case elf.R_RISCV_64:
		writeU64(place, uint64(int64(symValue)+addend))
		return nil
	default:
		return fmt.Errorf("unsupported riscv64 relocation type: %d", relocType)
	}
}

func applyPPC64LEReloc(relocType uint32, place uintptr, loadBias uintptr, symValue uintptr, addend int64, tlsOffset int64, hasTLS bool) error {
	switch elf.R_PPC64(relocType) {
	case elf.R_PPC64_NONE:
		return nil
	case elf.R_PPC64_RELATIVE:
		writeU64(place, uint64(int64(loadBias)+addend))
		return nil
	case elf.R_PPC64_TPREL64:
		if !hasTLS {
			return errors.New("ppc64le static TLS relocation has no reserved host TLS slot")
		}
		writeU64(place, uint64(tlsOffset+addend))
		return nil
	case elf.R_PPC64_JMP_SLOT, elf.R_PPC64_GLOB_DAT:
		writeU64(place, uint64(symValue))
		return nil
	case elf.R_PPC64_ADDR64:
		writeU64(place, uint64(int64(symValue)+addend))
		return nil
	default:
		return fmt.Errorf("unsupported ppc64le relocation type: %d", relocType)
	}
}

func resolveRelocationSymbol(symIndex uint32, dynSyms []elf.Symbol, loadBias uintptr, resolver *symbolResolver) (uintptr, error) {
	if symIndex == 0 {
		return 0, nil
	}

	sym, ok := dynSymbolByIndex(dynSyms, symIndex)
	if !ok {
		return 0, fmt.Errorf("relocation references invalid symbol index %d", symIndex)
	}
	bind := elf.ST_BIND(sym.Info)
	if sym.Section != elf.SHN_UNDEF && sym.Value != 0 {
		return loadBias + uintptr(sym.Value), nil
	}
	if sym.Name == "" {
		return 0, fmt.Errorf("relocation symbol index %d is undefined and unnamed", symIndex)
	}

	addr, err := resolver.ResolveSymbol(sym)
	if err != nil {
		if bind == elf.STB_WEAK {
			// Undefined weak symbols are optional, but they still bind to an
			// available definition before falling back to zero.
			return 0, nil
		}
		return 0, fmt.Errorf("resolve external symbol %q: %w", sym.Name, err)
	}
	if addr == 0 && bind == elf.STB_WEAK {
		return 0, nil
	}
	if addr == 0 {
		return 0, fmt.Errorf("resolved external symbol %q to nil address", sym.Name)
	}
	return addr, nil
}

func dynSymbolByIndex(dynSyms []elf.Symbol, symIndex uint32) (elf.Symbol, bool) {
	// debug/elf.DynamicSymbols omits the null symbol at dynsym index 0.
	if symIndex == 0 {
		return elf.Symbol{}, false
	}
	idx := int(symIndex - 1)
	if idx < 0 || idx >= len(dynSyms) {
		return elf.Symbol{}, false
	}
	return dynSyms[idx], true
}

func applySegmentProtections(mapped mappedELF) error {
	pageSize := uint64(unix.Getpagesize())
	if pageSize == 0 {
		return errors.New("invalid page size")
	}

	for _, p := range mapped.progs {
		if p.Type != elf.PT_LOAD || p.Memsz == 0 {
			continue
		}
		if p.Flags&elf.PF_W != 0 && p.Flags&elf.PF_X != 0 {
			return fmt.Errorf("PT_LOAD vaddr=%#x requests writable and executable memory", p.Vaddr)
		}
		start := alignDown64(p.Vaddr, pageSize)
		end := alignUp64(p.Vaddr+p.Memsz, pageSize)
		if end <= start {
			continue
		}
		length, err := u64ToInt(end - start)
		if err != nil {
			return err
		}
		addr := mapped.loadBias + uintptr(start)
		if !mappedAddressInRange(mapped.mapping, addr, length) {
			return fmt.Errorf("segment protection range out of mapped image vaddr=%#x len=%#x", start, end-start)
		}
		seg := unsafe.Slice((*byte)(unsafe.Pointer(addr)), length)
		if err := unix.Mprotect(seg, progFlagsToProt(p.Flags)); err != nil {
			return fmt.Errorf("mprotect PT_LOAD vaddr=%#x memsz=%#x: %w", p.Vaddr, p.Memsz, err)
		}
	}
	return nil
}

func collectELFInitializers(mapped mappedELF, class elf.Class, info dynamicInitInfo) ([]uintptr, error) {
	initializers := make([]uintptr, 0)
	var err error
	initializers, err = appendDynamicInitArray(initializers, mapped, class, info.preinitArr, info.preinitSz, "DT_PREINIT_ARRAY")
	if err != nil {
		return nil, err
	}
	initializers, err = appendDynamicInitFn(initializers, mapped, uintptr(info.init), "DT_INIT")
	if err != nil {
		return nil, err
	}
	initializers, err = appendDynamicInitArray(initializers, mapped, class, info.initArray, info.initArraySz, "DT_INIT_ARRAY")
	if err != nil {
		return nil, err
	}
	return initializers, nil
}

func collectELFFinalizers(mapped mappedELF, class elf.Class, info dynamicInitInfo) ([]uintptr, error) {
	finalizers, err := appendDynamicInitArray(nil, mapped, class, info.finiArray, info.finiArraySz, "DT_FINI_ARRAY")
	if err != nil {
		return nil, err
	}
	// The System V ABI requires DT_FINI_ARRAY entries to run in reverse order.
	for left, right := 0, len(finalizers)-1; left < right; left, right = left+1, right-1 {
		finalizers[left], finalizers[right] = finalizers[right], finalizers[left]
	}
	return appendDynamicInitFn(finalizers, mapped, uintptr(info.fini), "DT_FINI")
}

func parseDynamicInitInfo(f *elf.File) (dynamicInitInfo, error) {
	var info dynamicInitInfo
	if f == nil {
		return info, nil
	}
	sec := f.Section(".dynamic")
	if sec == nil {
		return info, nil
	}
	data, err := sec.Data()
	if err != nil {
		return info, fmt.Errorf("read .dynamic section: %w", err)
	}
	if len(data) == 0 {
		return info, nil
	}

	switch f.Class {
	case elf.ELFCLASS64:
		const ent = 16
		if len(data)%ent != 0 {
			return info, fmt.Errorf("malformed .dynamic section: size %d is not a multiple of %d", len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			tag := int64(binary.LittleEndian.Uint64(data[i : i+8]))
			val := binary.LittleEndian.Uint64(data[i+8 : i+16])
			if tag == dynTagNull {
				break
			}
			switch tag {
			case dynTagInit:
				info.init = val
			case dynTagFini:
				info.fini = val
			case dynTagInitArray:
				info.initArray = val
			case dynTagFiniArray:
				info.finiArray = val
			case dynTagInitArraySz:
				info.initArraySz = val
			case dynTagFiniArraySz:
				info.finiArraySz = val
			case dynTagPreinitArr:
				info.preinitArr = val
			case dynTagPreinitSz:
				info.preinitSz = val
			}
		}
	case elf.ELFCLASS32:
		const ent = 8
		if len(data)%ent != 0 {
			return info, fmt.Errorf("malformed .dynamic section: size %d is not a multiple of %d", len(data), ent)
		}
		for i := 0; i < len(data); i += ent {
			tag := int64(int32(binary.LittleEndian.Uint32(data[i : i+4])))
			val := uint64(binary.LittleEndian.Uint32(data[i+4 : i+8]))
			if tag == dynTagNull {
				break
			}
			switch tag {
			case dynTagInit:
				info.init = val
			case dynTagFini:
				info.fini = val
			case dynTagInitArray:
				info.initArray = val
			case dynTagFiniArray:
				info.finiArray = val
			case dynTagInitArraySz:
				info.initArraySz = val
			case dynTagFiniArraySz:
				info.finiArraySz = val
			case dynTagPreinitArr:
				info.preinitArr = val
			case dynTagPreinitSz:
				info.preinitSz = val
			}
		}
	default:
		return info, fmt.Errorf("unsupported ELF class for .dynamic parsing: %s", f.Class)
	}
	return info, nil
}

func appendDynamicInitFn(initializers []uintptr, mapped mappedELF, fn uintptr, source string) ([]uintptr, error) {
	if fn == 0 {
		return initializers, nil
	}
	resolved, ok := normalizeInitFnAddress(mapped, fn)
	if !ok {
		return nil, fmt.Errorf("%s points outside mapped image: %#x", source, fn)
	}
	return append(initializers, resolved), nil
}

func appendDynamicInitArray(initializers []uintptr, mapped mappedELF, class elf.Class, arrayVAddr uint64, arraySz uint64, source string) ([]uintptr, error) {
	if arrayVAddr == 0 || arraySz == 0 {
		return initializers, nil
	}

	entrySize := 8
	if class == elf.ELFCLASS32 {
		entrySize = 4
	}
	if arraySz%uint64(entrySize) != 0 {
		return nil, fmt.Errorf("%s has malformed size %#x for entry size %d", source, arraySz, entrySize)
	}
	arrayLen, err := u64ToInt(arraySz)
	if err != nil {
		return nil, fmt.Errorf("%s size does not fit in int: %w", source, err)
	}

	arrayAddr := mapped.loadBias + uintptr(arrayVAddr)
	if !mappedAddressInRange(mapped.mapping, arrayAddr, arrayLen) {
		return nil, fmt.Errorf("%s range %#x..%#x is outside mapped image", source, arrayVAddr, arrayVAddr+arraySz)
	}

	count := int(arraySz / uint64(entrySize))
	for i := 0; i < count; i++ {
		entryAddr := arrayAddr + uintptr(i*entrySize)
		var fn uintptr
		if entrySize == 8 {
			fn = uintptr(readU64(entryAddr))
		} else {
			fn = uintptr(readU32(entryAddr))
		}
		if fn == 0 || fn == ^uintptr(0) {
			continue
		}
		resolved, ok := normalizeInitFnAddress(mapped, fn)
		if !ok {
			return nil, fmt.Errorf("%s[%d] points outside mapped image: %#x", source, i, fn)
		}
		initializers = append(initializers, resolved)
	}
	return initializers, nil
}

func normalizeInitFnAddress(mapped mappedELF, fn uintptr) (uintptr, bool) {
	if fn == 0 {
		return 0, false
	}
	if mappedAddressInRange(mapped.mapping, fn, 1) {
		return fn, true
	}
	rebased := mapped.loadBias + fn
	if mappedAddressInRange(mapped.mapping, rebased, 1) {
		return rebased, true
	}
	return 0, false
}

func buildExportedSymbolTable(f *elf.File, loadBias uintptr) map[string]uintptr {
	out := make(map[string]uintptr)
	if dynSyms, err := f.DynamicSymbols(); err == nil {
		addELFSymbols(out, dynSyms, loadBias)
	}
	if syms, err := f.Symbols(); err == nil {
		addELFSymbols(out, syms, loadBias)
	}
	return out
}

func addELFSymbols(dst map[string]uintptr, symbols []elf.Symbol, loadBias uintptr) {
	for _, sym := range symbols {
		if sym.Name == "" || sym.Value == 0 || sym.Section == elf.SHN_UNDEF {
			continue
		}
		bind := elf.ST_BIND(sym.Info)
		if bind != elf.STB_GLOBAL && bind != elf.STB_WEAK {
			continue
		}
		typ := elf.ST_TYPE(sym.Info)
		if typ != elf.STT_FUNC && typ != elf.STT_NOTYPE {
			continue
		}
		addr := loadBias + uintptr(sym.Value)
		if _, ok := dst[sym.Name]; !ok {
			dst[sym.Name] = addr
		}
		if at := strings.IndexByte(sym.Name, '@'); at > 0 {
			base := sym.Name[:at]
			if _, ok := dst[base]; !ok {
				dst[base] = addr
			}
		}
	}
}

func newSymbolResolver() *symbolResolver {
	resolver := &symbolResolver{
		resolved:     make(map[string]uintptr),
		misses:       make(map[string]error),
		opened:       make(map[string]uintptr),
		openLibrary:  openWithDlopen,
		closeLibrary: closeWithDlclose,
	}
	if modules, err := runtimeModules(); err == nil {
		resolver.modules = modules
	}
	if api, err := getLinuxDynAPI(); err == nil {
		resolver.api = api
	}
	return resolver
}

func (resolver *symbolResolver) primeDependencies(f *elf.File) error {
	needed, err := collectNeededLibraries(f)
	if err != nil {
		return err
	}
	if err := resolver.primeNeededLibraries(needed); err != nil {
		return err
	}
	for _, library := range commonLinuxDependencies() {
		_ = resolver.ensureLibraryLoaded(library, false)
	}
	return nil
}

func (resolver *symbolResolver) primeNeededLibraries(libraries []string) error {
	for _, library := range libraries {
		if err := resolver.ensureLibraryLoaded(library, true); err != nil {
			return fmt.Errorf("load DT_NEEDED %q: %w", library, err)
		}
	}
	return nil
}

func collectNeededLibraries(f *elf.File) ([]string, error) {
	if f == nil {
		return nil, nil
	}
	imports, err := f.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("read DT_NEEDED entries: %w", err)
	}
	if len(imports) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(imports))
	seen := make(map[string]struct{}, len(imports))
	for _, lib := range imports {
		lib = strings.TrimSpace(lib)
		if lib == "" {
			continue
		}
		if _, exists := seen[lib]; exists {
			continue
		}
		seen[lib] = struct{}{}
		out = append(out, lib)
	}
	return out, nil
}

func commonLinuxDependencies() []string {
	if runtime.GOOS == "freebsd" {
		return []string{
			"libc.so.7",
			"libthr.so.3",
		}
	}

	deps := []string{
		"libc.so.6",
		"libdl.so.2",
		"libpthread.so.0",
	}
	switch runtime.GOARCH {
	case "amd64":
		deps = append(deps, "ld-linux-x86-64.so.2", "ld-musl-x86_64.so.1")
	case "386":
		deps = append(deps, "ld-linux.so.2", "ld-musl-i386.so.1")
	case "arm64":
		deps = append(deps, "ld-linux-aarch64.so.1", "ld-musl-aarch64.so.1")
	case "arm":
		deps = append(deps, "ld-linux-armhf.so.3", "ld-linux.so.3", "ld-musl-armhf.so.1")
	case "riscv64":
		deps = append(deps, "ld-linux-riscv64-lp64d.so.1", "ld-musl-riscv64.so.1")
	case "ppc64le":
		deps = append(deps, "ld64.so.2", "ld-musl-powerpc64le.so.1")
	}
	return deps
}

func (resolver *symbolResolver) ensureLibraryLoaded(name string, requireOwnedReference bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if handle, ok := resolver.openedLibrary(name); ok {
		resolver.opened[name] = handle
		return nil
	}
	if !requireOwnedReference && resolver.hasModule(name) {
		return nil
	}
	if resolver.api == nil || resolver.api.dlopen == 0 || resolver.api.dlclose == 0 {
		return errors.New("dlopen/dlclose is unavailable")
	}
	if resolver.openLibrary == nil || resolver.closeLibrary == nil {
		return errors.New("dynamic library ownership hooks are unavailable")
	}

	var lastErr error
	for _, candidate := range dlopenCandidates(name) {
		if candidate == "" {
			continue
		}
		if handle, ok := resolver.openedLibrary(candidate); ok {
			resolver.opened[name] = handle
			return nil
		}
		if !requireOwnedReference && resolver.hasModule(candidate) {
			return nil
		}

		handle, err := resolver.openLibrary(resolver.api, candidate)
		if err != nil {
			lastErr = err
			continue
		}
		if handle == 0 {
			continue
		}
		resolver.rememberOpenedLibrary(name, candidate, handle)
		// Every successful dlopen owns one reference, including when the dynamic
		// loader returns the same numeric handle for two distinct acquisitions.
		resolver.ownedDlopen = append(resolver.ownedDlopen, handle)
		resolver.refreshModules()
		return nil
	}
	if !requireOwnedReference && resolver.hasModule(name) {
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("dlopen(%s): returned nil handle", name)
	}
	return lastErr
}

func (resolver *symbolResolver) openedLibrary(name string) (uintptr, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, false
	}
	if handle, ok := resolver.opened[name]; ok && handle != 0 {
		return handle, true
	}
	base := filepath.Base(name)
	if base != name {
		if handle, ok := resolver.opened[base]; ok && handle != 0 {
			return handle, true
		}
	}
	return 0, false
}

func (resolver *symbolResolver) rememberOpenedLibrary(name string, candidate string, handle uintptr) {
	for _, alias := range []string{name, candidate, filepath.Base(name), filepath.Base(candidate)} {
		alias = strings.TrimSpace(alias)
		if alias != "" && alias != "." {
			resolver.opened[alias] = handle
		}
	}
}

func (resolver *symbolResolver) takeOwnedLibraries() []uintptr {
	handles := resolver.ownedDlopen
	resolver.ownedDlopen = nil
	return handles
}

func (resolver *symbolResolver) closeOwnedLibraries() {
	closeDlopenHandles(resolver.api, resolver.closeLibrary, resolver.takeOwnedLibraries())
}

func closeDlopenHandles(api *linuxDynAPI, closeLibrary func(*linuxDynAPI, uintptr) error, handles []uintptr) {
	if closeLibrary == nil {
		return
	}
	for index := len(handles) - 1; index >= 0; index-- {
		_ = closeLibrary(api, handles[index])
	}
}

func (resolver *symbolResolver) refreshModules() {
	if modules, err := runtimeModules(); err == nil {
		resolver.modules = modules
	}
}

func (resolver *symbolResolver) hasModule(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	base := filepath.Base(name)
	for _, module := range resolver.modules {
		if module.path == name {
			return true
		}
		if base != "" && filepath.Base(module.path) == base {
			return true
		}
	}
	return false
}

func dlopenCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	out := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	add(name)
	base := filepath.Base(name)
	add(base)

	if runtime.GOOS == "freebsd" {
		switch base {
		case "libc.so":
			add("libc.so.7")
		case "libm.so":
			add("libm.so.5")
		case "libpthread.so", "libthr.so":
			add("libthr.so.3")
		}
	} else {
		switch base {
		case "libc.so":
			add("libc.so.6")
		case "libdl.so":
			add("libdl.so.2")
		case "libpthread.so":
			add("libpthread.so.0")
		}
	}
	if idx := strings.Index(base, ".so."); idx > 0 {
		add(base[:idx+3])
	}

	for _, dir := range linuxLibrarySearchDirs() {
		add(filepath.Join(dir, base))
	}
	return out
}

func linuxLibrarySearchDirs() []string {
	if runtime.GOOS == "freebsd" {
		return []string{
			"/lib",
			"/usr/lib",
			"/usr/local/lib",
			"/usr/local/lib/compat",
			"/libexec",
			"/usr/libexec",
		}
	}

	dirs := []string{"/lib", "/lib64", "/usr/lib", "/usr/lib64"}
	switch runtime.GOARCH {
	case "amd64":
		dirs = append(dirs, "/lib/x86_64-linux-gnu", "/usr/lib/x86_64-linux-gnu")
	case "386":
		dirs = append(dirs, "/lib/i386-linux-gnu", "/usr/lib/i386-linux-gnu")
	case "arm64":
		dirs = append(dirs, "/lib/aarch64-linux-gnu", "/usr/lib/aarch64-linux-gnu")
	case "arm":
		dirs = append(dirs, "/lib/arm-linux-gnueabihf", "/usr/lib/arm-linux-gnueabihf")
	case "riscv64":
		dirs = append(dirs, "/lib/riscv64-linux-gnu", "/usr/lib/riscv64-linux-gnu")
	case "ppc64le":
		dirs = append(dirs, "/lib/powerpc64le-linux-gnu", "/usr/lib/powerpc64le-linux-gnu")
	}
	return dirs
}

func (resolver *symbolResolver) Resolve(name string) (uintptr, error) {
	if addr, ok := resolver.resolved[name]; ok {
		return addr, nil
	}
	if err, ok := resolver.misses[name]; ok {
		return 0, err
	}

	if resolver.api != nil {
		// Prefer the native loader once dlsym has been bootstrapped. In addition
		// to honoring loader scope and interposition, dlsym evaluates GNU
		// IFUNC resolvers. Returning base+st_value for an IFUNC would bind its
		// resolver as the callable symbol and crash on the first invocation.
		if addr, err := resolveWithDLSym(resolver.api, name); err == nil && addr != 0 {
			resolver.resolved[name] = addr
			return addr, nil
		}
	}

	if addr, err := resolveFromRuntimeModules(resolver.modules, name); err == nil && addr != 0 {
		resolver.resolved[name] = addr
		return addr, nil
	}

	if resolver.api != nil && resolver.api.dlopen != 0 {
		for _, dep := range commonLinuxDependencies() {
			_ = resolver.ensureLibraryLoaded(dep, false)
		}
		if addr, err := resolveWithDLSym(resolver.api, name); err == nil && addr != 0 {
			resolver.resolved[name] = addr
			return addr, nil
		}
		if addr, err := resolveFromRuntimeModules(resolver.modules, name); err == nil && addr != 0 {
			resolver.resolved[name] = addr
			return addr, nil
		}
	}

	if at := strings.IndexByte(name, '@'); at > 0 {
		base := name[:at]
		if base != "" && base != name {
			if addr, err := resolver.Resolve(base); err == nil && addr != 0 {
				resolver.resolved[name] = addr
				return addr, nil
			}
		}
	}

	err := fmt.Errorf("unresolved external symbol %q", name)
	resolver.misses[name] = err
	return 0, err
}

func (resolver *symbolResolver) ResolveSymbol(sym elf.Symbol) (uintptr, error) {
	if resolver.resolveSymbol != nil {
		return resolver.resolveSymbol(sym)
	}
	return resolver.resolveDynamicSymbol(sym, runtime.GOOS == "freebsd")
}

func (resolver *symbolResolver) resolveDynamicSymbol(sym elf.Symbol, requireExactVersion bool) (uintptr, error) {
	if requireExactVersion && sym.Version != "" {
		if resolver.api == nil {
			return 0, fmt.Errorf("resolve versioned external symbol %s@%s: dynamic loader API is unavailable", sym.Name, sym.Version)
		}
		addr, err := resolveWithDLHandle(resolver.api, resolver.api.defaultHandle, sym.Name, sym.Version)
		if err != nil {
			return 0, fmt.Errorf("resolve versioned external symbol %s@%s: %w", sym.Name, sym.Version, err)
		}
		return addr, nil
	}
	if sym.Section == elf.SHN_UNDEF && elf.ST_BIND(sym.Info) == elf.STB_WEAK {
		// Preserve the legacy loader's weak-import behavior. Recursive loads use
		// resolveSymbol above and attempt graph lookup before resolving to zero.
		return 0, nil
	}
	return resolver.Resolve(sym.Name)
}

func resolveFromRuntimeModules(modules []runtimeELFModule, name string) (uintptr, error) {
	for _, module := range modules {
		off, err := findELFSymbolOffset(module.path, name)
		if err != nil || off == 0 {
			continue
		}
		return module.base + off, nil
	}
	return 0, fmt.Errorf("symbol %q not found in loaded ELF modules", name)
}

func runtimeModules() ([]runtimeELFModule, error) {
	if runtime.GOOS != "linux" {
		// FreeBSD normally runs without procfs mounted. Its dynamic API is
		// bootstrapped through native dlfcn entrypoints instead.
		return nil, nil
	}

	entries, err := readProcMaps()
	if err != nil {
		return nil, err
	}

	byPath := make(map[string]runtimeELFModule)
	for _, entry := range entries {
		if entry.path == "" || !strings.HasPrefix(entry.path, "/") {
			continue
		}
		if entry.start < entry.offset {
			continue
		}
		base := entry.start - entry.offset
		current, exists := byPath[entry.path]
		if !exists || base < current.base {
			byPath[entry.path] = runtimeELFModule{
				path:  entry.path,
				base:  base,
				score: libcPathScore(entry.path),
			}
		}
	}

	modules := make([]runtimeELFModule, 0, len(byPath))
	for _, module := range byPath {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].score != modules[j].score {
			return modules[i].score > modules[j].score
		}
		return modules[i].path < modules[j].path
	})
	return modules, nil
}

func resolveWithDLSym(api *linuxDynAPI, name string) (uintptr, error) {
	if api == nil || api.dlsym == 0 {
		return 0, errors.New("dlsym is unavailable")
	}
	cName, err := cStringBytes(name)
	if err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}
	sym := callExportFunction(api.dlsym, api.defaultHandle, cStringPtr(cName))
	runtime.KeepAlive(cName)
	if api.dlerror != 0 {
		if err := lastDLErrorLocked(api); err != nil {
			return 0, fmt.Errorf("dlsym(%s): %w", name, err)
		}
	}
	if sym == 0 {
		return 0, fmt.Errorf("dlsym(%s): symbol address is nil", name)
	}
	return sym, nil
}

func resolveWithDLHandle(api *linuxDynAPI, handle uintptr, name string, version string) (uintptr, error) {
	if api == nil || api.dlsym == 0 {
		return 0, errors.New("dlsym is unavailable")
	}
	nameBytes, err := cStringBytes(name)
	if err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}

	var address uintptr
	if version != "" {
		if api.dlvsym == 0 {
			return 0, fmt.Errorf("dlvsym is unavailable for versioned symbol %s@%s", name, version)
		}
		versionBytes, err := cStringBytes(version)
		if err != nil {
			return 0, err
		}
		address = callExportFunction(api.dlvsym, handle, cStringPtr(nameBytes), cStringPtr(versionBytes))
		runtime.KeepAlive(versionBytes)
	} else {
		address = callExportFunction(api.dlsym, handle, cStringPtr(nameBytes))
	}
	runtime.KeepAlive(nameBytes)
	if api.dlerror != 0 {
		if err := lastDLErrorLocked(api); err != nil {
			return 0, err
		}
	}
	if address == 0 {
		return 0, fmt.Errorf("symbol %q resolved to nil", name)
	}
	return address, nil
}

func openWithDlopen(api *linuxDynAPI, name string) (uintptr, error) {
	if api == nil || api.dlopen == 0 {
		return 0, errors.New("dlopen is unavailable")
	}
	cName, err := cStringBytes(name)
	if err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}
	handle := callExportFunction(api.dlopen, cStringPtr(cName), uintptr(rtldNow|rtldGlobal))
	runtime.KeepAlive(cName)
	if api.dlerror != 0 {
		if err := lastDLErrorLocked(api); err != nil {
			return 0, fmt.Errorf("dlopen(%s): %w", name, err)
		}
	}
	if handle == 0 {
		return 0, fmt.Errorf("dlopen(%s): symbol handle is nil", name)
	}
	return handle, nil
}

func closeWithDlclose(api *linuxDynAPI, handle uintptr) error {
	if api == nil || api.dlclose == 0 {
		return errors.New("dlclose is unavailable")
	}
	if handle == 0 {
		return nil
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}
	status := callExportFunction(api.dlclose, handle)
	if status != 0 {
		if err := lastDLErrorLocked(api); err != nil {
			return fmt.Errorf("dlclose(%#x): %w", handle, err)
		}
		return fmt.Errorf("dlclose(%#x) failed with status %d", handle, status)
	}
	return nil
}

func mappedAddressInRange(mapping []byte, addr uintptr, size int) bool {
	if len(mapping) == 0 || size < 0 {
		return false
	}
	start := uintptr(unsafe.Pointer(&mapping[0]))
	end := start + uintptr(len(mapping))
	if addr < start {
		return false
	}
	if uintptr(size) > end-addr {
		return false
	}
	return true
}

func progFlagsToProt(flags elf.ProgFlag) int {
	prot := 0
	if flags&elf.PF_R != 0 {
		prot |= unix.PROT_READ
	}
	if flags&elf.PF_W != 0 {
		prot |= unix.PROT_WRITE
	}
	if flags&elf.PF_X != 0 {
		prot |= unix.PROT_EXEC
	}
	return prot
}

func alignDown64(v, a uint64) uint64 {
	if a == 0 {
		return v
	}
	return v &^ (a - 1)
}

func alignUp64(v, a uint64) uint64 {
	if a == 0 {
		return v
	}
	return (v + (a - 1)) &^ (a - 1)
}

func u64ToInt(v uint64) (int, error) {
	max := ^uint(0) >> 1
	if v > uint64(max) {
		return 0, fmt.Errorf("value %d does not fit in int", v)
	}
	return int(v), nil
}

func readU32(addr uintptr) uint32 {
	b := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 4)
	return binary.LittleEndian.Uint32(b)
}

func writeU32(addr uintptr, v uint32) {
	b := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 4)
	binary.LittleEndian.PutUint32(b, v)
}

func readU64(addr uintptr) uint64 {
	b := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 8)
	return binary.LittleEndian.Uint64(b)
}

func writeU64(addr uintptr, v uint64) {
	b := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 8)
	binary.LittleEndian.PutUint64(b, v)
}

func cStringBytes(s string) ([]byte, error) {
	if strings.ContainsRune(s, '\x00') {
		return nil, errors.New("string contains NUL")
	}
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b, nil
}

func cStringPtr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func cStringFromPtr(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	const maxLen = 1 << 20
	buf := make([]byte, 0, 64)
	for i := 0; i < maxLen; i++ {
		ch := *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
		if ch == 0 {
			return string(buf)
		}
		buf = append(buf, ch)
	}
	return string(buf)
}

// lastDLErrorLocked must be called on the same locked OS thread as the loader
// operation whose error it reads.
func lastDLErrorLocked(api *linuxDynAPI) error {
	if api == nil || api.dlerror == 0 {
		return nil
	}
	msg := cStringFromPtr(callExportFunction(api.dlerror))
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

func getLinuxDynAPI() (*linuxDynAPI, error) {
	linuxAPIOnce.Do(func() {
		linuxAPIErr = initLinuxDynAPI()
	})
	if linuxAPIErr != nil {
		return nil, linuxAPIErr
	}
	return &linuxAPI, nil
}

type procMapEntry struct {
	start  uintptr
	offset uintptr
	perms  string
	path   string
}

func resolveRuntimeAPISymbol(modules []runtimeELFModule, symbol string) (uintptr, error) {
	for _, module := range modules {
		off, err := findELFSymbolOffset(module.path, symbol)
		if err != nil || off == 0 {
			continue
		}
		return module.base + off, nil
	}
	return 0, fmt.Errorf("symbol %q not found in runtime modules", symbol)
}

func libcPathScore(path string) int {
	p := strings.ToLower(path)
	switch {
	case strings.Contains(p, "libc.so"):
		return 100
	case strings.Contains(p, "libc-"):
		return 95
	case strings.Contains(p, "ld-musl"):
		return 90
	case strings.Contains(p, "musl"):
		return 85
	case strings.Contains(p, "ld-linux"):
		return 80
	default:
		return -1
	}
}

func readProcMaps() ([]procMapEntry, error) {
	raw, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return nil, fmt.Errorf("read /proc/self/maps: %w", err)
	}

	lines := strings.Split(string(raw), "\n")
	entries := make([]procMapEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.Contains(fields[1], "x") {
			continue
		}

		rangeParts := strings.SplitN(fields[0], "-", 2)
		if len(rangeParts) != 2 {
			continue
		}
		start, startErr := parseHexUintptr(rangeParts[0])
		offset, offsetErr := parseHexUintptr(fields[2])
		if startErr != nil || offsetErr != nil {
			continue
		}

		path := ""
		if len(fields) >= 6 {
			path = strings.Join(fields[5:], " ")
			path = strings.TrimSuffix(path, " (deleted)")
		}
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}

		entries = append(entries, procMapEntry{
			start:  start,
			offset: offset,
			perms:  fields[1],
			path:   path,
		})
	}
	return entries, nil
}

func parseHexUintptr(s string) (uintptr, error) {
	var out uintptr
	for _, r := range s {
		out <<= 4
		switch {
		case r >= '0' && r <= '9':
			out += uintptr(r - '0')
		case r >= 'a' && r <= 'f':
			out += uintptr(r-'a') + 10
		case r >= 'A' && r <= 'F':
			out += uintptr(r-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex string %q", s)
		}
	}
	return out, nil
}

func findELFSymbolOffset(path string, symbol string) (uintptr, error) {
	f, err := elf.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open elf %s: %w", path, err)
	}
	defer f.Close()

	if syms, err := f.DynamicSymbols(); err == nil {
		if off, ok := matchSymbolOffset(syms, symbol); ok {
			return off, nil
		}
	}
	if syms, err := f.Symbols(); err == nil {
		if off, ok := matchSymbolOffset(syms, symbol); ok {
			return off, nil
		}
	}
	return 0, fmt.Errorf("symbol %s not found in %s", symbol, path)
}

func matchSymbolOffset(symbols []elf.Symbol, want string) (uintptr, bool) {
	for _, s := range symbols {
		if s.Value == 0 {
			continue
		}
		// An IFUNC's st_value addresses its resolver, not the callable function.
		// Only the native dynamic loader can safely select its implementation.
		if elf.ST_TYPE(s.Info) == elf.STT_GNU_IFUNC {
			continue
		}
		if s.Name == want || strings.HasPrefix(s.Name, want+"@") {
			return uintptr(s.Value), true
		}
	}
	return 0, false
}

func validateELFForCurrentArch(data []byte) error {
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid ELF image: %w", err)
	}
	defer f.Close()
	return validateELFImage(data, f)
}

func validateELFImage(data []byte, f *elf.File) error {
	if err := validateELFHeaders(f); err != nil {
		return err
	}
	flagsOffset := 48
	if f.Class == elf.ELFCLASS32 {
		flagsOffset = 36
	}
	if len(data) < flagsOffset+4 {
		return errors.New("ELF header is truncated before e_flags")
	}
	flags := binary.LittleEndian.Uint32(data[flagsOffset : flagsOffset+4])
	return validateELFArchitectureFlags(f.Machine, flags)
}

func validateELFArchitectureFlags(machine elf.Machine, flags uint32) error {
	switch machine {
	case elf.EM_ARM:
		if flags&armELFEABIMask != armELFEABI5 || flags&(armELFFloatSoft|armELFFloatHard) != armELFFloatHard {
			return fmt.Errorf("ELF/arm shared libraries require EABI5 hard-float flags, got %#08x", flags)
		}
	case elf.EM_RISCV:
		if flags&riscvELFFloatABIMask != riscvELFFloatABIDouble {
			return fmt.Errorf("ELF/riscv64 shared libraries require the LP64D double-float ABI, got flags %#08x", flags)
		}
		if flags&riscvELFRVE != 0 {
			return fmt.Errorf("ELF/riscv64 shared libraries cannot use the RV32E register ABI, got flags %#08x", flags)
		}
		if flags&riscvELFTSO != 0 {
			return fmt.Errorf("ELF/riscv64 shared libraries requiring RVTSO are unsupported, got flags %#08x", flags)
		}
		if unknown := flags &^ riscvELFKnownFlags; unknown != 0 {
			return fmt.Errorf("ELF/riscv64 shared library has unknown flags %#08x", unknown)
		}
	case elf.EM_PPC64:
		if flags&ppc64ELFABI != ppc64ELFABI2 || flags&^uint32(ppc64ELFABI) != 0 {
			return fmt.Errorf("ELF/ppc64le shared libraries require the ELFv2 ABI flags, got %#08x", flags)
		}
	}
	return nil
}

func validateELFHeaders(f *elf.File) error {
	machine, err := currentELFMachine()
	if err != nil {
		return err
	}
	if f.Machine != machine {
		return fmt.Errorf("foreign platform (provided: %s, expected: %s)", f.Machine, machine)
	}
	if f.Type != elf.ET_DYN {
		return fmt.Errorf("unsupported ELF file type: %s", f.Type)
	}
	if f.Data != elf.ELFDATA2LSB {
		return fmt.Errorf("unsupported ELF endianness: %s", f.Data)
	}
	wantClass := elf.ELFCLASS64
	if runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
		wantClass = elf.ELFCLASS32
	}
	if f.Class != wantClass {
		return fmt.Errorf("unsupported ELF class for linux/%s: provided %s, expected %s", runtime.GOARCH, f.Class, wantClass)
	}
	return nil
}

func currentELFMachine() (elf.Machine, error) {
	switch runtime.GOARCH {
	case "386":
		return elf.EM_386, nil
	case "amd64":
		return elf.EM_X86_64, nil
	case "arm64":
		return elf.EM_AARCH64, nil
	case "arm":
		return elf.EM_ARM, nil
	case "riscv64":
		return elf.EM_RISCV, nil
	case "ppc64le":
		return elf.EM_PPC64, nil
	default:
		return 0, fmt.Errorf("unsupported linux architecture: %s", runtime.GOARCH)
	}
}
