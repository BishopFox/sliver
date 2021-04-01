// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package macho implements access to Mach-O object files.
package macho

// High level access to low level data structures.

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

// A File represents an open Mach-O file.
type File struct {
	FileHeader
	ByteOrder binary.ByteOrder
	Loads     []Load
	Sections  []*Section

	Symtab     *Symtab
	Dysymtab   *Dysymtab
	SigBlock   *SigBlock
	FuncStarts *FuncStarts
	DataInCode *DataInCode
	DylinkInfo *DylinkInfo

	EntryPoint uint64
	Insertion  []byte

	closer io.Closer
}

// A Load represents any Mach-O load command.
type Load interface {
	Raw() []byte
}

// A LoadBytes is the uninterpreted bytes of a Mach-O load command.
type LoadBytes []byte

func (b LoadBytes) Raw() []byte { return b }

// A SegmentHeader is the header for a Mach-O 32-bit or 64-bit load segment command.
type SegmentHeader struct {
	Cmd     LoadCmd
	Len     uint32
	Name    string
	Addr    uint64
	Memsz   uint64
	Offset  uint64
	Filesz  uint64
	Maxprot uint32
	Prot    uint32
	Nsect   uint32
	Flag    uint32
}

type SigBlock struct {
	Len    uint32
	Offset uint64
	RawDat []byte
}

type FuncStarts struct {
	Len    uint32
	Offset uint64
	RawDat []byte
}

type DataInCode struct {
	Len    uint32
	Offset uint64
	RawDat []byte
}

type DylinkInfo struct {
	RebaseLen         uint32
	RebaseOffset      uint64
	RebaseDat         []byte
	BindingInfoLen    uint32
	BindingInfoOffset uint64
	BindingInfoDat    []byte
	WeakBindingLen    uint32
	WeakBindingOffset uint64
	WeakBindingDat    []byte
	LazyBindingLen    uint32
	LazyBindingOffset uint64
	LazyBindingDat    []byte
	ExportInfoLen     uint32
	ExportInfoOffset  uint64
	ExportInfoDat     []byte
}

// A Segment represents a Mach-O 32-bit or 64-bit load segment command.
type Segment struct {
	LoadBytes
	SegmentHeader

	// Embed ReaderAt for ReadAt method.
	// Do not embed SectionReader directly
	// to avoid having Read and Seek.
	// If a client wants Read and Seek it must use
	// Open() to avoid fighting over the seek offset
	// with other clients.
	io.ReaderAt
	sr *io.SectionReader
}

// Data reads and returns the contents of the segment.
func (s *Segment) Data() ([]byte, error) {
	dat := make([]byte, s.sr.Size())
	n, err := s.sr.ReadAt(dat, 0)
	if n == len(dat) {
		err = nil
	}
	return dat[0:n], err
}

// Open returns a new ReadSeeker reading the segment.
func (s *Segment) Open() io.ReadSeeker { return io.NewSectionReader(s.sr, 0, 1<<63-1) }

type SectionHeader struct {
	Name   string
	Seg    string
	Addr   uint64
	Size   uint64
	Offset uint32
	Align  uint32
	Reloff uint32
	Nreloc uint32
	Flags  uint32
}

// A Reloc represents a Mach-O relocation.
type Reloc struct {
	Addr  uint32
	Value uint32
	// when Scattered == false && Extern == true, Value is the symbol number.
	// when Scattered == false && Extern == false, Value is the section number.
	// when Scattered == true, Value is the value that this reloc refers to.
	Type      uint8
	Len       uint8 // 0=byte, 1=word, 2=long, 3=quad
	Pcrel     bool
	Extern    bool // valid if Scattered == false
	Scattered bool
}

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

// Data reads and returns the contents of the Mach-O section.
func (s *Section) Data() ([]byte, error) {
	dat := make([]byte, s.sr.Size())
	n, err := s.sr.ReadAt(dat, 0)
	if n == len(dat) {
		err = nil
	}
	return dat[0:n], err
}

