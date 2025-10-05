// Copyright (c) 2024-2025 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package bart provides a high-performance Balanced Routing Table (BART).
//
// BART is balanced in terms of memory usage and lookup time
// for longest-prefix match (LPM) queries on IPv4 and IPv6 addresses.
//
// Internally, BART is implemented as a multibit trie with a fixed stride of 8 bits.
// Each level node uses a fast mapping function (adapted from D. E. Knuths ART algorithm)
// to arrange all 256 possible prefixes in a complete binary tree structure.
//
// Instead of allocating full arrays, BART uses popcount-compressed sparse arrays
// and aggressive path compression. This results in up to 100x less memory usage
// than ART, while maintaining or even improving lookup speed.
//
// Lookup operations are entirely bit-vector based and rely on precomputed
// lookup tables. Because the data fits within 256-bit blocks, it allows
// for extremely efficient, cacheline-aligned access and is accelerated by
// CPU instructions such as POPCNT, LZCNT, and TZCNT.
//
// The fixed 256-bit representation (4x uint64) permits loop unrolling in hot paths,
// ensuring predictable and fast performance even under high routing load.
package bart

import (
	"iter"
	"net/netip"
	"sync"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
)

// Table represents a thread-safe IPv4 and IPv6 routing table with payload V.
//
// The zero value is ready to use.
//
// The Table is safe for concurrent reads, but concurrent reads and writes
// must be externally synchronized. Mutation via Insert/Delete requires locks,
// or alternatively, use ...Persist methods which return a modified copy
// without altering the original table (copy-on-write).
//
// A Table must not be copied by value; always pass by pointer.
type Table[V any] struct {
	// used by -copylocks checker from `go vet`.
	_ [0]sync.Mutex

	// the root nodes, implemented as popcount compressed multibit tries
	root4 node[V]
	root6 node[V]

	// the number of prefixes in the routing table
	size4 int
	size6 int
}

// rootNodeByVersion, root node getter for ip version.
func (t *Table[V]) rootNodeByVersion(is4 bool) *node[V] {
	if is4 {
		return &t.root4
	}

	return &t.root6
}

// maxDepthAndLastBits, get the max depth in the trie and remaining bits
// for a given CIDR at max depth.
//
// ATTENTION: Split the IP prefixes at 8bit borders, count from 0.
//
//	/7, /15, /23, /31, ..., /127
//
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32]
//	BitPos: [0-7],[8-15],[16-23],[24-31],[32-39],[40-47],[48-55],[56-63],...,[120-127],[128]
//
//	0.0.0.0/0          => maxDepth:  0, lastBits: 0 (default route)
//	0.0.0.0/7          => maxDepth:  0, lastBits: 7
//	0.0.0.0/8          => maxDepth:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/8         => maxDepth:  1, lastBits: 0 (possible fringe)
//	10.0.0.0/22        => maxDepth:  2, lastBits: 6
//	10.0.0.0/29        => maxDepth:  3, lastBits: 5
//	10.0.0.0/32        => maxDepth:  4, lastBits: 0 (possible fringe)
//
//	::/0               => maxDepth:  0, lastBits: 0 (default route)
//	::1/128            => maxDepth: 16, lastBits: 0 (possible fringe)
//	2001:db8::/42      => maxDepth:  5, lastBits: 2
//	2001:db8::/56      => maxDepth:  7, lastBits: 0 (possible fringe)
//
//	/32 and /128 are special, they never form a new node, they are always inserted
//	as path-compressed leaf.
//
// We are not splitting at /8, /16, ..., because this would mean that the
// first node would have 512 prefixes, 9 bits from [0-8]. All remaining nodes
// would then only have 8 bits from [9-16], [17-24], [25..32], ...
// but the algorithm would then require a variable length bitset.
//
// If you can commit to a fixed size of [4]uint64, then the algorithm is
// much faster due to modern CPUs.
//
// Perhaps a future Go version that supports SIMD instructions for the [4]uint64 vectors
// will make the algorithm even faster on suitable hardware.
func maxDepthAndLastBits(bits int) (maxDepth int, lastBits uint8) {
	// maxDepth:  range from 0..4 or 0..16 !ATTENTION: not 0..3 or 0..15
	// lastBits:  range from 0..7
	return bits >> 3, uint8(bits & 7)
}

