// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
	"slices"

	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/lpm"
	"github.com/gaissmai/bart/internal/sparse"
)

const (
	strideLen    = 8   // byte, a multibit trie with stride len 8
	maxTreeDepth = 16  // max 16 bytes for IPv6
	maxItems     = 256 // max 256 prefixes or children in node
)

// stridePath, max 16 octets deep
type stridePath [maxTreeDepth]uint8

// node is a trie level node in the multibit routing table.
//
// Each node contains two conceptually different arrays:
//   - prefixes: representing routes, using a complete binary tree layout
//     driven by the baseIndex() function from the ART algorithm.
//   - children: holding subtries or path-compressed leaves/fringes with
//     a branching factor of 256 (8 bits per stride).
//
// Unlike the original ART, this implementation uses popcount-compressed sparse arrays
// instead of fixed-size arrays. Array slots are not pre-allocated; insertion
// and lookup rely on fast bitset operations and precomputed rank indexes.
//
// See doc/artlookup.pdf for the mapping mechanics and prefix tree details.
type node[V any] struct {
	// prefixes stores routing entries (prefix -> value),
	// laid out as a complete binary tree using baseIndex().
	prefixes sparse.Array256[V]

	// children holds subnodes for the 256 possible next-hop paths
	// at this trie level (8-bit stride).
	//
	// Entries in children may be:
	//   - *node[V]       -> internal child node for further traversal
	//   - *leafNode[V]   -> path-comp. node (depth < maxDepth - 1)
	//   - *fringeNode[V] -> path-comp. node (depth == maxDepth - 1, stride-aligned: /8, /16, ... /128))
	//
	// Note: Both *leafNode and *fringeNode entries are only created by path compression.
	// Prefixes that match exactly at the maximum trie depth (depth == maxDepth) are
	// never stored as children, but always directly in the prefixes array at that level.
	children sparse.Array256[any]
}

// isEmpty returns true if node has neither prefixes nor children
func (n *node[V]) isEmpty() bool {
	return n.prefixes.Len() == 0 && n.children.Len() == 0
}

// leafNode is a prefix with value, used as a path compressed child.
type leafNode[V any] struct {
	prefix netip.Prefix
	value  V
}

func newLeafNode[V any](pfx netip.Prefix, val V) *leafNode[V] {
	return &leafNode[V]{prefix: pfx, value: val}
}

// fringeNode is a path-compressed leaf with value but without a prefix.
// The prefix of a fringe is solely defined by the position in the trie.
// The fringe-compressiion (no stored prefix) saves a lot of memory,
// but the algorithm is more complex.
type fringeNode[V any] struct {
	value V
}

func newFringeNode[V any](val V) *fringeNode[V] {
	return &fringeNode[V]{value: val}
}

// isFringe determines whether a prefix qualifies as a "fringe node" -
// that is, a special kind of path-compressed leaf inserted at the final
// possible trie level (depth == maxDepth - 1).
//
// Both "leaves" and "fringes" are path-compressed terminal entries;
// the distinction lies in their position within the trie:
//
//   - A leaf is inserted at any intermediate level if no further stride
//     boundary matches (depth < maxDepth - 1).
//
//   - A fringe is inserted at the last possible stride level
//     (depth == maxDepth - 1) before a prefix would otherwise land
//     as a direct prefix (depth == maxDepth).
//
// Special property:
//   - A fringe acts as a default route for all downstream bit patterns
//     extending beyond its prefix.
//
// Examples:
//
//	e.g. prefix is addr/8, or addr/16, or ... addr/128
//	depth <  maxDepth-1 : a leaf, path-compressed
//	depth == maxDepth-1 : a fringe, path-compressed
//	depth == maxDepth   : a prefix with octet/pfx == 0/0 => idx == 1, a strides default route
//
// Logic:
//   - A prefix qualifies as a fringe if:
//     depth == maxDepth - 1 &&
//     lastBits == 0 (i.e., aligned on stride boundary, /8, /16, ... /128 bits)
func isFringe(depth, bits int) bool {
	maxDepth, lastBits := maxDepthAndLastBits(bits)
	return depth == maxDepth-1 && lastBits == 0
}

