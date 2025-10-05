// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"

	"github.com/gaissmai/bart/internal/allot"
	"github.com/gaissmai/bart/internal/art"
	"github.com/gaissmai/bart/internal/bitset"
)

// overlaps recursively compares two trie nodes and returns true
// if any of their prefixes or descendants overlap.
//
// The implementation checks for:
// 1. Direct overlapping prefixes on this node level
// 2. Prefixes in one node overlapping with children in the other
// 3. Matching child addresses in both nodes, which are recursively compared
//
// All 12 possible type combinations for child entries (node, leaf, fringe) are supported.
//
// The function is optimized for early exit on first match and uses heuristics to
// choose between set-based and loop-based matching for performance.
func (n *node[V]) overlaps(o *node[V], depth int) bool {
	nPfxCount := n.prefixes.Len()
	oPfxCount := o.prefixes.Len()

	nChildCount := n.children.Len()
	oChildCount := o.children.Len()

	// ##############################
	// 1. Test if any routes overlaps
	// ##############################

	// full cross check
	if nPfxCount > 0 && oPfxCount > 0 {
		if n.overlapsRoutes(o) {
			return true
		}
	}

	// ####################################
	// 2. Test if routes overlaps any child
	// ####################################

	// swap nodes to help chance on its way,
	// if the first call to expensive overlapsChildrenIn() is already true,
	// if both orders are false it doesn't help either
	if nChildCount > oChildCount {
		n, o = o, n

		nPfxCount = n.prefixes.Len()
		oPfxCount = o.prefixes.Len()

		nChildCount = n.children.Len()
		oChildCount = o.children.Len()
	}

	if nPfxCount > 0 && oChildCount > 0 {
		if n.overlapsChildrenIn(o) {
			return true
		}
	}

	// symmetric reverse
	if oPfxCount > 0 && nChildCount > 0 {
		if o.overlapsChildrenIn(n) {
			return true
		}
	}

	// ###########################################
	// 3. childs with same octet in nodes n and o
	// ###########################################

	// stop condition, n or o have no childs
	if nChildCount == 0 || oChildCount == 0 {
		return false
	}

	// stop condition, no child with identical octet in n and o
	if !n.children.Intersects(&o.children.BitSet256) {
		return false
	}

	return n.overlapsSameChildren(o, depth)
}

// overlapsRoutes compares the prefix sets of two nodes (n and o).
//
// It first checks for direct bitset intersection (identical indices),
// then walks both prefix sets using lpmTest to detect if any
// of the n-prefixes is contained in o, or vice versa.
func (n *node[V]) overlapsRoutes(o *node[V]) bool {
	// some prefixes are identical, trivial overlap
	if n.prefixes.Intersects(&o.prefixes.BitSet256) {
		return true
	}

	// get the lowest idx (biggest prefix)
	nFirstIdx, _ := n.prefixes.FirstSet()
	oFirstIdx, _ := o.prefixes.FirstSet()

	// start with other min value
	nIdx := oFirstIdx
	oIdx := nFirstIdx

	nOK := true
	oOK := true

	// zip, range over n and o together to help chance on its way
	for nOK || oOK {
		if nOK {
			// does any route in o overlap this prefix from n
			if nIdx, nOK = n.prefixes.NextSet(nIdx); nOK {
				if o.lpmTest(uint(nIdx)) {
					return true
				}

				if nIdx == 255 {
					// stop, don't overflow uint8!
					nOK = false
				} else {
					nIdx++
				}
			}
		}

		if oOK {
			// does any route in n overlap this prefix from o
			if oIdx, oOK = o.prefixes.NextSet(oIdx); oOK {
				if n.lpmTest(uint(oIdx)) {
					return true
				}

				if oIdx == 255 {
					// stop, don't overflow uint8!
					oOK = false
				} else {
					oIdx++
				}
			}
		}
	}

	return false
}

// overlapsChildrenIn checks whether the prefixes in node n
// overlap with any children (by address range) in node o.
//
// Uses bitset intersection or manual iteration heuristically,
// depending on prefix and child count.
//
// Bitset-based matching uses precomputed coverage tables
// to avoid per-address looping. This is critical for high fan-out nodes.
func (n *node[V]) overlapsChildrenIn(o *node[V]) bool {
	pfxCount := n.prefixes.Len()
	childCount := o.children.Len()

	// heuristic, compare benchmarks
	// when will we range over the children and when will we do bitset calc?
	magicNumber := 15
	doRange := childCount < magicNumber || pfxCount > magicNumber

	// do range over, not so many childs and maybe too many prefixes for other algo below
	if doRange {
		for _, addr := range o.children.AsSlice(&[256]uint8{}) {
			if n.lpmTest(art.OctetToIdx(addr)) {
				return true
			}
		}
		return false
	}

	// do bitset intersection, alloted route table with child octets
	// maybe too many childs for range-over or not so many prefixes to
	// build the alloted routing table from them

	// make allot table with prefixes as bitsets, bitsets are precalculated.
	// Just union the bitsets to one bitset (allot table) for all prefixes
	// in this node
	hostRoutes := bitset.BitSet256{}

	allIndices := n.prefixes.AsSlice(&[256]uint8{})

	// union all pre alloted bitsets
	for _, idx := range allIndices {
		hostRoutes = hostRoutes.Union(allot.IdxToFringeRoutes(idx))
	}

	return hostRoutes.Intersects(&o.children.BitSet256)
}