// Open returns a new ReadSeeker reading the Mach-O section.
func (s *Section) Open() io.ReadSeeker { return io.NewSectionReader(s.sr, 0, 1<<63-1) }

// A Dylinker represents a Mach-O load dynamic library command.
type Dylinker struct {
	LoadBytes
	Name string
}

// A Dylib represents a Mach-O load dynamic library command.
type Dylib struct {
	LoadBytes
	Name           string
	Time           uint32
	CurrentVersion uint32
	CompatVersion  uint32
}

// A Symtab represents a Mach-O symbol table command.
type Symtab struct {
	LoadBytes
	SymtabCmd
	Syms         []Symbol
	RawSymtab    []byte
	RawStringtab []byte
}

// A Dysymtab represents a Mach-O dynamic symbol table command.
type Dysymtab struct {
	LoadBytes
	DysymtabCmd
	IndirectSyms []uint32 // indices into Symtab.Syms
	RawDysymtab  []byte
}

// A Rpath represents a Mach-O rpath command.
type Rpath struct {
	LoadBytes
	Path string
}

// A Symbol is a Mach-O 32-bit or 64-bit symbol table entry.
type Symbol struct {
	Name  string
	Type  uint8
	Sect  uint8
	Desc  uint16
	Value uint64
}

/*
 * Mach-O reader
 */

// FormatError is returned by some operations if the data does
// not have the correct format for an object file.
type FormatError struct {
	off int64
	msg string
	val interface{}
}

func (e *FormatError) Error() string {
	msg := e.msg
	if e.val != nil {
		msg += fmt.Sprintf(" '%v'", e.val)
	}
	msg += fmt.Sprintf(" in record at byte %#x", e.off)
	return msg
}

// Open opens the named file using os.Open and prepares it for use as a Mach-O binary.
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

// NewFile creates a new macho.File for accessing a Mach-o binary file in an underlying reader.
func NewFile(r io.ReaderAt) (*File, error) {
	return newFileInternal(r, false)
}

// NewFileFromMemory creates a new macho.File for accessing a Mach-O binary in-memory image in an underlying reader.
func NewFileFromMemory(r io.ReaderAt) (*File, error) {
	return newFileInternal(r, true)
}

