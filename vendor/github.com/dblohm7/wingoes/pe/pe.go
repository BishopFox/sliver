// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build windows

// Package pe provides facilities for extracting information from PE binaries.
package pe

import (
	"bufio"
	"bytes"
	dpe "debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"os"
	"reflect"
	"strings"
	"unsafe"

	"golang.org/x/exp/constraints"
	"golang.org/x/sys/windows"
)

// The following constants are from the PE spec
const (
	offsetIMAGE_DOS_HEADERe_lfanew = 0x3C
	maxNumSections                 = 96
)

var (
	ErrBadLength       = errors.New("effective length did not match expected length")
	ErrBadCodeView     = errors.New("invalid CodeView debug info")
	ErrNotCodeView     = errors.New("debug info is not CodeView")
	ErrNotPresent      = errors.New("not present in this PE image")
	ErrIndexOutOfRange = errors.New("index out of range")
	// ErrInvalidBinary is returned whenever the headers do not parse as expected,
	// or reference locations outside the bounds of the PE file or module.
	// The headers might be corrupt, malicious, or have been tampered with.
	ErrInvalidBinary       = errors.New("invalid PE binary")
	ErrResolvingFileRVA    = errors.New("could not resolve file RVA")
	ErrUnavailableInModule = errors.New("this information is unavailable from loaded modules; the PE file itself must be examined")
	ErrUnsupportedMachine  = errors.New("unsupported machine")
)

type peReader interface {
	Base() uintptr
	io.Closer
	io.ReaderAt
	io.ReadSeeker
	Limit() uintptr
}

// PEHeaders represents the partially-parsed headers from a PE binary.
type PEHeaders struct {
	r              peReader
	fileHeader     *dpe.FileHeader
	optionalHeader *optionalHeader
	sections       []peSectionHeader
}

type peBounds struct {
	base  uintptr
	limit uintptr
}

type peFile struct {
	*os.File
	peBounds
}

func (pef *peFile) Base() uintptr {
	return pef.peBounds.base
}

func (pef *peFile) Limit() uintptr {
	if pef.limit == 0 {
		if fi, err := pef.Stat(); err == nil {
			pef.limit = uintptr(fi.Size())
		}
	}
	return pef.limit
}

type peModule struct {
	*bytes.Reader
	peBounds
	modLock windows.Handle
}

func (pei *peModule) Base() uintptr {
	return pei.peBounds.base
}

func (pei *peModule) Close() error {
	return windows.FreeLibrary(pei.modLock)
}

func (pei *peModule) Limit() uintptr {
	return pei.peBounds.limit
}

// NewPEFromBaseAddressAndSize parses the headers in a PE binary loaded
// into the current process's address space at address baseAddr with known
// size. If you do not have the size, use NewPEFromBaseAddress instead.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromBaseAddressAndSize(baseAddr uintptr, size uint32) (*PEHeaders, error) {
	// Grab a strong reference to the module until we're done with it.
	var modLock windows.Handle
	if err := windows.GetModuleHandleEx(
		windows.GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
		(*uint16)(unsafe.Pointer(baseAddr)),
		&modLock,
	); err != nil {
		return nil, err
	}

	slc := unsafe.Slice((*byte)(unsafe.Pointer(baseAddr)), size)
	r := bytes.NewReader(slc)
	peMod := &peModule{
		Reader: r,
		peBounds: peBounds{
			base:  baseAddr,
			limit: baseAddr + uintptr(size),
		},
		modLock: modLock,
	}
	return loadHeaders(peMod)
}

// NewPEFromBaseAddress parses the headers in a PE binary loaded into the
// current process's address space at address baseAddr.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromBaseAddress(baseAddr uintptr) (*PEHeaders, error) {
	var modInfo windows.ModuleInfo
	if err := windows.GetModuleInformation(
		windows.CurrentProcess(),
		windows.Handle(baseAddr),
		&modInfo,
		uint32(unsafe.Sizeof(modInfo)),
	); err != nil {
		return nil, fmt.Errorf("querying module handle: %w", err)
	}

	return NewPEFromBaseAddressAndSize(baseAddr, modInfo.SizeOfImage)
}

// NewPEFromHMODULE parses the headers in a PE binary identified by hmodule that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromHMODULE(hmodule windows.Handle) (*PEHeaders, error) {
	// HMODULEs are just a loaded module's base address with the lowest two
	// bits used for flags (see docs for LoadLibraryExW).
	return NewPEFromBaseAddress(uintptr(hmodule) & ^uintptr(3))
}

