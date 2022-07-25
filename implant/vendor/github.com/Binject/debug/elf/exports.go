package elf

/*
  Any symbol in the dynamic symbol table (in .dynsym) for which .st_shndx == SHN_UNDEF
  (references undefined section) is an import, and every other symbol is defined and exported.
*/

// Export - describes a single export entry
type Export struct {
	Name           string
	VirtualAddress uint64
}

// Exports - gets exports
func (f *File) Exports() ([]Export, error) {

	var exports []Export
	symbols, err := f.DynamicSymbols()
	if err != nil {
		return nil, err
	}
	for _, s := range symbols {
		if s.Section != SHN_UNDEF {
			exports = append(exports, Export{
				Name:           s.Name,
				VirtualAddress: s.Value,
			})
		}
	}

	return exports, nil
}
