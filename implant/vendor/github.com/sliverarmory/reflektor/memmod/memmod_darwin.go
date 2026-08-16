//go:build darwin && (amd64 || arm64)

package memmod

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	syscallSharedRegionCheckNP = uintptr(294)
	dyldScratchSize            = 0x4000
	rtldNow                    = 0x2
	rtldGlobal                 = 0x8
	rtldNoDelete               = 0x80

	lcSegment64 = 0x19
	lcSymtab    = 0x2
)

var (
	errDarwinLibraryClosed  = errors.New("library is closed")
	darwinLoaderDetailMu    sync.Mutex
	darwinLoaderDetail      string
	darwinDependencyHandles sync.Map
	darwinLoaderTransaction sync.Mutex
)

type Module struct {
	mu     sync.RWMutex
	image  []byte
	closed bool
}

// LoadLibrary loads a Mach-O image into the darwin in-memory loader context.
func LoadLibrary(data []byte) (*Module, error) {
	if len(data) == 0 {
		return nil, errors.New("empty Mach-O image")
	}

	image, err := selectCurrentArchMachOSlice(data)
	if err != nil {
		return nil, err
	}

	cloned := make([]byte, len(image))
	copy(cloned, image)
	return &Module{image: cloned}, nil
}

// Free releases the in-memory Mach-O bytes.
func (module *Module) Free() {
	module.mu.Lock()
	defer module.mu.Unlock()

	if module.closed {
		return
	}

	module.closed = true
	if module.image != nil {
		for i := range module.image {
			module.image[i] = 0
		}
		module.image = nil
	}
}

// CallExport loads the image and invokes the named exported symbol.
func (module *Module) CallExport(name string) error {
	symbol, err := normalizeMachOSymbol(name)
	if err != nil {
		return err
	}

	module.mu.RLock()
	if module.closed {
		module.mu.RUnlock()
		return errDarwinLibraryClosed
	}
	if len(module.image) == 0 {
		module.mu.RUnlock()
		return errors.New("library image is empty")
	}
	image := module.image
	module.mu.RUnlock()

	rc := memmodLoader(image, symbol)
	runtime.KeepAlive(image)

	if rc != 0 {
		return fmt.Errorf("call export %q: %w", name, loaderStatusError(rc))
	}
	return nil
}

// ProcAddressByName is not supported by the darwin loader path.
func (module *Module) ProcAddressByName(name string) (uintptr, error) {
	_ = name
	return 0, errors.New("ProcAddressByName is not supported on darwin; use CallExport")
}

// ProcAddressByOrdinal is not supported by the darwin loader path.
func (module *Module) ProcAddressByOrdinal(ordinal uint16) (uintptr, error) {
	_ = ordinal
	return 0, errors.New("ProcAddressByOrdinal is not supported on darwin; use CallExport")
}

type dyldCacheHeader struct {
	Magic                         [16]byte
	MappingOffset                 uint32
	MappingCount                  uint32
	ImagesOffsetOld               uint32
	ImagesCountOld                uint32
	DyldBaseAddress               uint64
	CodeSignatureOffset           uint64
	CodeSignatureSize             uint64
	SlideInfoOffsetUnused         uint64
	SlideInfoSizeUnused           uint64
	LocalSymbolsOffset            uint64
	LocalSymbolsSize              uint64
	UUID                          [16]byte
	CacheType                     uint64
	BranchPoolsOffset             uint32
	BranchPoolsCount              uint32
	AccelerateInfoAddr            uint64
	AccelerateInfoSize            uint64
	ImagesTextOffset              uint64
	ImagesTextCount               uint64
	PatchInfoAddr                 uint64
	PatchInfoSize                 uint64
	OtherImageGroupAddrUnused     uint64
	OtherImageGroupSizeUnused     uint64
	ProgClosuresAddr              uint64
	ProgClosuresSize              uint64
	ProgClosuresTrieAddr          uint64
	ProgClosuresTrieSize          uint64
	Platform                      uint32
	FormatVersionAndFlags         uint32
	SharedRegionStart             uint64
	SharedRegionSize              uint64
	MaxSlide                      uint64
	DylibsImageArrayAddr          uint64
	DylibsImageArraySize          uint64
	DylibsTrieAddr                uint64
	DylibsTrieSize                uint64
	OtherImageArrayAddr           uint64
	OtherImageArraySize           uint64
	OtherTrieAddr                 uint64
	OtherTrieSize                 uint64
	MappingWithSlideOffset        uint32
	MappingWithSlideCount         uint32
	DylibsPBLStateArrayAddrUnused uint64
	DylibsPBLSetAddr              uint64
	ProgramsPBLSetPoolAddr        uint64
	ProgramsPBLSetPoolSize        uint64
	ProgramTrieAddr               uint64
	ProgramTrieSize               uint32
	OSVersion                     uint32
	AltPlatform                   uint32
	AltOSVersion                  uint32
	SwiftOptsOffset               uint64
	SwiftOptsSize                 uint64
	SubCacheArrayOffset           uint32
	SubCacheArrayCount            uint32
	SymbolFileUUID                [16]byte
	RosettaReadOnlyAddr           uint64
	RosettaReadOnlySize           uint64
	RosettaReadWriteAddr          uint64
	RosettaReadWriteSize          uint64
	ImagesOffset                  uint32
	ImagesCount                   uint32
}

type dyldCacheImageInfo struct {
	Address        uint64
	ModTime        uint64
	Inode          uint64
	PathFileOffset uint32
	Pad            uint32
}

type sharedFileMapping struct {
	Address    uint64
	Size       uint64
	FileOffset uint64
	MaxProt    uint32
	InitProt   uint32
}

type machHeader64 struct {
	Magic      uint32
	CPUType    int32
	CPUSubType int32
	FileType   uint32
	NCmds      uint32
	SizeCmds   uint32
	Flags      uint32
	Reserved   uint32
}

type loadCommand struct {
	Cmd     uint32
	CmdSize uint32
}

type segmentCommand64 struct {
	Cmd      uint32
	CmdSize  uint32
	SegName  [16]byte
	VMAddr   uint64
	VMSize   uint64
	FileOff  uint64
	FileSize uint64
	MaxProt  uint32
	InitProt uint32
	NSects   uint32
	Flags    uint32
}