// NewPEFromDLL parses the headers in a PE binary identified by dll that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
// If the DLL is Release()d while the returned *PEHeaders is still in use,
// its behaviour will become undefined.
func NewPEFromDLL(dll *windows.DLL) (*PEHeaders, error) {
	if dll == nil || dll.Handle == 0 {
		return nil, os.ErrInvalid
	}

	return NewPEFromHMODULE(dll.Handle)
}

// NewPEFromLazyDLL parses the headers in a PE binary identified by ldll that
// is currently loaded into the current process's address space.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a nil
// *PEHeaders and a non-nil error.
func NewPEFromLazyDLL(ldll *windows.LazyDLL) (*PEHeaders, error) {
	if ldll == nil {
		return nil, os.ErrInvalid
	}
	if err := ldll.Load(); err != nil {
		return nil, err
	}

	return NewPEFromHMODULE(windows.Handle(ldll.Handle()))
}

// NewPEFromFileName opens a PE binary located at filename and parses its PE
// headers. Upon success it returns a non-nil *PEHeaders, otherwise it returns a
// nil *PEHeaders and a non-nil error.
// Call Close() on the returned *PEHeaders when it is no longer needed.
func NewPEFromFileName(filename string) (*PEHeaders, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return newPEFromFile(f)
}

// NewPEFromFileHandle parses the PE headers from hfile, an open Win32 file handle.
// It does *not* consume hfile.
// Upon success it returns a non-nil *PEHeaders, otherwise it returns a
// nil *PEHeaders and a non-nil error.
// Call Close() on the returned *PEHeaders when it is no longer needed.
func NewPEFromFileHandle(hfile windows.Handle) (*PEHeaders, error) {
	if hfile == 0 || hfile == windows.InvalidHandle {
		return nil, os.ErrInvalid
	}

	// Duplicate hfile so that we don't consume it.
	var hfileDup windows.Handle
	cp := windows.CurrentProcess()
	if err := windows.DuplicateHandle(
		cp,
		hfile,
		cp,
		&hfileDup,
		0,
		false,
		windows.DUPLICATE_SAME_ACCESS,
	); err != nil {
		return nil, err
	}

	return newPEFromFile(os.NewFile(uintptr(hfileDup), "PEFromFileHandle"))
}

func newPEFromFile(f *os.File) (*PEHeaders, error) {
	// peBounds base is 0, limit is loaded lazily
	pef := &peFile{File: f}
	return loadHeaders(pef)
}

func (peh *PEHeaders) Close() error {
	return peh.r.Close()
}

type rvaType interface {
	~int8 | ~int16 | ~int32 | ~uint8 | ~uint16 | ~uint32
}

// addOffset ensures that, if off is negative, it does not underflow base.
func addOffset[O rvaType](base uintptr, off O) uintptr {
	if off >= 0 {
		return base + uintptr(off)
	}

	negation := uintptr(-off)
	if negation >= base {
		return 0
	}
	return base - negation
}

func binaryRead(r io.Reader, data any) (err error) {
	// Windows is always LittleEndian
	err = binary.Read(r, binary.LittleEndian, data)
	if err == io.ErrUnexpectedEOF {
		err = ErrBadLength
	}
	return err
}

// readStruct reads a T from offset rva. If r is a *peModule, the returned *T
// points to the data in-place.
// Note that currently this function will fail if rva references memory beyond
// the bounds of the binary; in the case of modules, this may need to be relaxed
// in some cases due to tampering by third-party crapware.
func readStruct[T any, O rvaType](r peReader, rva O) (*T, error) {
	switch v := r.(type) {
	case *peFile:
		if _, err := r.Seek(int64(rva), io.SeekStart); err != nil {
			return nil, err
		}

		result := new(T)
		if err := binaryRead(r, result); err != nil {
			return nil, err
		}

		return result, nil
	case *peModule:
		addr := addOffset(r.Base(), rva)
		szT := unsafe.Sizeof(*((*T)(nil)))
		if addr+szT >= v.Limit() {
			return nil, ErrInvalidBinary
		}

		return (*T)(unsafe.Pointer(addr)), nil
	default:
		return nil, os.ErrInvalid
	}
}

