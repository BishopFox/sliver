package macho

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"
	"sort"
)

// Bytes - Returns the bytes of an assembled *macho.File
func (machoFile *File) Bytes() ([]byte, error) {
	var bytesWritten uint64
	w := bytes.NewBuffer(nil)

	// Write entire file header.
	buf := &bytes.Buffer{}
	err := binary.Write(buf, machoFile.ByteOrder, machoFile.FileHeader)
	if err != nil {
		panic(err)
	}
	headerLength := len(buf.Bytes())
	binary.Write(w, machoFile.ByteOrder, machoFile.FileHeader)
	bytesWritten += uint64(headerLength)
	//log.Printf("%x: Wrote file header of size: %v", bytesWritten, bytesWritten)

	// Reserved 4 bytes at end of header
	w.Write([]byte{0, 0, 0, 0})
	bytesWritten += 4

	// Write Load Commands Loop
	for _, singleLoad := range machoFile.Loads {
		buf2 := &bytes.Buffer{}
		err = binary.Write(buf2, machoFile.ByteOrder, singleLoad.Raw())
		if err != nil {
			panic(err)
		}
		LoadCmdLen := len(buf2.Bytes())
		binary.Write(w, machoFile.ByteOrder, singleLoad.Raw())
		bytesWritten += uint64(LoadCmdLen)
		//log.Printf("%x: Wrote Load Command, total size of: %v", bytesWritten, LoadCmdLen)
	}

	// Shellcode gets caved in between the final load command and the first section
	if len(machoFile.Insertion) > 0 {
		binary.Write(w, machoFile.ByteOrder, machoFile.Insertion)
		bytesWritten += uint64(len(machoFile.Insertion))
	}

	// Sort Sections
	sortedSections := machoFile.Sections[:]
	sort.Slice(sortedSections, func(a, b int) bool { return machoFile.Sections[a].Offset < machoFile.Sections[b].Offset })

	/*
		var caveOffset, caveSize uint64
		for _, s := range sortedSections {
			if s.SectionHeader.Seg == "__TEXT" && s.Name == "__text" {
				caveOffset = bytesWritten
				caveSize = uint64(s.Offset) - caveOffset
				log.Printf("Code Cave Size: %d - %d = %d\n", s.Offset, caveOffset, caveSize)
				log.Println("Shellcode Size: ", len(machoFile.Insertion))
				break
			}
		}
	*/

	// Write Sections
	for _, s := range sortedSections {

		if bytesWritten > uint64(s.Offset) {
			log.Printf("Overlapping Sections in Generated macho: %+v\n", s.Name)
			continue
		}
		if bytesWritten < uint64(s.Offset) {
			pad := make([]byte, uint64(s.Offset)-bytesWritten)
			w.Write(pad)
			bytesWritten += uint64(len(pad))
			//log.Printf("%x: wrote %d padding bytes\n", bytesWritten, len(pad))
		}
		section, err := ioutil.ReadAll(s.Open())
		if err != nil {
			return nil, err
		}
		binary.Write(w, machoFile.ByteOrder, section)
		bytesWritten += uint64(len(section))
		//log.Printf("%x: wrote %d bytes section/segment named: %s %s\n", bytesWritten, uint64(len(section)), s.Name, s.Seg)
	}
	// Write Dynamic Loader Info if it exists
	if machoFile.DylinkInfo != nil {
		// Write Rebase if it exists
		if len(machoFile.DylinkInfo.RebaseDat) > 0 {
			//log.Printf("Rebase Offset: %d", machoFile.DylinkInfo.RebaseOffset)
			if int64(machoFile.DylinkInfo.RebaseOffset)-int64(bytesWritten) > 0 {
				padA := make([]byte, machoFile.DylinkInfo.RebaseOffset-bytesWritten)
				w.Write(padA)
				bytesWritten += uint64(len(padA))
				//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padA))
			}
			//log.Printf("Rebase: %+v \n", machoFile.DylinkInfo.RebaseDat)
			w.Write(machoFile.DylinkInfo.RebaseDat)
			bytesWritten += uint64(machoFile.DylinkInfo.RebaseLen)
			//log.Printf("%x: Wrote raw Rebase, length of: %d", bytesWritten, machoFile.DylinkInfo.RebaseLen)
		}
		//Binding
		if len(machoFile.DylinkInfo.BindingInfoDat) > 0 {
			//log.Printf("Binding Offset: %d", machoFile.DylinkInfo.BindingInfoOffset)
			if int64(machoFile.DylinkInfo.BindingInfoOffset)-int64(bytesWritten) > 0 {
				padB := make([]byte, machoFile.DylinkInfo.BindingInfoOffset-bytesWritten)
				w.Write(padB)
				bytesWritten += uint64(len(padB))
				//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padB))
			}
			//log.Printf("Binding Info: %+v \n", machoFile.DylinkInfo.BindingInfoDat)
			w.Write(machoFile.DylinkInfo.BindingInfoDat)
			bytesWritten += uint64(machoFile.DylinkInfo.BindingInfoLen)
			//log.Printf("%x: Wrote raw Binding Info, length of: %d", bytesWritten, machoFile.DylinkInfo.BindingInfoLen)
		}
		//Lazy
		if len(machoFile.DylinkInfo.LazyBindingDat) > 0 {
			//log.Printf("Lazy Offset: %d", machoFile.DylinkInfo.LazyBindingOffset)
			if int64(machoFile.DylinkInfo.LazyBindingOffset)-int64(bytesWritten) > 0 {
				padD := make([]byte, machoFile.DylinkInfo.LazyBindingOffset-bytesWritten)
				w.Write(padD)
				bytesWritten += uint64(len(padD))
				//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padD))
			}
			//log.Printf("Lazy Binding Data: %+v \n", machoFile.DylinkInfo.LazyBindingDat)
			w.Write(machoFile.DylinkInfo.LazyBindingDat)
			bytesWritten += uint64(machoFile.DylinkInfo.LazyBindingLen)
			//log.Printf("%x: Wrote raw lazybinding, length of: %d", bytesWritten, machoFile.DylinkInfo.LazyBindingLen)
		}
		//Export
		if len(machoFile.DylinkInfo.ExportInfoDat) > 0 {
			//log.Printf("Export Offset: %d", machoFile.DylinkInfo.ExportInfoOffset)
			if int64(machoFile.DylinkInfo.ExportInfoOffset)-int64(bytesWritten) > 0 {
				padE := make([]byte, machoFile.DylinkInfo.ExportInfoOffset-bytesWritten)
				w.Write(padE)
				bytesWritten += uint64(len(padE))
				//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padE))
			}
			//log.Printf("Export Info: %+v \n", machoFile.DylinkInfo.ExportInfoDat)
			w.Write(machoFile.DylinkInfo.ExportInfoDat)
			bytesWritten += uint64(machoFile.DylinkInfo.ExportInfoLen)
			//log.Printf("%x: Wrote raw Export Info, length of: %d", bytesWritten, machoFile.DylinkInfo.ExportInfoLen)
		}
		//Weak
		if len(machoFile.DylinkInfo.WeakBindingDat) > 0 {
			//log.Printf("Weak Offset: %d", machoFile.DylinkInfo.WeakBindingOffset)
			if int64(machoFile.DylinkInfo.WeakBindingOffset)-int64(bytesWritten) > 0 {
				padC := make([]byte, machoFile.DylinkInfo.WeakBindingOffset-bytesWritten)
				w.Write(padC)
				bytesWritten += uint64(len(padC))
				//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padC))
			}
			//log.Printf("Weak Binding: %+v \n", machoFile.DylinkInfo.WeakBindingDat)
			w.Write(machoFile.DylinkInfo.WeakBindingDat)
			bytesWritten += uint64(machoFile.DylinkInfo.WeakBindingLen)
			//log.Printf("%x: Wrote raw Weak Binding, length of: %d", bytesWritten, machoFile.DylinkInfo.WeakBindingLen)
		}
	}

	// Write the Func Starts if they exist
	if machoFile.FuncStarts != nil {
		//log.Printf("new pad: %d", machoFile.FuncStarts.Offset-bytesWritten)
		if int64(machoFile.FuncStarts.Offset)-int64(bytesWritten) > 0 {
			padY := make([]byte, machoFile.FuncStarts.Offset-bytesWritten)
			w.Write(padY)
			bytesWritten += uint64(len(padY))
			//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padY))
		}
		//log.Printf("FuncStarts: %+v \n", machoFile.FuncStarts)
		w.Write(machoFile.FuncStarts.RawDat)
		bytesWritten += uint64(machoFile.FuncStarts.Len)
		//log.Printf("%x: Wrote raw funcstarts, length of: %d", bytesWritten, machoFile.FuncStarts.Len)
	}

	// Write the Data in Code Entries if they exist
	if machoFile.DataInCode != nil {
		if int64(machoFile.DataInCode.Offset)-int64(bytesWritten) > 0 {
			padZ := make([]byte, machoFile.DataInCode.Offset-bytesWritten)
			w.Write(padZ)
			bytesWritten += uint64(len(padZ))
			//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padZ))
		}
		//log.Printf("DataInCode: %+v \n", machoFile.DataInCode)
		w.Write(machoFile.DataInCode.RawDat)
		bytesWritten += uint64(machoFile.DataInCode.Len)
		//log.Printf("%x: Wrote raw dataincode, length of: %d", bytesWritten, machoFile.DataInCode.Len)
	}

	// Write Symbols is next I think
	symtab := machoFile.Symtab
	//log.Printf("Bytes written: %d", bytesWritten)
	//log.Printf("Indirect symbol offset: %d", machoFile.Dysymtab.DysymtabCmd.Indirectsymoff)
	//log.Printf("Locrel offset: %d", machoFile.Dysymtab.Locreloff)
	//log.Printf("Symtab offset: %d", symtab.Symoff)
	//log.Printf("String table offset: %d", symtab.Stroff)
	if int64(symtab.Symoff)-int64(bytesWritten) > 0 {
		pad := make([]byte, uint64(symtab.Symoff)-bytesWritten)
		w.Write(pad)
		bytesWritten += (uint64(symtab.Symoff) - bytesWritten)
		//log.Printf("%x: wrote pad of: %d", bytesWritten, uint64(symtab.Symoff)-bytesWritten)
	}
	w.Write(symtab.RawSymtab)
	bytesWritten += uint64(len(symtab.RawSymtab))
	//log.Printf("%x: Wrote raw symtab, length of: %d", bytesWritten, len(symtab.RawSymtab))

	// Write DySymTab next!
	dysymtab := machoFile.Dysymtab
	if int64(dysymtab.Indirectsymoff)-int64(bytesWritten) > 0 {
		pad2 := make([]byte, uint64(dysymtab.Indirectsymoff)-bytesWritten)
		w.Write(pad2)
		bytesWritten += uint64(len(pad2))
		//log.Printf("%x: wrote pad of: %d", bytesWritten, len(pad2))
	}
	w.Write(dysymtab.RawDysymtab)
	bytesWritten += uint64(len(dysymtab.RawDysymtab))
	//log.Printf("%x: Wrote raw indirect symbols, length of: %d", bytesWritten, len(dysymtab.RawDysymtab))

	// Write StringTab!
	if int64(symtab.Stroff)-int64(bytesWritten) > 0 {
		pad3 := make([]byte, uint64(symtab.Stroff)-bytesWritten)
		w.Write(pad3)
		bytesWritten += uint64(len(pad3))
		//log.Printf("%x: wrote pad of: %d", bytesWritten, len(pad3))
	}
	w.Write(symtab.RawStringtab)
	bytesWritten += uint64(len(symtab.RawStringtab))
	//log.Printf("%x: Wrote raw stringtab, length of: %d", bytesWritten, len(symtab.RawStringtab))

	// Write The Signature Block, if it exists
	//log.Printf("SigBlock Dat: %v", machoFile.SigBlock)
	if machoFile.SigBlock != nil {
		if int64(machoFile.SigBlock.Offset)-int64(bytesWritten) > 0 {
			padX := make([]byte, int64(machoFile.SigBlock.Offset)-int64(bytesWritten))
			w.Write(padX)
			bytesWritten += uint64(len(padX))
			//log.Printf("%x: wrote pad of: %d", bytesWritten, len(padX))
		}
		w.Write(machoFile.SigBlock.RawDat)
		bytesWritten += uint64(machoFile.SigBlock.Len)
		//log.Printf("%x: Wrote raw sigblock, length of: %d", bytesWritten, machoFile.SigBlock.Len)
	}

	// Write 0s to the end of the final segment
	if int64(FinalSegEnd)-int64(bytesWritten) > 0 {
		pad4 := make([]byte, uint64(FinalSegEnd)-bytesWritten)
		w.Write(pad4)
		bytesWritten += uint64(len(pad4))
		//log.Printf("%x: wrote pad of: %d", bytesWritten, len(pad4))
	}

	//log.Println("All done!")
	machoBytes := w.Bytes()
	return machoBytes, nil
}