type section64 struct {
	SectName  [16]byte
	SegName   [16]byte
	Addr      uint64
	Size      uint64
	Offset    uint32
	Align     uint32
	RelOff    uint32
	NReloc    uint32
	Flags     uint32
	Reserved1 uint32
	Reserved2 uint32
	Reserved3 uint32
}

type symtabCommand struct {
	Cmd     uint32
	CmdSize uint32
	SymOff  uint32
	NSyms   uint32
	StrOff  uint32
	StrSize uint32
}

type nlist64 struct {
	Strx  uint32
	Type  uint8
	Sect  uint8
	Desc  uint16
	Value uint64
}

type fileID struct {
	INode   uint64
	ModTime uint64
	IsValid bool
	_       [7]byte
}

type loadChain struct {
	Previous uintptr
	Image    uintptr
}

type loadOptions struct {
	Launching               bool
	StaticLinkage           bool
	CanBeMissing            bool
	RtldLocal               bool
	RtldNoDelete            bool
	RtldNoLoad              bool
	InsertedDylib           bool
	CanBeDylib              bool
	CanBeBundle             bool
	CanBeExecutable         bool
	ForceUnloadable         bool
	RequestorNeedsFallbacks bool
	_                       [4]byte
	RpathStack              uintptr
	Finder                  uintptr
	PathNotFoundHandler     uintptr
}

type loadedVector struct {
	Allocator uintptr
	Elements  uintptr
	Size      uintptr
	Capacity  uintptr
}

type dyldCacheDataConstLazyScopedWriter struct {
	State           uintptr
	WasMadeWritable bool
	_               [7]byte
}

type mappedImage struct {
	mapping     []byte
	loadAddress uintptr
}

// preloadDarwinDependencies asks public dyld to load absolute system
// dependencies before the private in-memory transaction starts. Public dyld
// performs the complete registration, interposition, cache-patch, TLS, and
// initializer lifecycle for those dependency graphs. The private loader then
// only needs to bind the anonymous root image to already-registered libraries.
func preloadDarwinDependencies(buffer []byte, libdyld uintptr, slide uint64) error {
	file, err := macho.NewFile(bytes.NewReader(buffer))
	if err != nil {
		return fmt.Errorf("inspect Mach-O dependencies: %w", err)
	}
	defer file.Close()

	imports, err := file.ImportedLibraries()
	if err != nil {
		return fmt.Errorf("read Mach-O dependencies: %w", err)
	}

	pending := make([]string, 0, len(imports))
	seen := make(map[string]struct{}, len(imports))
	for _, installName := range imports {
		// System install names have process-independent resolution. Anonymous
		// images have no meaningful @loader_path, @rpath would use the process'
		// runpaths, and arbitrary absolute libraries may rely on root cycles.
		// Leave all of those names to Loader::loadDependents.
		if !isDarwinSystemInstallName(installName) {
			continue
		}
		if _, ok := seen[installName]; ok {
			continue
		}
		seen[installName] = struct{}{}
		if _, pinned := darwinDependencyHandles.Load(installName); pinned {
			continue
		}
		pending = append(pending, installName)
	}
	if len(pending) == 0 {
		return nil
	}

	dlopen := findFirstAvailableSymbol(libdyld, slide, "", "_dlopen")
	if dlopen == 0 {
		return errors.New("resolve public dyld symbol _dlopen")
	}
	dlerror := findFirstAvailableSymbol(libdyld, slide, "", "_dlerror")

	// dlerror state is native-thread-local. Keep the clear, load, and error
	// read on one OS thread while constructors are allowed to run freely.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for _, installName := range pending {
		if _, pinned := darwinDependencyHandles.Load(installName); pinned {
			continue
		}

		name, err := cStringBytes(installName)
		if err != nil {
			return fmt.Errorf("encode dependency install name %q: %w", installName, err)
		}
		if dlerror != 0 {
			_ = callDlerror(dlerror)
		}
		handle := callDlopen(dlopen, cStringPtr(name), rtldNow|rtldGlobal|rtldNoDelete)
		runtime.KeepAlive(name)
		if handle == 0 {
			if dlerror != 0 {
				if message := cStringAt(callDlerror(dlerror)); message != "" {
					return fmt.Errorf("dlopen(%q): %s", installName, message)
				}
			}
			return fmt.Errorf("dlopen(%q) returned a nil handle", installName)
		}

		// The mapped root is intentionally process-resident, so its dependency
		// handles remain process-resident too and are never passed to dlclose.
		darwinDependencyHandles.LoadOrStore(installName, handle)
	}

	return nil
}

func isDarwinSystemInstallName(installName string) bool {
	return strings.HasPrefix(installName, "/usr/lib/") ||
		strings.HasPrefix(installName, "/System/Library/")
}