// NewFile creates a new File for accessing a PE binary in an underlying reader.
func newFileInternal(r io.ReaderAt, memoryMode bool) (*File, error) {

	f := new(File)
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	// Read and decode Mach magic to determine byte order, size.
	// Magic32 and Magic64 differ only in the bottom bit.
	var ident [4]byte
	if _, err := r.ReadAt(ident[0:], 0); err != nil {
		return nil, err
	}
	be := binary.BigEndian.Uint32(ident[0:])
	le := binary.LittleEndian.Uint32(ident[0:])
	switch Magic32 &^ 1 {
	case be &^ 1:
		f.ByteOrder = binary.BigEndian
		f.Magic = be
	case le &^ 1:
		f.ByteOrder = binary.LittleEndian
		f.Magic = le
	default:
		return nil, &FormatError{0, "invalid magic number", nil}
	}

	// Read entire file header.
	if err := binary.Read(sr, f.ByteOrder, &f.FileHeader); err != nil {
		return nil, err
	}

	// Then load commands.
	offset := int64(fileHeaderSize32)
	if f.Magic == Magic64 {
		offset = fileHeaderSize64
	}
	dat := make([]byte, f.Cmdsz)
	if _, err := r.ReadAt(dat, offset); err != nil {
		return nil, err
	}
	f.Loads = make([]Load, f.Ncmd)
	bo := f.ByteOrder
	for i := range f.Loads {
		// Each load command begins with uint32 command and length.
		if len(dat) < 8 {
			return nil, &FormatError{offset, "command block too small", nil}
		}
		cmd, siz := LoadCmd(bo.Uint32(dat[0:4])), bo.Uint32(dat[4:8])
		if siz < 8 || siz > uint32(len(dat)) {
			return nil, &FormatError{offset, "invalid command block size", nil}
		}
		var cmddat []byte
		cmddat, dat = dat[0:siz], dat[siz:]
		offset += int64(siz)
		var s *Segment
		//fmt.Printf("LoadCmdVal: %+v\n", cmd)
		switch cmd {
		default:
			f.Loads[i] = LoadBytes(cmddat)

		case LoadCmdRpath:
			var hdr RpathCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &hdr); err != nil {
				return nil, err
			}
			l := new(Rpath)
			if hdr.Path >= uint32(len(cmddat)) {
				return nil, &FormatError{offset, "invalid path in rpath command", hdr.Path}
			}
			l.Path = cstring(cmddat[hdr.Path:])
			l.LoadBytes = LoadBytes(cmddat)
			f.Loads[i] = l

		case LoadCmdDylinker:
			var hdr DylinkerCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &hdr); err != nil {
				return nil, err
			}
			l := new(Dylinker)
			if hdr.Name >= uint32(len(cmddat)) {
				return nil, &FormatError{offset, "invalid name in dynamic library command", hdr.Name}
			}
			l.Name = cstring(cmddat[hdr.Name:])
			l.LoadBytes = LoadBytes(cmddat)
			f.Loads[i] = l

		case LoadCmdDylib:
			var hdr DylibCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &hdr); err != nil {
				return nil, err
			}
			l := new(Dylib)
			if hdr.Name >= uint32(len(cmddat)) {
				return nil, &FormatError{offset, "invalid name in dynamic library command", hdr.Name}
			}
			l.Name = cstring(cmddat[hdr.Name:])
			l.Time = hdr.Time
			l.CurrentVersion = hdr.CurrentVersion
			l.CompatVersion = hdr.CompatVersion
			l.LoadBytes = LoadBytes(cmddat)
			f.Loads[i] = l

		case LoadCmdSymtab:
			var hdr SymtabCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &hdr); err != nil {
				return nil, err
			}
			strtab := make([]byte, hdr.Strsize)

			var linkeditAddr, textAddr, linkeditOffset int64
			if !memoryMode {
				if _, err := r.ReadAt(strtab, int64(hdr.Stroff)); err != nil {
					return nil, err
				}
			} else {
				// in memory, we have to translate the file offsets for strtab/symtab into offsets into LINKEDIT segment
				for _, load := range f.Loads {
					switch segment := load.(type) {
					case *Segment:
						if segment == nil {
							continue
						}
						if segment.Name == "__LINKEDIT" {
							linkeditAddr = int64(segment.Addr)
							linkeditOffset = int64(segment.Offset)
						} else if segment.Name == "__TEXT" {
							textAddr = int64(segment.Addr)
						}
					}
				}
				strtabAddr := (linkeditAddr - textAddr) + (int64(hdr.Stroff) - linkeditOffset)
				if _, err := r.ReadAt(strtab, strtabAddr); err != nil {
					return nil, err
				}
			}

			var symsz int
			if f.Magic == Magic64 {
				symsz = 16
			} else {
				symsz = 12
			}
			symdat := make([]byte, int(hdr.Nsyms)*symsz)

			if !memoryMode {
				if _, err := r.ReadAt(symdat, int64(hdr.Symoff)); err != nil {
					return nil, err
				}
			} else {
				if _, err := r.ReadAt(symdat, (linkeditAddr-textAddr)+(int64(hdr.Symoff)-linkeditOffset)); err != nil {
					return nil, err
				}
			}

			st, err := f.parseSymtab(symdat, strtab, cmddat, &hdr, offset)
			if err != nil {
				return nil, err
			}
			f.Loads[i] = st
			f.Symtab = st
			f.Symtab.Symoff = hdr.Symoff
			f.Symtab.Stroff = hdr.Stroff

		case LoadCmdSignature:
			var sigCmd SigBlockCmd
			s := bytes.NewReader(cmddat)
			if err := binary.Read(s, bo, &sigCmd); err != nil {
				return nil, err
			}
			//fmt.Printf("SigData: %+v\n", sigCmd)
			sig := make([]byte, sigCmd.Sigsize)
			if _, err := r.ReadAt(sig, int64(sigCmd.Sigoff)); err != nil {
				return nil, err
			}
			var block SigBlock
			block.Offset = uint64(sigCmd.Sigoff)
			block.Len = sigCmd.Sigsize
			block.RawDat = sig
			f.SigBlock = &block
			f.Loads[i] = LoadBytes(cmddat)

		case LoadCmdFuncStarts:
			var funcCmd FuncStartsCmd
			fsc := bytes.NewReader(cmddat)
			if err := binary.Read(fsc, bo, &funcCmd); err != nil {
				return nil, err
			}
			//fmt.Printf("FuncStartsData: %+v\n", funcCmd)
			fs := make([]byte, funcCmd.Datasize)
			if _, err := r.ReadAt(fs, int64(funcCmd.Dataoff)); err != nil {
				return nil, err
			}
			var funcs FuncStarts
			funcs.Offset = uint64(funcCmd.Dataoff)
			funcs.Len = funcCmd.Datasize
			funcs.RawDat = fs
			f.FuncStarts = &funcs
			f.Loads[i] = LoadBytes(cmddat)

		case LoadCmdDataInCode:
			var dataCmd DataInCodeCmd
			dcc := bytes.NewReader(cmddat)
			if err := binary.Read(dcc, bo, &dataCmd); err != nil {
				return nil, err
			}
			//fmt.Printf("DataInCode: %+v\n", dataCmd)
			dc := make([]byte, dataCmd.Datasize)
			if _, err := r.ReadAt(dc, int64(dataCmd.Dataoff)); err != nil {
				return nil, err
			}
			var datacode DataInCode
			datacode.Offset = uint64(dataCmd.Dataoff)
			datacode.Len = dataCmd.Datasize
			datacode.RawDat = dc
			f.DataInCode = &datacode
			f.Loads[i] = LoadBytes(cmddat)

		case LoadCmdDylinkInfo:
			var dylinkInfoCmd DylinkInfoCmd
			dic := bytes.NewReader(cmddat)
			if err := binary.Read(dic, bo, &dylinkInfoCmd); err != nil {
				return nil, err
			}
			//fmt.Printf("LoadCmdDylinkInfo: %+v\n", dylinkInfoCmd)
			// Copy each section next
			var dylinkInfo DylinkInfo
			// Rebase deets
			if dylinkInfoCmd.Rebasesize > 0 {
				if !memoryMode { // this data is in LINKEDIT already
					rebase := make([]byte, dylinkInfoCmd.Rebasesize)
					if _, err := r.ReadAt(rebase, int64(dylinkInfoCmd.Rebaseoff)); err != nil {
						return nil, err
					}
					dylinkInfo.RebaseDat = rebase
				}
				dylinkInfo.RebaseLen = dylinkInfoCmd.Rebasesize
				dylinkInfo.RebaseOffset = uint64(dylinkInfoCmd.Rebaseoff)
			}
			// BindingInfo deets
			if dylinkInfoCmd.Bindinginfosize > 0 {
				if !memoryMode { // this data is in LINKEDIT already
					binding := make([]byte, dylinkInfoCmd.Bindinginfosize)
					if _, err := r.ReadAt(binding, int64(dylinkInfoCmd.Bindinginfooff)); err != nil {
						return nil, err
					}
					dylinkInfo.BindingInfoDat = binding
				}
				dylinkInfo.BindingInfoLen = dylinkInfoCmd.Bindinginfosize
				dylinkInfo.BindingInfoOffset = uint64(dylinkInfoCmd.Bindinginfooff)
			}
			// Weak deets
			if dylinkInfoCmd.Weakbindingsize > 0 {
				if !memoryMode { // this data is in LINKEDIT already
					weak := make([]byte, dylinkInfoCmd.Weakbindingsize)
					if _, err := r.ReadAt(weak, int64(dylinkInfoCmd.Weakbindingoff)); err != nil {
						return nil, err
					}
					dylinkInfo.WeakBindingDat = weak
				}
				dylinkInfo.WeakBindingLen = dylinkInfoCmd.Weakbindingsize
				dylinkInfo.WeakBindingOffset = uint64(dylinkInfoCmd.Weakbindingoff)
			}
			// Lazy deets
			if dylinkInfoCmd.Lazybindingsize > 0 {
				if !memoryMode { // this data is in LINKEDIT already
					lazy := make([]byte, dylinkInfoCmd.Lazybindingsize)
					if _, err := r.ReadAt(lazy, int64(dylinkInfoCmd.Lazybindingoff)); err != nil {
						return nil, err
					}
					dylinkInfo.LazyBindingDat = lazy
				}
				dylinkInfo.LazyBindingLen = dylinkInfoCmd.Lazybindingsize
				dylinkInfo.LazyBindingOffset = uint64(dylinkInfoCmd.Lazybindingoff)
			}
			// ExportInfo deets
			if dylinkInfoCmd.Exportinfosize > 0 {
				if !memoryMode { // this data is in LINKEDIT already
					export := make([]byte, dylinkInfoCmd.Exportinfosize)
					if _, err := r.ReadAt(export, int64(dylinkInfoCmd.Exportinfooff)); err != nil {
						return nil, err
					}
					dylinkInfo.ExportInfoDat = export
				}
				dylinkInfo.ExportInfoLen = dylinkInfoCmd.Exportinfosize
				dylinkInfo.ExportInfoOffset = uint64(dylinkInfoCmd.Exportinfooff)
			}
			// Finalize the object
			f.DylinkInfo = &dylinkInfo
			f.Loads[i] = LoadBytes(cmddat)

		case LoadCmdDysymtab:
			var hdr DysymtabCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &hdr); err != nil {
				return nil, err
			}
			dat := make([]byte, hdr.Nindirectsyms*4)
			if _, err := r.ReadAt(dat, int64(hdr.Indirectsymoff)); err != nil {
				return nil, err
			}
			x := make([]uint32, hdr.Nindirectsyms)
			if err := binary.Read(bytes.NewReader(dat), bo, x); err != nil {
				return nil, err
			}
			st := new(Dysymtab)
			st.LoadBytes = LoadBytes(cmddat)
			st.DysymtabCmd = hdr
			st.IndirectSyms = x
			f.Loads[i] = st
			f.Dysymtab = st
			f.Dysymtab.Indirectsymoff = hdr.Indirectsymoff
			f.Dysymtab.RawDysymtab = dat

		case LoadCmdSegment:
			var seg32 Segment32
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &seg32); err != nil {
				return nil, err
			}
			s = new(Segment)
			s.LoadBytes = cmddat
			s.Cmd = cmd
			s.Len = siz
			s.Name = cstring(seg32.Name[0:])
			s.Addr = uint64(seg32.Addr)
			s.Memsz = uint64(seg32.Memsz)
			s.Offset = uint64(seg32.Offset)
			s.Filesz = uint64(seg32.Filesz)
			s.Maxprot = seg32.Maxprot
			s.Prot = seg32.Prot
			s.Nsect = seg32.Nsect
			s.Flag = seg32.Flag
			if uint64((seg32.Offset + seg32.Filesz)) > FinalSegEnd {
				FinalSegEnd = uint64((seg32.Offset + seg32.Filesz))
			}
			f.Loads[i] = s
			for i := 0; i < int(s.Nsect); i++ {
				var sh32 Section32
				if err := binary.Read(b, bo, &sh32); err != nil {
					return nil, err
				}
				sh := new(Section)
				sh.Name = cstring(sh32.Name[0:])
				sh.Seg = cstring(sh32.Seg[0:])
				sh.Addr = uint64(sh32.Addr)
				sh.Size = uint64(sh32.Size)
				sh.Offset = sh32.Offset
				sh.Align = sh32.Align
				sh.Reloff = sh32.Reloff
				sh.Nreloc = sh32.Nreloc
				sh.Flags = sh32.Flags
				if err := f.pushSection(sh, r); err != nil {
					return nil, err
				}
			}

		case LoadCmdSegment64:
			var seg64 Segment64
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &seg64); err != nil {
				return nil, err
			}
			s = new(Segment)
			s.LoadBytes = cmddat
			s.Cmd = cmd
			s.Len = siz
			s.Name = cstring(seg64.Name[0:])
			s.Addr = seg64.Addr
			s.Memsz = seg64.Memsz
			s.Offset = seg64.Offset
			s.Filesz = seg64.Filesz
			s.Maxprot = seg64.Maxprot
			s.Prot = seg64.Prot
			s.Nsect = seg64.Nsect
			s.Flag = seg64.Flag
			if uint64((seg64.Offset + seg64.Filesz)) > FinalSegEnd {
				FinalSegEnd = uint64((seg64.Offset + seg64.Filesz))
			}
			f.Loads[i] = s
			for i := 0; i < int(s.Nsect); i++ {
				var sh64 Section64
				if err := binary.Read(b, bo, &sh64); err != nil {
					return nil, err
				}
				sh := new(Section)
				sh.Name = cstring(sh64.Name[0:])
				sh.Seg = cstring(sh64.Seg[0:])
				sh.Addr = sh64.Addr
				sh.Size = sh64.Size
				sh.Offset = sh64.Offset
				sh.Align = sh64.Align
				sh.Reloff = sh64.Reloff
				sh.Nreloc = sh64.Nreloc
				sh.Flags = sh64.Flags
				if err := f.pushSection(sh, r); err != nil {
					return nil, err
				}
			}

		//case LoadCmdUnixThread:
		// todo: do we have to support thread_command here for older binaries? or is the LC_MAIN handling backwards compatible?

		case LoadCmdMain:
			var entryPoint EntryPointCmd
			b := bytes.NewReader(cmddat)
			if err := binary.Read(b, bo, &entryPoint); err != nil {
				return nil, err
			}
			f.EntryPoint = entryPoint.EntryOff
		}
		if s != nil {
			if !memoryMode {
				s.sr = io.NewSectionReader(r, int64(s.Offset), int64(s.Filesz))
			} else {
				s.sr = io.NewSectionReader(r, int64(s.Addr), int64(s.Filesz))
			}
			s.ReaderAt = s.sr
		}
	}
	return f, nil
}