// overlapsSameChildren compares all matching child addresses (octets)
// between node n and node o recursively.
//
// For each shared address, the corresponding child nodes (of any type)
// are compared using overlapsTwoChilds, which handles all
// node/leaf/fringe combinations.
func (n *node[V]) overlapsSameChildren(o *node[V], depth int) bool {
	// intersect the child bitsets from n with o
	commonChildren := n.children.Intersection(&o.children.BitSet256)

	addr := uint8(0)
	ok := true
	for ok {
		if addr, ok = commonChildren.NextSet(addr); ok {
			nChild := n.children.MustGet(addr)
			oChild := o.children.MustGet(addr)

			if overlapsTwoChilds[V](nChild, oChild, depth+1) {
				return true
			}

			if addr == 255 {
				// stop, don't overflow uint8!
				ok = false
			} else {
				addr++
			}
		}
	}
	return false
}

// overlapsTwoChilds checks two child entries for semantic overlap.
//
// Handles all 3x3 combinations of node kinds (node, leaf, fringe).
//
// Recurses into subtrees for (node, node), delegates to overlapsPrefixAtDepth
// for node/leaf mismatches, and returns true immediately if either side is fringe.
//
// Supports path-compressed routing structures without requiring full expansion.
func overlapsTwoChilds[V any](nChild, oChild any, depth int) bool {
	//  3x3 possible different combinations for n and o
	//
	//  node, node    --> overlaps rec descent
	//  node, leaf    --> overlapsPrefixAtDepth
	//  node, fringe  --> true
	//
	//  leaf, node    --> overlapsPrefixAtDepth
	//  leaf, leaf    --> netip.Prefix.Overlaps
	//  leaf, fringe  --> true
	//
	//  fringe, node    --> true
	//  fringe, leaf    --> true
	//  fringe, fringe  --> true
	//
	switch nKind := nChild.(type) {
	case *node[V]: // node, ...
		switch oKind := oChild.(type) {
		case *node[V]: // node, node
			return nKind.overlaps(oKind, depth)
		case *leafNode[V]: // node, leaf
			return nKind.overlapsPrefixAtDepth(oKind.prefix, depth)
		case *fringeNode[V]: // node, fringe
			return true
		default:
			panic("logic error, wrong node type")
		}

	case *leafNode[V]:
		switch oKind := oChild.(type) {
		case *node[V]: // leaf, node
			return oKind.overlapsPrefixAtDepth(nKind.prefix, depth)
		case *leafNode[V]: // leaf, leaf
			return oKind.prefix.Overlaps(nKind.prefix)
		case *fringeNode[V]: // leaf, fringe
			return true
		default:
			panic("logic error, wrong node type")
		}

	case *fringeNode[V]:
		return true

	default:
		panic("logic error, wrong node type")
	}
}

// overlapsPrefixAtDepth returns true if any route in the subtree rooted at this node
// overlaps with the given pfx, starting the comparison at the specified depth.
//
// This function supports structural overlap detection even in compressed or sparse
// paths within the trie, including fringe and leaf nodes. Matching is directional:
// it returns true if a route fully covers pfx, or if pfx covers an existing route.
//
// At each step, it checks for visible prefixes and children that may intersect the
// target prefix via stride-based longest-prefix test. The walk terminates early as
// soon as a structural overlap is found.
//
// This function underlies the top-level OverlapsPrefix behavior and handles details of
// trie traversal across varying prefix lengths and compression levels.
func (n *node[V]) overlapsPrefixAtDepth(pfx netip.Prefix, depth int) bool {
	ip := pfx.Addr()
	bits := pfx.Bits()
	octets := ip.AsSlice()
	maxDepth, lastBits := maxDepthAndLastBits(bits)

	for ; depth < len(octets); depth++ {
		if depth > maxDepth {
			break
		}

		octet := octets[depth]

		// full octet path in node trie, check overlap with last prefix octet
		if depth == maxDepth {
			return n.overlapsIdx(art.PfxToIdx(octet, lastBits))
		}

		// test if any route overlaps prefix´ so far
		// no best match needed, forward tests without backtracking
		if n.prefixes.Len() != 0 && n.lpmTest(art.OctetToIdx(octet)) {
			return true
		}

		if !n.children.Test(octet) {
			return false
		}

		// next child, node or leaf
		switch kid := n.children.MustGet(octet).(type) {
		case *node[V]:
			n = kid
			continue

		case *leafNode[V]:
			return kid.prefix.Overlaps(pfx)

		case *fringeNode[V]:
			return true

		default:
			panic("logic error, wrong node type")
		}
	}

	panic("unreachable: " + pfx.String())
}

// overlapsIdx returns true if the given prefix index overlaps with any entry in this node.
//
// The overlap detection considers three categories:
//
//  1. Whether any stored prefix in this node covers the requested prefix (LPM test)
//  2. Whether the requested prefix covers any stored route in the node
//  3. Whether the requested prefix overlaps with any fringe or child entry
//
// Internally, it leverages precomputed bitsets from the allotment model,
// using fast bitwise set intersections instead of explicit range comparisons.
// This enables high-performance overlap checks on a single stride level
// without descending further into the trie.
func (n *node[V]) overlapsIdx(idx uint8) bool {
	// 1. Test if any route in this node overlaps prefix?
	if n.lpmTest(uint(idx)) {
		return true
	}

	// 2. Test if prefix overlaps any route in this node

	// use bitset intersections instead of range loops
	// shallow copy pre alloted bitset for idx
	allotedPrefixRoutes := allot.IdxToPrefixRoutes(idx)
	if allotedPrefixRoutes.Intersects(&n.prefixes.BitSet256) {
		return true
	}

	// 3. Test if prefix overlaps any child in this node

	allotedHostRoutes := allot.IdxToFringeRoutes(idx)
	return allotedHostRoutes.Intersects(&n.children.BitSet256)
}