func memmodLoader(bufferRO []byte, entrySymbol string) int {
	if len(bufferRO) == 0 || entrySymbol == "" {
		return 1
	}

	// The private dyld transaction reads and extends RuntimeState.loaded without
	// going through dyld's public loader lock. Serialize complete transactions so
	// concurrent Reflektor calls cannot snapshot and re-fix each other's roots.
	darwinLoaderTransaction.Lock()
	transactionLocked := true
	defer func() {
		if transactionLocked {
			darwinLoaderTransaction.Unlock()
		}
	}()

	sharedRegionStart, err := sharedRegionStartAddr()
	if err != nil || sharedRegionStart == 0 {
		return 2
	}

	header := (*dyldCacheHeader)(unsafe.Pointer(sharedRegionStart))
	sfm := (*sharedFileMapping)(unsafe.Pointer(sharedRegionStart + uintptr(header.MappingOffset)))
	if sfm == nil {
		return 2
	}

	imagesCount := header.ImagesCountOld
	if imagesCount == 0 {
		imagesCount = header.ImagesCount
	}
	imagesOffset := header.ImagesOffsetOld
	if imagesOffset == 0 {
		imagesOffset = header.ImagesOffset
	}
	if imagesCount == 0 || imagesOffset == 0 {
		return 2
	}

	slide := uint64(sharedRegionStart) - sfm.Address

	libdyld := findCacheImage(sharedRegionStart, header, "/usr/lib/system/libdyld.dylib", slide)
	if libdyld == 0 {
		return 2
	}
	dyld := findCacheImage(sharedRegionStart, header, "/usr/lib/dyld", slide)
	if dyld == 0 {
		return 2
	}
	setDarwinLoaderDetail("")
	if err := preloadDarwinDependencies(bufferRO, uintptr(libdyld), slide); err != nil {
		setDarwinLoaderDetail(err.Error())
		return 13
	}

	apis := resolveDyldRuntimeAPIs(libdyld, slide)
	if apis == 0 {
		return 3
	}
	setDarwinLoaderDetail("")

	buffer := bufferRO

	justInTimeLoaderMake2 := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZN5dyld416JustInTimeLoader4makeERNS_12RuntimeStateEPKN5dyld39MachOFileEPKcRKNS_6FileIDEybbbtPKN6mach_o6LayoutE",
	)
	loadDependents := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZN5dyld46Loader14loadDependentsER11DiagnosticsRNS_12RuntimeStateERKNS0_11LoadOptionsE",
		"__ZN5dyld416JustInTimeLoader14loadDependentsER11DiagnosticsRNS_12RuntimeStateERKNS_6Loader11LoadOptionsE",
		"__ZN5dyld414PrebuiltLoader14loadDependentsER11DiagnosticsRNS_12RuntimeStateERKNS_6Loader11LoadOptionsE",
	)
	if loadDependents == 0 {
		loadDependents = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"Loader14loadDependentsER11DiagnosticsRNS_12RuntimeStateE",
		)
	}
	applyFixups := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZNK5dyld46Loader11applyFixupsER11DiagnosticsRNS_12RuntimeStateERNS_34DyldCacheDataConstLazyScopedWriterEbPN3lsl6VectorINSt3__14pairIPKS0_PKcEEEE",
		"__ZNK5dyld416JustInTimeLoader11applyFixupsER11DiagnosticsRNS_12RuntimeStateERNS_34DyldCacheDataConstLazyScopedWriterEbPN3lsl6VectorINSt3__14pairIPKNS_6LoaderEPKcEEEE",
		"__ZNK5dyld414PrebuiltLoader11applyFixupsER11DiagnosticsRNS_12RuntimeStateERNS_34DyldCacheDataConstLazyScopedWriterEbPN3lsl6VectorINSt3__14pairIPKNS_6LoaderEPKcEEEE",
	)
	if applyFixups == 0 {
		applyFixups = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"Loader11applyFixupsER11DiagnosticsRNS_12RuntimeStateE",
		)
	}
	incDlRefCount := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZN5dyld412RuntimeState13incDlRefCountEPKNS_6LoaderE",
	)
	if incDlRefCount == 0 {
		incDlRefCount = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"RuntimeState13incDlRefCount",
		)
	}
	runInitializers := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZNK5dyld46Loader38runInitializersBottomUpPlusUpwardLinksERNS_12RuntimeStateE",
		"__ZNK5dyld46Loader15runInitializersERNS_12RuntimeStateE",
		"__ZNK5dyld416JustInTimeLoader15runInitializersERNS_12RuntimeStateE",
		"__ZNK5dyld414PrebuiltLoader15runInitializersERNS_12RuntimeStateE",
	)
	if runInitializers == 0 {
		runInitializers = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"runInitializers",
			"RuntimeState",
		)
	}

	diagnosticsCtor := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZN11DiagnosticsC1Ev",
		"__ZN11DiagnosticsC2Ev",
	)
	if diagnosticsCtor == 0 {
		diagnosticsCtor = findFirstAvailableSymbol(uintptr(libdyld), slide, "",
			"__ZN11DiagnosticsC1Ev",
			"__ZN11DiagnosticsC2Ev",
		)
	}
	if diagnosticsCtor == 0 {
		diagnosticsCtor = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"DiagnosticsC",
			"Ev",
		)
	}
	if diagnosticsCtor == 0 {
		diagnosticsCtor = findFirstMatchingSymbol(uintptr(libdyld), slide, "",
			"DiagnosticsC",
			"Ev",
		)
	}
	diagnosticsClearError := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZN11Diagnostics10clearErrorEv",
	)
	if diagnosticsClearError == 0 {
		diagnosticsClearError = findFirstAvailableSymbol(uintptr(libdyld), slide, "",
			"__ZN11Diagnostics10clearErrorEv",
		)
	}
	if diagnosticsClearError == 0 {
		diagnosticsClearError = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"Diagnostics10clearErrorEv",
		)
	}
	if diagnosticsClearError == 0 {
		diagnosticsClearError = findFirstMatchingSymbol(uintptr(libdyld), slide, "",
			"Diagnostics10clearErrorEv",
		)
	}
	diagnosticsHasError := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZNK11Diagnostics8hasErrorEv",
	)
	if diagnosticsHasError == 0 {
		diagnosticsHasError = findFirstAvailableSymbol(uintptr(libdyld), slide, "",
			"__ZNK11Diagnostics8hasErrorEv",
		)
	}
	if diagnosticsHasError == 0 {
		diagnosticsHasError = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"Diagnostics8hasErrorEv",
		)
	}
	if diagnosticsHasError == 0 {
		diagnosticsHasError = findFirstMatchingSymbol(uintptr(libdyld), slide, "",
			"Diagnostics8hasErrorEv",
		)
	}
	diagnosticsErrorMessage := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
		"__ZNK11Diagnostics12errorMessageEv",
	)
	if diagnosticsErrorMessage == 0 {
		diagnosticsErrorMessage = findFirstAvailableSymbol(uintptr(libdyld), slide, "",
			"__ZNK11Diagnostics12errorMessageEv",
		)
	}
	if diagnosticsErrorMessage == 0 {
		diagnosticsErrorMessage = findFirstMatchingSymbol(uintptr(dyld), slide, "/usr/lib/dyld",
			"Diagnostics12errorMessageEv",
		)
	}
	if diagnosticsErrorMessage == 0 {
		diagnosticsErrorMessage = findFirstMatchingSymbol(uintptr(libdyld), slide, "",
			"Diagnostics12errorMessageEv",
		)
	}

	missing := make([]string, 0, 8)
	if justInTimeLoaderMake2 == 0 {
		missing = append(missing, "JustInTimeLoader::make")
	}
	if loadDependents == 0 {
		missing = append(missing, "Loader::loadDependents")
	}
	if applyFixups == 0 {
		missing = append(missing, "Loader::applyFixups")
	}
	if incDlRefCount == 0 {
		missing = append(missing, "RuntimeState::incDlRefCount")
	}
	if runInitializers == 0 {
		missing = append(missing, "Loader::runInitializers")
	}
	if diagnosticsClearError == 0 {
		missing = append(missing, "Diagnostics::clearError")
	}
	if diagnosticsHasError == 0 {
		missing = append(missing, "Diagnostics::hasError")
	}
	if len(missing) != 0 {
		setDarwinLoaderDetail(strings.Join(missing, ", "))
		return 4
	}
	setDarwinLoaderDetail("")

	memoryManager := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld", "__ZN3lsl13MemoryManager13memoryManagerEv")
	lockLock := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld", "__ZN3lsl4Lock4lockEv")
	writeProtect := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld", "__ZN3lsl13MemoryManager12writeProtectEb")
	lockUnlock := findFirstAvailableSymbol(uintptr(dyld), slide, "/usr/lib/dyld", "__ZN3lsl4Lock6unlockEv")

	mapped, rc := mapMachOImage(buffer)
	if rc != 0 {
		return rc
	}

	scratch, mapErr := unix.Mmap(-1, 0, dyldScratchSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if mapErr != nil || len(scratch) < dyldScratchSize {
		return 7
	}
	structspace := uintptr(unsafe.Pointer(&scratch[0]))

	rtopLoader := (*uintptr)(unsafe.Pointer(structspace))
	cursor := structspace + unsafe.Sizeof(uintptr(0))

	fileid := (*fileID)(unsafe.Pointer(cursor))
	cursor += unsafe.Sizeof(fileID{})
	fileid.INode = 0
	fileid.ModTime = 0
	fileid.IsValid = false

	diag := unsafe.Pointer(cursor)
	cursor += 0x1000
	diagnosticsReady := diagnosticsCtor != 0
	if diagnosticsReady {
		call1(diagnosticsCtor, uintptr(diag))
	}

	loadChainMain := (*loadChain)(unsafe.Pointer(cursor))
	cursor += unsafe.Sizeof(loadChain{})
	loadChainCaller := (*loadChain)(unsafe.Pointer(cursor))
	cursor += unsafe.Sizeof(loadChain{})
	loadChainCur := (*loadChain)(unsafe.Pointer(cursor))
	cursor += unsafe.Sizeof(loadChain{})

	depOptions := (*loadOptions)(unsafe.Pointer(cursor))

	loaded := (*loadedVector)(unsafe.Pointer(apis + 32))
	startLoaderCount := loaded.Size

	entryName, err := cStringBytes(fmt.Sprintf("memmod-%x-%x", uintptr(unsafe.Pointer(&buffer[0])), len(buffer)))
	if err != nil {
		setDarwinLoaderDetail("failed to build temporary loader name")
		return 8
	}

	enteredWritable := false
	memoryManagerInstance := uintptr(0)
	if memoryManager != 0 {
		memoryManagerInstance = call0(memoryManager)
	}
	if memoryManagerInstance != 0 && lockLock != 0 && writeProtect != 0 && lockUnlock != 0 {
		enteredWritable = enterWritableDyldStateLock(memoryManagerInstance, lockLock, writeProtect, lockUnlock)
	}
	if enteredWritable {
		defer func() {
			if enteredWritable {
				exitWritableDyldStateLock(memoryManagerInstance, lockLock, writeProtect, lockUnlock)
			}
		}()
	}

	if diagnosticsReady {
		call1(diagnosticsClearError, uintptr(diag))
	}
	*rtopLoader = 0

	topLoader := call10(
		justInTimeLoaderMake2,
		apis,
		mapped.loadAddress,
		cStringPtr(entryName),
		uintptr(unsafe.Pointer(fileid)),
		0,
		0,
		1,
		0,
		0,
		0,
	)
	runtime.KeepAlive(entryName)
	if diagnosticsReady && call1(diagnosticsHasError, uintptr(diag)) != 0 {
		msg := diagnosticsMessage(diag, diagnosticsErrorMessage)
		if msg != "" {
			setDarwinLoaderDetail(fmt.Sprintf("JustInTimeLoader::make returned diagnostics error: %s", msg))
		} else {
			setDarwinLoaderDetail("JustInTimeLoader::make returned diagnostics error")
		}
		return 8
	}
	if topLoader == 0 {
		setDarwinLoaderDetail("JustInTimeLoader::make returned null loader")
		return 8
	}
	setDarwinLoaderDetail("")
	*rtopLoader = topLoader

	loadChainMain.Previous = 0
	loadChainMain.Image = *(*uintptr)(unsafe.Pointer(apis + 24))

	loadChainCaller.Previous = uintptr(unsafe.Pointer(loadChainMain))
	if loaded.Elements != 0 {
		loadChainCaller.Image = *(*uintptr)(unsafe.Pointer(loaded.Elements))
	}

	loadChainCur.Previous = uintptr(unsafe.Pointer(loadChainCaller))
	loadChainCur.Image = topLoader

	// LC_LOAD_DYLIB dependencies use static linkage semantics. The root was
	// already created with leaveMapped=true, so no private Loader flag writes
	// are needed here (their offsets are dyld-version-specific).
	depOptions.StaticLinkage = true
	depOptions.RtldLocal = false
	depOptions.RtldNoDelete = true
	depOptions.CanBeDylib = true
	depOptions.RpathStack = uintptr(unsafe.Pointer(loadChainCur))
	depOptions.RequestorNeedsFallbacks = false

	if diagnosticsReady {
		call1(diagnosticsClearError, uintptr(diag))
	}
	call4(loadDependents, topLoader, uintptr(diag), apis, uintptr(unsafe.Pointer(depOptions)))
	if diagnosticsReady && call1(diagnosticsHasError, uintptr(diag)) != 0 {
		if msg := diagnosticsMessage(diag, diagnosticsErrorMessage); msg != "" {
			setDarwinLoaderDetail(fmt.Sprintf("Loader::loadDependents reported diagnostics error: %s", msg))
		} else {
			setDarwinLoaderDetail("Loader::loadDependents reported diagnostics error")
		}
		return 9
	}

	newLoadersCount := loaded.Size - startLoaderCount
	if newLoadersCount != 0 {
		dcd := dyldCacheDataConstLazyScopedWriter{State: apis}
		for i := uintptr(0); i < newLoadersCount; i++ {
			ldr := loadedElement(loaded, startLoaderCount+i)
			call6(applyFixups, ldr, uintptr(diag), apis, uintptr(unsafe.Pointer(&dcd)), 1, 0)
		}
		if diagnosticsReady && call1(diagnosticsHasError, uintptr(diag)) != 0 {
			if msg := diagnosticsMessage(diag, diagnosticsErrorMessage); msg != "" {
				setDarwinLoaderDetail(fmt.Sprintf("Loader::applyFixups reported diagnostics error: %s", msg))
			} else {
				setDarwinLoaderDetail("Loader::applyFixups reported diagnostics error")
			}
			return 9
		}
	}

	setDarwinLoaderDetail("")
	call2(incDlRefCount, apis, topLoader)
	call2(runInitializers, topLoader, apis)

	loadedText := findLoadedTextSegment(mapped.loadAddress)
	if loadedText == nil {
		return 10
	}
	if mapped.loadAddress < uintptr(loadedText.VMAddr) {
		return 11
	}
	imageSlide := mapped.loadAddress - uintptr(loadedText.VMAddr)
	addrEntry := findSymbol(mapped.loadAddress, entrySymbol, uint64(imageSlide))
	if addrEntry == 0 {
		return 12
	}

	// Restore dyld's normal write-protected state before calling arbitrary
	// library code. The export may create threads or enter dyld itself.
	if enteredWritable {
		exitWritableDyldStateLock(memoryManagerInstance, lockLock, writeProtect, lockUnlock)
		enteredWritable = false
	}
	// The loader transaction is complete. Do not retain Reflektor's mutex while
	// running arbitrary export code, which may re-enter Reflektor.
	transactionLocked = false
	darwinLoaderTransaction.Unlock()

	// Exported fixture entry points use the C void(void) ABI. Keep that call
	// signature exact when cgo supplies the foreign-function bridge.
	callVoid0(addrEntry)
	// Keep mapped and scratch memory reachable until after entry returns.
	runtime.KeepAlive(mapped.mapping)
	runtime.KeepAlive(scratch)
	return 0
}

