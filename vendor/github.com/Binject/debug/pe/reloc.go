package pe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// RelocationTable - for base relocation entries
type RelocationTableEntry struct {
	RelocationBlock
	BlockItems []BlockItem
}

// RelocationBlock - for base relocation entries
type RelocationBlock struct {
	VirtualAddress uint32
	SizeOfBlock    uint32
}

// BlockItem - relocation block item
type BlockItem struct {
	Type   byte   // 4 bits
	Offset uint16 // 12 bits
}

// Reloc represents a PE COFF relocation.
// Each section contains its own relocation list.
type Reloc struct {
	VirtualAddress   uint32
	SymbolTableIndex uint32
	Type             uint16
}

const (
	//IMAGE_REL_BASED_ABSOLUTE - The base relocation is skipped. This type can be used to pad a block.
	IMAGE_REL_BASED_ABSOLUTE = 0

	//IMAGE_REL_BASED_HIGH           = 1
	//IMAGE_REL_BASED_LOW            = 2

	//IMAGE_REL_BASED_HIGHLOW - The base relocation applies all 32 bits of the difference to the 32-bit field at offset.
	IMAGE_REL_BASED_HIGHLOW = 3

	//IMAGE_REL_BASED_HIGHADJ        = 4
	//IMAGE_REL_BASED_MIPS_JMPADDR   = 5
	//IMAGE_REL_BASED_ARM_MOV32      = 5
	//IMAGE_REL_BASED_RISCV_HIGH20   = 5
	//IMAGE_REL_BASED_THUMB_MOV32    = 7
	//IMAGE_REL_BASED_RISCV_LOW12I   = 7
	//IMAGE_REL_BASED_RISCV_LOW12S   = 8
	//IMAGE_REL_BASED_MIPS_JMPADDR16 = 9
	IMAGE_REL_BASED_DIR64 = 10
)

// readBaseRelocationTable - reads the base relocation table from the file and stores it
func (f *File) readBaseRelocationTable() (*[]RelocationTableEntry, error) {

	if f.OptionalHeader == nil { // Optional header is optional, might not exist
		return nil, nil
	}

	var dd DataDirectory
	if f.Machine == IMAGE_FILE_MACHINE_AMD64 {
		dd = f.OptionalHeader.(*OptionalHeader64).DataDirectory[IMAGE_DIRECTORY_ENTRY_BASERELOC]
	} else {
		dd = f.OptionalHeader.(*OptionalHeader32).DataDirectory[IMAGE_DIRECTORY_ENTRY_BASERELOC]
	}
	var sectionData []byte
	var err error
	for _, section := range f.Sections {
		if section.VirtualAddress == dd.VirtualAddress {
			sectionData, err = section.Data()
			if err != nil {
				return nil, err
			}
			break
		}
	}
	r := bytes.NewReader(sectionData)
	var reloBlocks []RelocationTableEntry
	bytesRead := uint32(0)
	for bytesRead < dd.Size {
		var reloBlock RelocationBlock
		err = binary.Read(r, binary.LittleEndian, &reloBlock)
		bytesRead += 8
		if err != nil {
			return nil, fmt.Errorf("fail to read relocation block: %v", err)
		}
		numBlocks := (reloBlock.SizeOfBlock - 8) / 2
		blocks := make([]BlockItem, numBlocks)
		for i := uint32(0); i < numBlocks; i++ {
			var buf [2]byte
			err = binary.Read(r, binary.LittleEndian, &buf)
			bytesRead += 2
			if err != nil {
				return nil, fmt.Errorf("fail to read relocation block item %d: %v", i, err)
			}
			var item BlockItem
			val := binary.LittleEndian.Uint16(buf[:2])
			item.Type = byte(val >> 12)
			item.Offset = val & 0x0fff
			blocks[i] = item
		}
		reloBlocks = append(reloBlocks, RelocationTableEntry{reloBlock, blocks})
	}
	return &reloBlocks, nil
}

// Relocate - performs base relocations on this image to the given offset
func (f *File) Relocate(baseAddr uint64, image *[]byte) {
	var imageBase uint64
	pe64 := f.Machine == IMAGE_FILE_MACHINE_AMD64
	if pe64 {
		imageBase = f.OptionalHeader.(*OptionalHeader64).ImageBase
	} else {
		imageBase = uint64(f.OptionalHeader.(*OptionalHeader32).ImageBase)
	}
	for _, block := range *f.BaseRelocationTable {
		pageRVA := block.VirtualAddress
		for _, item := range block.BlockItems {
			if item.Type == IMAGE_REL_BASED_HIGHLOW { // 32 bit
				delta := uint32(baseAddr - imageBase)
				fileOffset := f.RVAToFileOffset(pageRVA)
				idx := fileOffset + uint32(item.Offset)
				originalAddress := binary.LittleEndian.Uint32((*image)[idx : idx+4])
				b := make([]byte, 4)
				binary.LittleEndian.PutUint32(b, originalAddress+delta)
				copy((*image)[idx:idx+4], b)
			} else if item.Type == IMAGE_REL_BASED_DIR64 { // 64 bit
				delta := baseAddr - imageBase
				fileOffset := f.RVAToFileOffset(pageRVA)
				idx := fileOffset + uint32(item.Offset)
				originalAddress := binary.LittleEndian.Uint64((*image)[idx : idx+8])
				b := make([]byte, 8)
				binary.LittleEndian.PutUint64(b, originalAddress+delta)
				copy((*image)[idx:idx+8], b)
			}
		}
	}

	// update imageBase in the optional header
	if pe64 {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, baseAddr)
		idx := f.OptionalHeaderOffset + 24
		copy((*image)[idx:idx+8], b)
	} else {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(baseAddr))
		idx := f.OptionalHeaderOffset + 28
		copy((*image)[idx:idx+4], b)
	}
}

func readRelocs(sh *SectionHeader, r io.ReadSeeker) ([]Reloc, error) {
	if sh.NumberOfRelocations <= 0 {
		return nil, nil
	}
	_, err := r.Seek(int64(sh.PointerToRelocations), seekStart)
	if err != nil {
		return nil, fmt.Errorf("fail to seek to %q section relocations: %v", sh.Name, err)
	}
	relocs := make([]Reloc, sh.NumberOfRelocations)
	err = binary.Read(r, binary.LittleEndian, relocs)
	if err != nil {
		return nil, fmt.Errorf("fail to read section relocations: %v", err)
	}
	return relocs, nil
}
