package elf

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"
)

// Bytes - returns the bytes of an Elf file
func (elfFile *File) Bytes() ([]byte, error) {

	bytesWritten := uint64(0)
	elfBuf := bytes.NewBuffer(nil)
	w := bufio.NewWriter(elfBuf)

	// Write Elf Magic
	w.WriteByte('\x7f')
	w.WriteByte('E')
	w.WriteByte('L')
	w.WriteByte('F')
	bytesWritten += 4

	// ident[EI_CLASS]
	w.WriteByte(byte(elfFile.Class))
	// ident[EI_DATA]
	w.WriteByte(byte(elfFile.Data))
	// ident[EI_VERSION]
	w.WriteByte(byte(elfFile.Version))
	// ident[EI_OSABI]
	w.WriteByte(byte(elfFile.OSABI))
	// ident[EI_ABIVERSION]
	w.WriteByte(byte(elfFile.ABIVersion))
	// ident[EI_PAD] ( 7 bytes )
	w.Write([]byte{0, 0, 0, 0, 0, 0, 0})
	bytesWritten += 12

	// Type
	binary.Write(w, elfFile.ByteOrder, uint16(elfFile.Type))
	// Machine
	binary.Write(w, elfFile.ByteOrder, uint16(elfFile.Machine))
	// Version
	binary.Write(w, elfFile.ByteOrder, uint32(elfFile.Version))
	bytesWritten += 8

	phsize := 0

	switch elfFile.Class {
	case ELFCLASS32:
		phsize = 0x20
		// Entry 32
		binary.Write(w, elfFile.ByteOrder, uint32(elfFile.Entry))
		// PH Offset 32
		binary.Write(w, elfFile.ByteOrder, uint32(0x34))
		// SH Offset 32 //   0x20	0x28	4	8	e_shoff	Points to the start of the section header table.
		binary.Write(w, elfFile.ByteOrder, int32(elfFile.FileHeader.SHTOffset))
		// Flags
		binary.Write(w, elfFile.ByteOrder, uint32(0)) // todo
		// EH Size
		binary.Write(w, elfFile.ByteOrder, uint16(52))
		// PH Size //		0x2A	0x36	2	e_phentsize	Contains the size of a program header table entry.
		binary.Write(w, elfFile.ByteOrder, uint16(phsize))
		// PH Num // 0x2C	0x38	2	e_phnum	Contains the number of entries in the program header table.
		binary.Write(w, elfFile.ByteOrder, uint16(len(elfFile.Progs)))
		// SH Size //	0x2E	0x3A	2	e_shentsize	Contains the size of a section header table entry.
		binary.Write(w, elfFile.ByteOrder, uint16(0x28))
		bytesWritten += 24

	case ELFCLASS64:
		phsize = 0x38
		// Entry 64
		binary.Write(w, elfFile.ByteOrder, uint64(elfFile.Entry))
		// PH Offset 64
		binary.Write(w, elfFile.ByteOrder, uint64(0x40))
		// SH Offset 64 //   0x20	0x28	4	8	e_shoff	Points to the start of the section header table.
		binary.Write(w, elfFile.ByteOrder, int64(elfFile.FileHeader.SHTOffset))
		// Flags
		binary.Write(w, elfFile.ByteOrder, uint32(0)) // I think right?
		// EH Size
		binary.Write(w, elfFile.ByteOrder, uint16(64))
		// PH Size //		0x2A	0x36	2	e_phentsize	Contains the size of a program header table entry.
		binary.Write(w, elfFile.ByteOrder, uint16(phsize))
		// PH Num // 0x2C	0x38	2	e_phnum	Contains the number of entries in the program header table.
		binary.Write(w, elfFile.ByteOrder, uint16(len(elfFile.Progs)))
		// SH Size //	0x2E	0x3A	2	e_shentsize	Contains the size of a section header table entry.
		binary.Write(w, elfFile.ByteOrder, uint16(0x40))
		bytesWritten += 36
	}

	// SH Num //	0x30	0x3C	2	e_shnum	Contains the number of entries in the section header table.
	binary.Write(w, elfFile.ByteOrder, uint16(len(elfFile.Sections)))
	// SH Str Ndx	// 0x32	0x3E	2	e_shstrndx	Contains index of the section header table entry that contains the section names.
	binary.Write(w, elfFile.ByteOrder, uint16(elfFile.ShStrIndex))
	bytesWritten += 4

	// Program Header
	for _, p := range elfFile.Progs {
		// Type (segment)
		binary.Write(w, elfFile.ByteOrder, uint32(p.Type))
		bytesWritten += 4

		switch elfFile.Class {
		case ELFCLASS32:
			// Offset of Segment in File
			binary.Write(w, elfFile.ByteOrder, uint32(p.Off))

			// Vaddr
			binary.Write(w, elfFile.ByteOrder, uint32(p.Vaddr))

			// Paddr
			binary.Write(w, elfFile.ByteOrder, uint32(p.Paddr))

			// File Size
			binary.Write(w, elfFile.ByteOrder, uint32(p.Filesz))

			// Memory Size
			binary.Write(w, elfFile.ByteOrder, uint32(p.Memsz))

			// Flags (segment)
			binary.Write(w, elfFile.ByteOrder, uint32(p.Flags))

			// Alignment
			binary.Write(w, elfFile.ByteOrder, uint32(p.Align))

			bytesWritten += 28

		case ELFCLASS64:
			// Flags (segment)
			binary.Write(w, elfFile.ByteOrder, uint32(p.Flags))

			// Offset of Segment in File
			binary.Write(w, elfFile.ByteOrder, uint64(p.Off))

			// Vaddr
			binary.Write(w, elfFile.ByteOrder, uint64(p.Vaddr))

			// Paddr
			binary.Write(w, elfFile.ByteOrder, uint64(p.Paddr))

			// File Size
			binary.Write(w, elfFile.ByteOrder, uint64(p.Filesz))

			// Memory Size
			binary.Write(w, elfFile.ByteOrder, uint64(p.Memsz))

			// Alignment
			binary.Write(w, elfFile.ByteOrder, uint64(p.Align))

			bytesWritten += 52
		}
	}

	sortedSections := elfFile.Sections[:]
	//sort.Slice(sortedSections, func(a, b int) bool { return elfFile.Sections[a].Link < elfFile.Sections[b].Link })
	for _, s := range sortedSections {

		//log.Printf("Writing section: %s type: %+v\n", s.Name, s.Type)
		//log.Printf("written: %x offset: %x\n", bytesWritten, s.Offset)

		if s.Type == SHT_NULL || s.Type == SHT_NOBITS || s.FileSize == 0 {
			//log.Println("continuing...")
			continue
		}

		if bytesWritten > s.Offset {
			log.Printf("Overlapping Sections in Generated Elf: %+v\n", s.Name)
			continue
		}
		if s.Offset != 0 && bytesWritten < s.Offset {
			pad := make([]byte, s.Offset-bytesWritten)
			w.Write(pad)
			//log.Printf("Padding before section %s at %x: length:%x to:%x\n", s.Name, bytesWritten, len(pad), s.Offset)
			bytesWritten += uint64(len(pad))
		}

		slen := 0
		switch s.Type {
		case SHT_DYNAMIC:
			for _, taggedValue := range elfFile.DynTags {
				//log.Printf("writing %d (%x) -> %d (%x)\n", taggedValue.Tag, taggedValue.Tag, taggedValue.Value, taggedValue.Value)
				switch elfFile.Class {
				case ELFCLASS32:
					binary.Write(w, elfFile.ByteOrder, uint32(taggedValue.Tag))
					binary.Write(w, elfFile.ByteOrder, uint32(taggedValue.Value))
					bytesWritten += 8
				case ELFCLASS64:
					binary.Write(w, elfFile.ByteOrder, uint64(taggedValue.Tag))
					binary.Write(w, elfFile.ByteOrder, uint64(taggedValue.Value))
					bytesWritten += 16
				}
			}
		default:
			section, err := ioutil.ReadAll(s.Open())
			if err != nil {
				return nil, err
			}
			binary.Write(w, elfFile.ByteOrder, section)
			slen = len(section)
			//log.Printf("Wrote %s section at %x, length %x\n", s.Name, bytesWritten, slen)
			bytesWritten += uint64(slen)
		}

		// todo:  elfFile.Insertion should be renamed InsertionLoadEnd or similar
		if s.Type == SHT_PROGBITS && len(elfFile.Insertion) > 0 && s.Size-uint64(slen) >= uint64(len(elfFile.Insertion)) {
			binary.Write(w, elfFile.ByteOrder, elfFile.Insertion)
			bytesWritten += uint64(len(elfFile.Insertion))
		}
		w.Flush()
	}

	// Pad to Section Header Table
	if bytesWritten < uint64(elfFile.FileHeader.SHTOffset) {
		pad := make([]byte, uint64(elfFile.FileHeader.SHTOffset)-bytesWritten)
		w.Write(pad)
		//log.Printf("Padding before SHT at %x: length:%x to:%x\n", bytesWritten, len(pad), elfFile.FileHeader.SHTOffset)
		bytesWritten += uint64(len(pad))
	}

	// Write Section Header Table

	for _, s := range elfFile.Sections[:] {

		switch elfFile.Class {
		case ELFCLASS32:
			binary.Write(w, elfFile.ByteOrder, &Section32{
				Name:      s.Shname,
				Type:      uint32(s.Type),
				Flags:     uint32(s.Flags),
				Addr:      uint32(s.Addr),
				Off:       uint32(s.Offset),
				Size:      uint32(s.Size),
				Link:      s.Link,
				Info:      s.Info,
				Addralign: uint32(s.Addralign),
				Entsize:   uint32(s.Entsize)})
		case ELFCLASS64:
			binary.Write(w, elfFile.ByteOrder, &Section64{
				Name:      s.Shname,
				Type:      uint32(s.Type),
				Flags:     uint64(s.Flags),
				Addr:      s.Addr,
				Off:       s.Offset,
				Size:      s.Size,
				Link:      s.Link,
				Info:      s.Info,
				Addralign: s.Addralign,
				Entsize:   s.Entsize})
		}
	}

	// Do I have a PT_NOTE segment to add at the end?

	if len(elfFile.InsertionEOF) > 0 {
		binary.Write(w, elfFile.ByteOrder, elfFile.InsertionEOF)
		bytesWritten += uint64(len(elfFile.InsertionEOF))
	}

	w.Flush()
	return elfBuf.Bytes(), nil
}

// WriteFile - Creates a new file and writes it using the Bytes func above
func (elfFile *File) WriteFile(destFile string) error {
	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()
	elfData, err := elfFile.Bytes()
	if err != nil {
		return err
	}
	_, err = f.Write(elfData)
	if err != nil {
		return err
	}

	return nil
}
