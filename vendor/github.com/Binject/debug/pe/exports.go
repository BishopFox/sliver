package pe

import (
	"encoding/binary"
)

// ExportDirectory - data directory definition for exported functions
type ExportDirectory struct {
	ExportFlags       uint32 // reserved, must be zero
	TimeDateStamp     uint32
	MajorVersion      uint16
	MinorVersion      uint16
	NameRVA           uint32 // pointer to the name of the DLL
	OrdinalBase       uint32
	NumberOfFunctions uint32
	NumberOfNames     uint32 // also Ordinal Table Len
	AddressTableAddr  uint32 // RVA of EAT, relative to image base
	NameTableAddr     uint32 // RVA of export name pointer table, relative to image base
	OrdinalTableAddr  uint32 // address of the ordinal table, relative to iamge base

	DllName string
}

// Export - describes a single export entry
type Export struct {
	Ordinal        uint32
	Name           string
	VirtualAddress uint32
}

// Exports - gets exports
func (f *File) Exports() ([]Export, error) {
	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64

	// grab the number of data directory entries
	var ddLength uint32
	if pe64 {
		ddLength = f.OptionalHeader.(*OptionalHeader64).NumberOfRvaAndSizes
	} else {
		ddLength = f.OptionalHeader.(*OptionalHeader32).NumberOfRvaAndSizes
	}

	// check that the length of data directory entries is large
	// enough to include the exports directory.
	if ddLength < IMAGE_DIRECTORY_ENTRY_EXPORT+1 {
		return nil, nil
	}

	// grab the export data directory entry
	var edd DataDirectory
	if pe64 {
		edd = f.OptionalHeader.(*OptionalHeader64).DataDirectory[IMAGE_DIRECTORY_ENTRY_EXPORT]
	} else {
		edd = f.OptionalHeader.(*OptionalHeader32).DataDirectory[IMAGE_DIRECTORY_ENTRY_EXPORT]
	}

	// figure out which section contains the export directory table
	var ds *Section
	ds = nil
	for _, s := range f.Sections {
		if s.VirtualAddress <= edd.VirtualAddress && edd.VirtualAddress < s.VirtualAddress+s.VirtualSize {
			ds = s
			break
		}
	}

	// didn't find a section, so no exports were found
	if ds == nil {
		return nil, nil
	}

	d, err := ds.Data()
	if err != nil {
		return nil, err
	}

	exportDirOffset := edd.VirtualAddress - ds.VirtualAddress

	// seek to the virtual address specified in the export data directory
	dxd := d[exportDirOffset:]

	// deserialize export directory
	var dt ExportDirectory
	dt.ExportFlags = binary.LittleEndian.Uint32(dxd[0:4])
	dt.TimeDateStamp = binary.LittleEndian.Uint32(dxd[4:8])
	dt.MajorVersion = binary.LittleEndian.Uint16(dxd[8:10])
	dt.MinorVersion = binary.LittleEndian.Uint16(dxd[10:12])
	dt.NameRVA = binary.LittleEndian.Uint32(dxd[12:16])
	dt.OrdinalBase = binary.LittleEndian.Uint32(dxd[16:20])
	dt.NumberOfFunctions = binary.LittleEndian.Uint32(dxd[20:24])
	dt.NumberOfNames = binary.LittleEndian.Uint32(dxd[24:28])
	dt.AddressTableAddr = binary.LittleEndian.Uint32(dxd[28:32])
	dt.NameTableAddr = binary.LittleEndian.Uint32(dxd[32:36])
	dt.OrdinalTableAddr = binary.LittleEndian.Uint32(dxd[36:40])

	dt.DllName, _ = getString(d, int(dt.NameRVA-ds.VirtualAddress))

	// seek to ordinal table
	dno := d[dt.OrdinalTableAddr-ds.VirtualAddress:]
	// seek to names table
	dnn := d[dt.NameTableAddr-ds.VirtualAddress:]

	// build whole ordinal->name table
	ordinalTable := make(map[uint16]uint32)
	for n := uint32(0); n < dt.NumberOfNames; n++ {
		ord := binary.LittleEndian.Uint16(dno[n*2 : (n*2)+2])
		nameRVA := binary.LittleEndian.Uint32(dnn[n*4 : (n*4)+4])
		ordinalTable[ord] = nameRVA
	}
	dno = nil
	dnn = nil

	// seek to ordinal table
	dna := d[dt.AddressTableAddr-ds.VirtualAddress:]

	var exports []Export
	for i := uint32(0); i < dt.NumberOfFunctions; i++ {
		var export Export
		export.VirtualAddress =
			binary.LittleEndian.Uint32(dna[i*4 : (i*4)+4])
		export.Ordinal = dt.OrdinalBase + i

		// check the entire ordinal table looking for this index to see if we have a name
		_, ok := ordinalTable[uint16(i)]
		if ok { // a name exists for this exported function
			nameRVA, _ := ordinalTable[uint16(i)]
			export.Name, _ = getString(d, int(nameRVA-ds.VirtualAddress))
		}
		exports = append(exports, export)
	}
	return exports, nil
}
