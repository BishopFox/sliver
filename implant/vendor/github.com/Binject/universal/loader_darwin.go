package universal

import (
	"bytes"
	"errors"
	"fmt"
	"unsafe"

	"github.com/Binject/debug/macho"
	"github.com/awgh/cppgo/asmcall/cdecl"
	"github.com/awgh/rawreader"
	"golang.org/x/sys/unix"
)

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)

// LoadLibraryImpl - loads a single library to memory, without trying to check or load required imports
func LoadLibraryImpl(name string, image *[]byte) (*Library, error) {

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

	if len(functionName) > 0 && functionName[0] != '_' {
		functionName = "_" + functionName
	} // OSX has underscore-prefixed exports

	proc, ok := l.FindProc(functionName)
	if !ok {
		return 0, errors.New("Call did not find export " + functionName)
	}
	val, err := cdecl.Call(proc, args...)
	return val, err
}