func sharedRegionStartAddr() (uintptr, error) {
	var address uintptr
	_, _, errno := unix.Syscall(syscallSharedRegionCheckNP, uintptr(unsafe.Pointer(&address)), 0, 0)
	if errno != 0 {
		return 0, errno
	}
	if address == 0 {
		return 0, errors.New("shared region address is nil")
	}
	return address, nil
}

func mapMachOImage(data []byte) (mappedImage, int) {
	if len(data) == 0 {
		return mappedImage{}, 5
	}

	f, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return mappedImage{}, 5
	}
	defer f.Close()

	var (
		textSeg *macho.Segment
		minVM   uint64 = math.MaxUint64
		maxVM   uint64
	)
	segments := make([]*macho.Segment, 0, len(f.Loads))
	for _, load := range f.Loads {
		seg, ok := load.(*macho.Segment)
		if !ok {
			continue
		}
		segments = append(segments, seg)
		if seg.Name == "__TEXT" {
			textSeg = seg
		}
		if seg.Memsz == 0 {
			continue
		}
		if seg.Addr < minVM {
			minVM = seg.Addr
		}
		end := seg.Addr + seg.Memsz
		if end > maxVM {
			maxVM = end
		}
	}
	if textSeg == nil {
		return mappedImage{}, 10
	}
	if minVM == math.MaxUint64 || maxVM <= minVM {
		return mappedImage{}, 5
	}

	vmSpace := maxVM - minVM
	if vmSpace == 0 || vmSpace > uint64(math.MaxInt) {
		return mappedImage{}, 5
	}

	mapped, mmapErr := unix.Mmap(-1, 0, int(vmSpace), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if mmapErr != nil || len(mapped) == 0 {
		return mappedImage{}, 6
	}
	base := uintptr(unsafe.Pointer(&mapped[0]))
	imageBase := base - uintptr(minVM)

	for _, seg := range segments {
		if seg.Filesz == 0 {
			continue
		}
		if seg.Offset > uint64(len(data)) || seg.Filesz > uint64(len(data))-seg.Offset {
			return mappedImage{}, 5
		}
		if seg.Filesz > uint64(math.MaxInt) {
			return mappedImage{}, 5
		}
		dst := imageBase + uintptr(seg.Addr)
		sz := int(seg.Filesz)
		dstSlice := unsafe.Slice((*byte)(unsafe.Pointer(dst)), sz)
		src := data[seg.Offset : seg.Offset+seg.Filesz]
		copy(dstSlice, src)
	}

	pageSize := uintptr(unix.Getpagesize())
	for _, seg := range segments {
		if seg.Memsz == 0 {
			continue
		}
		start := imageBase + uintptr(seg.Addr)
		end := start + uintptr(seg.Memsz)
		pageStart := alignDown(start, pageSize)
		pageEnd := alignUp(end, pageSize)
		if pageEnd <= pageStart {
			continue
		}
		protLen := pageEnd - pageStart
		if protLen > uintptr(math.MaxInt) {
			return mappedImage{}, 6
		}
		protSlice := unsafe.Slice((*byte)(unsafe.Pointer(pageStart)), int(protLen))
		if err := unix.Mprotect(protSlice, int(seg.Prot)); err != nil {
			return mappedImage{}, 6
		}
	}

	if textSeg.Offset > textSeg.Addr+vmSpace {
		return mappedImage{}, 5
	}
	loadAddress := imageBase + uintptr(textSeg.Addr) - uintptr(textSeg.Offset)
	if loadAddress < base || loadAddress >= base+uintptr(len(mapped)) {
		return mappedImage{}, 5
	}

	return mappedImage{mapping: mapped, loadAddress: loadAddress}, 0
}

