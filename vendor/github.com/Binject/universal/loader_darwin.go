package universal

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"unsafe"

	"github.com/Binject/debug/macho"
	"github.com/awgh/cppgo/asmcall/cdecl"
	"github.com/awgh/rawreader"
	"golang.org/x/sys/unix"
)

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)
const LIB_DYLD_PATH = "/usr/lib/system/libdyld.dylib"

func syscallSharedRegionCheckNp() (uint64, error) {
	var address uint64
	r1, _, _ := unix.RawSyscall(unix.SYS_SHARED_REGION_CHECK_NP, uintptr(unsafe.Pointer(&address)), 0, 0)
	if r1 != 0 {
		return 0, errors.New("shared region check failed")
	}
	return address, nil
}

func isSierra() bool {
	var utsName unix.Utsname
	err := unix.Uname(&utsName)
	if err != nil {
		return true
	}
	if utsName.Release[0] == '1' && utsName.Release[1] < '6' {
		return false
	}
	if utsName.Release[0] == '9' && utsName.Release[1] == '.' {
		return false
	}
	return true
}

// LoadLibraryImpl - loads a single library to memory, without trying to check or load required imports
func LoadLibraryImpl(name string, image *[]byte) (*Library, error) {
	// On MacOS Sierra and up, NSLinkModule is just a wrapper around
	// dlopen, pwrite and unlink. There's no "in-memory" only method using
	// dyld4 APIs, so there's no point in using NSCreateObjectFileImageFromMemory and NSLinkModule anymore.
	// We can just write our own version that:
	// - writes the binary image to a temporary file
	// - calls dlopen on the temporary file
	// - unlinks the temporary file
	// - uses dlsym to resolve all the symbols
	// See https://github.com/apple-oss-distributions/dyld/blob/main/dyld/DyldAPIs.cpp#L2785-L2847 for more details.
	if isSierra() {
		return LoadLibraryDlopen(name, image)
	}
	// force input image to be a bundle
	// this is not the most efficient method, but it does create the abililty to do more fixups down the road,
	// like if we added universal file support
	machoFile, err := macho.NewFile(bytes.NewReader(*image))
	if err != nil {
		return nil, err
	}
	machoFile.FileHeader.Type = macho.TypeBundle
	newImage, err := machoFile.Bytes()
	if err != nil {
		return nil, err
	}
	image = &newImage

	// Find dyld, loop through all the Mach-O images in memory until we find
	// one that exports NSCreateObjectFileImageFromMemory and NSLinkModule
	var execBase uintptr = uintptr(0x0000000001000000)
	ptr := execBase
	var createObjectAddr, linkModuleAddr uint64

	for i := 0; i < 10; i++ { // quit after the first ten images
		ptr, err = findMacho(ptr, 0x1000)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Found image at 0x%08x\n", ptr)
		rawr := rawreader.New(ptr, MaxInt)
		machoFile, err := macho.NewFileFromMemory(rawr)
		if err != nil {
			return nil, err
		}
		exports := machoFile.Exports()

		foundCreateObj := false
		foundLinkModule := false
		for _, export := range exports {
			if export.Name == "_NSCreateObjectFileImageFromMemory" {
				fmt.Println("found NSCreateObjectFileImageFromMemory")
				createObjectAddr = export.VirtualAddress
				foundCreateObj = true
			}
			if export.Name == "_NSLinkModule" {
				fmt.Println("found NSLinkModule")
				linkModuleAddr = export.VirtualAddress
				foundLinkModule = true
			}
		}
		if foundCreateObj && foundLinkModule {
			fmt.Println("Found dyld!")
			break
		}
		ptr = ptr + 0x1000
	}

	createObjectProc := ptr + uintptr(createObjectAddr)
	linkModuleProc := ptr + uintptr(linkModuleAddr)

	imageLen := uint64(len(*image))
	var NSObjectFileImage uintptr
	_, err = cdecl.Call(createObjectProc,
		uintptr(unsafe.Pointer(&(*image)[0])),       // pointer to image (*char)
		uintptr(unsafe.Pointer(&imageLen)),          // length of image (size_t)
		uintptr(unsafe.Pointer(&NSObjectFileImage))) // output pointer to struct
	if err != nil {
		return nil, err
	}
	options := uint64(3) // NSLINKMODULE_OPTION_PRIVATE | NSLINKMODULE_OPTION_BINDNOW
	modPtr, err := cdecl.Call(linkModuleProc,
		uintptr(unsafe.Pointer(NSObjectFileImage)), // file image struct from NSCreateObjectFileImageFromMemory
		uintptr(unsafe.Pointer(&name)),             // image name
		uintptr(unsafe.Pointer(&options)))          // image options flags
	if err != nil {
		return nil, err
	}

	libPtr, err := findMachoPtr(modPtr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Printf("Found image at 0x%08x\n", libPtr)

	// parse the Mach-o we just loaded to get the exports
	rawr := rawreader.New(libPtr, MaxInt)
	machoFile, err = macho.NewFileFromMemory(rawr)
	if err != nil {
		return nil, err
	}
	exports := machoFile.Exports()
	if err != nil {
		return nil, err
	}
	lib := Library{
		BaseAddress: libPtr,
		Exports:     make(map[string]uint64),
	}
	for _, x := range exports {
		lib.Exports[x.Name] = uint64(x.VirtualAddress)
	}
	return &lib, nil
}

// LoadLibraryDlopen - loads a single library using dlopen and resolves
// all exported symbols using dlsym. This function writes to a temporary
// file and then loads the file into memory, and unlinks it.
// This is similar to what NSLinkModule does nowadays.
func LoadLibraryDlopen(name string, image *[]byte) (*Library, error) {
	var (
		dlopenAddr  uint64
		dlSymAddr   uint64
		dyldAddress uint64
	)

	machoFile, err := macho.NewFile(bytes.NewReader(*image))
	if err != nil {
		return nil, err
	}

	// We use the shared library cache to find the libdyld.dylib wrapper loaded into our process.
	// Once we have the address of the libdyld.dylib wrapper, we can parse it in memory using debug/macho (thanks Awgh)
	// and find the exports we're interested in: dlopen and dlsym.
	// This part of the code is a port from the awesome work of usiegl00: https://github.com/usiegl00/metasploit-framework/blob/osx-stage-monterey/external/source/shellcode/osx/stager/main.c#L159-L183
	sharedRegionStart, err := syscallSharedRegionCheckNp()
	if err != nil {
		return nil, err
	}
	dyldCacheheader := (*DyldCacheHeader)(unsafe.Pointer(uintptr(sharedRegionStart)))

	imagesCount := dyldCacheheader.ImagesCountOld
	if imagesCount == 0 {
		imagesCount = dyldCacheheader.ImagesCount
	}
	imagesOffset := dyldCacheheader.ImagesOffsetOld
	if imagesOffset == 0 {
		imagesOffset = dyldCacheheader.ImagesOffset
	}
	sharedFileMapping := (*SharedFileMapping)(unsafe.Pointer(uintptr(sharedRegionStart + uint64(dyldCacheheader.MappingOffset))))

	dyldCacheImageInfo := (*DyldCacheImageInfo)(unsafe.Pointer(uintptr(sharedRegionStart + uint64(imagesOffset))))
	for i := uint32(0); i < imagesCount; i++ {
		pathFileOffset := sharedRegionStart + uint64(dyldCacheImageInfo.PathFileOffset)
		dylibPath := readCString(pathFileOffset)
		if dylibPath == LIB_DYLD_PATH {
			dyldAddress = dyldCacheImageInfo.Address
			break
		}
		dyldCacheImageInfo = (*DyldCacheImageInfo)(unsafe.Pointer(uintptr(sharedRegionStart+uint64(imagesOffset)) + uintptr(i)*unsafe.Sizeof(DyldCacheImageInfo{})))
	}

	offset := sharedRegionStart - sharedFileMapping.Address
	dyldAddress += offset

	rawr := rawreader.New(uintptr(dyldAddress), MaxInt)
	dyldFile, err := macho.NewFileFromMemory(rawr)
	if err != nil {
		return nil, err
	}
	exports := dyldFile.Exports()

	for _, export := range exports {
		if export.Name == "_dlopen" {
			dlopenAddr = export.VirtualAddress + offset
		}
		if export.Name == "_dlsym" {
			dlSymAddr = export.VirtualAddress + offset
		}
	}
	dlopenProc := uintptr(dlopenAddr)
	dlSymProc := uintptr(dlSymAddr)

	options := 0x80000080 // RTLD_UNLOADABLE | RTLD_NODELETE per https://github.dev/apple-oss-distributions/dyld/blob/419f8cbca6fb3420a248f158714a9d322af2aa5a/dyld/DyldAPIs.cpp#L2818-L2819

	// Create a temp file to write the image to so we can dlopen it.
	// I wish MacOS had a memfd_create equivalent ...
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}

	filename := tmpFile.Name()
	imagePath := append([]byte(filename), 0x00) // C strings are the worst

	_, err = tmpFile.Write(*image)
	if err != nil {
		return nil, err
	}

	// We need to sign the written file otherwise dlopen will fail on MacOS 11 and up on Apple Silicon machines.
	// We're just using ad-hoc code signing, so we're not going to do anything fancy here.
	err = machoCodeSign(filename)
	if err != nil {
		return nil, err
	}

	pseudoHandle, err := cdecl.Call(dlopenProc,
		uintptr(unsafe.Pointer(&imagePath[0])), // image name
		uintptr(options))                       // image options flags
	if err != nil {
		return nil, err
	}

	if pseudoHandle == 0 {
		unix.Unlink(filename)
		return nil, errors.New("dlopen returned 0")
	}

	// Unlink the temp file, we don't need it anymore
	err = unix.Unlink(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	exports = machoFile.Exports()
	if err != nil {
		return nil, err
	}

	lib := Library{
		BaseAddress: pseudoHandle, // library pseudo handle, do not use directly
		Exports:     make(map[string]uint64),
	}

	// Since the return value from dlopen is a pseudo handle
	// and not the address of the dylib in memory, we can't
	// really use it to resolve symbols directly. Instead, we
	// use dlsym to resolve all symbols ahead of time, that way
	// the end user can just use Library.Call.
	// The downside is that Library.BaseAddress is unusable and
	// any attempt to use it will result in a crash.
	for _, x := range exports {
		exportName := append([]byte(x.Name[1:]), 0x00) // strip the _
		exportPtr, err := cdecl.Call(dlSymProc,
			pseudoHandle,
			uintptr(unsafe.Pointer(&exportName[0])))
		if err != nil {
			return nil, err
		}
		lib.Exports[x.Name] = uint64(exportPtr)
	}
	return &lib, nil
}

func readCString(pathFileOffset uint64) string {
	path := ""
	pathSize := 0
	for {
		b := (*byte)(unsafe.Pointer(uintptr(pathFileOffset + uint64(pathSize))))
		if *b == 0 {
			break
		}
		path += string(*b)
		pathSize++
	}
	return path
}

func isPtrValid(p uintptr) bool {
	r1, _, err := unix.RawSyscall(unix.SYS_CHMOD, p, 0777, 0)
	if r1 != 0 && err != 2 {
		return false
	}
	return err == 2
}

func findMacho(addr uintptr, increment uint32) (uintptr, error) {
	var (
		base   uintptr
		header uint32
		err    error = fmt.Errorf("could not find macho")
	)
	ptr := addr
	for {
		if isPtrValid(ptr) {
			header = *(*uint32)(unsafe.Pointer(ptr))
			if header == macho.Magic64 {
				base = ptr
				err = nil
				break
			}
		}
		ptr += uintptr(increment)
	}
	return base, err
}

func findMachoPtr(addr uintptr) (uintptr, error) {
	var (
		base   uintptr
		header uint32
		err    error = fmt.Errorf("could not find macho")
	)
	idx := addr
	ptr := uintptr(*(*uint64)(unsafe.Pointer(idx)))
	for {
		if isPtrValid(ptr) {
			header = *(*uint32)(unsafe.Pointer(ptr))
			if header == macho.Magic64 {
				base = ptr
				err = nil
				break
			}
		}
		idx += 8
		ptr = uintptr(*(*uint64)(unsafe.Pointer(idx)))
	}
	return base, err
}

// Call - call a function in a loaded library
func (l *Library) Call(functionName string, args ...uintptr) (uintptr, error) {
	var (
		proc    uintptr
		funcPtr uint64
		ok      bool
	)
	if len(functionName) > 0 && functionName[0] != '_' {
		functionName = "_" + functionName
	} // OSX has underscore-prefixed exports

	if !isSierra() {
		proc, ok = l.FindProc(functionName)
	} else {
		// Can't rely on FindProc for Sierra and up since
		// the BaseAddress is a pseudo handle to the lib,
		// not the actual address.
		funcPtr, ok = l.Exports[functionName]
		proc = uintptr(funcPtr)
	}
	if !ok {
		return 0, errors.New("Call did not find export " + functionName)
	}
	val, err := cdecl.Call(proc, args...)
	return val, err
}
