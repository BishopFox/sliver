package generate

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	SECTION_NAME            = 8
	DEFAULT_CHARACTERISTICS = 0x40000040
)

// SpoofMetadataConfig describes metadata mutation inputs by executable format.
// Additional executable formats can be added as new fields without changing the
// high-level API.
type SpoofMetadataConfig struct {
	PE *PESpoofMetadataConfig `json:"pe,omitempty" yaml:"pe,omitempty"`
}

// PESpoofMetadataConfig contains PE-specific metadata mutation inputs.
type PESpoofMetadataConfig struct {
	// Source is the donor PE for metadata/resource cloning.
	Source *SpoofMetadataFile `json:"source,omitempty" yaml:"source,omitempty"`
	// Icon is reserved for future standalone icon mutation support.
	Icon *SpoofMetadataFile `json:"icon,omitempty" yaml:"icon,omitempty"`
	// Optional PE structure overrides applied after donor metadata cloning.
	ResourceDirectory        *IMAGE_RESOURCE_DIRECTORY         `json:"resource_directory,omitempty" yaml:"resource_directory,omitempty"`
	ResourceDirectoryEntries []*IMAGE_RESOURCE_DIRECTORY_ENTRY `json:"resource_directory_entries,omitempty" yaml:"resource_directory_entries,omitempty"`
	ResourceDataEntries      []*IMAGE_RESOURCE_DATA_ENTRY      `json:"resource_data_entries,omitempty" yaml:"resource_data_entries,omitempty"`
	ExportDirectory          *IMAGE_EXPORT_DIRECTORY           `json:"export_directory,omitempty" yaml:"export_directory,omitempty"`
}

// SpoofMetadataFile is binary metadata content passed from the client.
type SpoofMetadataFile struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Data []byte `json:"data,omitempty" yaml:"data,omitempty"`
}

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
func SpoofMetadata(targetPath string, metadata *SpoofMetadataConfig) error {
	if metadata == nil {
		return nil
	}

	if metadata.PE != nil {
		if metadata.PE.Source == nil || len(metadata.PE.Source.Data) == 0 {
			return errors.New("pe spoof metadata source data is required")
		}
		return spoofPEMetadata(targetPath, metadata.PE)
	}

	return errors.New("unsupported spoof metadata format")
}

