//go:build windows

package memmod

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type recursiveLoadState uint8

const (
	recursiveLoadLoading recursiveLoadState = iota
	recursiveLoadReady
	windowsMaxRecursiveModules = 512
	windowsMaxRecursiveBytes   = uint64(1 << 30)
)

type recursiveModuleRecord struct {
	key    string
	path   string
	state  recursiveLoadState
	module *Module
}

type recursiveLoadSession struct {
	rootPath   string
	reader     DependencyReader
	records    map[string]*recursiveModuleRecord
	stack      []*recursiveModuleRecord
	loaded     []*Module
	pinned     bool
	freed      bool
	totalBytes uint64
}

type recursiveImport struct {
	handle windows.Handle
	module *Module
}

// resolvedWindowsExport retains the custom module that owns an export address.
// System-loader exports have a nil owner. Keeping this metadata through PE
// forwarder chains lets callers apply module-specific invocation constraints.
type resolvedWindowsExport struct {
	address uintptr
	owner   *Module
}

// LoadLibraryRecursive loads a PE image and recursively memory-loads non-system
// dependencies returned by reader. Windows system modules retain the legacy
// LOAD_LIBRARY_SEARCH_SYSTEM32 behavior.
func LoadLibraryRecursive(data []byte, origin string, reader DependencyReader) (*Module, error) {
	if len(data) == 0 {
		return nil, errors.New("empty library image")
	}
	if reader == nil {
		return nil, errors.New("recursive dependency reader is nil")
	}
	if uint64(len(data)) > windowsMaxRecursiveBytes {
		return nil, fmt.Errorf("recursive PE graph exceeds %d bytes", windowsMaxRecursiveBytes)
	}

	rootPath, rootKey, err := canonicalRecursivePath(origin)
	if err != nil {
		return nil, fmt.Errorf("invalid recursive library origin: %w", err)
	}
	session := &recursiveLoadSession{
		rootPath: rootPath,
		reader:   reader,
		records:  make(map[string]*recursiveModuleRecord),
	}
	module, err := session.loadModule(data, rootPath, rootKey)
	if err != nil {
		session.free()
		return nil, err
	}
	module.recursiveOwner = true
	return module, nil
}

func (session *recursiveLoadSession) loadModule(data []byte, path string, key string) (*Module, error) {
	if record, ok := session.records[key]; ok {
		if record.state == recursiveLoadLoading {
			return nil, session.cycleError(record)
		}
		return record.module, nil
	}
	if len(session.records) >= windowsMaxRecursiveModules {
		return nil, fmt.Errorf("recursive PE graph exceeds %d images", windowsMaxRecursiveModules)
	}
	if uint64(len(data)) > windowsMaxRecursiveBytes-session.totalBytes {
		return nil, fmt.Errorf("recursive PE graph exceeds %d bytes", windowsMaxRecursiveBytes)
	}
	session.totalBytes += uint64(len(data))

	record := &recursiveModuleRecord{key: key, path: path, state: recursiveLoadLoading}
	session.records[key] = record
	session.stack = append(session.stack, record)
	defer func() {
		session.stack = session.stack[:len(session.stack)-1]
	}()

	var candidate *Module
	module, err := loadLibrary(data, func(module *Module) error {
		candidate = module
		module.recursive = session
		module.recursivePath = path
		return module.buildImportTableRecursive()
	})
	if err != nil {
		if candidate != nil && candidate.goRuntime && candidate.runtimeStarted {
			session.pinned = true
		}
		return nil, fmt.Errorf("load recursive module %q: %w", path, err)
	}

	record.state = recursiveLoadReady
	record.module = module
	session.loaded = append(session.loaded, module)
	if module.goRuntime && module.runtimeStarted {
		session.pinned = true
	}
	return module, nil
}

func (session *recursiveLoadSession) cycleError(record *recursiveModuleRecord) error {
	start := 0
	for i := range session.stack {
		if session.stack[i].key == record.key {
			start = i
			break
		}
	}
	cycle := make([]string, 0, len(session.stack)-start+1)
	for _, entry := range session.stack[start:] {
		cycle = append(cycle, entry.path)
	}
	cycle = append(cycle, record.path)
	return fmt.Errorf("recursive dependency cycle: %s", strings.Join(cycle, " -> "))
}

func (session *recursiveLoadSession) free() {
	if session.freed {
		return
	}
	session.freed = true
	if session.pinned {
		// A started Go runtime can continue to execute code from any node in the
		// dependency graph. Pin the whole graph, matching legacy Go DLL behavior.
		for _, module := range session.loaded {
			if module.blockedMemory != nil {
				module.blockedMemory.free()
				module.blockedMemory = nil
			}
		}
		return
	}
	for i := len(session.loaded) - 1; i >= 0; i-- {
		module := session.loaded[i]
		module.recursiveOwner = false
		module.freeSelf()
	}
}

