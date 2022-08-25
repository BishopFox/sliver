package universal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"strconv"
	"syscall"

	"unsafe"

	"github.com/Binject/debug/pe"
)

const (
	MEM_COMMIT                             = 0x001000
	MEM_RESERVE                            = 0x002000
	GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS = 0x00000004
	INFINITE                               = 0xFFFFFFFF
)

// LoadLibraryImpl - loads a single library to memory, without trying to check or load required imports
func LoadLibraryImpl(name string, image *[]byte) (*Library, error) {
	const PtrSize = 32 << uintptr(^uintptr(0)>>63) // are we on a 32bit or 64bit system?
	pelib, err := pe.NewFile(bytes.NewReader(*image))
	if err != nil {
		return nil, err
	}
	pe64 := pelib.Machine == pe.IMAGE_FILE_MACHINE_AMD64
	if pe64 && PtrSize != 64 {
		return nil, errors.New("Cannot load a 64bit DLL from a 32bit process")
	} else if !pe64 && PtrSize != 32 {
		return nil, errors.New("Cannot load a 32bit DLL from a 64bit process")
	}

	var sizeOfImage uint32
	if pe64 {
		sizeOfImage = pelib.OptionalHeader.(*pe.OptionalHeader64).SizeOfImage
	} else {
		sizeOfImage = pelib.OptionalHeader.(*pe.OptionalHeader32).SizeOfImage
	}
	r, err := virtualAlloc(0, sizeOfImage, MEM_RESERVE, syscall.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	dst, err := virtualAlloc(r, sizeOfImage, MEM_COMMIT, syscall.PAGE_EXECUTE_READWRITE)
	if err != nil {
		return nil, err
	}

	//perform base relocations
	pelib.Relocate(uint64(dst), image)

	//write to memory
	CopySections(pelib, image, dst)

	exports, err := pelib.Exports()
	if err != nil {
		return nil, err
	}
	lib := Library{
		BaseAddress: dst,
		Exports:     make(map[string]uint64),
	}
	for _, x := range exports {
		lib.Exports[x.Name] = uint64(x.VirtualAddress)
	}

	return &lib, nil
}

// CopySections - writes the sections of a PE image to the given base address in memory
func CopySections(pefile *pe.File, image *[]byte, loc uintptr) error {
	// Copy Headers
	var sizeOfHeaders uint32
	if pefile.Machine == pe.IMAGE_FILE_MACHINE_AMD64 {
		sizeOfHeaders = pefile.OptionalHeader.(*pe.OptionalHeader64).SizeOfHeaders
	} else {
		sizeOfHeaders = pefile.OptionalHeader.(*pe.OptionalHeader32).SizeOfHeaders
	}
	hbuf := (*[^uint32(0)]byte)(unsafe.Pointer(uintptr(loc)))
	for index := uint32(0); index < sizeOfHeaders; index++ {
		hbuf[index] = (*image)[index]
	}

	// Copy Sections
	for _, section := range pefile.Sections {
		//fmt.Println("Writing:", fmt.Sprintf("%s %x %x", section.Name, loc, uint32(loc)+section.VirtualAddress))
		if section.Size == 0 {
			continue
		}
		d, err := section.Data()
		if err != nil {
			return err
		}
		dataLen := uint32(len(d))
		dst := uint64(loc) + uint64(section.VirtualAddress)
		buf := (*[^uint32(0)]byte)(unsafe.Pointer(uintptr(dst)))
		for index := uint32(0); index < dataLen; index++ {
			buf[index] = d[index]
		}
	}

	// Write symbol and string tables
	bbuf := bytes.NewBuffer(nil)
	binary.Write(bbuf, binary.LittleEndian, pefile.COFFSymbols)
	binary.Write(bbuf, binary.LittleEndian, pefile.StringTable)
	b := bbuf.Bytes()
	blen := uint32(len(b))
	for index := uint32(0); index < blen; index++ {
		hbuf[index+pefile.FileHeader.PointerToSymbolTable] = b[index]
	}

	return nil
}

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procVirtualAlloc = kernel32.MustFindProc("VirtualAlloc")
)

func virtualAlloc(addr uintptr, size, allocType, protect uint32) (uintptr, error) {
	r1, _, e1 := procVirtualAlloc.Call(
		addr,
		uintptr(size),
		uintptr(allocType),
		uintptr(protect))

	if int(r1) == 0 {
		return r1, os.NewSyscallError("VirtualAlloc", e1)
	}
	return r1, nil
}

// Call - call a function in a loaded library
//
// cribbed from: https://golang.org/src/Syscall/dll_windows.go
//
// On amd64, Call can pass and return floating-point values. To pass
// an argument x with C type "float", use
// uintptr(math.Float32bits(x)). To pass an argument with C type
// "double", use uintptr(math.Float64bits(x)). Floating-point return
// values are returned in r2. The return value for C type "float" is
// math.Float32frombits(uint32(r2)). For C type "double", it is
// math.Float64frombits(uint64(r2)).
func (l *Library) Call(functionName string, args ...uintptr) (uintptr, error) {

	proc, ok := l.FindProc(functionName)
	if !ok {
		return 0, errors.New("Call did not find export " + functionName)
	}
	var r uintptr
	var errno syscall.Errno
	var err error
	switch len(args) {
	case 0:
		r, _, errno = syscall.Syscall(proc, uintptr(len(args)), 0, 0, 0)
	case 1:
		r, _, errno = syscall.Syscall(proc, uintptr(len(args)), args[0], 0, 0)
	case 2:
		r, _, errno = syscall.Syscall(proc, uintptr(len(args)), args[0], args[1], 0)
	case 3:
		r, _, errno = syscall.Syscall(proc, uintptr(len(args)), args[0], args[1], args[2])
	case 4:
		r, _, errno = syscall.Syscall6(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], 0, 0)
	case 5:
		r, _, errno = syscall.Syscall6(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], 0)
	case 6:
		r, _, errno = syscall.Syscall6(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5])
	case 7:
		r, _, errno = syscall.Syscall9(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], 0, 0)
	case 8:
		r, _, errno = syscall.Syscall9(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], 0)
	case 9:
		r, _, errno = syscall.Syscall9(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8])
	case 10:
		r, _, errno = syscall.Syscall12(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], 0, 0)
	case 11:
		r, _, errno = syscall.Syscall12(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], 0)
	case 12:
		r, _, errno = syscall.Syscall12(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11])
	case 13:
		r, _, errno = syscall.Syscall15(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], 0, 0)
	case 14:
		r, _, errno = syscall.Syscall15(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], 0)
	case 15:
		r, _, errno = syscall.Syscall15(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14])
	case 16:
		r, _, errno = syscall.Syscall18(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14], args[15], 0, 0)
	case 17:
		r, _, errno = syscall.Syscall18(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14], args[15], args[16], 0)
	case 18:
		r, _, errno = syscall.Syscall18(proc, uintptr(len(args)), args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14], args[15], args[16], args[17])
	default:
		return 0, errors.New("Call " + functionName + " with too many arguments " + strconv.Itoa(len(args)) + ".")
	}
	if errno != 0 {
		errString := errno.Error()
		if errString != "" {
			err = errors.New(errString)
		}
	}
	return r, err
}