func spoofPEMetadata(targetPath string, metadata *PESpoofMetadataConfig) error {
	if metadata == nil || metadata.Source == nil {
		return errors.New("pe spoof metadata source data is required")
	}
	spoofData := metadata.Source.Data
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

	if err := applyPEMetadataOverrides(tgtFile, metadata); err != nil {
		return fmt.Errorf("failed to apply PE metadata overrides: %w", err)
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

func applyPEMetadataOverrides(peFile *pe.File, metadata *PESpoofMetadataConfig) error {
	if metadata == nil {
		return nil
	}
	if err := applyPEResourceOverrides(peFile, metadata); err != nil {
		return err
	}
	if err := applyPEExportOverrides(peFile, metadata.ExportDirectory); err != nil {
		return err
	}
	return nil
}

func applyPEResourceOverrides(peFile *pe.File, metadata *PESpoofMetadataConfig) error {
	if metadata == nil {
		return nil
	}
	hasOverrides := metadata.ResourceDirectory != nil ||
		len(metadata.ResourceDirectoryEntries) > 0 ||
		len(metadata.ResourceDataEntries) > 0
	if !hasOverrides {
		return nil
	}

	rsrcSection := peFile.Section(".rsrc")
	if rsrcSection == nil {
		return errors.New("resource overrides requested but target PE has no .rsrc section")
	}
	resourceData, err := rsrcSection.Data()
	if err != nil {
		return err
	}

	modifiedData, err := applyResourceSectionOverrides(resourceData, metadata.ResourceDirectory, metadata.ResourceDirectoryEntries, metadata.ResourceDataEntries)
	if err != nil {
		return err
	}
	rsrcSection.Replace(bytes.NewReader(modifiedData), int64(len(modifiedData)))
	return nil
}

func applyResourceSectionOverrides(
	sectionData []byte,
	resourceDirectory *IMAGE_RESOURCE_DIRECTORY,
	resourceDirectoryEntries []*IMAGE_RESOURCE_DIRECTORY_ENTRY,
	resourceDataEntries []*IMAGE_RESOURCE_DATA_ENTRY,
) ([]byte, error) {
	hasOverrides := resourceDirectory != nil ||
		len(resourceDirectoryEntries) > 0 ||
		len(resourceDataEntries) > 0
	if !hasOverrides {
		return append([]byte(nil), sectionData...), nil
	}
	if len(sectionData) < 16 {
		return nil, errors.New("resource section too small")
	}

	modifiedData := append([]byte(nil), sectionData...)
	if resourceDirectory != nil {
		binary.LittleEndian.PutUint32(modifiedData[0:4], resourceDirectory.Characteristics)
		binary.LittleEndian.PutUint32(modifiedData[4:8], resourceDirectory.TimeDateStamp)
		binary.LittleEndian.PutUint16(modifiedData[8:10], resourceDirectory.MajorVersion)
		binary.LittleEndian.PutUint16(modifiedData[10:12], resourceDirectory.MinorVersion)
		binary.LittleEndian.PutUint16(modifiedData[12:14], resourceDirectory.NumberOfNamedEntries)
		binary.LittleEndian.PutUint16(modifiedData[14:16], resourceDirectory.NumberOfIdEntries)
	}

	if len(resourceDirectoryEntries) > 0 {
		rootEntriesCount := int(binary.LittleEndian.Uint16(modifiedData[12:14])) +
			int(binary.LittleEndian.Uint16(modifiedData[14:16]))
		if len(resourceDirectoryEntries) > rootEntriesCount {
			return nil, fmt.Errorf(
				"resource_directory_entries has %d entries but root directory only has %d entries",
				len(resourceDirectoryEntries),
				rootEntriesCount,
			)
		}
		for i, entry := range resourceDirectoryEntries {
			if entry == nil {
				continue
			}
			entryOffset := 16 + (i * 8)
			if entryOffset+8 > len(modifiedData) {
				return nil, errors.New("resource directory entry out of bounds")
			}
			binary.LittleEndian.PutUint32(modifiedData[entryOffset:entryOffset+4], entry.Name)
			binary.LittleEndian.PutUint32(modifiedData[entryOffset+4:entryOffset+8], entry.OffsetToData)
		}
	}

	if len(resourceDataEntries) > 0 {
		dataEntryOffsets, err := collectResourceDataEntryOffsets(modifiedData)
		if err != nil {
			return nil, err
		}
		if len(resourceDataEntries) > len(dataEntryOffsets) {
			return nil, fmt.Errorf(
				"resource_data_entries has %d entries but resource tree only has %d data entries",
				len(resourceDataEntries),
				len(dataEntryOffsets),
			)
		}
		for i, entry := range resourceDataEntries {
			if entry == nil {
				continue
			}
			dataEntryOffset := dataEntryOffsets[i]
			if dataEntryOffset+16 > uint32(len(modifiedData)) {
				return nil, errors.New("resource data entry out of bounds")
			}
			offset := int(dataEntryOffset)
			binary.LittleEndian.PutUint32(modifiedData[offset:offset+4], entry.OffsetToData)
			binary.LittleEndian.PutUint32(modifiedData[offset+4:offset+8], entry.Size)
			binary.LittleEndian.PutUint32(modifiedData[offset+8:offset+12], entry.CodePage)
			binary.LittleEndian.PutUint32(modifiedData[offset+12:offset+16], entry.Reserved)
		}
	}

	return modifiedData, nil
}

func collectResourceDataEntryOffsets(data []byte) ([]uint32, error) {
	offsets := make([]uint32, 0)
	seen := map[uint32]struct{}{}
	if err := traverseCollectResourceDataEntryOffsets(data, 0, 0, seen, &offsets); err != nil {
		return nil, err
	}
	return offsets, nil
}

func traverseCollectResourceDataEntryOffsets(data []byte, offset uint32, depth int, seen map[uint32]struct{}, offsets *[]uint32) error {
	if depth > 10 {
		return nil
	}
	if offset+16 > uint32(len(data)) {
		return errors.New("resource directory out of bounds")
	}

	var dir IMAGE_RESOURCE_DIRECTORY
	br := bytes.NewReader(data[offset:])
	if err := binary.Read(br, binary.LittleEndian, &dir); err != nil {
		return err
	}

	entries := int(dir.NumberOfNamedEntries + dir.NumberOfIdEntries)
	entryOffset := offset + 16
	for i := 0; i < entries; i++ {
		if entryOffset+8 > uint32(len(data)) {
			return errors.New("resource directory entry out of bounds")
		}

		entryDataOffset := binary.LittleEndian.Uint32(data[entryOffset+4 : entryOffset+8])
		if entryDataOffset&0x80000000 != 0 {
			subDirOffset := entryDataOffset & 0x7FFFFFFF
			if err := traverseCollectResourceDataEntryOffsets(data, subDirOffset, depth+1, seen, offsets); err != nil {
				return err
			}
		} else {
			if entryDataOffset+16 > uint32(len(data)) {
				return errors.New("resource data entry out of bounds")
			}
			if _, exists := seen[entryDataOffset]; !exists {
				seen[entryDataOffset] = struct{}{}
				*offsets = append(*offsets, entryDataOffset)
			}
		}

		entryOffset += 8
	}
	return nil
}

func applyPEExportOverrides(peFile *pe.File, exportDirectory *IMAGE_EXPORT_DIRECTORY) error {
	if exportDirectory == nil {
		return nil
	}

	section, sectionData, sectionOffset, err := exportDirectorySectionData(peFile)
	if err != nil {
		return err
	}
	if section == nil {
		return nil
	}

	modifiedData, err := applyExportDirectoryOverrides(sectionData, sectionOffset, exportDirectory)
	if err != nil {
		return err
	}
	section.Replace(bytes.NewReader(modifiedData), int64(len(modifiedData)))
	return nil
}

func applyExportDirectoryOverrides(sectionData []byte, sectionOffset uint32, exportDirectory *IMAGE_EXPORT_DIRECTORY) ([]byte, error) {
	if exportDirectory == nil {
		return append([]byte(nil), sectionData...), nil
	}

	const exportDirectorySize = 40
	if sectionOffset+exportDirectorySize > uint32(len(sectionData)) {
		return nil, errors.New("export directory out of bounds")
	}

	modifiedData := append([]byte(nil), sectionData...)
	offset := int(sectionOffset)

	binary.LittleEndian.PutUint32(modifiedData[offset:offset+4], exportDirectory.Characteristics)
	binary.LittleEndian.PutUint32(modifiedData[offset+4:offset+8], exportDirectory.TimeDateStamp)
	binary.LittleEndian.PutUint16(modifiedData[offset+8:offset+10], exportDirectory.MajorVersion)
	binary.LittleEndian.PutUint16(modifiedData[offset+10:offset+12], exportDirectory.MinorVersion)
	binary.LittleEndian.PutUint32(modifiedData[offset+12:offset+16], exportDirectory.Name)
	binary.LittleEndian.PutUint32(modifiedData[offset+16:offset+20], exportDirectory.Base)
	binary.LittleEndian.PutUint32(modifiedData[offset+20:offset+24], exportDirectory.NumberOfFunctions)
	binary.LittleEndian.PutUint32(modifiedData[offset+24:offset+28], exportDirectory.NumberOfNames)
	binary.LittleEndian.PutUint32(modifiedData[offset+28:offset+32], exportDirectory.AddressOfFunctions)
	binary.LittleEndian.PutUint32(modifiedData[offset+32:offset+36], exportDirectory.AddressOfNames)
	binary.LittleEndian.PutUint32(modifiedData[offset+36:offset+40], exportDirectory.AddressOfNameOrdinals)

	return modifiedData, nil
}

func exportDirectorySectionData(peFile *pe.File) (*pe.Section, []byte, uint32, error) {
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
		return nil, nil, 0, nil
	}

	for _, section := range peFile.Sections {
		if va >= section.VirtualAddress && va < section.VirtualAddress+section.Size {
			sectionOffset := va - section.VirtualAddress
			sectionData, err := section.Data()
			if err != nil {
				return nil, nil, 0, err
			}
			if sectionOffset > uint32(len(sectionData)) {
				return nil, nil, 0, errors.New("export directory out of bounds")
			}
			return section, sectionData, sectionOffset, nil
		}
	}
	return nil, nil, 0, nil
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
	section, sectionData, sectionOffset, err := exportDirectorySectionData(peFile)
	if err != nil {
		return err
	}
	if section == nil {
		return nil
	}
	if uint32(len(sectionData)) > sectionOffset+8 {
		binary.LittleEndian.PutUint32(sectionData[sectionOffset+4:], timestamp)
		section.Replace(bytes.NewReader(sectionData), int64(len(sectionData)))
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