func alignDown(v, a uintptr) uintptr {
	if a == 0 {
		return v
	}
	return v &^ (a - 1)
}

func alignUp(v, a uintptr) uintptr {
	if a == 0 {
		return v
	}
	return (v + (a - 1)) &^ (a - 1)
}

func findCacheImage(sharedRegionStart uintptr, header *dyldCacheHeader, wantPath string, slide uint64) uint64 {
	imagesCount := header.ImagesCountOld
	if imagesCount == 0 {
		imagesCount = header.ImagesCount
	}
	imagesOffset := header.ImagesOffsetOld
	if imagesOffset == 0 {
		imagesOffset = header.ImagesOffset
	}
	if imagesCount == 0 || imagesOffset == 0 {
		return 0
	}

	entrySize := unsafe.Sizeof(dyldCacheImageInfo{})
	base := sharedRegionStart + uintptr(imagesOffset)
	for i := uint32(0); i < imagesCount; i++ {
		entry := (*dyldCacheImageInfo)(unsafe.Pointer(base + uintptr(i)*entrySize))
		path := sharedRegionStart + uintptr(entry.PathFileOffset)
		if cStringEqual(path, wantPath) {
			return entry.Address + slide
		}
	}
	return 0
}

func findSection(base uint64, segName string, sectName string, slide uint64) uintptr {
	mh := (*machHeader64)(unsafe.Pointer(uintptr(base)))
	lc := uintptr(base) + unsafe.Sizeof(machHeader64{})

	for i := uint32(0); i < mh.NCmds; i++ {
		cmd := (*loadCommand)(unsafe.Pointer(lc))
		if cmd.Cmd == lcSegment64 {
			seg := (*segmentCommand64)(unsafe.Pointer(lc))
			if fixedCString(seg.SegName[:]) == segName {
				sect := lc + unsafe.Sizeof(segmentCommand64{})
				for j := uint32(0); j < seg.NSects; j++ {
					s := (*section64)(unsafe.Pointer(sect + uintptr(j)*unsafe.Sizeof(section64{})))
					if fixedCString(s.SectName[:]) == sectName {
						return uintptr(s.Addr + slide)
					}
				}
			}
		}
		lc += uintptr(cmd.CmdSize)
	}
	return 0
}

