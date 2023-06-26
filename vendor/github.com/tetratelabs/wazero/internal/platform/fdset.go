package platform

// Set adds the given fd to the set.
func (f *FdSet) Set(fd int) {
	f.Bits[fd/nfdbits] |= (1 << (uintptr(fd) % nfdbits))
}

// Clear removes the given fd from the set.
func (f *FdSet) Clear(fd int) {
	f.Bits[fd/nfdbits] &^= (1 << (uintptr(fd) % nfdbits))
}

// IsSet returns true when fd is in the set.
func (f *FdSet) IsSet(fd int) bool {
	return f.Bits[fd/nfdbits]&(1<<(uintptr(fd)%nfdbits)) != 0
}

// Zero clears the set.
func (f *FdSet) Zero() {
	for i := range f.Bits {
		f.Bits[i] = 0
	}
}
