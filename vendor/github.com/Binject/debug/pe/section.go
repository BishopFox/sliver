// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pe

import (
	"io"
	"strconv"
)

// SectionHeader32 represents real PE COFF section header.
type SectionHeader32 struct {
	Name                 [8]uint8
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
	Characteristics      uint32
}

// fullName finds real name of section sh. Normally name is stored
// in sh.Name, but if it is longer then 8 characters, it is stored
// in COFF string table st instead.
func (sh *SectionHeader32) fullName(st StringTable) (string, error) {
	if sh.Name[0] != '/' {
		return cstring(sh.Name[:]), nil
	}
	i, err := strconv.Atoi(cstring(sh.Name[1:]))
	if err != nil {
		return "", err
	}
	return st.String(uint32(i))
}

// SectionHeader is similar to SectionHeader32 with Name
// field replaced by Go string. OriginalName is the
// original name of the section on disk.
type SectionHeader struct {
	Name                 string
	OriginalName         [8]uint8
	VirtualSize          uint32
	VirtualAddress       uint32
	Size                 uint32
	Offset               uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
	Characteristics      uint32
}

// Section provides access to PE COFF section.
type Section struct {
	SectionHeader
	Relocs []Reloc

	// Embed ReaderAt for ReadAt method.
	// Do not embed SectionReader directly
	// to avoid having Read and Seek.
	// If a client wants Read and Seek it must use
	// Open() to avoid fighting over the seek offset
	// with other clients.
	io.ReaderAt
	sr *io.SectionReader
}

// Data reads and returns the contents of the PE section s.
func (s *Section) Data() ([]byte, error) {

	if s.sr == nil { // This section was added from code, the internal SectionReader is nil
		return nil, nil
	}

	dat := make([]byte, s.sr.Size())
	n, err := s.sr.ReadAt(dat, 0)
	if n == len(dat) {
		err = nil
	}
	return dat[0:n], err
}

// Open returns a new ReadSeeker reading the PE section s.
func (s *Section) Open() io.ReadSeeker {
	return io.NewSectionReader(s.sr, 0, 1<<63-1)
}

// Section Flags (Characteristics field)
const (
	IMAGE_SCN_CNT_CODE    = 0x00000020 // Section contains code
	IMAGE_SCN_MEM_EXECUTE = 0x20000000 // Section is executable
	IMAGE_SCN_MEM_READ    = 0x40000000 // Section is readable

	IMAGE_FILE_RELOCS_STRIPPED = 0x0001 // Relocation info stripped from file

	IMAGE_DLLCHARACTERISTICS_NX_COMPAT = 0x0100 // Image is NX compatable

	IMAGE_DLLCHARACTERISTICS_DYNAMIC_BASE = 0x0040 // DLL can move
)
