package pe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

func (peFile *File) Bytes() ([]byte, error) {
	var bytesWritten uint64
	peBuf := bytes.NewBuffer(nil)

	// write DOS header and stub
	binary.Write(peBuf, binary.LittleEndian, peFile.DosHeader)
	bytesWritten += uint64(binary.Size(peFile.DosHeader))
	if peFile.DosExists {
		binary.Write(peBuf, binary.LittleEndian, peFile.DosStub)
		bytesWritten += uint64(binary.Size(peFile.DosStub))
	}

	// write Rich header
	if peFile.RichHeader != nil {
		binary.Write(peBuf, binary.LittleEndian, peFile.RichHeader)
		bytesWritten += uint64(len(peFile.RichHeader))
	}

	// apply padding before PE header if necessary
	if uint32(bytesWritten) != peFile.DosHeader.AddressOfNewExeHeader {
		padding := make([]byte, peFile.DosHeader.AddressOfNewExeHeader-uint32(bytesWritten))
		binary.Write(peBuf, binary.LittleEndian, padding)
		bytesWritten += uint64(len(padding))
	}

	// write PE header
	peMagic := []byte{'P', 'E', 0x00, 0x00}
	binary.Write(peBuf, binary.LittleEndian, peMagic)
	binary.Write(peBuf, binary.LittleEndian, peFile.FileHeader)
	bytesWritten += uint64(binary.Size(peFile.FileHeader) + len(peMagic))

	var (
		is32bit                              bool
		oldCertTableOffset, oldCertTableSize uint32
	)

	switch peFile.FileHeader.Machine {
	case IMAGE_FILE_MACHINE_I386:
		is32bit = true
		optionalHeader := peFile.OptionalHeader.(*OptionalHeader32)
		binary.Write(peBuf, binary.LittleEndian, peFile.OptionalHeader.(*OptionalHeader32))
		bytesWritten += uint64(binary.Size(optionalHeader))

		oldCertTableOffset = optionalHeader.DataDirectory[CERTIFICATE_TABLE].VirtualAddress
		oldCertTableSize = optionalHeader.DataDirectory[CERTIFICATE_TABLE].Size
	case IMAGE_FILE_MACHINE_AMD64:
		is32bit = false
		optionalHeader := peFile.OptionalHeader.(*OptionalHeader64)
		binary.Write(peBuf, binary.LittleEndian, optionalHeader)
		bytesWritten += uint64(binary.Size(optionalHeader))

		oldCertTableOffset = optionalHeader.DataDirectory[CERTIFICATE_TABLE].VirtualAddress
		oldCertTableSize = optionalHeader.DataDirectory[CERTIFICATE_TABLE].Size
	default:
		return nil, errors.New("architecture not supported")
	}

	// write section headers
	sectionHeaders := make([]SectionHeader32, len(peFile.Sections))
	for idx, section := range peFile.Sections {
		// write section header
		sectionHeader := SectionHeader32{
			Name:                 section.OriginalName,
			VirtualSize:          section.VirtualSize,
			VirtualAddress:       section.VirtualAddress,
			SizeOfRawData:        section.Size,
			PointerToRawData:     section.Offset,
			PointerToRelocations: section.PointerToRelocations,
			PointerToLineNumbers: section.PointerToLineNumbers,
			NumberOfRelocations:  section.NumberOfRelocations,
			NumberOfLineNumbers:  section.NumberOfLineNumbers,
			Characteristics:      section.Characteristics,
		}

		// if the PE file was pulled from memory, the symbol table offset in the header will be wrong.
		// Fix it up by picking the section that lines up, and use the raw offset instead.
		if peFile.FileHeader.PointerToSymbolTable == sectionHeader.VirtualAddress {
			peFile.FileHeader.PointerToSymbolTable = sectionHeader.PointerToRawData
		}

		sectionHeaders[idx] = sectionHeader

		//log.Printf("section: %+v\nsectionHeader: %+v\n", section, sectionHeader)

		binary.Write(peBuf, binary.LittleEndian, sectionHeader)
		bytesWritten += uint64(binary.Size(sectionHeader))
	}

	// write sections' data
	for idx, sectionHeader := range sectionHeaders {
		section := peFile.Sections[idx]
		sectionData, err := section.Data()
		if err != nil {
			return nil, err
		}
		if sectionData == nil { // for sections that weren't in the original file
			sectionData = []byte{}
		}
		if section.Offset != 0 && bytesWritten < uint64(section.Offset) {
			pad := make([]byte, uint64(section.Offset)-bytesWritten)
			peBuf.Write(pad)
			//log.Printf("Padding before section %s at %x: length:%x to:%x\n", section.Name, bytesWritten, len(pad), section.Offset)
			bytesWritten += uint64(len(pad))
		}
		// if our shellcode insertion address is inside this section, insert it at the correct offset in sectionData
		if peFile.InsertionAddr >= section.Offset && int64(peFile.InsertionAddr) < (int64(section.Offset)+int64(section.Size)-int64(len(peFile.InsertionBytes))) {
			sectionData = append(sectionData, peFile.InsertionBytes[:]...)
			datalen := len(sectionData)
			if sectionHeader.SizeOfRawData > uint32(datalen) {
				paddingSize := sectionHeader.SizeOfRawData - uint32(datalen)
				padding := make([]byte, paddingSize, paddingSize)
				sectionData = append(sectionData, padding...)
				//log.Printf("Padding after section %s: length:%d\n", section.Name, paddingSize)
			}
		}

		binary.Write(peBuf, binary.LittleEndian, sectionData)
		bytesWritten += uint64(len(sectionData))
	}

	// write symbols
	binary.Write(peBuf, binary.LittleEndian, peFile.COFFSymbols)
	bytesWritten += uint64(binary.Size(peFile.COFFSymbols))

	// write the string table
	binary.Write(peBuf, binary.LittleEndian, peFile.StringTable)
	bytesWritten += uint64(binary.Size(peFile.StringTable))

	var newCertTableOffset, newCertTableSize uint32

	// write the certificate table
	if peFile.CertificateTable != nil {
		newCertTableOffset = uint32(bytesWritten)
		newCertTableSize = uint32(len(peFile.CertificateTable))
	} else {
		newCertTableOffset = 0
		newCertTableSize = 0
	}

	binary.Write(peBuf, binary.LittleEndian, peFile.CertificateTable)
	bytesWritten += uint64(len(peFile.CertificateTable))

	peData := peBuf.Bytes()

	// write the offset and size of the new Certificate Table if it changed
	if newCertTableOffset != oldCertTableOffset || newCertTableSize != oldCertTableSize {
		certTableInfo := &DataDirectory{
			VirtualAddress: newCertTableOffset,
			Size:           newCertTableSize,
		}

		var certTableInfoBuf bytes.Buffer
		binary.Write(&certTableInfoBuf, binary.LittleEndian, certTableInfo)

		var certTableLoc int64
		if is32bit {
			certTableLoc = int64(peFile.DosHeader.AddressOfNewExeHeader) + 24 + 128
		} else {
			certTableLoc = int64(peFile.DosHeader.AddressOfNewExeHeader) + 24 + 144
		}

		peData = append(peData[:certTableLoc], append(certTableInfoBuf.Bytes(), peData[int(certTableLoc)+binary.Size(certTableInfo):]...)...)
	}

	return peData, nil
}

func (peFile *File) WriteFile(destFile string) error {
	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()

	peData, err := peFile.Bytes()
	if err != nil {
		return err
	}

	_, err = f.Write(peData)
	if err != nil {
		return err
	}

	return nil
}
