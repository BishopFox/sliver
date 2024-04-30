// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pe implements access to PE (Microsoft Windows Portable Executable) files.
package pe

import (
	"bytes"
	"compress/zlib"
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// Avoid use of post-Go 1.4 io features, to make safe for toolchain bootstrap.
const (
	seekStart   = 0
	seekCurrent = 1
)

// A File represents an open PE file.
type File struct {
	DosHeader
	DosExists  bool
	DosStub    [64]byte // TODO(capnspacehook) make slice and correctly parse any DOS stub
	RichHeader []byte
	FileHeader
	OptionalHeader      interface{} // of type *OptionalHeader32 or *OptionalHeader64
	Sections            []*Section
	BaseRelocationTable *[]RelocationTableEntry
	Symbols             []*Symbol    // COFF symbols with auxiliary symbol records removed
	COFFSymbols         []COFFSymbol // all COFF symbols (including auxiliary symbol records)
	StringTable         StringTable
	CertificateTable    []byte

	OptionalHeaderOffset int64 // offset of the start of the Optional Header
	InsertionAddr        uint32
	InsertionBytes       []byte

	Net Net //If a managed executable, Net provides an interface to some of the metadata

	closer io.Closer
}

// Open opens the named file using os.Open and prepares it for use as a PE binary.
func Open(name string) (*File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	ff, err := NewFile(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	ff.closer = f
	return ff, nil
}

// Close closes the File.
// If the File was created using NewFile directly instead of Open,
// Close has no effect.
func (f *File) Close() error {
	var err error
	if f.closer != nil {
		err = f.closer.Close()
		f.closer = nil
	}
	return err
}

var (
	sizeofOptionalHeader32 = uint16(binary.Size(OptionalHeader32{}))
	sizeofOptionalHeader64 = uint16(binary.Size(OptionalHeader64{}))
)

// TODO(brainman): add Load function, as a replacement for NewFile, that does not call removeAuxSymbols (for performance)

// NewFile creates a new pe.File for accessing a PE binary file in an underlying reader.
func NewFile(r io.ReaderAt) (*File, error) {
	return newFileInternal(r, false)
}

// NewFileFromMemory creates a new pe.File for accessing a PE binary in-memory image in an underlying reader.
func NewFileFromMemory(r io.ReaderAt) (*File, error) {
	return newFileInternal(r, true)
}

// NewFile creates a new File for accessing a PE binary in an underlying reader.
func newFileInternal(r io.ReaderAt, memoryMode bool) (*File, error) {

	f := new(File)
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	binary.Read(sr, binary.LittleEndian, &f.DosHeader)
	dosHeaderSize := binary.Size(f.DosHeader)
	if dosHeaderSize < int(f.DosHeader.AddressOfNewExeHeader) {
		binary.Read(sr, binary.LittleEndian, &f.DosStub)
		f.DosExists = true
	} else {
		f.DosExists = false
	}

	possibleRichHeaderStart := dosHeaderSize
	if f.DosExists {
		possibleRichHeaderStart += binary.Size(f.DosStub)
	}
	possibleRichHeaderEnd := int(f.DosHeader.AddressOfNewExeHeader)
	if possibleRichHeaderEnd > possibleRichHeaderStart {
		richHeader := make([]byte, possibleRichHeaderEnd-possibleRichHeaderStart)
		binary.Read(sr, binary.LittleEndian, richHeader)

		if richIndex := bytes.Index(richHeader, []byte("Rich")); richIndex != -1 {
			f.RichHeader = richHeader[:richIndex+8]
		}
	}

	var peHeaderOffset int64
	if f.DosHeader.MZSignature == 0x5a4d {
		peHeaderOffset = int64(f.DosHeader.AddressOfNewExeHeader)
		var sign [4]byte
		r.ReadAt(sign[:], peHeaderOffset)
		if !(sign[0] == 'P' && sign[1] == 'E' && sign[2] == 0 && sign[3] == 0) {
			return nil, fmt.Errorf("Invalid PE COFF file signature of %v", sign)
		}
		peHeaderOffset += int64(4)
	} else {
		peHeaderOffset = int64(0)
	}

	sr.Seek(peHeaderOffset, seekStart)
	if err := binary.Read(sr, binary.LittleEndian, &f.FileHeader); err != nil {
		return nil, err
	}
	switch f.FileHeader.Machine {
	case IMAGE_FILE_MACHINE_UNKNOWN, IMAGE_FILE_MACHINE_ARMNT, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_MACHINE_I386:
	default:
		return nil, fmt.Errorf("Unrecognised COFF file header machine value of 0x%x", f.FileHeader.Machine)
	}

	var err error

	if memoryMode {
		//get strings table location - offset is wrong in the header because we are in memory mode. Can we fix it? Yes we can!
		restore, err := sr.Seek(0, seekCurrent)
		if err != nil {
			return nil, fmt.Errorf("Had a bad time getting restore point: %v", err)
		}
		//seek to table start (skip the headers)
		sr.Seek(peHeaderOffset+int64(binary.Size(f.FileHeader))+int64(f.FileHeader.SizeOfOptionalHeader), seekStart)

		//iterate through the sections to find the raw offset value that matches the original symbol table value
		for i := 0; i < int(f.FileHeader.NumberOfSections); i++ {
			sh := new(SectionHeader32)
			if err := binary.Read(sr, binary.LittleEndian, sh); err != nil {
				return nil, err
			}
			//original offset matches the pointer to the symbol table, update the header so other things can reference it good again
			if sh.PointerToRawData == f.FileHeader.PointerToSymbolTable {
				f.FileHeader.PointerToSymbolTable = sh.VirtualAddress
			}
		}
		//restore the original location of sr (this shouldn't actually be required, but just in case)
		sr.Seek(restore, seekStart)
	}

	// Read string table.
	f.StringTable, err = readStringTable(&f.FileHeader, sr)
	if err != nil {
		return nil, err
	}

	// Read symbol table.
	f.COFFSymbols, err = readCOFFSymbols(&f.FileHeader, sr)
	if err != nil {
		return nil, err
	}
	f.Symbols, err = removeAuxSymbols(f.COFFSymbols, f.StringTable)
	if err != nil {
		return nil, err
	}

	// Read optional header.
	f.OptionalHeaderOffset = peHeaderOffset + int64(binary.Size(f.FileHeader))
	sr.Seek(f.OptionalHeaderOffset, seekStart)

	var oh32 OptionalHeader32
	var oh64 OptionalHeader64
	switch f.FileHeader.SizeOfOptionalHeader {
	case sizeofOptionalHeader32:
		if err := binary.Read(sr, binary.LittleEndian, &oh32); err != nil {
			return nil, err
		}
		if oh32.Magic != 0x10b { // PE32
			return nil, fmt.Errorf("pe32 optional header has unexpected Magic of 0x%x", oh32.Magic)
		}
		f.OptionalHeader = &oh32
	case sizeofOptionalHeader64:
		if err := binary.Read(sr, binary.LittleEndian, &oh64); err != nil {
			return nil, err
		}
		if oh64.Magic != 0x20b { // PE32+
			return nil, fmt.Errorf("pe32+ optional header has unexpected Magic of 0x%x", oh64.Magic)
		}
		f.OptionalHeader = &oh64
	}

	// Process sections.
	f.Sections = make([]*Section, f.FileHeader.NumberOfSections)
	for i := 0; i < int(f.FileHeader.NumberOfSections); i++ {
		sh := new(SectionHeader32)
		if err := binary.Read(sr, binary.LittleEndian, sh); err != nil {
			return nil, err
		}
		name, err := sh.fullName(f.StringTable)
		if err != nil {
			return nil, err
		}
		s := new(Section)
		s.SectionHeader = SectionHeader{
			Name:                 name,
			OriginalName:         sh.Name,
			VirtualSize:          sh.VirtualSize,
			VirtualAddress:       sh.VirtualAddress,
			Size:                 sh.SizeOfRawData,
			Offset:               sh.PointerToRawData,
			PointerToRelocations: sh.PointerToRelocations,
			PointerToLineNumbers: sh.PointerToLineNumbers,
			NumberOfRelocations:  sh.NumberOfRelocations,
			NumberOfLineNumbers:  sh.NumberOfLineNumbers,
			Characteristics:      sh.Characteristics,
		}
		r2 := r
		if sh.PointerToRawData == 0 { // .bss must have all 0s
			r2 = zeroReaderAt{}
		}
		if !memoryMode {
			s.sr = io.NewSectionReader(r2, int64(s.SectionHeader.Offset), int64(s.SectionHeader.Size))
		} else {
			s.sr = io.NewSectionReader(r2, int64(s.SectionHeader.VirtualAddress), int64(s.SectionHeader.Size))
		}
		s.ReaderAt = s.sr
		f.Sections[i] = s
	}
	for i := range f.Sections {
		var err error
		f.Sections[i].Relocs, err = readRelocs(&f.Sections[i].SectionHeader, sr)
		if err != nil {
			return nil, err
		}
	}

	// Read Base Relocation Block and Items
	f.BaseRelocationTable, err = f.readBaseRelocationTable()
	if err != nil {
		return nil, err
	}

	// Read certificate table (only in disk mode)
	if !memoryMode {
		f.CertificateTable, err = readCertTable(f, sr)
		if err != nil {
			return nil, err
		}
	}

	//fill net info
	if f.IsManaged() {
		var va, size uint32

		//determine location of the COM descriptor directory
		switch v := f.OptionalHeader.(type) {
		case *OptionalHeader32:
			va = v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress
			size = v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].Size
		case *OptionalHeader64:
			va = v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress
			size = v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].Size
		}

		//I'm unsure how to get a reader (not a readerat) for a particular thing, so copying buffers around.. this could be more optimal
		buff := make([]byte, size)

		//none of these reads have errors being caught either, which is probably not ideal
		if !memoryMode {
			r.ReadAt(buff, int64(f.RVAToFileOffset(va)))
		} else {
			r.ReadAt(buff, int64(va))
		}
		binary.Read(bytes.NewReader(buff), binary.LittleEndian, &f.Net.NetDirectory)

		//Now that we have the COR20 header (COM descriptor directory header), we can get the metadata section header, which has the version
		buff = make([]byte, f.Net.NetDirectory.MetaDataSize)
		//again, none of the reads are error checked :shrug:
		if !memoryMode {
			r.ReadAt(buff, int64(f.RVAToFileOffset(f.Net.NetDirectory.MetaDataRVA)))
		} else {
			r.ReadAt(buff, int64(f.Net.NetDirectory.MetaDataRVA))
		}
		f.Net.MetaData, _ = newMetadataHeader(bytes.NewReader(buff))

	}

	return f, nil
}

