package elf

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// applyRelocations applies relocations to dst. rels is a relocations section
// in REL or RELA format.
func (f *File) applyRelocations(dst []byte, rels []byte) error {
	switch {
	case f.Class == ELFCLASS64 && f.Machine == EM_X86_64:
		return f.applyRelocationsAMD64(dst, rels)
	case f.Class == ELFCLASS32 && f.Machine == EM_386:
		return f.applyRelocations386(dst, rels)
	case f.Class == ELFCLASS32 && f.Machine == EM_ARM:
		return f.applyRelocationsARM(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_AARCH64:
		return f.applyRelocationsARM64(dst, rels)
	case f.Class == ELFCLASS32 && f.Machine == EM_PPC:
		return f.applyRelocationsPPC(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_PPC64:
		return f.applyRelocationsPPC64(dst, rels)
	case f.Class == ELFCLASS32 && f.Machine == EM_MIPS:
		return f.applyRelocationsMIPS(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_MIPS:
		return f.applyRelocationsMIPS64(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_RISCV:
		return f.applyRelocationsRISCV64(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_S390:
		return f.applyRelocationss390x(dst, rels)
	case f.Class == ELFCLASS64 && f.Machine == EM_SPARCV9:
		return f.applyRelocationsSPARC64(dst, rels)
	default:
		return errors.New("applyRelocations: not implemented")
	}
}

func (f *File) applyRelocationsAMD64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_X86_64(rela.Info & 0xffff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		// There are relocations, so this must be a normal
		// object file, and we only look at section symbols,
		// so we assume that the symbol value is 0.

		switch t {
		case R_X86_64_64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], uint64(rela.Addend))
		case R_X86_64_32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}

func (f *File) applyRelocations386(dst []byte, rels []byte) error {
	// 8 is the size of Rel32.
	if len(rels)%8 != 0 {
		return errors.New("length of relocation section is not a multiple of 8")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rel Rel32

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rel)
		symNo := rel.Info >> 8
		t := R_386(rel.Info & 0xff)

		if symNo == 0 || symNo > uint32(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]

		if t == R_386_32 {
			if rel.Off+4 >= uint32(len(dst)) {
				continue
			}
			val := f.ByteOrder.Uint32(dst[rel.Off : rel.Off+4])
			val += uint32(sym.Value)
			f.ByteOrder.PutUint32(dst[rel.Off:rel.Off+4], val)
		}
	}

	return nil
}

func (f *File) applyRelocationsARM(dst []byte, rels []byte) error {
	// 8 is the size of Rel32.
	if len(rels)%8 != 0 {
		return errors.New("length of relocation section is not a multiple of 8")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rel Rel32

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rel)
		symNo := rel.Info >> 8
		t := R_ARM(rel.Info & 0xff)

		if symNo == 0 || symNo > uint32(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]

		switch t {
		case R_ARM_ABS32:
			if rel.Off+4 >= uint32(len(dst)) {
				continue
			}
			val := f.ByteOrder.Uint32(dst[rel.Off : rel.Off+4])
			val += uint32(sym.Value)
			f.ByteOrder.PutUint32(dst[rel.Off:rel.Off+4], val)
		}
	}

	return nil
}

func (f *File) applyRelocationsARM64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_AARCH64(rela.Info & 0xffff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		// There are relocations, so this must be a normal
		// object file, and we only look at section symbols,
		// so we assume that the symbol value is 0.

		switch t {
		case R_AARCH64_ABS64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], uint64(rela.Addend))
		case R_AARCH64_ABS32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}

func (f *File) applyRelocationsPPC(dst []byte, rels []byte) error {
	// 12 is the size of Rela32.
	if len(rels)%12 != 0 {
		return errors.New("length of relocation section is not a multiple of 12")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela32

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 8
		t := R_PPC(rela.Info & 0xff)

		if symNo == 0 || symNo > uint32(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		switch t {
		case R_PPC_ADDR32:
			if rela.Off+4 >= uint32(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}

func (f *File) applyRelocationsPPC64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_PPC64(rela.Info & 0xffff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		switch t {
		case R_PPC64_ADDR64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], uint64(rela.Addend))
		case R_PPC64_ADDR32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}

func (f *File) applyRelocationsMIPS(dst []byte, rels []byte) error {
	// 8 is the size of Rel32.
	if len(rels)%8 != 0 {
		return errors.New("length of relocation section is not a multiple of 8")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rel Rel32

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rel)
		symNo := rel.Info >> 8
		t := R_MIPS(rel.Info & 0xff)

		if symNo == 0 || symNo > uint32(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]

		switch t {
		case R_MIPS_32:
			if rel.Off+4 >= uint32(len(dst)) {
				continue
			}
			val := f.ByteOrder.Uint32(dst[rel.Off : rel.Off+4])
			val += uint32(sym.Value)
			f.ByteOrder.PutUint32(dst[rel.Off:rel.Off+4], val)
		}
	}

	return nil
}

func (f *File) applyRelocationsMIPS64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		var symNo uint64
		var t R_MIPS
		if f.ByteOrder == binary.BigEndian {
			symNo = rela.Info >> 32
			t = R_MIPS(rela.Info & 0xff)
		} else {
			symNo = rela.Info & 0xffffffff
			t = R_MIPS(rela.Info >> 56)
		}

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		switch t {
		case R_MIPS_64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], uint64(rela.Addend))
		case R_MIPS_32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}

func (f *File) applyRelocationsRISCV64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_RISCV(rela.Info & 0xffff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		switch SymType(sym.Info & 0xf) {
		case STT_SECTION, STT_NOTYPE:
			break
		default:
			continue
		}

		switch t {
		case R_RISCV_64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			val := sym.Value + uint64(rela.Addend)
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], val)
		case R_RISCV_32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			val := uint32(sym.Value) + uint32(rela.Addend)
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], val)
		}
	}

	return nil
}

func (f *File) applyRelocationss390x(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_390(rela.Info & 0xffff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		switch SymType(sym.Info & 0xf) {
		case STT_SECTION, STT_NOTYPE:
			break
		default:
			continue
		}

		switch t {
		case R_390_64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			val := sym.Value + uint64(rela.Addend)
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], val)
		case R_390_32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			val := uint32(sym.Value) + uint32(rela.Addend)
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], val)
		}
	}

	return nil
}

func (f *File) applyRelocationsSPARC64(dst []byte, rels []byte) error {
	// 24 is the size of Rela64.
	if len(rels)%24 != 0 {
		return errors.New("length of relocation section is not a multiple of 24")
	}

	symbols, _, err := f.getSymbols(SHT_SYMTAB)
	if err != nil {
		return err
	}

	b := bytes.NewReader(rels)
	var rela Rela64

	for b.Len() > 0 {
		binary.Read(b, f.ByteOrder, &rela)
		symNo := rela.Info >> 32
		t := R_SPARC(rela.Info & 0xff)

		if symNo == 0 || symNo > uint64(len(symbols)) {
			continue
		}
		sym := &symbols[symNo-1]
		if SymType(sym.Info&0xf) != STT_SECTION {
			// We don't handle non-section relocations for now.
			continue
		}

		switch t {
		case R_SPARC_64, R_SPARC_UA64:
			if rela.Off+8 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint64(dst[rela.Off:rela.Off+8], uint64(rela.Addend))
		case R_SPARC_32, R_SPARC_UA32:
			if rela.Off+4 >= uint64(len(dst)) || rela.Addend < 0 {
				continue
			}
			f.ByteOrder.PutUint32(dst[rela.Off:rela.Off+4], uint32(rela.Addend))
		}
	}

	return nil
}