func (module *Module) buildImportTableRecursive() error {
	if delay := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT); delay.Size != 0 {
		return errors.New("recursive loading does not support delay-load imports")
	}
	directory := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_IMPORT)
	if directory.Size == 0 {
		return nil
	}

	importDesc := (*IMAGE_IMPORT_DESCRIPTOR)(a2p(module.codeBase + uintptr(directory.VirtualAddress)))
	for importDesc.Name != 0 {
		name := windows.BytePtrToString((*byte)(a2p(module.codeBase + uintptr(importDesc.Name))))
		dependency, err := module.recursive.resolveImport(module, name)
		if err != nil {
			return fmt.Errorf("resolve dependency %q for %q: %w", name, module.recursivePath, err)
		}

		var thunkRef, funcRef *uintptr
		if importDesc.OriginalFirstThunk() != 0 {
			thunkRef = (*uintptr)(a2p(module.codeBase + uintptr(importDesc.OriginalFirstThunk())))
			funcRef = (*uintptr)(a2p(module.codeBase + uintptr(importDesc.FirstThunk)))
		} else {
			thunkRef = (*uintptr)(a2p(module.codeBase + uintptr(importDesc.FirstThunk)))
			funcRef = (*uintptr)(a2p(module.codeBase + uintptr(importDesc.FirstThunk)))
		}

		for *thunkRef != 0 {
			var resolved resolvedWindowsExport
			if IMAGE_SNAP_BY_ORDINAL(*thunkRef) {
				ordinal := uint16(IMAGE_ORDINAL(*thunkRef))
				resolved, err = dependency.procAddressByOrdinal(ordinal, make(map[string]struct{}))
			} else {
				thunkData := (*IMAGE_IMPORT_BY_NAME)(a2p(module.codeBase + *thunkRef))
				functionName := windows.BytePtrToString(&thunkData.Name[0])
				resolved, err = dependency.procAddressByName(functionName, make(map[string]struct{}))
			}
			if err != nil {
				if dependency.handle != 0 {
					_ = windows.FreeLibrary(dependency.handle)
				}
				return fmt.Errorf("resolve import from %q: %w", name, err)
			}
			*funcRef = resolved.address
			thunkRef = (*uintptr)(a2p(uintptr(unsafe.Pointer(thunkRef)) + unsafe.Sizeof(*thunkRef)))
			funcRef = (*uintptr)(a2p(uintptr(unsafe.Pointer(funcRef)) + unsafe.Sizeof(*funcRef)))
		}

		if dependency.handle != 0 {
			module.modules = append(module.modules, dependency.handle)
		}
		importDesc = (*IMAGE_IMPORT_DESCRIPTOR)(a2p(uintptr(unsafe.Pointer(importDesc)) + unsafe.Sizeof(*importDesc)))
	}
	return nil
}