func findSectionAnySegment(base uint64, sectName string, slide uint64) uintptr {
	mh := (*machHeader64)(unsafe.Pointer(uintptr(base)))
	lc := uintptr(base) + unsafe.Sizeof(machHeader64{})

	for i := uint32(0); i < mh.NCmds; i++ {
		cmd := (*loadCommand)(unsafe.Pointer(lc))
		if cmd.Cmd == lcSegment64 {
			seg := (*segmentCommand64)(unsafe.Pointer(lc))
			sect := lc + unsafe.Sizeof(segmentCommand64{})
			for j := uint32(0); j < seg.NSects; j++ {
				s := (*section64)(unsafe.Pointer(sect + uintptr(j)*unsafe.Sizeof(section64{})))
				if fixedCString(s.SectName[:]) == sectName {
					return uintptr(s.Addr + slide)
				}
			}
		}
		lc += uintptr(cmd.CmdSize)
	}
	return 0
}

func resolveDyldRuntimeAPIs(libdyld uint64, slide uint64) uintptr {
	if libdyld == 0 {
		return 0
	}

	// Keep legacy-first order for older dyld cache layouts, then probe newer
	// common data segments and finally a segment-agnostic section search.
	candidates := [][2]string{
		{"__TPRO_CONST", "__dyld_apis"},
		{"__DATA_CONST", "__dyld_apis"},
		{"__AUTH_CONST", "__dyld_apis"},
		{"__DATA", "__dyld_apis"},
	}

	for _, candidate := range candidates {
		sec := findSection(libdyld, candidate[0], candidate[1], slide)
		if apis := dyldRuntimeAPIsFromSection(sec); apis != 0 {
			return apis
		}
	}

	if sec := findSectionAnySegment(libdyld, "__dyld_apis", slide); sec != 0 {
		if apis := dyldRuntimeAPIsFromSection(sec); apis != 0 {
			return apis
		}
	}

	return 0
}

func dyldRuntimeAPIsFromSection(sectionAddr uintptr) uintptr {
	if sectionAddr == 0 {
		return 0
	}

	apis := *(*uintptr)(unsafe.Pointer(sectionAddr))
	if apis != 0 {
		return apis
	}

	// Some layouts may expose the APIs struct directly at section base.
	// Validate the expected loaded-vector pointers (offsets used below).
	imagePtr := *(*uintptr)(unsafe.Pointer(sectionAddr + 24))
	vectorElemPtr := *(*uintptr)(unsafe.Pointer(sectionAddr + 32))
	if imagePtr != 0 || vectorElemPtr != 0 {
		return sectionAddr
	}

	return 0
}

func findSymbol(base uintptr, symbol string, offset uint64) uintptr {
	mh := (*machHeader64)(unsafe.Pointer(base))
	lc := base + unsafe.Sizeof(machHeader64{})

	var (
		symtab   *symtabCommand
		linkedit *segmentCommand64
		text     *segmentCommand64
	)

	for i := uint32(0); i < mh.NCmds; i++ {
		cmd := (*loadCommand)(unsafe.Pointer(lc))
		switch cmd.Cmd {
		case lcSymtab:
			symtab = (*symtabCommand)(unsafe.Pointer(lc))
		case lcSegment64:
			seg := (*segmentCommand64)(unsafe.Pointer(lc))
			switch fixedCString(seg.SegName[:]) {
			case "__LINKEDIT":
				linkedit = seg
			case "__TEXT":
				text = seg
			}
		}
		lc += uintptr(cmd.CmdSize)
	}

	if symtab == nil || linkedit == nil || text == nil {
		return 0
	}

	fileSlide := int64(linkedit.VMAddr) - int64(text.VMAddr) - int64(linkedit.FileOff)
	strtab := base + uintptr(fileSlide+int64(symtab.StrOff))
	nlBase := base + uintptr(fileSlide+int64(symtab.SymOff))

	nlSize := unsafe.Sizeof(nlist64{})
	for i := uint32(0); i < symtab.NSyms; i++ {
		nl := (*nlist64)(unsafe.Pointer(nlBase + uintptr(i)*nlSize))
		if nl.Strx == 0 {
			continue
		}
		name := strtab + uintptr(nl.Strx)
		if cStringEqual(name, symbol) {
			if nl.Value == 0 {
				continue
			}
			return uintptr(nl.Value + offset)
		}
	}
	return 0
}