// Insert adds a pfx to the tree, with given val.
// If pfx is already present in the tree, its value is set to val.
func (t *Table[V]) Insert(pfx netip.Prefix, val V) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	if exists := n.insertAtDepth(pfx, val, 0); exists {
		return
	}

	// true insert, update size
	t.sizeUpdate(is4, 1)
}

// Deprecated: use [Table.Modify] instead.
//
// Update or set the value at pfx with a callback function.
// The callback function is called with (value, found) and returns a new value.
//
// If the pfx does not already exist, it is set with the new value.
func (t *Table[V]) Update(pfx netip.Prefix, cb func(val V, found bool) V) (newVal V) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	n := t.rootNodeByVersion(is4)

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// last octet from prefix, update/insert prefix into node
		if depth == maxDepth {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.prefixes.Get(idx)
			newVal := cb(oldVal, existed)
			n.prefixes.InsertAt(idx, newVal)

			if !existed {
				t.sizeUpdate(is4, 1)
			}
			return newVal
		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed
			newVal := cb(zero, false)
			if isFringe(depth, bits) {
				n.children.InsertAt(octet, newFringeNode(newVal))
			} else {
				n.children.InsertAt(octet, newLeafNode(pfx, newVal))
			}
			t.sizeUpdate(is4, 1)
			return newVal
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.children.InsertAt(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// update existing value if prefix is fringe
			if isFringe(depth, bits) {
				kid.value = cb(kid.value, true)
				return kid.value
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Modify applies an insert, update, or delete operation for the value
// associated with the given prefix. The supplied callback decides the
// operation: it is called with the current value (or zero if not found)
// and a boolean indicating whether the prefix exists. The callback must
// return a new value and a delete flag: del == false inserts or updates,
// del == true deletes the entry if it exists (otherwise no-op). Modify
// returns the resulting value and a boolean indicating whether the
// entry was actually deleted.
//
// The operation is determined by the callback function, which is called with:
//
//	val:   the current value (or zero value if not found)
//	found: true if the prefix currently exists, false otherwise
//
// The callback returns:
//
//	val: the new value to insert or update (ignored if del == true)
//	del: true to delete the entry, false to insert or update
//
// Modify returns:
//
//	val:     the zero, old, or new value depending on the operation (see table)
//	deleted: true if the entry was deleted, false otherwise
//
// Summary:
//
//	Operation | cb-input        | cb-return       | Modify-return
//	---------------------------------------------------------------
//	No-op:    | (zero,   false) | (_,      true)  | (zero,   false)
//	Insert:   | (zero,   false) | (newVal, false) | (newVal, false)
//	Update:   | (oldVal, true)  | (newVal, false) | (oldVal, false)
//	Delete:   | (oldVal, true)  | (_,      true)  | (oldVal, true)
func (t *Table[V]) Modify(pfx netip.Prefix, cb func(val V, found bool) (_ V, del bool)) (_ V, deleted bool) {
	var zero V

	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	n := t.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*node[V]{}

	// find the proper trie node to update prefix
	for depth, octet := range octets {
		// push current node on stack for path recording
		stack[depth] = n

		// last octet from prefix, update/insert/delete prefix
		if depth == maxDepth {
			idx := art.PfxToIdx(octet, lastBits)

			oldVal, existed := n.prefixes.Get(idx)
			newVal, del := cb(oldVal, existed)

			// update size if necessary
			switch {
			case !existed && del: // no-op
				return zero, false

			case existed && del: // delete
				n.prefixes.DeleteAt(idx)
				t.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)
				return oldVal, true

			case !existed: // insert
				n.prefixes.InsertAt(idx, newVal)
				t.sizeUpdate(is4, 1)
				return newVal, false

			case existed: // update
				n.prefixes.InsertAt(idx, newVal)
				return oldVal, false

			default:
				panic("unreachable")
			}

		}

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// insert prefix path compressed

			newVal, del := cb(zero, false)
			if del {
				return zero, false // no-op
			}

			// insert
			if isFringe(depth, bits) {
				n.children.InsertAt(octet, newFringeNode(newVal))
			} else {
				n.children.InsertAt(octet, newLeafNode(pfx, newVal))
			}

			t.sizeUpdate(is4, 1)
			return newVal, false
		}

		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			oldVal := kid.value

			// update existing value if prefixes are equal
			if kid.prefix == pfx {
				newVal, del := cb(oldVal, true)

				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.children.DeleteAt(octet)

				t.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.children.InsertAt(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			oldVal := kid.value

			// update existing value if prefix is fringe
			if isFringe(depth, bits) {
				newVal, del := cb(kid.value, true)
				if !del {
					kid.value = newVal
					return oldVal, false // update
				}

				// delete
				n.children.DeleteAt(octet)

				t.sizeUpdate(is4, -1)
				n.purgeAndCompress(stack[:depth], octets, is4)

				return oldVal, true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (octet
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Deprecated: use [Table.Delete] instead.
func (t *Table[V]) GetAndDelete(pfx netip.Prefix) (val V, found bool) {
	return t.Delete(pfx)
}

// Delete the prefix and returns the associated payload for prefix and true if found
// or the zero value and false if prefix is not set in the routing table.
func (t *Table[V]) Delete(pfx netip.Prefix) (val V, found bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	n := t.rootNodeByVersion(is4)

	// record the nodes on the path to the deleted node, needed to purge
	// and/or path compress nodes after the deletion of a prefix
	stack := [maxTreeDepth]*node[V]{}

	// find the trie node
	for depth, octet := range octets {
		depth = depth & 0xf // BCE, Delete must be fast

		// push current node on stack for path recording
		stack[depth] = n

		if depth == maxDepth {
			// try to delete prefix in trie node
			val, found = n.prefixes.DeleteAt(art.PfxToIdx(octet, lastBits))
			if !found {
				return
			}

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)
			return val, true
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// if pfx is no fringe at this depth, fast exit
			if !isFringe(depth, bits) {
				return
			}

			// pfx is fringe at depth, delete fringe
			n.children.DeleteAt(octet)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		case *leafNode[V]:
			// Attention: pfx must be masked to be comparable!
			if kid.prefix != pfx {
				return
			}

			// prefix is equal leaf, delete leaf
			n.children.DeleteAt(octet)

			t.sizeUpdate(is4, -1)
			n.purgeAndCompress(stack[:depth], octets, is4)

			return kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	return
}

// Get returns the associated payload for prefix and true, or false if
// prefix is not set in the routing table.
func (t *Table[V]) Get(pfx netip.Prefix) (val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	// values derived from pfx
	ip := pfx.Addr()
	is4 := ip.Is4()
	bits := pfx.Bits()

	n := t.rootNodeByVersion(is4)

	maxDepth, lastBits := maxDepthAndLastBits(bits)

	octets := ip.AsSlice()

	// find the trie node
	for depth, octet := range octets {
		if depth == maxDepth {
			return n.prefixes.Get(art.PfxToIdx(octet, lastBits))
		}

		if !n.children.Test(octet) {
			return
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// reached a path compressed fringe, stop traversing
			if isFringe(depth, bits) {
				return kid.value, true
			}
			return

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix == pfx {
				return kid.value, true
			}
			return

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable")
}

// Contains does a route lookup for IP and
// returns true if any route matched.
//
// Contains does not return the value nor the prefix of the matching item,
// but as a test against a black- or whitelist it's often sufficient
// and even few nanoseconds faster than [Table.Lookup].
func (t *Table[V]) Contains(ip netip.Addr) bool {
	// if ip is invalid, Is4() returns false and AsSlice() returns nil
	is4 := ip.Is4()
	n := t.rootNodeByVersion(is4)

	for _, octet := range ip.AsSlice() {
		// for contains, any lpm match is good enough, no backtracking needed
		if n.prefixes.Len() != 0 && n.lpmTest(art.OctetToIdx(octet)) {
			return true
		}

		// stop traversing?
		if !n.children.Test(octet) {
			return false
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible octets below
			return true

		case *leafNode[V]:
			return kid.prefix.Contains(ip)

		default:
			panic("logic error, wrong node type")
		}
	}

	return false
}

// Lookup does a route lookup (longest prefix match) for IP and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) Lookup(ip netip.Addr) (val V, ok bool) {
	if !ip.IsValid() {
		return
	}

	is4 := ip.Is4()
	octets := ip.AsSlice()

	n := t.rootNodeByVersion(is4)

	// stack of the traversed nodes for fast backtracking, if needed
	stack := [maxTreeDepth]*node[V]{}

	// run variable, used after for loop
	var depth int
	var octet byte

LOOP:
	// find leaf node
	for depth, octet = range octets {
		depth = depth & 0xf // BCE, Lookup must be fast

		// push current node on stack for fast backtracking
		stack[depth] = n

		// go down in tight loop to last octet
		if !n.children.Test(octet) {
			// no more nodes below octet
			break LOOP
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *fringeNode[V]:
			// fringe is the default-route for all possible nodes below
			return kid.value, true

		case *leafNode[V]:
			if kid.prefix.Contains(ip) {
				return kid.value, true
			}
			// reached a path compressed prefix, stop traversing
			break LOOP

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack, bounds check eliminated
	for ; depth >= 0; depth-- {
		depth = depth & 0xf // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() != 0 {
			idx := art.OctetToIdx(octets[depth])
			// lpmGet(idx), manually inlined
			// --------------------------------------------------------------
			if topIdx, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
				return n.prefixes.MustGet(topIdx), true
			}
			// --------------------------------------------------------------
		}
	}

	return
}

// LookupPrefix does a route lookup (longest prefix match) for pfx and
// returns the associated value and true, or false if no route matched.
func (t *Table[V]) LookupPrefix(pfx netip.Prefix) (val V, ok bool) {
	_, val, ok = t.lookupPrefixLPM(pfx, false)
	return val, ok
}

// LookupPrefixLPM is similar to [Table.LookupPrefix],
// but it returns the lpm prefix in addition to value,ok.
//
// This method is about 20-30% slower than LookupPrefix and should only
// be used if the matching lpm entry is also required for other reasons.
//
// If LookupPrefixLPM is to be used for IP address lookups,
// they must be converted to /32 or /128 prefixes.
func (t *Table[V]) LookupPrefixLPM(pfx netip.Prefix) (lpmPfx netip.Prefix, val V, ok bool) {
	return t.lookupPrefixLPM(pfx, true)
}

func (t *Table[V]) lookupPrefixLPM(pfx netip.Prefix, withLPM bool) (lpmPfx netip.Prefix, val V, ok bool) {
	if !pfx.IsValid() {
		return
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	ip := pfx.Addr()
	bits := pfx.Bits()
	is4 := ip.Is4()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	n := t.rootNodeByVersion(is4)

	// record path to leaf node
	stack := [maxTreeDepth]*node[V]{}

	var depth int
	var octet byte

LOOP:
	// find the last node on the octets path in the trie,
	for depth, octet = range octets {
		depth = depth & 0xf // BCE

		if depth > maxDepth {
			depth--
			break
		}
		// push current node on stack
		stack[depth] = n

		// go down in tight loop to leaf node
		if !n.children.Test(octet) {
			break LOOP
		}
		kid := n.children.MustGet(octet)

		// kid is node or leaf or fringe at octet
		switch kid := kid.(type) {
		case *node[V]:
			n = kid
			continue LOOP // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix, stop traversing
			if kid.prefix.Bits() > bits || !kid.prefix.Contains(ip) {
				break LOOP
			}
			return kid.prefix, kid.value, true

		case *fringeNode[V]:
			// the bits of the fringe are defined by the depth
			// maybe the LPM isn't needed, saves some cycles
			fringeBits := (depth + 1) << 3
			if fringeBits > bits {
				break LOOP
			}

			// the LPM isn't needed, saves some cycles
			if !withLPM {
				return netip.Prefix{}, kid.value, true
			}

			// sic, get the LPM prefix back, it costs some cycles!
			fringePfx := cidrForFringe(octets, depth, is4, octet)
			return fringePfx, kid.value, true

		default:
			panic("logic error, wrong node type")
		}
	}

	// start backtracking, unwind the stack
	for ; depth >= 0; depth-- {
		depth = depth & 0xf // BCE

		n = stack[depth]

		// longest prefix match, skip if node has no prefixes
		if n.prefixes.Len() == 0 {
			continue
		}

		// only the lastOctet may have a different prefix len
		// all others are just host routes
		var idx uint
		octet = octets[depth]
		if depth == maxDepth {
			idx = uint(art.PfxToIdx(octet, lastBits))
		} else {
			idx = art.OctetToIdx(octet)
		}

		// manually inlined: lpmGet(idx)
		if topIdx, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
			val = n.prefixes.MustGet(topIdx)

			// called from LookupPrefix
			if !withLPM {
				return netip.Prefix{}, val, ok
			}

			// called from LookupPrefixLPM

			// get the bits from depth and top idx
			pfxBits := int(art.PfxBits(depth, topIdx))

			// calculate the lpmPfx from incoming ip and new mask
			lpmPfx, _ = ip.Prefix(pfxBits)
			return lpmPfx, val, ok
		}
	}

	return
}

// Supernets returns an iterator over all supernet routes that cover the given prefix pfx.
//
// The traversal searches both exact-length and shorter (less specific) prefixes that
// overlap or include pfx. Starting from the most specific position in the trie,
// it walks upward through parent nodes and yields any matching entries found at each level.
//
// The iteration order is reverse-CIDR: from longest prefix match (LPM) towards
// least-specific routes.
//
// The search is protocol-specific (IPv4 or IPv6) and stops immediately if the yield
// function returns false. If pfx is invalid, the function silently returns.
//
// This can be used to enumerate all covering supernet routes in routing-based
// policy engines, diagnostics tools, or fallback resolution logic.
//
// Example:
//
//	for supernet, val := range table.Supernets(netip.MustParsePrefix("192.0.2.128/25")) {
//	    fmt.Println("Matched covering route:", supernet, "->", val)
//	}
func (t *Table[V]) Supernets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()
		octets := ip.AsSlice()
		maxDepth, lastBits := maxDepthAndLastBits(bits)

		n := t.rootNodeByVersion(is4)

		// stack of the traversed nodes for reverse ordering of supernets
		stack := [maxTreeDepth]*node[V]{}

		// run variable, used after for loop
		var depth int
		var octet byte

		// find last node along this octet path
	LOOP:
		for depth, octet = range octets {
			if depth > maxDepth {
				depth--
				break
			}
			// push current node on stack
			stack[depth] = n

			// descend down the trie
			if !n.children.Test(octet) {
				break LOOP
			}
			kid := n.children.MustGet(octet)

			// kid is node or leaf or fringe at octet
			switch kid := kid.(type) {
			case *node[V]:
				n = kid
				continue LOOP // descend down to next trie level

			case *leafNode[V]:
				if kid.prefix.Bits() > pfx.Bits() {
					break LOOP
				}

				if kid.prefix.Overlaps(pfx) {
					if !yield(kid.prefix, kid.value) {
						// early exit
						return
					}
				}
				// end of trie along this octets path
				break LOOP

			case *fringeNode[V]:
				fringePfx := cidrForFringe(octets, depth, is4, octet)
				if fringePfx.Bits() > pfx.Bits() {
					break LOOP
				}

				if fringePfx.Overlaps(pfx) {
					if !yield(fringePfx, kid.value) {
						// early exit
						return
					}
				}
				// end of trie along this octets path
				break LOOP

			default:
				panic("logic error, wrong node type")
			}
		}

		// start backtracking, unwind the stack
		for ; depth >= 0; depth-- {
			n = stack[depth]

			// only the lastOctet may have a different prefix len
			// all others are just host routes
			var idx uint
			octet = octets[depth]
			if depth == maxDepth {
				idx = uint(art.PfxToIdx(octet, lastBits))
			} else {
				idx = art.OctetToIdx(octet)
			}

			// micro benchmarking, skip if there is no match
			if !n.lpmTest(idx) {
				continue
			}

			// yield all the matching prefixes, not just the lpm
			if !n.eachLookupPrefix(octets, depth, is4, idx, yield) {
				// early exit
				return
			}
		}
	}
}

// Subnets returns an iterator over all prefix–value pairs in the routing table
// that are fully contained within the given prefix pfx.
//
// Entries are returned in CIDR sort order.
//
// Example:
//
//	for sub, val := range table.Subnets(netip.MustParsePrefix("10.0.0.0/8")) {
//	    fmt.Println("Covered:", sub, "->", val)
//	}
func (t *Table[V]) Subnets(pfx netip.Prefix) iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		if !pfx.IsValid() {
			return
		}

		// canonicalize the prefix
		pfx = pfx.Masked()

		// values derived from pfx
		ip := pfx.Addr()
		is4 := ip.Is4()
		bits := pfx.Bits()
		octets := ip.AsSlice()
		maxDepth, lastBits := maxDepthAndLastBits(bits)

		n := t.rootNodeByVersion(is4)

		// find the trie node
		for depth, octet := range octets {
			if depth == maxDepth {
				idx := art.PfxToIdx(octet, lastBits)
				_ = n.eachSubnet(octets, depth, is4, idx, yield)
				return
			}

			if !n.children.Test(octet) {
				return
			}
			kid := n.children.MustGet(octet)

			// kid is node or leaf or fringe at octet
			switch kid := kid.(type) {
			case *node[V]:
				n = kid
				continue // descend down to next trie level

			case *leafNode[V]:
				if pfx.Bits() <= kid.prefix.Bits() && pfx.Overlaps(kid.prefix) {
					_ = yield(kid.prefix, kid.value)
				}
				return

			case *fringeNode[V]:
				fringePfx := cidrForFringe(octets, depth, is4, octet)
				if pfx.Bits() <= fringePfx.Bits() && pfx.Overlaps(fringePfx) {
					_ = yield(fringePfx, kid.value)
				}
				return

			default:
				panic("logic error, wrong node type")
			}
		}
	}
}

// OverlapsPrefix reports whether any route in the table overlaps with the given pfx or vice versa.
//
// The check is bidirectional: it returns true if the input prefix is covered by an existing
// route, or if any stored route is itself contained within the input prefix.
//
// Internally, the function normalizes the prefix and descends the relevant trie branch,
// using stride-based logic to identify overlap without performing a full lookup.
//
// This is useful for containment tests, route validation, or policy checks using prefix
// semantics without retrieving exact matches.
func (t *Table[V]) OverlapsPrefix(pfx netip.Prefix) bool {
	if !pfx.IsValid() {
		return false
	}

	// canonicalize the prefix
	pfx = pfx.Masked()

	is4 := pfx.Addr().Is4()
	n := t.rootNodeByVersion(is4)

	return n.overlapsPrefixAtDepth(pfx, 0)
}

// Overlaps reports whether any route in the receiver table overlaps
// with a route in the other table, in either direction.
//
// The overlap check is bidirectional: it returns true if any IP prefix
// in the receiver is covered by the other table, or vice versa.
// This includes partial overlaps, exact matches, and supernet/subnet relationships.
//
// Both IPv4 and IPv6 route trees are compared independently. If either
// tree has overlapping routes, the function returns true.
//
// This is useful for conflict detection, policy enforcement,
// or validating mutually exclusive routing domains.
func (t *Table[V]) Overlaps(o *Table[V]) bool {
	return t.Overlaps4(o) || t.Overlaps6(o)
}

// Overlaps4 is like [Table.Overlaps] but for the v4 routing table only.
func (t *Table[V]) Overlaps4(o *Table[V]) bool {
	if t.size4 == 0 || o.size4 == 0 {
		return false
	}
	return t.root4.overlaps(&o.root4, 0)
}

// Overlaps6 is like [Table.Overlaps] but for the v6 routing table only.
func (t *Table[V]) Overlaps6(o *Table[V]) bool {
	if t.size6 == 0 || o.size6 == 0 {
		return false
	}
	return t.root6.overlaps(&o.root6, 0)
}

// Union merges another routing table into the receiver table, modifying it in-place.
//
// All prefixes and values from the other table (o) are inserted into the receiver.
// If a duplicate prefix exists in both tables, the value from o replaces the existing entry.
// This duplicate is shallow-copied by default, but if the value type V implements the
// Cloner interface, the value is deeply cloned before insertion. See also Table.Clone.
func (t *Table[V]) Union(o *Table[V]) {
	// Create a cloning function for deep copying values;
	// returns nil if V does not implement the Cloner interface.
	cloneFn := cloneFnFactory[V]()
	if cloneFn == nil {
		cloneFn = copyVal
	}

	dup4 := t.root4.unionRec(cloneFn, &o.root4, 0)
	dup6 := t.root6.unionRec(cloneFn, &o.root6, 0)

	t.size4 += o.size4 - dup4
	t.size6 += o.size6 - dup6
}

// Equal checks whether two tables are structurally and semantically equal.
// It ensures both trees (IPv4-based and IPv6-based) have the same sizes and
// recursively compares their root nodes.
func (t *Table[V]) Equal(o *Table[V]) bool {
	if t.size4 != o.size4 || t.size6 != o.size6 {
		return false
	}

	return t.root4.equalRec(&o.root4) && t.root6.equalRec(&o.root6)
}

// Clone returns a copy of the routing table.
// The payload of type V is shallow copied, but if type V implements the [Cloner] interface,
// the values are cloned.
func (t *Table[V]) Clone() *Table[V] {
	if t == nil {
		return nil
	}

	c := new(Table[V])

	cloneFn := cloneFnFactory[V]()

	c.root4 = *t.root4.cloneRec(cloneFn)
	c.root6 = *t.root6.cloneRec(cloneFn)

	c.size4 = t.size4
	c.size6 = t.size6

	return c
}

func (t *Table[V]) sizeUpdate(is4 bool, n int) {
	if is4 {
		t.size4 += n
		return
	}
	t.size6 += n
}

// Size returns the prefix count.
func (t *Table[V]) Size() int {
	return t.size4 + t.size6
}

// Size4 returns the IPv4 prefix count.
func (t *Table[V]) Size4() int {
	return t.size4
}

// Size6 returns the IPv6 prefix count.
func (t *Table[V]) Size6() int {
	return t.size6
}

// All returns an iterator over all prefix–value pairs in the table.
//
// The entries from both IPv4 and IPv6 subtries are yielded using an internal recursive traversal.
// The iteration order is unspecified and may vary between calls; for a stable order, use AllSorted.
//
// You can use All directly in a for-range loop without providing a yield function.
// The Go compiler automatically synthesizes the yield callback for you:
//
//	for prefix, value := range t.All() {
//	    fmt.Println(prefix, value)
//	}
//
// Under the hood, the loop body is passed as a yield function to the iterator.
// If you break or return from the loop, iteration stops early as expected.
//
// IMPORTANT: Modifying or deleting entries during iteration is not allowed,
// as this would interfere with the internal traversal and may corrupt or
// prematurely terminate the iteration.
//
// If mutation of the table during traversal is required,
// use [Table.WalkPersist] instead.
func (t *Table[V]) All() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield) && t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// All4 is like [Table.All] but only for the v4 routing table.
func (t *Table[V]) All4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRec(stridePath{}, 0, true, yield)
	}
}

// All6 is like [Table.All] but only for the v6 routing table.
func (t *Table[V]) All6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRec(stridePath{}, 0, false, yield)
	}
}

// AllSorted returns an iterator over all prefix–value pairs in the table,
// ordered in canonical CIDR prefix sort order.
//
// This can be used directly with a for-range loop; the Go compiler provides the yield function implicitly.
//
//	for prefix, value := range t.AllSorted() {
//	    fmt.Println(prefix, value)
//	}
//
// The traversal is stable and predictable across calls.
// Iteration stops early if you break out of the loop.
func (t *Table[V]) AllSorted() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield) &&
			t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}

// AllSorted4 is like [Table.AllSorted] but only for the v4 routing table.
func (t *Table[V]) AllSorted4() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root4.allRecSorted(stridePath{}, 0, true, yield)
	}
}

// AllSorted6 is like [Table.AllSorted] but only for the v6 routing table.
func (t *Table[V]) AllSorted6() iter.Seq2[netip.Prefix, V] {
	return func(yield func(netip.Prefix, V) bool) {
		_ = t.root6.allRecSorted(stridePath{}, 0, false, yield)
	}
}
