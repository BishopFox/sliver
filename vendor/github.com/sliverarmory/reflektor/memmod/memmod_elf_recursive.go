//go:build (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) || (freebsd && (amd64 || arm64))

package memmod

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	recursiveDynTagFini        = 13
	recursiveDynTagFiniArray   = 26
	recursiveDynTagFiniArraySz = 28
	linuxMaxRecursiveImages    = 512
	linuxMaxRecursiveBytes     = uint64(1 << 30)
)

type linuxRecursiveNodeState uint8

const (
	linuxRecursiveNodeNew linuxRecursiveNodeState = iota
	linuxRecursiveNodeVisiting
	linuxRecursiveNodeDiscovered
)

type linuxRecursiveExport struct {
	symbol  elf.Symbol
	address uintptr
}

type linuxRecursiveDependency struct {
	name   string
	custom *linuxRecursiveNode
	system *linuxRecursiveSystem
}

type linuxRecursiveNode struct {
	path         string
	soname       string
	aliases      map[string]struct{}
	raw          []byte
	file         *elf.File
	dynamicSyms  []elf.Symbol
	runPaths     []string
	rPaths       []string
	needed       []string
	dependencies []linuxRecursiveDependency
	state        linuxRecursiveNodeState

	initInfo dynamicInitInfo
	finiInfo linuxRecursiveFiniInfo
	mapped   mappedELF
	exports  map[string][]linuxRecursiveExport

	goRuntime       bool
	tlsSlot         uintptr
	tlsSlotReserved bool
	initializers    []uintptr
	finalizers      []uintptr
	initialized     bool
}

type linuxRecursiveSystem struct {
	path    string
	handle  uintptr
	aliases map[string]struct{}
}

type linuxRecursiveScopeEntry struct {
	custom *linuxRecursiveNode
	system *linuxRecursiveSystem
}

type linuxRecursiveFiniInfo struct {
	fini        uint64
	finiArray   uint64
	finiArraySz uint64
}

type linuxRecursiveGroup struct {
	api       *linuxDynAPI
	nodes     []*linuxRecursiveNode
	initOrder []*linuxRecursiveNode
	systems   []*linuxRecursiveSystem
	pinned    bool
	freed     bool
}

type linuxRecursiveLoader struct {
	reader       DependencyReader
	group        *linuxRecursiveGroup
	root         *linuxRecursiveNode
	byPath       map[string]*linuxRecursiveNode
	bySONAME     map[string]*linuxRecursiveNode
	systemByPath map[string]*linuxRecursiveSystem
	systemRoots  []string
	scope        []linuxRecursiveScopeEntry
	totalBytes   uint64
}

// LoadLibraryRecursive maps an ELF root and each non-system DT_NEEDED
// dependency into one private Reflektor load group. The legacy LoadLibrary path
// intentionally remains separate and continues to resolve dependencies through
// the process loader.
func LoadLibraryRecursive(data []byte, origin string, reader DependencyReader) (_ *Module, retErr error) {
	if len(data) == 0 {
		return nil, errors.New("empty ELF image")
	}
	if reader == nil {
		return nil, errors.New("recursive ELF load requires a dependency reader")
	}
	if uint64(len(data)) > linuxMaxRecursiveBytes {
		return nil, fmt.Errorf("recursive ELF graph exceeds %d bytes", linuxMaxRecursiveBytes)
	}

	rootPath, err := canonicalRecursivePath(origin)
	if err != nil {
		return nil, fmt.Errorf("resolve recursive root origin: %w", err)
	}
	api, err := getLinuxDynAPI()
	if err != nil {
		return nil, fmt.Errorf("initialize ELF dynamic API for recursive load: %w", err)
	}

	loader := &linuxRecursiveLoader{
		reader:       reader,
		group:        &linuxRecursiveGroup{api: api},
		byPath:       make(map[string]*linuxRecursiveNode),
		bySONAME:     make(map[string]*linuxRecursiveNode),
		systemByPath: make(map[string]*linuxRecursiveSystem),
		systemRoots:  linuxRecursiveSystemRoots(),
		totalBytes:   uint64(len(data)),
	}
	defer func() {
		if retErr != nil {
			loader.group.free()
		}
	}()

	root, err := loader.parseCustomNode(data, rootPath)
	if err != nil {
		return nil, fmt.Errorf("parse recursive root %q: %w", rootPath, err)
	}
	loader.root = root
	loader.registerCustomNode(root)
	if err := loader.discover(root, nil); err != nil {
		return nil, err
	}
	loader.group.initOrder = linuxRecursivePostOrder(root)
	loader.scope = linuxRecursiveScope(root)

	if err := loader.mapAll(); err != nil {
		return nil, err
	}
	if err := loader.relocateAll(); err != nil {
		return nil, err
	}
	if err := loader.protectAndCollectLifecycle(); err != nil {
		return nil, err
	}

	rootSymbols := buildExportedSymbolTable(root.file, root.mapped.loadBias)
	if err := loader.initializeAll(); err != nil {
		return nil, err
	}
	loader.releaseLoadMetadata()

	return &Module{
		mapping:   root.mapped.mapping,
		loadBias:  root.mapped.loadBias,
		symbols:   rootSymbols,
		goRuntime: root.goRuntime,
		recursive: loader.group,
	}, nil
}