func findFirstAvailableSymbol(base uintptr, offset uint64, diskPath string, symbols ...string) uintptr {
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}
		if addr := findSymbol(base, symbol, offset); addr != 0 {
			return addr
		}
	}
	if diskPath == "" {
		return 0
	}
	for _, symbol := range symbols {
		if symbol == "" {
			continue
		}
		if addr := findSymbolInMachOFile(diskPath, symbol, offset); addr != 0 {
			return addr
		}
	}
	return 0
}

func findFirstMatchingSymbol(base uintptr, offset uint64, diskPath string, required ...string) uintptr {
	if len(required) == 0 {
		return 0
	}
	if addr := findSymbolByContains(base, offset, required...); addr != 0 {
		return addr
	}
	if diskPath == "" {
		return 0
	}
	return findSymbolInMachOFileByContains(diskPath, offset, required...)
}

func findSymbolByContains(base uintptr, offset uint64, required ...string) uintptr {
	mh := (*machHeader64)(unsafe.Pointer(base))
	lc := base + unsafe.Sizeof(machHeader64{})

	var (
		symtab   *symtabCommand
		linkedit *segmentCommand64
		text     *segmentCommand64
	)

	for i := uint32(0); i < mh.NCmds; i++ {
		cmd := (*loadCommand)(unsafe.Pointer(lc))
		switch cmd.Cmd {
		case lcSymtab:
			symtab = (*symtabCommand)(unsafe.Pointer(lc))
		case lcSegment64:
			seg := (*segmentCommand64)(unsafe.Pointer(lc))
			switch fixedCString(seg.SegName[:]) {
			case "__LINKEDIT":
				linkedit = seg
			case "__TEXT":
				text = seg
			}
		}
		lc += uintptr(cmd.CmdSize)
	}

	if symtab == nil || linkedit == nil || text == nil {
		return 0
	}

	fileSlide := int64(linkedit.VMAddr) - int64(text.VMAddr) - int64(linkedit.FileOff)
	strtab := base + uintptr(fileSlide+int64(symtab.StrOff))
	nlBase := base + uintptr(fileSlide+int64(symtab.SymOff))
	nlSize := unsafe.Sizeof(nlist64{})

	bestLen := math.MaxInt
	bestAddr := uintptr(0)
	for i := uint32(0); i < symtab.NSyms; i++ {
		nl := (*nlist64)(unsafe.Pointer(nlBase + uintptr(i)*nlSize))
		if nl.Strx == 0 || nl.Value == 0 {
			continue
		}
		name := cStringAt(strtab + uintptr(nl.Strx))
		if !isUsableSymbolCandidate(name) || !containsAll(name, required...) {
			continue
		}
		if len(name) < bestLen {
			bestLen = len(name)
			bestAddr = uintptr(nl.Value + offset)
		}
	}
	return bestAddr
}

func findSymbolInMachOFileByContains(path string, slide uint64, required ...string) uintptr {
	file, closeFn, err := openCurrentArchMachOFile(path)
	if err != nil || file == nil {
		return 0
	}
	defer closeFn()

	if file.Symtab == nil || len(file.Symtab.Syms) == 0 {
		return 0
	}
	bestLen := math.MaxInt
	bestAddr := uintptr(0)
	for _, sym := range file.Symtab.Syms {
		if sym.Value == 0 {
			continue
		}
		if !isUsableSymbolCandidate(sym.Name) || !containsAll(sym.Name, required...) {
			continue
		}
		if len(sym.Name) < bestLen {
			bestLen = len(sym.Name)
			bestAddr = uintptr(sym.Value + slide)
		}
	}
	return bestAddr
}

func cStringAt(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	buf := make([]byte, 0, 64)
	for i := 0; i < 4096; i++ {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
		if b == 0 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf)
}

func diagnosticsMessage(diag unsafe.Pointer, errorMessageFn uintptr) string {
	if diag == nil || errorMessageFn == 0 {
		return ""
	}
	msgPtr := call1(errorMessageFn, uintptr(diag))
	if msgPtr == 0 {
		return ""
	}
	msg := strings.TrimSpace(cStringAt(msgPtr))
	if msg == "" {
		return ""
	}
	return msg
}

func isUsableSymbolCandidate(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "block_invoke") || strings.Contains(name, ".cold") {
		return false
	}
	return true
}

func containsAll(name string, required ...string) bool {
	for _, needle := range required {
		if needle == "" {
			continue
		}
		if !strings.Contains(name, needle) {
			return false
		}
	}
	return true
}

func findSymbolInMachOFile(path string, symbol string, slide uint64) uintptr {
	file, closeFn, err := openCurrentArchMachOFile(path)
	if err != nil || file == nil {
		return 0
	}
	defer closeFn()

	if file.Symtab == nil || len(file.Symtab.Syms) == 0 {
		return 0
	}
	for _, sym := range file.Symtab.Syms {
		if sym.Name != symbol || sym.Value == 0 {
			continue
		}
		return uintptr(sym.Value + slide)
	}
	return 0
}

func openCurrentArchMachOFile(path string) (*macho.File, func(), error) {
	cpu, err := currentMachOCPU()
	if err != nil {
		return nil, func() {}, err
	}

	if fat, err := macho.OpenFat(path); err == nil {
		for _, arch := range fat.Arches {
			if arch.Cpu == cpu {
				return arch.File, func() { _ = fat.Close() }, nil
			}
		}
		_ = fat.Close()
		return nil, func() {}, fmt.Errorf("no matching architecture in %s", path)
	}

	file, err := macho.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	if file.Cpu != cpu {
		_ = file.Close()
		return nil, func() {}, fmt.Errorf("wrong architecture in %s: got %s want %s", path, file.Cpu, cpu)
	}
	return file, func() { _ = file.Close() }, nil
}