func (f *File) parseSymtab(symdat, strtab, cmddat []byte, hdr *SymtabCmd, offset int64) (*Symtab, error) {
	bo := f.ByteOrder
	symtab := make([]Symbol, hdr.Nsyms)
	b := bytes.NewReader(symdat)
	for i := range symtab {
		var n Nlist64
		if f.Magic == Magic64 {
			if err := binary.Read(b, bo, &n); err != nil {
				return nil, err
			}
		} else {
			var n32 Nlist32
			if err := binary.Read(b, bo, &n32); err != nil {
				return nil, err
			}
			n.Name = n32.Name
			n.Type = n32.Type
			n.Sect = n32.Sect
			n.Desc = n32.Desc
			n.Value = uint64(n32.Value)
		}
		sym := &symtab[i]
		if n.Name >= uint32(len(strtab)) {
			return nil, &FormatError{offset, "invalid name in symbol table", n.Name}
		}
		sym.Name = cstring(strtab[n.Name:])
		sym.Type = n.Type
		sym.Sect = n.Sect
		sym.Desc = n.Desc
		sym.Value = n.Value
	}
	st := new(Symtab)
	st.LoadBytes = LoadBytes(cmddat)
	st.Syms = symtab
	st.RawSymtab = symdat
	st.RawStringtab = strtab
	return st, nil
}