func (loader *linuxRecursiveLoader) parseCustomNode(data []byte, path string) (*linuxRecursiveNode, error) {
	if len(data) == 0 {
		return nil, errors.New("empty dependency ELF image")
	}
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("invalid ELF image: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = f.Close()
		}
	}()

	if err := validateELFHeaders(f, data); err != nil {
		return nil, err
	}
	initInfo, err := parseDynamicInitInfo(f)
	if err != nil {
		return nil, err
	}
	finiInfo, err := parseRecursiveFiniInfo(f)
	if err != nil {
		return nil, err
	}
	dynamicSyms, err := f.DynamicSymbols()
	if err != nil {
		return nil, fmt.Errorf("read dynamic symbol table: %w", err)
	}
	goRuntime := f.Section(".go.buildinfo") != nil
	if err := validateRecursiveELFFeatures(f, dynamicSyms, goRuntime); err != nil {
		return nil, err
	}

	soname := ""
	if values, err := f.DynString(elf.DT_SONAME); err == nil && len(values) != 0 {
		soname = strings.TrimSpace(values[0])
	}
	runPaths, rPaths := recursiveELFSearchPaths(f, path)
	node := &linuxRecursiveNode{
		path:        path,
		soname:      soname,
		aliases:     make(map[string]struct{}),
		raw:         data,
		file:        f,
		dynamicSyms: dynamicSyms,
		runPaths:    runPaths,
		rPaths:      rPaths,
		needed:      collectNeededLibraries(f),
		initInfo:    initInfo,
		finiInfo:    finiInfo,
		goRuntime:   goRuntime,
	}
	node.addAlias(path)
	node.addAlias(soname)
	cleanup = false
	return node, nil
}

func (loader *linuxRecursiveLoader) registerCustomNode(node *linuxRecursiveNode) {
	loader.byPath[node.path] = node
	if node.soname != "" {
		if _, exists := loader.bySONAME[node.soname]; !exists {
			loader.bySONAME[node.soname] = node
		}
	}
	loader.group.nodes = append(loader.group.nodes, node)
}