func (session *recursiveLoadSession) resolveImport(importer *Module, name string) (recursiveImport, error) {
	if isWindowsSystemDependency(name) {
		handle, err := windows.LoadLibraryEx(name, 0, windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
		if err != nil {
			return recursiveImport{}, fmt.Errorf("load system module %q: %w", name, err)
		}
		return recursiveImport{handle: handle}, nil
	}

	request := DependencyRequest{
		Name:         name,
		ImporterPath: importer.recursivePath,
		SearchPaths:  recursiveSearchPaths(importer.recursivePath, session.rootPath),
	}
	dependency, err := session.reader(request)
	if err != nil {
		if !errors.Is(err, ErrDependencyNotFound) || filepath.Base(name) != name {
			return recursiveImport{}, fmt.Errorf("read dependency %q: %w", name, err)
		}
		handle, systemErr := windows.LoadLibraryEx(name, 0, windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
		if systemErr != nil {
			return recursiveImport{}, fmt.Errorf("read dependency %q: %w; System32 fallback: %v", name, err, systemErr)
		}
		return recursiveImport{handle: handle}, nil
	}
	if len(dependency.Data) == 0 {
		return recursiveImport{}, errors.New("dependency reader returned an empty image")
	}
	path, key, err := canonicalRecursivePath(dependency.Path)
	if err != nil {
		return recursiveImport{}, fmt.Errorf("dependency reader returned invalid path for %q: %w", name, err)
	}
	module, err := session.loadModule(dependency.Data, path, key)
	if err != nil {
		return recursiveImport{}, err
	}
	return recursiveImport{module: module}, nil
}

func (dependency recursiveImport) procAddressByName(name string, chain map[string]struct{}) (resolvedWindowsExport, error) {
	if dependency.handle != 0 {
		address, err := windows.GetProcAddress(dependency.handle, name)
		return resolvedWindowsExport{address: address}, err
	}
	return dependency.module.recursiveProcAddressByName(name, chain)
}

func (dependency recursiveImport) procAddressByOrdinal(ordinal uint16, chain map[string]struct{}) (resolvedWindowsExport, error) {
	if dependency.handle != 0 {
		address, err := windows.GetProcAddressByOrdinal(dependency.handle, uintptr(ordinal))
		return resolvedWindowsExport{address: address}, err
	}
	return dependency.module.recursiveProcAddressByOrdinal(ordinal, chain)
}

func (module *Module) recursiveProcAddressByName(name string, chain map[string]struct{}) (resolvedWindowsExport, error) {
	directory := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_EXPORT)
	if directory.Size == 0 {
		return resolvedWindowsExport{}, errors.New("No export table found")
	}
	if module.nameExports == nil {
		return resolvedWindowsExport{}, errors.New("No functions exported by name")
	}
	idx, ok := module.nameExports[name]
	if !ok {
		return resolvedWindowsExport{}, errors.New("Function not found by name")
	}
	return module.recursiveProcAddressByIndex(uint32(idx), name, chain)
}

func (module *Module) recursiveProcAddressByOrdinal(ordinal uint16, chain map[string]struct{}) (resolvedWindowsExport, error) {
	directory := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_EXPORT)
	if directory.Size == 0 {
		return resolvedWindowsExport{}, errors.New("No export table found")
	}
	exports := (*IMAGE_EXPORT_DIRECTORY)(a2p(module.codeBase + uintptr(directory.VirtualAddress)))
	if uint32(ordinal) < exports.Base {
		return resolvedWindowsExport{}, errors.New("Ordinal number too low")
	}
	return module.recursiveProcAddressByIndex(uint32(ordinal)-exports.Base, "#"+strconv.FormatUint(uint64(ordinal), 10), chain)
}

func (module *Module) recursiveProcAddressByIndex(idx uint32, symbol string, chain map[string]struct{}) (resolvedWindowsExport, error) {
	directory := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_EXPORT)
	exports := (*IMAGE_EXPORT_DIRECTORY)(a2p(module.codeBase + uintptr(directory.VirtualAddress)))
	if idx >= exports.NumberOfFunctions {
		return resolvedWindowsExport{}, errors.New("Ordinal number too high")
	}

	cacheKey := strconv.FormatUint(uint64(idx), 10)
	module.recursiveMu.Lock()
	if resolved, ok := module.forwarders[cacheKey]; ok {
		module.recursiveMu.Unlock()
		return resolved, nil
	}
	module.recursiveMu.Unlock()

	rva := *(*uint32)(a2p(module.codeBase + uintptr(exports.AddressOfFunctions) + uintptr(idx)*4))
	if !rvaWithinDirectory(rva, directory) {
		return resolvedWindowsExport{address: module.codeBase + uintptr(rva), owner: module}, nil
	}
	forwarder := windows.BytePtrToString((*byte)(a2p(module.codeBase + uintptr(rva))))
	chainKey := strings.ToLower(module.recursivePath) + "!" + symbol
	if _, ok := chain[chainKey]; ok {
		return resolvedWindowsExport{}, fmt.Errorf("recursive export forwarder cycle at %s", chainKey)
	}
	chain[chainKey] = struct{}{}
	defer delete(chain, chainKey)

	moduleName, targetName, ordinal, byOrdinal, err := parseExportForwarder(forwarder)
	if err != nil {
		return resolvedWindowsExport{}, err
	}
	dependency, err := module.recursive.resolveImport(module, moduleName)
	if err != nil {
		return resolvedWindowsExport{}, fmt.Errorf("resolve forwarded module %q: %w", moduleName, err)
	}
	var resolved resolvedWindowsExport
	if byOrdinal {
		resolved, err = dependency.procAddressByOrdinal(ordinal, chain)
	} else {
		resolved, err = dependency.procAddressByName(targetName, chain)
	}
	if err != nil {
		if dependency.handle != 0 {
			_ = windows.FreeLibrary(dependency.handle)
		}
		return resolvedWindowsExport{}, fmt.Errorf("resolve forwarded export %q: %w", forwarder, err)
	}
	if dependency.handle != 0 {
		module.recursiveMu.Lock()
		module.modules = append(module.modules, dependency.handle)
		module.recursiveMu.Unlock()
	}
	module.recursiveMu.Lock()
	if module.forwarders == nil {
		module.forwarders = make(map[string]resolvedWindowsExport)
	}
	module.forwarders[cacheKey] = resolved
	module.recursiveMu.Unlock()
	return resolved, nil
}

