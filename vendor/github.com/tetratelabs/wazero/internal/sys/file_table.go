package sys

import "math/bits"

// FileTable is a data structure mapping 32 bit file descriptor to file objects.
//
// The data structure optimizes for memory density and lookup performance,
// trading off compute at insertion time. This is a useful compromise for the
// use cases we employ it with: files are usually read or written a lot more
// often than they are opened, each operation requires a table lookup so we are
// better off spending extra compute to insert files in the table in order to
// get cheaper lookups. Memory efficiency is also crucial to support scaling
// with programs that open thousands of files: having a high or non-linear
// memory-to-item ratio could otherwise be used as an attack vector by malicous
// applications attempting to damage performance of the host.
type FileTable struct {
	masks []uint64
	files []*FileEntry
}

// Len returns the number of files stored in the table.
func (t *FileTable) Len() (n int) {
	// We could make this a O(1) operation if we cached the number of files in
	// the table. More state usually means more problems, so until we have a
	// clear need for this, the simple implementation may be a better trade off.
	for _, mask := range t.masks {
		n += bits.OnesCount64(mask)
	}
	return n
}

// Grow ensures that t has enough room for n files, potentially reallocating the
// internal buffers if their capacity was too small to hold this many files.
func (t *FileTable) Grow(n int) {
	// Round up to a multiple of 64 since this is the smallest increment due to
	// using 64 bits masks.
	n = (n*64 + 63) / 64

	if n > len(t.masks) {
		masks := make([]uint64, n)
		copy(masks, t.masks)

		files := make([]*FileEntry, n*64)
		copy(files, t.files)

		t.masks = masks
		t.files = files
	}
}

// Insert inserts the given file to the table, returning the fd that it is
// mapped to.
//
// The method does not perform deduplication, it is possible for the same file
// to be inserted multiple times, each insertion will return a different fd.
func (t *FileTable) Insert(file *FileEntry) (fd uint32) {
	offset := 0
insert:
	// Note: this loop could be made a lot more efficient using vectorized
	// operations: 256 bits vector registers would yield a theoretical 4x
	// speed up (e.g. using AVX2).
	for index, mask := range t.masks[offset:] {
		if ^mask != 0 { // not full?
			shift := bits.TrailingZeros64(^mask)
			index += offset
			fd = uint32(index)*64 + uint32(shift)
			t.files[fd] = file
			t.masks[index] = mask | uint64(1<<shift)
			return fd
		}
	}

	offset = len(t.masks)
	n := 2 * len(t.masks)
	if n == 0 {
		n = 1
	}

	t.Grow(n)
	goto insert
}

// Lookup returns the file associated with the given fd (may be nil).
func (t *FileTable) Lookup(fd uint32) (file *FileEntry, found bool) {
	if i := int(fd); i >= 0 && i < len(t.files) {
		index := uint(fd) / 64
		shift := uint(fd) % 64
		if (t.masks[index] & (1 << shift)) != 0 {
			file, found = t.files[i], true
		}
	}
	return
}

// Delete deletes the file stored at the given fd from the table.
func (t *FileTable) Delete(fd uint32) {
	if index, shift := fd/64, fd%64; int(index) < len(t.masks) {
		mask := t.masks[index]
		if (mask & (1 << shift)) != 0 {
			t.files[fd] = nil
			t.masks[index] = mask & ^uint64(1<<shift)
		}
	}
}

// Range calls f for each file and its associated fd in the table. The function
// f might return false to interupt the iteration.
func (t *FileTable) Range(f func(uint32, *FileEntry) bool) {
	for i, mask := range t.masks {
		if mask == 0 {
			continue
		}
		for j := uint32(0); j < 64; j++ {
			if (mask & (1 << j)) == 0 {
				continue
			}
			if fd := uint32(i)*64 + j; !f(fd, t.files[fd]) {
				return
			}
		}
	}
}

// Reset clears the content of the table.
func (t *FileTable) Reset() {
	for i := range t.masks {
		t.masks[i] = 0
	}
	for i := range t.files {
		t.files[i] = nil
	}
}
