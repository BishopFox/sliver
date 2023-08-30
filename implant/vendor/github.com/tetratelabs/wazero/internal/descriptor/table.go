package descriptor

import "math/bits"

// Table is a data structure mapping 32 bit descriptor to items.
//
// # Negative keys are invalid.
//
// Negative keys (e.g. -1) are invalid inputs and will return a corresponding
// not-found value. This matches POSIX behavior of file descriptors.
// See https://pubs.opengroup.org/onlinepubs/9699919799/functions/dirfd.html#tag_16_90
//
// # Data structure design
//
// The data structure optimizes for memory density and lookup performance,
// trading off compute at insertion time. This is a useful compromise for the
// use cases we employ it with: items are usually accessed a lot more often
// than they are inserted, each operation requires a table lookup, so we are
// better off spending extra compute to insert items in the table in order to
// get cheaper lookups. Memory efficiency is also crucial to support scaling
// with programs that maintain thousands of items: having a high or non-linear
// memory-to-item ratio could otherwise be used as an attack vector by
// malicious applications attempting to damage performance of the host.
type Table[Key ~int32, Item any] struct {
	masks []uint64
	items []Item
}

// Len returns the number of items stored in the table.
func (t *Table[Key, Item]) Len() (n int) {
	// We could make this a O(1) operation if we cached the number of items in
	// the table. More state usually means more problems, so until we have a
	// clear need for this, the simple implementation may be a better trade off.
	for _, mask := range t.masks {
		n += bits.OnesCount64(mask)
	}
	return n
}

// grow ensures that t has enough room for n items, potentially reallocating the
// internal buffers if their capacity was too small to hold this many items.
func (t *Table[Key, Item]) grow(n int) {
	// Round up to a multiple of 64 since this is the smallest increment due to
	// using 64 bits masks.
	n = (n*64 + 63) / 64

	if n > len(t.masks) {
		masks := make([]uint64, n)
		copy(masks, t.masks)

		items := make([]Item, n*64)
		copy(items, t.items)

		t.masks = masks
		t.items = items
	}
}

// Insert inserts the given item to the table, returning the key that it is
// mapped to or false if the table was full.
//
// The method does not perform deduplication, it is possible for the same item
// to be inserted multiple times, each insertion will return a different key.
func (t *Table[Key, Item]) Insert(item Item) (key Key, ok bool) {
	offset := 0
insert:
	// Note: this loop could be made a lot more efficient using vectorized
	// operations: 256 bits vector registers would yield a theoretical 4x
	// speed up (e.g. using AVX2).
	for index, mask := range t.masks[offset:] {
		if ^mask != 0 { // not full?
			shift := bits.TrailingZeros64(^mask)
			index += offset
			key = Key(index)*64 + Key(shift)
			t.items[key] = item
			t.masks[index] = mask | uint64(1<<shift)
			return key, key >= 0
		}
	}

	offset = len(t.masks)
	n := 2 * len(t.masks)
	if n == 0 {
		n = 1
	}

	t.grow(n)
	goto insert
}

// Lookup returns the item associated with the given key (may be nil).
func (t *Table[Key, Item]) Lookup(key Key) (item Item, found bool) {
	if key < 0 { // invalid key
		return
	}
	if i := int(key); i >= 0 && i < len(t.items) {
		index := uint(key) / 64
		shift := uint(key) % 64
		if (t.masks[index] & (1 << shift)) != 0 {
			item, found = t.items[i], true
		}
	}
	return
}

// InsertAt inserts the given `item` at the item descriptor `key`. This returns
// false if the insert was impossible due to negative key.
func (t *Table[Key, Item]) InsertAt(item Item, key Key) bool {
	if key < 0 {
		return false
	}
	if diff := int(key) - t.Len(); diff > 0 {
		t.grow(diff)
	}
	index := uint(key) / 64
	shift := uint(key) % 64
	t.masks[index] |= 1 << shift
	t.items[key] = item
	return true
}

// Delete deletes the item stored at the given key from the table.
func (t *Table[Key, Item]) Delete(key Key) {
	if key < 0 { // invalid key
		return
	}
	if index, shift := key/64, key%64; int(index) < len(t.masks) {
		mask := t.masks[index]
		if (mask & (1 << shift)) != 0 {
			var zero Item
			t.items[key] = zero
			t.masks[index] = mask & ^uint64(1<<shift)
		}
	}
}

// Range calls f for each item and its associated key in the table. The function
// f might return false to interupt the iteration.
func (t *Table[Key, Item]) Range(f func(Key, Item) bool) {
	for i, mask := range t.masks {
		if mask == 0 {
			continue
		}
		for j := Key(0); j < 64; j++ {
			if (mask & (1 << j)) == 0 {
				continue
			}
			if key := Key(i)*64 + j; !f(key, t.items[key]) {
				return
			}
		}
	}
}

// Reset clears the content of the table.
func (t *Table[Key, Item]) Reset() {
	for i := range t.masks {
		t.masks[i] = 0
	}
	var zero Item
	for i := range t.items {
		t.items[i] = zero
	}
}
