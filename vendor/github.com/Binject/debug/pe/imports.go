package pe

import "encoding/binary"

// ImportDirectory entry
type ImportDirectory struct {
	OriginalFirstThunk uint32
	TimeDateStamp      uint32
	ForwarderChain     uint32
	NameRVA            uint32
	FirstThunk         uint32

	DllName string
}

// IAT returns the DataDirectory for the IAT
func (f *File) IAT() *DataDirectory {
	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64

	// grab the number of data directory entries
	var ddLength uint32
	if pe64 {
		ddLength = f.OptionalHeader.(*OptionalHeader64).NumberOfRvaAndSizes
	} else {
		ddLength = f.OptionalHeader.(*OptionalHeader32).NumberOfRvaAndSizes
	}

	// check that the length of data directory entries is large
	// enough to include the imports directory.
	if ddLength < IMAGE_DIRECTORY_ENTRY_IAT+1 {
		return nil
	}

	// grab the IAT entry
	var idd DataDirectory
	if pe64 {
		idd = f.OptionalHeader.(*OptionalHeader64).DataDirectory[IMAGE_DIRECTORY_ENTRY_IAT]
	} else {
		idd = f.OptionalHeader.(*OptionalHeader32).DataDirectory[IMAGE_DIRECTORY_ENTRY_IAT]
	}
	return &idd
}

// ImportDirectoryTable - returns the Import Directory Table, a pointer to the section, and the section raw data
func (f *File) ImportDirectoryTable() ([]ImportDirectory, *Section, *[]byte, error) {

	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64

	// grab the number of data directory entries
	var ddLength uint32
	if pe64 {
		ddLength = f.OptionalHeader.(*OptionalHeader64).NumberOfRvaAndSizes
	} else {
		ddLength = f.OptionalHeader.(*OptionalHeader32).NumberOfRvaAndSizes
	}

	// check that the length of data directory entries is large
	// enough to include the imports directory.
	if ddLength < IMAGE_DIRECTORY_ENTRY_IMPORT+1 {
		return nil, nil, nil, nil
	}

	// grab the import data directory entry
	var idd DataDirectory
	if pe64 {
		idd = f.OptionalHeader.(*OptionalHeader64).DataDirectory[IMAGE_DIRECTORY_ENTRY_IMPORT]
	} else {
		idd = f.OptionalHeader.(*OptionalHeader32).DataDirectory[IMAGE_DIRECTORY_ENTRY_IMPORT]
	}

	// figure out which section contains the import directory table
	var ds *Section
	ds = nil
	for _, s := range f.Sections {
		if s.VirtualAddress <= idd.VirtualAddress && idd.VirtualAddress < s.VirtualAddress+s.VirtualSize {
			ds = s
			break
		}
	}

	// didn't find a section, so no import libraries were found
	if ds == nil {
		return nil, nil, nil, nil
	}

	sectionData, err := ds.Data()
	if err != nil {
		return nil, nil, nil, err
	}

	// seek to the virtual address specified in the import data directory
	d := sectionData[idd.VirtualAddress-ds.VirtualAddress:]

	// start decoding the import directory
	var ida []ImportDirectory
	for len(d) > 0 {
		var dt ImportDirectory
		dt.OriginalFirstThunk = binary.LittleEndian.Uint32(d[0:4])
		dt.TimeDateStamp = binary.LittleEndian.Uint32(d[4:8])
		dt.ForwarderChain = binary.LittleEndian.Uint32(d[8:12])
		dt.NameRVA = binary.LittleEndian.Uint32(d[12:16])
		dt.FirstThunk = binary.LittleEndian.Uint32(d[16:20])
		dt.DllName, _ = getString(sectionData, int(dt.NameRVA-ds.VirtualAddress))
		d = d[20:]
		if dt.OriginalFirstThunk == 0 {
			break
		}
		ida = append(ida, dt)
	}
	return ida, ds, &sectionData, nil
}

// ImportedSymbols returns the names of all symbols
// referred to by the binary f that are expected to be
// satisfied by other libraries at dynamic load time.
// It does not return weak symbols.
func (f *File) ImportedSymbols() ([]string, error) {
	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64

	ida, ds, sectionData, err := f.ImportDirectoryTable()
	if err != nil {
		return nil, err
	}

	var all []string
	for _, dt := range ida {
		d := (*sectionData)[:]
		// seek to OriginalFirstThunk
		d = d[dt.OriginalFirstThunk-ds.VirtualAddress:]
		for len(d) > 0 {
			if pe64 { // 64bit
				va := binary.LittleEndian.Uint64(d[0:8])
				d = d[8:]
				if va == 0 {
					break
				}
				if va&0x8000000000000000 > 0 { // is Ordinal
					// TODO add dynimport ordinal support.
				} else {
					fn, _ := getString(*sectionData, int(uint32(va)-ds.VirtualAddress+2))
					all = append(all, fn+":"+dt.DllName)
				}
			} else { // 32bit
				va := binary.LittleEndian.Uint32(d[0:4])
				d = d[4:]
				if va == 0 {
					break
				}
				if va&0x80000000 > 0 { // is Ordinal
					// TODO add dynimport ordinal support.
					//ord := va&0x0000FFFF
				} else {
					fn, _ := getString(*sectionData, int(va-ds.VirtualAddress+2))
					all = append(all, fn+":"+dt.DllName)
				}
			}
		}
	}

	return all, nil
}

// ImportedLibraries returns the names of all libraries
// referred to by the binary f that are expected to be
// linked with the binary at dynamic link time.
func (f *File) ImportedLibraries() ([]string, error) {
	ida, _, _, err := f.ImportDirectoryTable()
	if err != nil {
		return nil, err
	}
	var all []string
	for _, dt := range ida {
		all = append(all, dt.DllName)
	}
	return all, nil
}