// readStructArray reads a []T with length count from offset rva. If r is a
// *peModule, the returned []T references the data in-place.
// Note that currently this function will fail if rva references memory beyond
// the bounds of the binary; in the case of modules, this may need to be relaxed
// in some cases due to tampering by third-party crapware.
func readStructArray[T any, O rvaType](r peReader, rva O, count int) ([]T, error) {
	switch v := r.(type) {
	case *peFile:
		if _, err := r.Seek(int64(rva), io.SeekStart); err != nil {
			return nil, err
		}

		result := make([]T, count)
		if err := binaryRead(r, result); err != nil {
			return nil, err
		}

		return result, nil
	case *peModule:
		addr := addOffset(r.Base(), rva)
		szT := reflect.ArrayOf(count, reflect.TypeOf((*T)(nil)).Elem()).Size()
		if addr+szT >= v.Limit() {
			return nil, ErrInvalidBinary
		}

		return unsafe.Slice((*T)(unsafe.Pointer(addr)), count), nil
	default:
		return nil, os.ErrInvalid
	}
}

type peSectionHeader dpe.SectionHeader32

func (s *peSectionHeader) NameAsString() string {
	// s.Name is UTF-8. When the string's length is < len(s.Name), the remaining
	// bytes are padded with zeros.
	for i, c := range s.Name {
		if c == 0 {
			return string(s.Name[:i])
		}
	}

	return string(s.Name[:])
}

func loadHeaders(r peReader) (*PEHeaders, error) {
	// Check the signature of the DOS stub header
	var mz [2]byte
	if _, err := r.ReadAt(mz[:], 0); err != nil {
		if err == io.EOF {
			err = ErrInvalidBinary
		}
		return nil, err
	}
	if mz[0] != 'M' || mz[1] != 'Z' {
		return nil, ErrInvalidBinary
	}

	// Seek to the offset of the value that points to the beginning of the PE headers
	if _, err := r.Seek(offsetIMAGE_DOS_HEADERe_lfanew, io.SeekStart); err != nil {
		return nil, err
	}

	// Load the offset to the beginning of the PE headers
	var e_lfanew int32
	if err := binaryRead(r, &e_lfanew); err != nil {
		if err == ErrBadLength {
			err = ErrInvalidBinary
		}
		return nil, err
	}
	if e_lfanew <= 0 {
		return nil, ErrInvalidBinary
	}
	if addOffset(r.Base(), e_lfanew) >= r.Limit() {
		return nil, ErrInvalidBinary
	}

	// Check the PE signature
	var peSig [4]byte
	if _, err := r.ReadAt(peSig[:], int64(e_lfanew)); err != nil {
		if err == io.EOF {
			err = ErrInvalidBinary
		}
		return nil, err
	}
	if peSig[0] != 'P' || peSig[1] != 'E' || peSig[2] != 0 || peSig[3] != 0 {
		return nil, ErrInvalidBinary
	}

	// Read the file header
	fileHeaderOffset := uint32(e_lfanew) + uint32(unsafe.Sizeof(peSig))
	if r.Base()+uintptr(fileHeaderOffset) >= r.Limit() {
		return nil, ErrInvalidBinary
	}

	fileHeader, err := readStruct[dpe.FileHeader](r, fileHeaderOffset)
	if err != nil {
		return nil, err
	}

	// In-memory modules should always have a machine type that matches our own.
	// (okay, so that's kinda sorta untrue with respect to WOW64, but that's
	// a _very_ obscure use case).
	_, isModule := r.(*peModule)
	// TODO(aaron): Uncomment once we can read binaries from disk whose archs
	// do not necessarily match our own.
	if /*isModule &&*/ fileHeader.Machine != expectedMachine {
		return nil, ErrUnsupportedMachine
	}

	// Read the optional header
	optionalHeaderOffset := uint32(fileHeaderOffset) + uint32(unsafe.Sizeof(*fileHeader))
	if r.Base()+uintptr(optionalHeaderOffset) >= r.Limit() {
		return nil, ErrInvalidBinary
	}

	// TODO(aaron): parameterize optional header type so we can read binaries
	// from disk whose archs do not necessarily match our own.
	optionalHeader, err := readStruct[optionalHeader](r, optionalHeaderOffset)
	if err != nil {
		return nil, err
	}

	// Check the optional header's Magic field
	expectedOptionalHeaderMagic := uint16(optionalHeaderMagic)
	if !isModule {
		switch fileHeader.Machine {
		case dpe.IMAGE_FILE_MACHINE_I386:
			expectedOptionalHeaderMagic = 0x010B
		case dpe.IMAGE_FILE_MACHINE_AMD64, dpe.IMAGE_FILE_MACHINE_ARM64:
			expectedOptionalHeaderMagic = 0x020B
		default:
			return nil, ErrUnsupportedMachine
		}
	}

	if optionalHeader.Magic != expectedOptionalHeaderMagic {
		return nil, ErrInvalidBinary
	}

	// Coarse-grained check that header sizes make sense
	totalEssentialHeaderLen := uint32(offsetIMAGE_DOS_HEADERe_lfanew) +
		uint32(unsafe.Sizeof(e_lfanew)) +
		uint32(unsafe.Sizeof(*fileHeader)) +
		uint32(fileHeader.SizeOfOptionalHeader)
	if optionalHeader.SizeOfImage < totalEssentialHeaderLen {
		return nil, ErrInvalidBinary
	}

	numSections := fileHeader.NumberOfSections
	if numSections > maxNumSections {
		// More than 96 sections?! Really?!
		return nil, ErrInvalidBinary
	}

	// Read in the section table
	sectionTableOffset := optionalHeaderOffset + uint32(fileHeader.SizeOfOptionalHeader)
	if r.Base()+uintptr(sectionTableOffset) >= r.Limit() {
		return nil, ErrInvalidBinary
	}

	sections, err := readStructArray[peSectionHeader](r, sectionTableOffset, int(numSections))
	if err != nil {
		return nil, err
	}

	return &PEHeaders{r: r, fileHeader: fileHeader, optionalHeader: optionalHeader, sections: sections}, nil
}