type relocInfo struct {
	Addr   uint32
	Symnum uint32
}

func (f *File) pushSection(sh *Section, r io.ReaderAt) error {
	f.Sections = append(f.Sections, sh)
	sh.sr = io.NewSectionReader(r, int64(sh.Offset), int64(sh.Size))
	sh.ReaderAt = sh.sr

	if sh.Nreloc > 0 {
		reldat := make([]byte, int(sh.Nreloc)*8)
		if _, err := r.ReadAt(reldat, int64(sh.Reloff)); err != nil {
			return err
		}
		b := bytes.NewReader(reldat)

		bo := f.ByteOrder

		sh.Relocs = make([]Reloc, sh.Nreloc)
		for i := range sh.Relocs {
			rel := &sh.Relocs[i]

			var ri relocInfo
			if err := binary.Read(b, bo, &ri); err != nil {
				return err
			}

			if ri.Addr&(1<<31) != 0 { // scattered
				rel.Addr = ri.Addr & (1<<24 - 1)
				rel.Type = uint8((ri.Addr >> 24) & (1<<4 - 1))
				rel.Len = uint8((ri.Addr >> 28) & (1<<2 - 1))
				rel.Pcrel = ri.Addr&(1<<30) != 0
				rel.Value = ri.Symnum
				rel.Scattered = true
			} else {
				switch bo {
				case binary.LittleEndian:
					rel.Addr = ri.Addr
					rel.Value = ri.Symnum & (1<<24 - 1)
					rel.Pcrel = ri.Symnum&(1<<24) != 0
					rel.Len = uint8((ri.Symnum >> 25) & (1<<2 - 1))
					rel.Extern = ri.Symnum&(1<<27) != 0
					rel.Type = uint8((ri.Symnum >> 28) & (1<<4 - 1))
				case binary.BigEndian:
					rel.Addr = ri.Addr
					rel.Value = ri.Symnum >> 8
					rel.Pcrel = ri.Symnum&(1<<7) != 0
					rel.Len = uint8((ri.Symnum >> 5) & (1<<2 - 1))
					rel.Extern = ri.Symnum&(1<<4) != 0
					rel.Type = uint8(ri.Symnum & (1<<4 - 1))
				default:
					panic("unreachable")
				}
			}
		}
	}

	return nil
}