// WriteFile - Creates a new file and writes it using the Bytes func above
func (machoFile *File) WriteFile(destFile string) error {
	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()
	machoData, err := machoFile.Bytes()
	if err != nil {
		return err
	}
	_, err = f.Write(machoData)
	if err != nil {
		return err
	}

	return nil
}

// WriteFatFile - Creates a new Fat file and multiple machos into it
func (FatyFile *FatFile) WriteFatFile(destFile string) error {
	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	bytesWritten := uint64(0)
	var FatyMachos []File
	var FatyMachosOffsets []uint32

	// Fat Header First
	// Magic Bytes
	binary.Write(w, binary.BigEndian, FatyFile.Magic)
	bytesWritten += 4
	// Number of Fat Arches [4 bytes]
	FatArches := uint32(len(FatyFile.Arches))
	binary.Write(w, binary.BigEndian, FatArches)
	bytesWritten += 4
	w.Flush()
	// Arch Size
	for _, arch := range FatyFile.Arches {
		log.Printf("Arch details: %v\n", arch)
		FatyMachos = append(FatyMachos, *(arch.File))
		//Cpu Type
		binary.Write(w, binary.BigEndian, uint32(arch.Cpu))
		bytesWritten += 4
		//Sub CPU type
		binary.Write(w, binary.BigEndian, uint32(arch.SubCpu))
		bytesWritten += 4
		//FileOffset
		FatyMachosOffsets = append(FatyMachosOffsets, arch.Offset)
		binary.Write(w, binary.BigEndian, uint32(arch.Offset))
		bytesWritten += 4
		//Size
		binary.Write(w, binary.BigEndian, uint32(arch.Size))
		bytesWritten += 4
		//Align
		binary.Write(w, binary.BigEndian, uint32(arch.Align))
		bytesWritten += 4
		w.Flush()
	}
	// End of Fat Headers

	// Write each Macho File at its Offset
	for index, mach := range FatyMachos {
		// Pad to the offset
		if bytesWritten < uint64(FatyMachosOffsets[index]) {
			pad := make([]byte, uint64(FatyMachosOffsets[index])-bytesWritten)
			w.Write(pad)
			bytesWritten += uint64(len(pad))
		}
		// Write the Mach-o file
		machOut, err := mach.Bytes()
		if err != nil {
			return err
		}
		binary.Write(w, binary.BigEndian, machOut)
		bytesWritten += uint64(len(machOut))
		w.Flush()
	}

	return nil
}