// insertAtDepth inserts a network prefix and its associated value into the
// trie starting at the specified byte depth.
//
// The function walks the prefix address from the given depth and inserts the value either directly into
// the node´s prefix table or as a compressed leaf or fringe node. If a conflicting leaf or fringe exists,
// it is pushed down via a new intermediate node. Existing entries with the same prefix are overwritten.
func (n *node[V]) insertAtDepth(pfx netip.Prefix, val V, depth int) (exists bool) {
	ip := pfx.Addr() // the pfx must be in canonical form
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	// find the proper trie node to insert prefix
	// start with prefix octet at depth
	for _, octet := range octets[depth:] {
		// last masked octet: insert/override prefix/val into node
		if depth == maxDepth {
			return n.prefixes.InsertAt(art.PfxToIdx(octet, lastBits), val)
		}

		// reached end of trie path ...
		if !n.children.Test(octet) {
			// insert prefix path compressed as leaf or fringe
			if isFringe(depth, bits) {
				return n.children.InsertAt(octet, newFringeNode(val))
			}
			return n.children.InsertAt(octet, newLeafNode(pfx, val))
		}

		// ... or decend down the trie
		kid := n.children.MustGet(octet)

		// kid is node or leaf at addr
		switch kid := kid.(type) {
		case *node[V]:
			n = kid // descend down to next trie level

		case *leafNode[V]:
			// reached a path compressed prefix
			// override value in slot if prefixes are equal
			if kid.prefix == pfx {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the leaf down
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.insertAtDepth(kid.prefix, kid.value, depth+1)

			n.children.InsertAt(octet, newNode)
			n = newNode

		case *fringeNode[V]:
			// reached a path compressed fringe
			// override value in slot if pfx is a fringe
			if isFringe(depth, bits) {
				kid.value = val
				// exists
				return true
			}

			// create new node
			// push the fringe down, it becomes a default route (idx=1)
			// insert new child at current leaf position (addr)
			// descend down, replace n with new child
			newNode := new(node[V])
			newNode.prefixes.InsertAt(1, kid.value)

			n.children.InsertAt(octet, newNode)
			n = newNode

		default:
			panic("logic error, wrong node type")
		}

		depth++
	}

	panic("unreachable")
}

// purgeAndCompress traverses the deletion path upward and removes empty or compressible nodes
// in the trie.
//
// After a route deletion, this function walks back through the recorded traversal stack
// and optimizes the trie by eliminating redundant intermediate nodes. A node is purged if it is empty,
// and compressed if it contains only a single leaf, fringe, or prefix.
//
// Compressible cases are handled by removing the node and reinserting its content (prefix or value)
// one level higher, preserving routing semantics while reducing structural depth. The child is then
// replaced in the parent, effectively flattening the trie where appropriate.
//
// The reconstruction of prefixes for fringe or prefix entries is based on
// the original `octets` traversal path and the parent´s depth.
func (n *node[V]) purgeAndCompress(stack []*node[V], octets []uint8, is4 bool) {
	// unwind the stack
	for depth := len(stack) - 1; depth >= 0; depth-- {
		parent := stack[depth]
		octet := octets[depth]

		pfxCount := n.prefixes.Len()
		childCount := n.children.Len()

		switch {
		case n.isEmpty():
			// just delete this empty node from parent
			parent.children.DeleteAt(octet)

		case pfxCount == 0 && childCount == 1:
			switch kid := n.children.Items[0].(type) {
			case *node[V]:
				// fast exit, we are at an intermediate path node
				// no further delete/compress upwards the stack is possible
				return
			case *leafNode[V]:
				// just one leaf, delete this node and reinsert the leaf above
				parent.children.DeleteAt(octet)

				// ... (re)insert the leaf at parents depth
				parent.insertAtDepth(kid.prefix, kid.value, depth)
			case *fringeNode[V]:
				// just one fringe, delete this node and reinsert the fringe as leaf above
				parent.children.DeleteAt(octet)

				// get the last octet back, the only item is also the first item
				lastOctet, _ := n.children.FirstSet()

				// rebuild the prefix with octets, depth, ip version and addr
				// depth is the parent's depth, so add +1 here for the kid
				fringePfx := cidrForFringe(octets, depth+1, is4, lastOctet)

				// ... (re)reinsert prefix/value at parents depth
				parent.insertAtDepth(fringePfx, kid.value, depth)
			}

		case pfxCount == 1 && childCount == 0:
			// just one prefix, delete this node and reinsert the idx as leaf above
			parent.children.DeleteAt(octet)

			// get prefix back from idx ...
			idx, _ := n.prefixes.FirstSet() // single idx must be first bit set
			val := n.prefixes.Items[0]      // single value must be at Items[0]

			// ... and octet path
			path := stridePath{}
			copy(path[:], octets)

			// depth is the parent's depth, so add +1 here for the kid
			pfx := cidrFromPath(path, depth+1, is4, idx)

			// ... (re)insert prefix/value at parents depth
			parent.insertAtDepth(pfx, val, depth)
		}

		// climb up the stack
		n = parent
	}
}

// lpmGet performs a longest-prefix match (LPM) lookup for the given index (idx)
// within the 8-bit stride-based prefix table at this trie depth.
//
// The function returns the matched base index, associated value, and true if a
// matching prefix exists at this level; otherwise, ok is false.
//
// Internally, the prefix table is organized as a complete binary tree (CBT) indexed
// via the baseIndex function. Unlike the original ART algorithm, this implementation
// does not use an allotment-based approach. Instead, it performs CBT backtracking
// using a bitset-based operation with a precomputed backtracking pattern specific to idx.
func (n *node[V]) lpmGet(idx uint) (baseIdx uint8, val V, ok bool) {
	// top is the idx of the longest-prefix-match
	if top, ok := n.prefixes.IntersectionTop(lpm.BackTrackingBitset(idx)); ok {
		return top, n.prefixes.MustGet(top), true
	}

	// not found (on this level)
	return
}

// lpmTest returns true if an index (idx) has any matching longest-prefix
// in the current node’s prefix table.
//
// This function performs a presence check without retrieving the associated value.
// It is faster than a full lookup, as it only tests for intersection with the
// backtracking bitset for the given index.
//
// The prefix table is structured as a complete binary tree (CBT), and LPM testing
// is done via a bitset operation that maps the traversal path from the given index
// toward its possible ancestors.
func (n *node[V]) lpmTest(idx uint) bool {
	return n.prefixes.Intersects(lpm.BackTrackingBitset(idx))
}

// allRec recursively traverses the trie starting at the current node,
// applying the provided yield function to every stored prefix and value.
//
// For each route entry (prefix and value), yield is invoked. If yield returns false,
// the traversal stops immediately, and false is propagated upwards,
// enabling early termination.
//
// The function handles all prefix entries in the current node, as well as any children -
// including sub-nodes, leaf nodes with full prefixes, and fringe nodes
// representing path-compressed prefixes. IP prefix reconstruction is performed on-the-fly
// from the current path and depth.
//
// The traversal order is not defined. This implementation favors simplicity
// and runtime efficiency over consistency of iteration sequence.
func (n *node[V]) allRec(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
		cidr := cidrFromPath(path, depth, is4, idx)

		// callback for this prefix and val
		if !yield(cidr, n.prefixes.MustGet(idx)) {
			// early exit
			return false
		}
	}

	// for all children (nodes and leaves) in this node do ...
	for i, addr := range n.children.AsSlice(&[256]uint8{}) {
		switch kid := n.children.Items[i].(type) {
		case *node[V]:
			// rec-descent with this node
			path[depth] = addr
			if !kid.allRec(path, depth+1, is4, yield) {
				// early exit
				return false
			}
		case *leafNode[V]:
			// callback for this leaf
			if !yield(kid.prefix, kid.value) {
				// early exit
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// allRecSorted recursively traverses the trie in prefix-sorted order and applies
// the given yield function to each stored prefix and value.
//
// Unlike allRec, this implementation ensures that route entries are visited in
// canonical prefix sort order. To achieve this,
// both the prefixes and children of the current node are gathered, sorted,
// and then interleaved during traversal based on logical octet positioning.
//
// The function first sorts relevant entries by their prefix index and address value,
// using a comparison function that ranks prefixes according to their mask length and position.
// Then it walks the trie, always yielding child entries that fall before the current prefix,
// followed by the prefix itself. Remaining children are processed once all prefixes have been visited.
//
// Prefixes are reconstructed on-the-fly from the traversal path, and iteration includes all child types:
// inner nodes (recursive descent), leaf nodes, and fringe (compressed) prefixes.
//
// If the yield callback returns false at any point, traversal stops early and false is returned,
// allowing for efficient filtered iteration. The order is stable and predictable, making the function
// suitable for use cases like table exports, comparisons, or serialization.
func (n *node[V]) allRecSorted(path stridePath, depth int, is4 bool, yield func(netip.Prefix, V) bool) bool {
	// get slice of all child octets, sorted by addr
	allChildAddrs := n.children.AsSlice(&[256]uint8{})

	// get slice of all indexes, sorted by idx
	allIndices := n.prefixes.AsSlice(&[256]uint8{})

	// sort indices in CIDR sort order
	slices.SortFunc(allIndices, cmpIndexRank)

	childCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all childs before idx
		for j := childCursor; j < len(allChildAddrs); j++ {
			childAddr := allChildAddrs[j]

			if childAddr >= pfxOctet {
				break
			}

			// yield the node (rec-descent) or leaf
			switch kid := n.children.Items[j].(type) {
			case *node[V]:
				path[depth] = childAddr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}
			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}
			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, childAddr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			childCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for j := childCursor; j < len(allChildAddrs); j++ {
		addr := allChildAddrs[j]
		switch kid := n.children.Items[j].(type) {
		case *node[V]:
			path[depth] = addr
			if !kid.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leafNode[V]:
			if !yield(kid.prefix, kid.value) {
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// eachLookupPrefix performs a hierarchical lookup of all matching prefixes
// in the current node’s 8-bit stride-based prefix table.
//
// The function walks up the trie-internal complete binary tree (CBT),
// testing each possible prefix length mask (in decreasing order of specificity),
// and invokes the yield function for every matching entry.
//
// The given idx refers to the position for this stride's prefix and is used
// to derive a backtracking path through the CBT by repeatedly halving the index.
// At each step, if a prefix exists in the table, its corresponding CIDR is
// reconstructed and yielded. If yield returns false, traversal stops early.
//
// This function is intended for internal use during supernet traversal and
// does not descend the trie further.
func (n *node[V]) eachLookupPrefix(octets []byte, depth int, is4 bool, pfxIdx uint, yield func(netip.Prefix, V) bool) (ok bool) {
	// path needed below more than once in loop
	var path stridePath
	copy(path[:], octets)

	// fast forward, it's a /8 route, too big for bitset256
	if pfxIdx > 255 {
		pfxIdx >>= 1
	}
	idx := uint8(pfxIdx) // now it fits into uint8

	for ; idx > 0; idx >>= 1 {
		if n.prefixes.Test(idx) {
			val := n.prefixes.MustGet(idx)
			cidr := cidrFromPath(path, depth, is4, idx)

			if !yield(cidr, val) {
				return false
			}
		}
	}

	return true
}

// eachSubnet yields all prefix entries and child nodes covered by a given parent prefix,
// sorted in natural CIDR order, within the current node.
//
// The function iterates through all prefixes and children from the node’s stride tables.
// Only entries that fall within the address range defined by the parent prefix index (pfxIdx)
// are included. Matching entries are buffered, sorted, and passed through to the yield function.
//
// Child entries (nodes, leaves, fringes) that fall under the covered address range
// are processed recursively via allRecSorted to ensure sorted traversal.
//
// This function is intended for internal use by Subnets(), and it assumes the
// current node is positioned at the point in the trie corresponding to the parent prefix.
func (n *node[V]) eachSubnet(octets []byte, depth int, is4 bool, pfxIdx uint8, yield func(netip.Prefix, V) bool) bool {
	// octets as array, needed below more than once
	var path stridePath
	copy(path[:], octets)

	pfxFirstAddr, pfxLastAddr := art.IdxToRange(pfxIdx)

	allCoveredIndices := make([]uint8, 0, maxItems)
	for _, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
		thisFirstAddr, thisLastAddr := art.IdxToRange(idx)

		if thisFirstAddr >= pfxFirstAddr && thisLastAddr <= pfxLastAddr {
			allCoveredIndices = append(allCoveredIndices, idx)
		}
	}

	// sort indices in CIDR sort order
	slices.SortFunc(allCoveredIndices, cmpIndexRank)

	// 2. collect all covered child addrs by prefix

	allCoveredChildAddrs := make([]uint8, 0, maxItems)
	for _, addr := range n.children.AsSlice(&[256]uint8{}) {
		if addr >= pfxFirstAddr && addr <= pfxLastAddr {
			allCoveredChildAddrs = append(allCoveredChildAddrs, addr)
		}
	}

	// 3. yield covered indices, pathcomp prefixes and childs in CIDR sort order

	addrCursor := 0

	// yield indices and childs in CIDR sort order
	for _, pfxIdx := range allCoveredIndices {
		pfxOctet, _ := art.IdxToPfx(pfxIdx)

		// yield all childs before idx
		for j := addrCursor; j < len(allCoveredChildAddrs); j++ {
			addr := allCoveredChildAddrs[j]
			if addr >= pfxOctet {
				break
			}

			// yield the node or leaf?
			switch kid := n.children.MustGet(addr).(type) {
			case *node[V]:
				path[depth] = addr
				if !kid.allRecSorted(path, depth+1, is4, yield) {
					return false
				}

			case *leafNode[V]:
				if !yield(kid.prefix, kid.value) {
					return false
				}

			case *fringeNode[V]:
				fringePfx := cidrForFringe(path[:], depth, is4, addr)
				// callback for this fringe
				if !yield(fringePfx, kid.value) {
					// early exit
					return false
				}

			default:
				panic("logic error, wrong node type")
			}

			addrCursor++
		}

		// yield the prefix for this idx
		cidr := cidrFromPath(path, depth, is4, pfxIdx)
		// n.prefixes.Items[i] not possible after sorting allIndices
		if !yield(cidr, n.prefixes.MustGet(pfxIdx)) {
			return false
		}
	}

	// yield the rest of leaves and nodes (rec-descent)
	for _, addr := range allCoveredChildAddrs[addrCursor:] {
		// yield the node or leaf?
		switch kid := n.children.MustGet(addr).(type) {
		case *node[V]:
			path[depth] = addr
			if !kid.allRecSorted(path, depth+1, is4, yield) {
				return false
			}
		case *leafNode[V]:
			if !yield(kid.prefix, kid.value) {
				return false
			}
		case *fringeNode[V]:
			fringePfx := cidrForFringe(path[:], depth, is4, addr)
			// callback for this fringe
			if !yield(fringePfx, kid.value) {
				// early exit
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}

// cmpIndexRank, sort indexes in prefix sort order.
func cmpIndexRank(aIdx, bIdx uint8) int {
	// convert idx [1..255] to prefix
	aOctet, aBits := art.IdxToPfx(aIdx)
	bOctet, bBits := art.IdxToPfx(bIdx)

	// cmp the prefixes, first by address and then by bits
	if aOctet == bOctet {
		if aBits <= bBits {
			return -1
		}

		return 1
	}

	if aOctet < bOctet {
		return -1
	}

	return 1
}

// cidrFromPath, helper function,
// get prefix back from stride path, depth and idx.
// The prefix is solely defined by the position in the trie and the baseIndex.
func cidrFromPath(path stridePath, depth int, is4 bool, idx uint8) netip.Prefix {
	depth = depth & 0xf // BCE

	octet, pfxLen := art.IdxToPfx(idx)

	// set masked byte in path at depth
	path[depth] = octet

	// zero/mask the bytes after prefix bits
	clear(path[depth+1:])

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// calc bits with pathLen and pfxLen
	bits := depth<<3 + int(pfxLen)

	// return a normalized prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}

// cidrForFringe, helper function,
// get prefix back from octets path, depth, IP version and last octet.
// The prefix of a fringe is solely defined by the position in the trie.
func cidrForFringe(octets []byte, depth int, is4 bool, lastOctet uint8) netip.Prefix {
	depth = depth & 0xf // BCE

	path := stridePath{}
	copy(path[:], octets[:depth+1])

	// replace last octet
	path[depth] = lastOctet

	// make ip addr from octets
	var ip netip.Addr
	if is4 {
		ip = netip.AddrFrom4([4]byte(path[:4]))
	} else {
		ip = netip.AddrFrom16(path)
	}

	// it's a fringe, bits are alway /8, /16, /24, ...
	bits := (depth + 1) << 3

	// return a (normalized) prefix from ip/bits
	return netip.PrefixFrom(ip, bits)
}