type rva32 interface {
	~int32 | ~uint32
}

// resolveRVA resolves rva, or returns 0 if unavailable.
func resolveRVA[O rva32](nfo *PEHeaders, rva O) O {
	if _, ok := nfo.r.(*peFile); !ok {
		// Just the identity function in this case.
		return rva
	}

	if rva <= 0 {
		return 0
	}

	// We walk the section table, locating the section that would contain rva if
	// we were mapped into memory. We then calculate the offset of rva from the
	// starting virtual address of the section, and then add that offset to the
	// section's starting file pointer.
	urva := uint32(rva)
	for _, s := range nfo.sections {
		if urva < s.VirtualAddress {
			continue
		}
		if urva >= (s.VirtualAddress + s.VirtualSize) {
			continue
		}
		voff := urva - s.VirtualAddress
		foff := s.PointerToRawData + voff
		if foff >= s.PointerToRawData+s.SizeOfRawData {
			return 0
		}
		return O(foff)
	}

	return 0
}

type DataDirectoryEntry = dpe.DataDirectory

func (nfo *PEHeaders) dataDirectory() []DataDirectoryEntry {
	cnt := nfo.optionalHeader.NumberOfRvaAndSizes
	if maxCnt := uint32(len(nfo.optionalHeader.DataDirectory)); cnt > maxCnt {
		cnt = maxCnt
	}
	return nfo.optionalHeader.DataDirectory[:cnt]
}

// DataDirectoryIndex is an enumeration specifying a particular entry in the
// data directory.
type DataDirectoryIndex int

const (
	IMAGE_DIRECTORY_ENTRY_EXPORT         = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_EXPORT)
	IMAGE_DIRECTORY_ENTRY_IMPORT         = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_IMPORT)
	IMAGE_DIRECTORY_ENTRY_RESOURCE       = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_RESOURCE)
	IMAGE_DIRECTORY_ENTRY_EXCEPTION      = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_EXCEPTION)
	IMAGE_DIRECTORY_ENTRY_SECURITY       = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_SECURITY)
	IMAGE_DIRECTORY_ENTRY_BASERELOC      = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_BASERELOC)
	IMAGE_DIRECTORY_ENTRY_DEBUG          = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_DEBUG)
	IMAGE_DIRECTORY_ENTRY_ARCHITECTURE   = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_ARCHITECTURE)
	IMAGE_DIRECTORY_ENTRY_GLOBALPTR      = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_GLOBALPTR)
	IMAGE_DIRECTORY_ENTRY_TLS            = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_TLS)
	IMAGE_DIRECTORY_ENTRY_LOAD_CONFIG    = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_LOAD_CONFIG)
	IMAGE_DIRECTORY_ENTRY_BOUND_IMPORT   = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_BOUND_IMPORT)
	IMAGE_DIRECTORY_ENTRY_IAT            = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_IAT)
	IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT   = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT)
	IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR = DataDirectoryIndex(dpe.IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR)
)