func cstring(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i == -1 {
		i = len(b)
	}
	return string(b[0:i])
}

// Segment returns the first Segment with the given name, or nil if no such segment exists.
func (f *File) Segment(name string) *Segment {
	for _, l := range f.Loads {
		if s, ok := l.(*Segment); ok && s.Name == name {
			return s
		}
	}
	return nil
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

// DWARF returns the DWARF debug information for the Mach-O file.
func (f *File) DWARF() (*dwarf.Data, error) {
	dwarfSuffix := func(s *Section) string {
		switch {
		case strings.HasPrefix(s.Name, "__debug_"):
			return s.Name[8:]
		case strings.HasPrefix(s.Name, "__zdebug_"):
			return s.Name[9:]
		default:
			return ""
		}

	}
	sectionData := func(s *Section) ([]byte, error) {
		b, err := s.Data()
		if err != nil && uint64(len(b)) < s.Size {
			return nil, err
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

// ImportedSymbols returns the names of all symbols
// referred to by the binary f that are expected to be
// satisfied by other libraries at dynamic load time.
func (f *File) ImportedSymbols() ([]string, error) {
	if f.Dysymtab == nil || f.Symtab == nil {
		return nil, &FormatError{0, "missing symbol table", nil}
	}

	st := f.Symtab
	dt := f.Dysymtab
	var all []string
	for _, s := range st.Syms[dt.Iundefsym : dt.Iundefsym+dt.Nundefsym] {
		all = append(all, s.Name)
	}
	return all, nil
}

// ImportedLibraries returns the paths of all libraries
// referred to by the binary f that are expected to be
// linked with the binary at dynamic link time.
func (f *File) ImportedLibraries() ([]string, error) {
	var all []string
	for _, l := range f.Loads {
		if lib, ok := l.(*Dylib); ok {
			all = append(all, lib.Name)
		}
	}
	return all, nil
}