// zeroReaderAt is ReaderAt that reads 0s.
type zeroReaderAt struct{}

// ReadAt writes len(p) 0s into p.
func (w zeroReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// getString extracts a string from symbol string table.
func getString(section []byte, start int) (string, bool) {
	if start < 0 || start >= len(section) {
		return "", false
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end]), true
		}
	}
	return "", false
}

// Section returns the first section with the given name, or nil if no such
// section exists.
func (f *File) Section(name string) *Section {
	for _, s := range f.Sections {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (f *File) DWARF() (*dwarf.Data, error) {
	dwarfSuffix := func(s *Section) string {
		switch {
		case strings.HasPrefix(s.Name, ".debug_"):
			return s.Name[7:]
		case strings.HasPrefix(s.Name, ".zdebug_"):
			return s.Name[8:]
		default:
			return ""
		}

	}

	// sectionData gets the data for s and checks its size.
	sectionData := func(s *Section) ([]byte, error) {
		b, err := s.Data()
		if err != nil && uint32(len(b)) < s.Size {
			return nil, err
		}

		if 0 < s.VirtualSize && s.VirtualSize < s.Size {
			b = b[:s.VirtualSize]
		}

		if len(b) >= 12 && string(b[:4]) == "ZLIB" {
			dlen := binary.BigEndian.Uint64(b[4:12])
			dbuf := make([]byte, dlen)
			r, err := zlib.NewReader(bytes.NewBuffer(b[12:]))
			if err != nil {
				return nil, err
			}
			if _, err := io.ReadFull(r, dbuf); err != nil {
				return nil, err
			}
			if err := r.Close(); err != nil {
				return nil, err
			}
			b = dbuf
		}
		return b, nil
	}

	// There are many other DWARF sections, but these
	// are the ones the debug/dwarf package uses.
	// Don't bother loading others.
	var dat = map[string][]byte{"abbrev": nil, "info": nil, "str": nil, "line": nil, "ranges": nil}
	for _, s := range f.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; !ok {
			continue
		}

		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}
		dat[suffix] = b
	}

	d, err := dwarf.New(dat["abbrev"], nil, nil, dat["info"], dat["line"], nil, dat["ranges"], dat["str"])
	if err != nil {
		return nil, err
	}

	// Look for DWARF4 .debug_types sections.
	for i, s := range f.Sections {
		suffix := dwarfSuffix(s)
		if suffix != "types" {
			continue
		}

		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}

		err = d.AddTypes(fmt.Sprintf("types-%d", i), b)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

// FormatError is unused.
// The type is retained for compatibility.
type FormatError struct {
}

func (e *FormatError) Error() string {
	return "unknown error"
}

// RVAToFileOffset Converts a Relative offset to the actual offset in the file.
func (f *File) RVAToFileOffset(rva uint32) uint32 {
	var offset uint32
	for _, section := range f.Sections {
		if rva >= section.SectionHeader.VirtualAddress && rva <= section.SectionHeader.VirtualAddress+section.SectionHeader.Size {
			offset = section.SectionHeader.Offset + (rva - section.SectionHeader.VirtualAddress)
		}
	}
	return offset
}

// IsManaged returns true if the loaded PE file references the CLR header (aka is a .net exe)
func (f *File) IsManaged() bool {
	switch v := f.OptionalHeader.(type) {
	case *OptionalHeader32:
		if v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress != 0 {
			return true
		}
	case *OptionalHeader64:
		if v.DataDirectory[IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress != 0 {
			return true
		}
	}

	return false
}