func findLoadedTextSegment(base uintptr) *segmentCommand64 {
	mh := (*machHeader64)(unsafe.Pointer(base))
	lc := base + unsafe.Sizeof(machHeader64{})
	for i := uint32(0); i < mh.NCmds; i++ {
		cmd := (*loadCommand)(unsafe.Pointer(lc))
		if cmd.Cmd == lcSegment64 {
			seg := (*segmentCommand64)(unsafe.Pointer(lc))
			if fixedCString(seg.SegName[:]) == "__TEXT" {
				return seg
			}
		}
		lc += uintptr(cmd.CmdSize)
	}
	return nil
}

func fixedCString(buf []byte) string {
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}
	return string(buf[:end])
}

func cStringEqual(ptr uintptr, want string) bool {
	if ptr == 0 {
		return false
	}
	for i := 0; i < len(want); i++ {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
		if b != want[i] {
			return false
		}
	}
	return *(*byte)(unsafe.Pointer(ptr + uintptr(len(want)))) == 0
}

func loadedElement(v *loadedVector, idx uintptr) uintptr {
	if v == nil || v.Elements == 0 {
		return 0
	}
	stride := unsafe.Sizeof(uintptr(0))
	return *(*uintptr)(unsafe.Pointer(v.Elements + idx*stride))
}

func enterWritableDyldStateLock(mm, lockFn, writeProtectFn, unlockFn uintptr) bool {
	if mm == 0 || lockFn == 0 || writeProtectFn == 0 || unlockFn == 0 {
		return false
	}
	call1(lockFn, mm)
	counter := (*uint64)(unsafe.Pointer(mm + 0x18))
	c := *counter
	if c == 0 {
		call2(writeProtectFn, mm, 0)
		c = *counter
	}
	*counter = c + 1
	call1(unlockFn, mm)
	return true
}

func exitWritableDyldStateLock(mm, lockFn, writeProtectFn, unlockFn uintptr) {
	if mm == 0 || lockFn == 0 || writeProtectFn == 0 || unlockFn == 0 {
		return
	}
	call1(lockFn, mm)
	counter := (*uint64)(unsafe.Pointer(mm + 0x18))
	c := *counter
	if c != 0 {
		c--
		*counter = c
		if c == 0 {
			call2(writeProtectFn, mm, 1)
		}
	}
	call1(unlockFn, mm)
}

func selectCurrentArchMachOSlice(data []byte) ([]byte, error) {
	cpu, err := currentMachOCPU()
	if err != nil {
		return nil, err
	}

	if fat, err := macho.NewFatFile(bytes.NewReader(data)); err == nil {
		defer fat.Close()
		for _, arch := range fat.Arches {
			if arch.Cpu != cpu {
				continue
			}
			offset := int(arch.Offset)
			size := int(arch.Size)
			if offset < 0 || size <= 0 || offset+size > len(data) {
				return nil, errors.New("invalid fat Mach-O slice bounds")
			}
			slice := data[offset : offset+size]
			if err := validateThinMachO(slice, cpu); err != nil {
				return nil, err
			}
			return slice, nil
		}
		return nil, fmt.Errorf("foreign platform: no %s slice in fat Mach-O", cpu)
	}

	if err := validateThinMachO(data, cpu); err != nil {
		return nil, err
	}
	return data, nil
}

func validateThinMachO(data []byte, expectedCPU macho.Cpu) error {
	file, err := macho.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid Mach-O image: %w", err)
	}
	defer file.Close()

	if file.Cpu != expectedCPU {
		return fmt.Errorf("foreign platform (provided: %s, expected: %s)", file.Cpu, expectedCPU)
	}
	switch file.Type {
	case macho.TypeDylib, macho.TypeBundle:
		return nil
	default:
		return fmt.Errorf("unsupported Mach-O file type: %v", file.Type)
	}
}

func currentMachOCPU() (macho.Cpu, error) {
	switch runtime.GOARCH {
	case "arm64":
		return macho.CpuArm64, nil
	case "amd64":
		return macho.CpuAmd64, nil
	default:
		return 0, fmt.Errorf("unsupported darwin architecture: %s", runtime.GOARCH)
	}
}

func normalizeMachOSymbol(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("export name cannot be empty")
	}
	if strings.ContainsRune(name, '\x00') {
		return "", errors.New("export name contains NUL")
	}
	if !strings.HasPrefix(name, "_") {
		name = "_" + name
	}
	return name, nil
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

func setDarwinLoaderDetail(detail string) {
	darwinLoaderDetailMu.Lock()
	defer darwinLoaderDetailMu.Unlock()
	darwinLoaderDetail = detail
}

func getDarwinLoaderDetail() string {
	darwinLoaderDetailMu.Lock()
	defer darwinLoaderDetailMu.Unlock()
	return darwinLoaderDetail
}

func loaderStatusError(code int) error {
	switch code {
	case 2:
		return errors.New("failed to locate required dyld cache images")
	case 3:
		return errors.New("failed to resolve dyld runtime API section")
	case 4:
		if detail := getDarwinLoaderDetail(); detail != "" {
			return fmt.Errorf("failed to resolve required dyld symbols: %s", detail)
		}
		return errors.New("failed to resolve required dyld symbols")
	case 5:
		return errors.New("failed to analyze Mach-O VM layout")
	case 6:
		return errors.New("failed to allocate mapped image space")
	case 7:
		return errors.New("failed to allocate dyld scratch space")
	case 8:
		if detail := getDarwinLoaderDetail(); detail != "" {
			return fmt.Errorf("failed to create top-level dyld loader: %s", detail)
		}
		return errors.New("failed to create top-level dyld loader")
	case 9:
		if detail := getDarwinLoaderDetail(); detail != "" {
			return fmt.Errorf("failed to load dependents or apply fixups: %s", detail)
		}
		return errors.New("failed to load dependents or apply fixups")
	case 10:
		return errors.New("failed to find __TEXT segment in loaded image")
	case 11:
		return errors.New("invalid loaded image slide")
	case 12:
		return errors.New("export symbol not found")
	case 13:
		if detail := getDarwinLoaderDetail(); detail != "" {
			return fmt.Errorf("failed to preload Mach-O dependencies: %s", detail)
		}
		return errors.New("failed to preload Mach-O dependencies")
	default:
		return fmt.Errorf("in-memory dyld loader failed with status %d", code)
	}
}