func (loader *linuxRecursiveLoader) discover(node *linuxRecursiveNode, inheritedRPaths []string) error {
	switch node.state {
	case linuxRecursiveNodeVisiting, linuxRecursiveNodeDiscovered:
		return nil
	}
	node.state = linuxRecursiveNodeVisiting
	searchPaths, childRPaths := linuxRecursiveDependencySearchPaths(node, inheritedRPaths)

	for _, needed := range node.needed {
		child := loader.bySONAME[needed]
		if child == nil {
			child = loader.byPath[needed]
		}
		if child != nil {
			child.addAlias(needed)
			node.dependencies = append(node.dependencies, linuxRecursiveDependency{name: needed, custom: child})
			if err := loader.discover(child, childRPaths); err != nil {
				return err
			}
			continue
		}

		dependency, err := loader.reader(DependencyRequest{
			Name:         needed,
			ImporterPath: node.path,
			SearchPaths:  searchPaths,
		})
		if err != nil {
			if errors.Is(err, ErrDependencyNotFound) && isLinuxNativeDependencyName(needed) {
				system, systemErr := loader.openSystem(needed, needed)
				if systemErr != nil {
					return fmt.Errorf("load native dependency %q for %q after reader miss: %w", needed, node.path, systemErr)
				}
				node.dependencies = append(node.dependencies, linuxRecursiveDependency{name: needed, system: system})
				continue
			}
			return fmt.Errorf("resolve dependency %q imported by %q: %w", needed, node.path, err)
		}
		if len(dependency.Data) == 0 {
			return fmt.Errorf("dependency %q imported by %q is empty", needed, node.path)
		}
		dependencyPath, err := canonicalRecursivePath(dependency.Path)
		if err != nil {
			return fmt.Errorf("resolve path for dependency %q imported by %q: %w", needed, node.path, err)
		}

		if loader.isSystemPath(dependencyPath) {
			system, err := loader.openSystem(needed, dependencyPath)
			if err != nil {
				return fmt.Errorf("load system dependency %q for %q: %w", needed, node.path, err)
			}
			node.dependencies = append(node.dependencies, linuxRecursiveDependency{name: needed, system: system})
			continue
		}

		child = loader.byPath[dependencyPath]
		if child == nil {
			if len(loader.group.nodes) >= linuxMaxRecursiveImages {
				return fmt.Errorf("recursive ELF graph exceeds %d images", linuxMaxRecursiveImages)
			}
			if uint64(len(dependency.Data)) > linuxMaxRecursiveBytes-loader.totalBytes {
				return fmt.Errorf("recursive ELF graph exceeds %d bytes", linuxMaxRecursiveBytes)
			}
			candidate, err := loader.parseCustomNode(dependency.Data, dependencyPath)
			if err != nil {
				return fmt.Errorf("parse dependency %q at %q imported by %q: %w", needed, dependencyPath, node.path, err)
			}
			if candidate.soname != "" {
				child = loader.bySONAME[candidate.soname]
			}
			if child == nil {
				child = candidate
				loader.registerCustomNode(child)
				loader.totalBytes += uint64(len(dependency.Data))
			} else {
				_ = candidate.file.Close()
				loader.byPath[dependencyPath] = child
			}
		}
		child.addAlias(needed)
		child.addAlias(dependencyPath)
		node.dependencies = append(node.dependencies, linuxRecursiveDependency{name: needed, custom: child})
		if err := loader.discover(child, childRPaths); err != nil {
			return err
		}
	}

	node.state = linuxRecursiveNodeDiscovered
	return nil
}

func (loader *linuxRecursiveLoader) openSystem(name string, path string) (*linuxRecursiveSystem, error) {
	if existing := loader.systemByPath[path]; existing != nil {
		existing.addAlias(name)
		return existing, nil
	}
	handle, err := openWithDlopen(loader.group.api, path)
	if err != nil {
		return nil, err
	}
	system := &linuxRecursiveSystem{
		path:    path,
		handle:  handle,
		aliases: make(map[string]struct{}),
	}
	system.addAlias(name)
	system.addAlias(path)
	loader.systemByPath[path] = system
	loader.group.systems = append(loader.group.systems, system)
	return system, nil
}