// DataDirectoryEntry returns information from nfo's data directory at index idx.
// The type of the return value depends on the value of idx. Most values for idx
// currently return the DataDirectoryEntry itself, however it will return more
// sophisticated information for the following values of idx:
//
// IMAGE_DIRECTORY_ENTRY_SECURITY returns []AuthenticodeCert
// IMAGE_DIRECTORY_ENTRY_DEBUG returns []IMAGE_DEBUG_DIRECTORY
//
// Note that other idx values _will_ be modified in the future to support more
// sophisticated return values, so be careful to structure your type assertions
// accordingly.
func (nfo *PEHeaders) DataDirectoryEntry(idx DataDirectoryIndex) (any, error) {
	dd := nfo.dataDirectory()
	if int(idx) >= len(dd) {
		return nil, ErrIndexOutOfRange
	}

	dde := dd[idx]
	if dde.VirtualAddress == 0 || dde.Size == 0 {
		return nil, ErrNotPresent
	}

	switch idx {
	/* TODO(aaron): (don't forget to sync tests!)
	case IMAGE_DIRECTORY_ENTRY_EXPORT:
	case IMAGE_DIRECTORY_ENTRY_IMPORT:
	case IMAGE_DIRECTORY_ENTRY_RESOURCE:
	*/
	case IMAGE_DIRECTORY_ENTRY_SECURITY:
		return nfo.extractAuthenticode(dde)
	case IMAGE_DIRECTORY_ENTRY_DEBUG:
		return nfo.extractDebugInfo(dde)
	// case IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR:
	default:
		return dde, nil
	}
}

// WIN_CERT_REVISION is an enumeration from the Windows SDK.
type WIN_CERT_REVISION uint16

const (
	WIN_CERT_REVISION_1_0 WIN_CERT_REVISION = 0x0100
	WIN_CERT_REVISION_2_0 WIN_CERT_REVISION = 0x0200
)

// WIN_CERT_TYPE is an enumeration from the Windows SDK.
type WIN_CERT_TYPE uint16

const (
	WIN_CERT_TYPE_X509             WIN_CERT_TYPE = 0x0001
	WIN_CERT_TYPE_PKCS_SIGNED_DATA WIN_CERT_TYPE = 0x0002
	WIN_CERT_TYPE_TS_STACK_SIGNED  WIN_CERT_TYPE = 0x0004
)

type _WIN_CERTIFICATE_HEADER struct {
	Length          uint32
	Revision        WIN_CERT_REVISION
	CertificateType WIN_CERT_TYPE
}

// AuthenticodeCert represents an authenticode signature that has been extracted
// from a signed PE binary but not fully parsed.
type AuthenticodeCert struct {
	header _WIN_CERTIFICATE_HEADER
	data   []byte
}

// Revision returns the revision of ac.
func (ac *AuthenticodeCert) Revision() WIN_CERT_REVISION {
	return ac.header.Revision
}

// Type returns the type of ac.
func (ac *AuthenticodeCert) Type() WIN_CERT_TYPE {
	return ac.header.CertificateType
}

// Data returns the raw bytes of ac's cert.
func (ac *AuthenticodeCert) Data() []byte {
	return ac.data
}

func alignUp[V constraints.Integer](v V, powerOfTwo uint8) V {
	if bits.OnesCount8(powerOfTwo) != 1 {
		panic("invalid powerOfTwo argument to alignUp")
	}
	return v + ((-v) & (V(powerOfTwo) - 1))
}

// IMAGE_DEBUG_DIRECTORY describes debug information embedded in the binary.
type IMAGE_DEBUG_DIRECTORY struct {
	Characteristics  uint32
	TimeDateStamp    uint32
	MajorVersion     uint16
	MinorVersion     uint16
	Type             uint32 // an IMAGE_DEBUG_TYPE constant
	SizeOfData       uint32
	AddressOfRawData uint32
	PointerToRawData uint32
}

// IMAGE_DEBUG_TYPE_CODEVIEW identifies the current IMAGE_DEBUG_DIRECTORY as
// pointing to CodeView debug information.
const IMAGE_DEBUG_TYPE_CODEVIEW = 2

// IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED contains CodeView debug information
// embedded in the PE file. Note that this structure's ABI does not match its C
// counterpart because the former uses a Go string and the latter is packed and
// also includes a signature field.
type IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED struct {
	GUID    windows.GUID
	Age     uint32
	PDBPath string
}

// String returns the data from u formatted in the same way that Microsoft
// debugging tools and symbol servers use to identify PDB files corresponding
// to a specific binary.
func (u *IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%08X%04X%04X", u.GUID.Data1, u.GUID.Data2, u.GUID.Data3)
	for _, v := range u.GUID.Data4 {
		fmt.Fprintf(&b, "%02X", v)
	}
	fmt.Fprintf(&b, "%X", u.Age)
	return b.String()
}

const codeViewSignature = 0x53445352

func (u *IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED) unpack(r *bufio.Reader) error {
	var signature uint32
	if err := binaryRead(r, &signature); err != nil {
		return err
	}
	if signature != codeViewSignature {
		return ErrBadCodeView
	}

	if err := binaryRead(r, &u.GUID); err != nil {
		return err
	}

	if err := binaryRead(r, &u.Age); err != nil {
		return err
	}

	pdbBytes := make([]byte, 0, 16)
	for b, err := r.ReadByte(); err == nil && b != 0; b, err = r.ReadByte() {
		pdbBytes = append(pdbBytes, b)
	}

	u.PDBPath = string(pdbBytes)
	return nil
}

func (nfo *PEHeaders) extractDebugInfo(dde DataDirectoryEntry) (any, error) {
	rva := resolveRVA(nfo, dde.VirtualAddress)
	if rva == 0 {
		return nil, ErrResolvingFileRVA
	}

	count := dde.Size / uint32(unsafe.Sizeof(IMAGE_DEBUG_DIRECTORY{}))
	return readStructArray[IMAGE_DEBUG_DIRECTORY](nfo.r, rva, int(count))
}

// ExtractCodeViewInfo obtains CodeView debug information from de, assuming that
// de represents CodeView debug info.
func (nfo *PEHeaders) ExtractCodeViewInfo(de IMAGE_DEBUG_DIRECTORY) (*IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED, error) {
	if de.Type != IMAGE_DEBUG_TYPE_CODEVIEW {
		return nil, ErrNotCodeView
	}

	cv := new(IMAGE_DEBUG_INFO_CODEVIEW_UNPACKED)
	var sr *io.SectionReader
	switch v := nfo.r.(type) {
	case *peFile:
		sr = io.NewSectionReader(v, int64(de.PointerToRawData), int64(de.SizeOfData))
	case *peModule:
		sr = io.NewSectionReader(v, int64(de.AddressOfRawData), int64(de.SizeOfData))
	default:
		return nil, ErrInvalidBinary
	}

	if err := cv.unpack(bufio.NewReader(sr)); err != nil {
		return nil, err
	}

	return cv, nil
}

func readFull(r io.Reader, buf []byte) (n int, err error) {
	n, err = io.ReadFull(r, buf)
	if err == io.ErrUnexpectedEOF {
		err = ErrBadLength
	}
	return n, err
}

func (nfo *PEHeaders) extractAuthenticode(dde DataDirectoryEntry) (any, error) {
	if _, ok := nfo.r.(*peFile); !ok {
		// Authenticode; only available in file, not loaded at runtime.
		return nil, ErrUnavailableInModule
	}

	var result []AuthenticodeCert
	// The VirtualAddress is a file offset.
	sr := io.NewSectionReader(nfo.r, int64(dde.VirtualAddress), int64(dde.Size))
	var curOffset int64
	szEntry := unsafe.Sizeof(_WIN_CERTIFICATE_HEADER{})

	for {
		var entry AuthenticodeCert
		if err := binaryRead(sr, &entry.header); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		curOffset += int64(szEntry)

		if uintptr(entry.header.Length) < szEntry {
			return nil, ErrInvalidBinary
		}

		entry.data = make([]byte, uintptr(entry.header.Length)-szEntry)
		n, err := readFull(sr, entry.data)
		if err != nil {
			return nil, err
		}
		curOffset += int64(n)

		result = append(result, entry)

		curOffset = alignUp(curOffset, 8)
		if _, err := sr.Seek(curOffset, io.SeekStart); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}

	return result, nil
}