func (module *Module) resolveRecursiveForwarders() error {
	directory := module.headerDirectory(IMAGE_DIRECTORY_ENTRY_EXPORT)
	if directory.Size == 0 {
		return nil
	}
	exports := (*IMAGE_EXPORT_DIRECTORY)(a2p(module.codeBase + uintptr(directory.VirtualAddress)))
	for idx := uint32(0); idx < exports.NumberOfFunctions; idx++ {
		rva := *(*uint32)(a2p(module.codeBase + uintptr(exports.AddressOfFunctions) + uintptr(idx)*4))
		if !rvaWithinDirectory(rva, directory) {
			continue
		}
		if _, err := module.recursiveProcAddressByIndex(idx, "#index"+strconv.FormatUint(uint64(idx), 10), make(map[string]struct{})); err != nil {
			return err
		}
	}
	return nil
}

func rvaWithinDirectory(rva uint32, directory *IMAGE_DATA_DIRECTORY) bool {
	end := uint64(directory.VirtualAddress) + uint64(directory.Size)
	return rva >= directory.VirtualAddress && uint64(rva) < end
}

func parseExportForwarder(forwarder string) (moduleName string, targetName string, ordinal uint16, byOrdinal bool, err error) {
	separator := strings.LastIndexByte(forwarder, '.')
	if separator <= 0 || separator == len(forwarder)-1 {
		err = fmt.Errorf("malformed export forwarder %q", forwarder)
		return
	}
	moduleName = forwarder[:separator]
	if !strings.HasSuffix(strings.ToLower(moduleName), ".dll") {
		moduleName += ".dll"
	}
	targetName = forwarder[separator+1:]
	if !strings.HasPrefix(targetName, "#") {
		return
	}
	value, parseErr := strconv.ParseUint(strings.TrimPrefix(targetName, "#"), 10, 16)
	if parseErr != nil {
		err = fmt.Errorf("malformed ordinal export forwarder %q", forwarder)
		return
	}
	ordinal = uint16(value)
	byOrdinal = true
	targetName = ""
	return
}

func recursiveSearchPaths(importerPath string, rootPath string) []string {
	candidates := []string{filepath.Dir(importerPath), filepath.Dir(rootPath)}
	paths := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, candidate)
	}
	return paths
}

func canonicalRecursivePath(path string) (string, string, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || !filepath.IsAbs(path) {
		return "", "", fmt.Errorf("path must be absolute: %q", path)
	}
	return path, strings.ToLower(path), nil
}

func isAPISetContract(name string) bool {
	name = strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	return strings.HasPrefix(name, "api-ms-") || strings.HasPrefix(name, "ext-ms-")
}

func isWindowsSystemDependency(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || filepath.Base(name) != name {
		return false
	}
	base := strings.ToLower(name)
	if isAPISetContract(base) {
		return true
	}
	_, ok := windowsSystemDependencies[base]
	return ok
}

var windowsSystemDependencies = map[string]struct{}{
	"advapi32.dll":         {},
	"bcrypt.dll":           {},
	"bcryptprimitives.dll": {},
	"cabinet.dll":          {},
	"cfgmgr32.dll":         {},
	"clbcatq.dll":          {},
	"combase.dll":          {},
	"comctl32.dll":         {},
	"comdlg32.dll":         {},
	"crypt32.dll":          {},
	"cryptbase.dll":        {},
	"cryptsp.dll":          {},
	"dnsapi.dll":           {},
	"dwmapi.dll":           {},
	"gdi32.dll":            {},
	"gdi32full.dll":        {},
	"imm32.dll":            {},
	"iphlpapi.dll":         {},
	"kernel32.dll":         {},
	"kernelbase.dll":       {},
	"mpr.dll":              {},
	"msasn1.dll":           {},
	"msvcp_win.dll":        {},
	"msvcrt.dll":           {},
	"ncrypt.dll":           {},
	"netapi32.dll":         {},
	"normaliz.dll":         {},
	"ntdll.dll":            {},
	"ole32.dll":            {},
	"oleacc.dll":           {},
	"oleaut32.dll":         {},
	"pdh.dll":              {},
	"powrprof.dll":         {},
	"profapi.dll":          {},
	"psapi.dll":            {},
	"rpcrt4.dll":           {},
	"sechost.dll":          {},
	"secur32.dll":          {},
	"setupapi.dll":         {},
	"shell32.dll":          {},
	"shlwapi.dll":          {},
	"ucrtbase.dll":         {},
	"user32.dll":           {},
	"userenv.dll":          {},
	"version.dll":          {},
	"winhttp.dll":          {},
	"wininet.dll":          {},
	"winmm.dll":            {},
	"winnsi.dll":           {},
	"winspool.drv":         {},
	"ws2_32.dll":           {},
	"wtsapi32.dll":         {},
}