func (loader *linuxRecursiveLoader) mapAll() error {
	for _, node := range loader.group.nodes {
		goRuntime, tlsSlot, tlsOffset, err := prepareLinuxGoRuntimeTLS(node.file)
		if err != nil {
			return fmt.Errorf("prepare TLS for recursive dependency %q: %w", node.path, err)
		}
		if goRuntime != node.goRuntime {
			return fmt.Errorf("inconsistent Go runtime detection for recursive dependency %q", node.path)
		}
		if goRuntime {
			node.tlsSlot = tlsSlot
			node.tlsSlotReserved = true
		}

		mapped, err := mapELFImage(node.raw, node.file)
		if err != nil {
			return fmt.Errorf("map recursive dependency %q: %w", node.path, err)
		}
		mapped.tlsOffset = tlsOffset
		mapped.hasTLS = goRuntime
		node.mapped = mapped
		node.exports, err = buildRecursiveExportTable(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (loader *linuxRecursiveLoader) relocateAll() error {
	for _, node := range loader.group.nodes {
		requester := node
		resolver := &symbolResolver{
			resolveSymbol: func(sym elf.Symbol) (uintptr, error) {
				return loader.resolveExternal(requester, sym)
			},
		}
		if err := applyDynamicRelocations(node.mapped, node.file, resolver); err != nil {
			return fmt.Errorf("relocate recursive dependency %q: %w", node.path, err)
		}
	}
	return nil
}

func (loader *linuxRecursiveLoader) protectAndCollectLifecycle() error {
	for _, node := range loader.group.nodes {
		if err := flushELFInstructionCache(node.mapped); err != nil {
			return fmt.Errorf("flush recursive dependency %q: %w", node.path, err)
		}
		if err := applySegmentProtections(node.mapped); err != nil {
			return fmt.Errorf("protect recursive dependency %q: %w", node.path, err)
		}
		initializers, err := collectRecursiveInitializers(node.mapped, node.file.Class, node.initInfo)
		if err != nil {
			return fmt.Errorf("collect initializers for recursive dependency %q: %w", node.path, err)
		}
		finalizers, err := collectRecursiveFinalizers(node.mapped, node.file.Class, node.finiInfo)
		if err != nil {
			return fmt.Errorf("collect finalizers for recursive dependency %q: %w", node.path, err)
		}
		node.initializers = initializers
		node.finalizers = finalizers
	}
	return nil
}

func (loader *linuxRecursiveLoader) initializeAll() error {
	for _, node := range loader.group.initOrder {
		argc, argv, envp := linuxInitCallArgs(node.goRuntime)
		if node.goRuntime && argv == 0 {
			return fmt.Errorf("allocate Go runtime argv/environment vector for recursive dependency %q", node.path)
		}
		if node.goRuntime {
			// Once a Go constructor starts it may retain both its own mapping and
			// raw addresses into every dependency in this load group.
			loader.group.pinned = true
			node.tlsSlotReserved = false
		}
		for _, initializer := range node.initializers {
			cCallVoid3(initializer, argc, argv, envp)
		}
		node.initialized = true
	}
	return nil
}

func (loader *linuxRecursiveLoader) releaseLoadMetadata() {
	for _, node := range loader.group.nodes {
		if node.file != nil {
			_ = node.file.Close()
			node.file = nil
		}
		node.raw = nil
		node.dynamicSyms = nil
		node.exports = nil
		node.dependencies = nil
		node.initializers = nil
	}
	loader.scope = nil
	loader.byPath = nil
	loader.bySONAME = nil
	loader.systemByPath = nil
}

func (loader *linuxRecursiveLoader) resolveExternal(requester *linuxRecursiveNode, symbol elf.Symbol) (uintptr, error) {
	if symbol.Name == "" {
		return 0, errors.New("external ELF symbol is unnamed")
	}

	tried := make(map[int]struct{}, len(loader.scope))
	if symbol.Library != "" {
		for index, entry := range loader.scope {
			if !entry.matchesLibrary(symbol.Library) {
				continue
			}
			tried[index] = struct{}{}
			if address, err := entry.lookup(loader.group.api, symbol); err == nil && address != 0 {
				return address, nil
			}
		}
	}
	for index, entry := range loader.scope {
		if _, alreadyTried := tried[index]; alreadyTried {
			continue
		}
		if address, err := entry.lookup(loader.group.api, symbol); err == nil && address != 0 {
			return address, nil
		}
	}

	if address, err := resolveWithLinuxHandle(loader.group.api, loader.group.api.defaultHandle, symbol.Name, symbol.Version); err == nil && address != 0 {
		return address, nil
	}
	return 0, fmt.Errorf("unresolved recursive external symbol %q (version=%q library=%q) for %q", symbol.Name, symbol.Version, symbol.Library, requester.path)
}

func (entry linuxRecursiveScopeEntry) lookup(api *linuxDynAPI, imported elf.Symbol) (uintptr, error) {
	if entry.custom != nil {
		return entry.custom.lookup(imported)
	}
	if entry.system != nil {
		return resolveWithLinuxHandle(api, entry.system.handle, imported.Name, imported.Version)
	}
	return 0, errors.New("empty recursive symbol-scope entry")
}

func (entry linuxRecursiveScopeEntry) matchesLibrary(name string) bool {
	if entry.custom != nil {
		return entry.custom.matchesAlias(name)
	}
	if entry.system != nil {
		return entry.system.matchesAlias(name)
	}
	return false
}

func (node *linuxRecursiveNode) lookup(imported elf.Symbol) (uintptr, error) {
	for _, candidate := range node.exports[imported.Name] {
		if !recursiveSymbolVersionMatches(imported, candidate.symbol) {
			continue
		}
		return candidate.address, nil
	}
	return 0, fmt.Errorf("symbol %q not exported by %q", imported.Name, node.path)
}

func buildRecursiveExportTable(node *linuxRecursiveNode) (map[string][]linuxRecursiveExport, error) {
	exports := make(map[string][]linuxRecursiveExport)
	for _, symbol := range node.dynamicSyms {
		if symbol.Name == "" || symbol.Section == elf.SHN_UNDEF {
			continue
		}
		binding := elf.ST_BIND(symbol.Info)
		if binding != elf.STB_GLOBAL && binding != elf.STB_WEAK {
			continue
		}
		visibility := elf.ST_VISIBILITY(symbol.Other)
		if visibility == elf.STV_HIDDEN || visibility == elf.STV_INTERNAL {
			continue
		}
		switch elf.ST_TYPE(symbol.Info) {
		case elf.STT_FUNC, elf.STT_OBJECT, elf.STT_NOTYPE:
		default:
			continue
		}

		var address uintptr
		if symbol.Section == elf.SHN_ABS {
			address = uintptr(symbol.Value)
		} else {
			address = node.mapped.loadBias + uintptr(symbol.Value)
			if !mappedAddressInRange(node.mapped.mapping, address, 1) {
				return nil, fmt.Errorf("recursive export %q in %q points outside mapped image", symbol.Name, node.path)
			}
		}
		exports[symbol.Name] = append(exports[symbol.Name], linuxRecursiveExport{symbol: symbol, address: address})
	}
	return exports, nil
}

func recursiveSymbolVersionMatches(imported elf.Symbol, exported elf.Symbol) bool {
	if imported.Version != "" {
		return exported.Version == imported.Version
	}
	if exported.HasVersion && exported.VersionIndex.IsHidden() {
		return false
	}
	return true
}

func linuxRecursiveScope(root *linuxRecursiveNode) []linuxRecursiveScopeEntry {
	if root == nil {
		return nil
	}
	queue := []linuxRecursiveScopeEntry{{custom: root}}
	seenCustom := make(map[*linuxRecursiveNode]struct{})
	seenSystem := make(map[*linuxRecursiveSystem]struct{})
	var scope []linuxRecursiveScopeEntry
	for len(queue) != 0 {
		entry := queue[0]
		queue = queue[1:]
		if entry.custom != nil {
			if _, exists := seenCustom[entry.custom]; exists {
				continue
			}
			seenCustom[entry.custom] = struct{}{}
			scope = append(scope, entry)
			for _, dependency := range entry.custom.dependencies {
				queue = append(queue, linuxRecursiveScopeEntry{custom: dependency.custom, system: dependency.system})
			}
			continue
		}
		if entry.system != nil {
			if _, exists := seenSystem[entry.system]; exists {
				continue
			}
			seenSystem[entry.system] = struct{}{}
			scope = append(scope, entry)
		}
	}
	return scope
}

func linuxRecursivePostOrder(root *linuxRecursiveNode) []*linuxRecursiveNode {
	seen := make(map[*linuxRecursiveNode]struct{})
	active := make(map[*linuxRecursiveNode]struct{})
	var order []*linuxRecursiveNode
	var visit func(*linuxRecursiveNode)
	visit = func(node *linuxRecursiveNode) {
		if node == nil {
			return
		}
		if _, exists := seen[node]; exists {
			return
		}
		if _, cycle := active[node]; cycle {
			return
		}
		active[node] = struct{}{}
		for _, dependency := range node.dependencies {
			visit(dependency.custom)
		}
		delete(active, node)
		seen[node] = struct{}{}
		order = append(order, node)
	}
	visit(root)
	return order
}

func (group *linuxRecursiveGroup) free() {
	if group == nil || group.freed {
		return
	}
	group.freed = true
	for _, node := range group.nodes {
		if node.file != nil {
			_ = node.file.Close()
			node.file = nil
		}
		node.raw = nil
	}
	if group.pinned {
		return
	}

	for index := len(group.initOrder) - 1; index >= 0; index-- {
		node := group.initOrder[index]
		if !node.initialized {
			continue
		}
		for _, finalizer := range node.finalizers {
			cCallVoid0(finalizer)
		}
		node.initialized = false
	}
	for index := len(group.initOrder) - 1; index >= 0; index-- {
		node := group.initOrder[index]
		if len(node.mapped.mapping) != 0 {
			_ = unix.Munmap(node.mapped.mapping)
			node.mapped.mapping = nil
		}
		if node.tlsSlotReserved {
			releaseLinuxGoTLSSlot(node.tlsSlot)
			node.tlsSlotReserved = false
		}
	}
	if group.api != nil && group.api.dlclose != 0 {
		for index := len(group.systems) - 1; index >= 0; index-- {
			closeWithDL(group.api, group.systems[index].handle)
		}
	}
	group.nodes = nil
	group.initOrder = nil
	group.systems = nil
}

func collectRecursiveInitializers(mapped mappedELF, class elf.Class, info dynamicInitInfo) ([]uintptr, error) {
	initializers, err := appendDynamicInitFn(nil, mapped, uintptr(info.init), "DT_INIT")
	if err != nil {
		return nil, err
	}
	return appendDynamicInitArray(initializers, mapped, class, info.initArray, info.initArraySz, "DT_INIT_ARRAY")
}

func collectRecursiveFinalizers(mapped mappedELF, class elf.Class, info linuxRecursiveFiniInfo) ([]uintptr, error) {
	array, err := appendDynamicInitArray(nil, mapped, class, info.finiArray, info.finiArraySz, "DT_FINI_ARRAY")
	if err != nil {
		return nil, err
	}
	finalizers := make([]uintptr, 0, len(array)+1)
	for index := len(array) - 1; index >= 0; index-- {
		finalizers = append(finalizers, array[index])
	}
	return appendDynamicInitFn(finalizers, mapped, uintptr(info.fini), "DT_FINI")
}

func parseRecursiveFiniInfo(f *elf.File) (linuxRecursiveFiniInfo, error) {
	var info linuxRecursiveFiniInfo
	if f == nil {
		return info, nil
	}
	section := f.Section(".dynamic")
	if section == nil {
		return info, nil
	}
	data, err := section.Data()
	if err != nil {
		return info, fmt.Errorf("read .dynamic section: %w", err)
	}

	switch f.Class {
	case elf.ELFCLASS64:
		const entrySize = 16
		if len(data)%entrySize != 0 {
			return info, fmt.Errorf("malformed .dynamic section: size %d is not a multiple of %d", len(data), entrySize)
		}
		for index := 0; index < len(data); index += entrySize {
			tag := int64(binary.LittleEndian.Uint64(data[index : index+8]))
			value := binary.LittleEndian.Uint64(data[index+8 : index+16])
			if tag == dynTagNull {
				break
			}
			setRecursiveFiniTag(&info, tag, value)
		}
	case elf.ELFCLASS32:
		const entrySize = 8
		if len(data)%entrySize != 0 {
			return info, fmt.Errorf("malformed .dynamic section: size %d is not a multiple of %d", len(data), entrySize)
		}
		for index := 0; index < len(data); index += entrySize {
			tag := int64(int32(binary.LittleEndian.Uint32(data[index : index+4])))
			value := uint64(binary.LittleEndian.Uint32(data[index+4 : index+8]))
			if tag == dynTagNull {
				break
			}
			setRecursiveFiniTag(&info, tag, value)
		}
	default:
		return info, fmt.Errorf("unsupported ELF class for .dynamic parsing: %s", f.Class)
	}
	return info, nil
}

func setRecursiveFiniTag(info *linuxRecursiveFiniInfo, tag int64, value uint64) {
	switch tag {
	case recursiveDynTagFini:
		info.fini = value
	case recursiveDynTagFiniArray:
		info.finiArray = value
	case recursiveDynTagFiniArraySz:
		info.finiArraySz = value
	}
}

func recursiveELFSearchPaths(f *elf.File, origin string) (runPaths []string, rPaths []string) {
	if f == nil {
		return nil, nil
	}
	return recursiveELFPathList(f, elf.DT_RUNPATH, origin), recursiveELFPathList(f, elf.DT_RPATH, origin)
}

func recursiveELFPathList(f *elf.File, tag elf.DynTag, origin string) []string {
	values, _ := f.DynString(tag)
	var paths []string
	for _, value := range values {
		for _, path := range strings.Split(value, ":") {
			path = strings.TrimSpace(path)
			path = strings.ReplaceAll(path, "${ORIGIN}", filepath.Dir(origin))
			path = strings.ReplaceAll(path, "$ORIGIN", filepath.Dir(origin))
			if path != "" {
				paths = append(paths, filepath.Clean(path))
			}
		}
	}
	return paths
}

func linuxRecursiveDependencySearchPaths(node *linuxRecursiveNode, inheritedRPaths []string) (searchPaths []string, childRPaths []string) {
	childRPaths = append([]string(nil), inheritedRPaths...)
	if len(node.runPaths) == 0 && len(node.rPaths) != 0 {
		childRPaths = appendUniqueLinuxPaths(node.rPaths, childRPaths)
	}
	searchPaths = appendUniqueLinuxPaths(searchPaths, childRPaths)
	if value := os.Getenv("LD_LIBRARY_PATH"); value != "" {
		for _, path := range filepath.SplitList(value) {
			path = strings.TrimSpace(path)
			if path != "" {
				searchPaths = appendUniqueLinuxPaths(searchPaths, []string{path})
			}
		}
	}
	searchPaths = appendUniqueLinuxPaths(searchPaths, node.runPaths)
	return searchPaths, childRPaths
}

func appendUniqueLinuxPaths(paths []string, additions []string) []string {
	seen := make(map[string]struct{}, len(paths)+len(additions))
	for _, path := range paths {
		seen[path] = struct{}{}
	}
	for _, path := range additions {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func validateRecursiveELFFeatures(f *elf.File, symbols []elf.Symbol, goRuntime bool) error {
	for _, program := range f.Progs {
		if program.Type == elf.PT_TLS && program.Memsz != 0 && !goRuntime {
			return fmt.Errorf("recursive ELF loading does not support general PT_TLS (filesz=%#x memsz=%#x)", program.Filesz, program.Memsz)
		}
	}
	for _, symbol := range symbols {
		if elf.ST_TYPE(symbol.Info) == elf.STT_GNU_IFUNC {
			return fmt.Errorf("recursive ELF loading does not support GNU IFUNC symbol %q", symbol.Name)
		}
		if elf.ST_TYPE(symbol.Info) == elf.STT_TLS && !goRuntime {
			return fmt.Errorf("recursive ELF loading does not support TLS symbol %q", symbol.Name)
		}
	}
	if section := f.Section(".relr.dyn"); section != nil && section.Size != 0 {
		return errors.New("recursive ELF loading does not support packed RELR relocations")
	}
	if hasRecursiveIRelative(f) {
		return errors.New("recursive ELF loading does not support IRELATIVE/IFUNC relocations")
	}
	return nil
}

func hasRecursiveIRelative(f *elf.File) bool {
	for _, section := range relocationSections(f) {
		data, err := section.Data()
		if err != nil {
			continue
		}
		entrySize := 0
		infoOffset := 0
		switch {
		case section.Type == elf.SHT_RELA && f.Class == elf.ELFCLASS64:
			entrySize, infoOffset = 24, 8
		case section.Type == elf.SHT_RELA && f.Class == elf.ELFCLASS32:
			entrySize, infoOffset = 12, 4
		case section.Type == elf.SHT_REL && f.Class == elf.ELFCLASS64:
			entrySize, infoOffset = 16, 8
		case section.Type == elf.SHT_REL && f.Class == elf.ELFCLASS32:
			entrySize, infoOffset = 8, 4
		default:
			continue
		}
		for offset := 0; offset+entrySize <= len(data); offset += entrySize {
			var relocationType uint32
			if f.Class == elf.ELFCLASS64 {
				info := binary.LittleEndian.Uint64(data[offset+infoOffset : offset+infoOffset+8])
				relocationType = uint32(elf.R_TYPE64(info))
			} else {
				info := binary.LittleEndian.Uint32(data[offset+infoOffset : offset+infoOffset+4])
				relocationType = elf.R_TYPE32(info)
			}
			switch f.Machine {
			case elf.EM_X86_64:
				if elf.R_X86_64(relocationType) == elf.R_X86_64_IRELATIVE {
					return true
				}
			case elf.EM_386:
				if elf.R_386(relocationType) == elf.R_386_IRELATIVE {
					return true
				}
			case elf.EM_AARCH64:
				if elf.R_AARCH64(relocationType) == elf.R_AARCH64_IRELATIVE {
					return true
				}
			case elf.EM_ARM:
				if elf.R_ARM(relocationType) == elf.R_ARM_IRELATIVE {
					return true
				}
			case elf.EM_RISCV:
				// The psABI assigns relocation 58 to R_RISCV_IRELATIVE.
				if relocationType == 58 {
					return true
				}
			case elf.EM_PPC64:
				if elf.R_PPC64(relocationType) == elf.R_PPC64_IRELATIVE {
					return true
				}
			}
		}
	}
	return false
}

func resolveWithLinuxHandle(api *linuxDynAPI, handle uintptr, name string, version string) (uintptr, error) {
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

func closeWithDL(api *linuxDynAPI, handle uintptr) {
	if api == nil || api.dlclose == 0 || handle == 0 {
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}
	_ = callExportFunction(api.dlclose, handle)
	if api.dlerror != 0 {
		_ = callExportFunction(api.dlerror)
	}
}

func canonicalRecursivePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("dependency path is empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	if evaluated, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = filepath.Clean(evaluated)
	}
	return absolute, nil
}

func linuxRecursiveSystemRoots() []string {
	configured := []string{"/lib", "/lib64", "/usr/lib", "/usr/lib64"}
	if runtime.GOOS == "freebsd" {
		configured = []string{"/lib", "/usr/lib", "/usr/local/lib", "/usr/local/lib/compat", "/libexec", "/usr/libexec"}
	}
	seen := make(map[string]struct{}, len(configured))
	var roots []string
	for _, root := range configured {
		canonical, err := canonicalRecursivePath(root)
		if err != nil {
			continue
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		roots = append(roots, canonical)
	}
	return roots
}

func (loader *linuxRecursiveLoader) isSystemPath(path string) bool {
	for _, root := range loader.systemRoots {
		if linuxPathWithinRoot(path, root) {
			return true
		}
	}
	return false
}

func linuxPathWithinRoot(path string, root string) bool {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	if path == root {
		return true
	}
	return strings.HasPrefix(path, root+string(os.PathSeparator))
}

func isLinuxNativeDependencyName(name string) bool {
	base := filepath.Base(strings.TrimSpace(name))
	if runtime.GOOS == "freebsd" {
		switch base {
		case "ld-elf.so.1",
			"libc++.so", "libc++.so.1",
			"libc.so", "libc.so.7",
			"libcxxrt.so", "libcxxrt.so.1",
			"libcrypto.so", "libcrypto.so.30", "libcrypto.so.111", "libcrypto.so.3",
			"libcurl.so", "libcurl.so.4",
			"libgcc_s.so", "libgcc_s.so.1",
			"libm.so", "libm.so.5",
			"libpthread.so", "libthr.so", "libthr.so.3",
			"libresolv.so", "libresolv.so.2",
			"librt.so", "librt.so.1",
			"libssl.so", "libssl.so.30", "libssl.so.111", "libssl.so.3",
			"libstdc++.so", "libstdc++.so.6",
			"libsys.so", "libsys.so.7",
			"libutil.so", "libutil.so.9", "libutil.so.10",
			"libz.so", "libz.so.6", "libz.so.1":
			return true
		default:
			return false
		}
	}
	if strings.HasPrefix(base, "ld-linux-") || strings.HasPrefix(base, "ld-musl-") {
		return true
	}
	switch base {
	case "linux-vdso.so.1",
		"libc.so", "libc.so.6",
		"libcrypto.so", "libcrypto.so.3",
		"libcurl.so", "libcurl.so.4",
		"libdl.so", "libdl.so.2",
		"libgcc_s.so", "libgcc_s.so.1",
		"libm.so", "libm.so.6",
		"libpthread.so", "libpthread.so.0",
		"libresolv.so", "libresolv.so.2",
		"librt.so", "librt.so.1",
		"libssl.so", "libssl.so.3",
		"libstdc++.so", "libstdc++.so.6",
		"libz.so", "libz.so.1":
		return true
	default:
		return false
	}
}

func (node *linuxRecursiveNode) addAlias(alias string) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	node.aliases[alias] = struct{}{}
	node.aliases[filepath.Base(alias)] = struct{}{}
}

func (node *linuxRecursiveNode) matchesAlias(alias string) bool {
	_, exact := node.aliases[alias]
	if exact {
		return true
	}
	_, base := node.aliases[filepath.Base(alias)]
	return base
}

func (system *linuxRecursiveSystem) addAlias(alias string) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	system.aliases[alias] = struct{}{}
	system.aliases[filepath.Base(alias)] = struct{}{}
}

func (system *linuxRecursiveSystem) matchesAlias(alias string) bool {
	_, exact := system.aliases[alias]
	if exact {
		return true
	}
	_, base := system.aliases[filepath.Base(alias)]
	return base
}
