// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"

	"github.com/gaissmai/bart/internal/art"
)

// DumpListNode contains CIDR, Value and Subnets, representing the trie
// in a sorted, recursive representation, especially useful for serialization.
type DumpListNode[V any] struct {
	CIDR    netip.Prefix      `json:"cidr"`
	Value   V                 `json:"value"`
	Subnets []DumpListNode[V] `json:"subnets,omitempty"`
}

// trieItem, a node has no path information about its predecessors,
// we collect this during the recursive descent.
type trieItem[V any] struct {
	// for traversing, path/depth/idx is needed to get the CIDR back from the trie.
	n     *node[V]
	is4   bool
	path  stridePath
	depth int
	idx   uint8

	// for printing
	cidr netip.Prefix
	val  V
}

// String returns a hierarchical tree diagram of the ordered CIDRs
// as string, just a wrapper for [Table.Fprint].
// If Fprint returns an error, String panics.
func (t *Table[V]) String() string {
	w := new(strings.Builder)
	if err := t.Fprint(w); err != nil {
		panic(err)
	}

	return w.String()
}

// Fprint writes a hierarchical tree diagram of the ordered CIDRs
// with default formatted payload V to w.
//
// The order from top to bottom is in ascending order of the prefix address
// and the subtree structure is determined by the CIDRs coverage.
//
//	▼
//	├─ 10.0.0.0/8 (V)
//	│  ├─ 10.0.0.0/24 (V)
//	│  └─ 10.0.1.0/24 (V)
//	├─ 127.0.0.0/8 (V)
//	│  └─ 127.0.0.1/32 (V)
//	├─ 169.254.0.0/16 (V)
//	├─ 172.16.0.0/12 (V)
//	└─ 192.168.0.0/16 (V)
//	   └─ 192.168.1.0/24 (V)
//	▼
//	└─ ::/0 (V)
//	   ├─ ::1/128 (V)
//	   ├─ 2000::/3 (V)
//	   │  └─ 2001:db8::/32 (V)
//	   └─ fe80::/10 (V)
func (t *Table[V]) Fprint(w io.Writer) error {
	if t == nil || w == nil {
		return nil
	}

	// v4
	if err := t.fprint(w, true); err != nil {
		return err
	}

	// v6
	if err := t.fprint(w, false); err != nil {
		return err
	}

	return nil
}

// fprint is the version dependent adapter to fprintRec.
func (t *Table[V]) fprint(w io.Writer, is4 bool) error {
	n := t.rootNodeByVersion(is4)
	if n.isEmpty() {
		return nil
	}

	if _, err := fmt.Fprint(w, "▼\n"); err != nil {
		return err
	}

	startParent := trieItem[V]{
		n:    nil,
		idx:  0,
		path: stridePath{},
		is4:  is4,
	}

	return n.fprintRec(w, startParent, "")
}

// fprintRec, the output is a hierarchical CIDR tree covered starting with this node
func (n *node[V]) fprintRec(w io.Writer, parent trieItem[V], pad string) error {
	// recursion stop condition
	if n == nil {
		return nil
	}

	// get direct covered childs for this parent ...
	directItems := n.directItemsRec(parent.idx, parent.path, parent.depth, parent.is4)

	// sort them by netip.Prefix, not by baseIndex
	slices.SortFunc(directItems, func(a, b trieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	// symbols used in tree
	glyphe := "├─ "
	spacer := "│  "

	// for all direct item under this node ...
	for i, item := range directItems {
		// ... treat last kid special
		if i == len(directItems)-1 {
			glyphe = "└─ "
			spacer = "   "
		}

		var err error
		// Lite: val is the empty struct, don't print it
		switch any(item.val).(type) {
		case struct{}:
			_, err = fmt.Fprintf(w, "%s%s\n", pad+glyphe, item.cidr)
		default:
			_, err = fmt.Fprintf(w, "%s%s (%v)\n", pad+glyphe, item.cidr, item.val)
		}

		if err != nil {
			return err
		}

		// rec-descent with this item as parent
		if err := item.n.fprintRec(w, item, pad+spacer); err != nil {
			return err
		}
	}

	return nil
}

// MarshalText implements the [encoding.TextMarshaler] interface,
// just a wrapper for [Table.Fprint].
func (t *Table[V]) MarshalText() ([]byte, error) {
	w := new(bytes.Buffer)
	if err := t.Fprint(w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// MarshalJSON dumps the table into two sorted lists: for ipv4 and ipv6.
// Every root and subnet is an array, not a map, because the order matters.
func (t *Table[V]) MarshalJSON() ([]byte, error) {
	if t == nil {
		return nil, nil
	}

	result := struct {
		Ipv4 []DumpListNode[V] `json:"ipv4,omitempty"`
		Ipv6 []DumpListNode[V] `json:"ipv6,omitempty"`
	}{
		Ipv4: t.DumpList4(),
		Ipv6: t.DumpList6(),
	}

	buf, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// DumpList4 dumps the ipv4 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build the text or json serialization.
func (t *Table[V]) DumpList4() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root4.dumpListRec(0, stridePath{}, 0, true)
}

// DumpList6 dumps the ipv6 tree into a list of roots and their subnets.
// It can be used to analyze the tree or build custom json representation.
func (t *Table[V]) DumpList6() []DumpListNode[V] {
	if t == nil {
		return nil
	}
	return t.root6.dumpListRec(0, stridePath{}, 0, false)
}

// dumpListRec, build the data structure rec-descent with the help
// of directItemsRec.
func (n *node[V]) dumpListRec(parentIdx uint8, path stridePath, depth int, is4 bool) []DumpListNode[V] {
	// recursion stop condition
	if n == nil {
		return nil
	}

	directItems := n.directItemsRec(parentIdx, path, depth, is4)

	// sort the items by prefix
	slices.SortFunc(directItems, func(a, b trieItem[V]) int {
		return cmpPrefix(a.cidr, b.cidr)
	})

	nodes := make([]DumpListNode[V], 0, len(directItems))

	for _, item := range directItems {
		nodes = append(nodes, DumpListNode[V]{
			CIDR:  item.cidr,
			Value: item.val,
			// build it rec-descent
			Subnets: item.n.dumpListRec(item.idx, item.path, item.depth, is4),
		})
	}

	return nodes
}

// directItemsRec, returns the direct covered items by parent.
// It's a complex recursive function, you have to know the data structure
// by heart to understand this function!
//
// See the  artlookup.pdf paper in the doc folder, the baseIndex function is the key.
func (n *node[V]) directItemsRec(parentIdx uint8, path stridePath, depth int, is4 bool) (directItems []trieItem[V]) {
	// recursion stop condition
	if n == nil {
		return nil
	}

	// prefixes:
	// for all idx's (prefixes mapped by baseIndex) in this node
	// do a longest-prefix-match
	for i, idx := range n.prefixes.AsSlice(&[256]uint8{}) {
		// tricky part, skip self
		// test with next possible lpm (idx>>1), it's a complete binary tree
		nextIdx := idx >> 1

		// fast skip, lpm not possible
		if nextIdx < parentIdx {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.lpmGet(uint(nextIdx))

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// prefix is directly covered by parent

			item := trieItem[V]{
				n:     n,
				is4:   is4,
				path:  path,
				depth: depth,
				idx:   idx,
				// get the prefix back from trie
				cidr: cidrFromPath(path, depth, is4, idx),
				val:  n.prefixes.Items[i],
			}

			directItems = append(directItems, item)
		}
	}

	// children:
	for i, addr := range n.children.AsSlice(&[256]uint8{}) {
		hostIdx := art.OctetToIdx(addr)

		// fast skip, lpm not possible
		if hostIdx < uint(parentIdx) {
			continue
		}

		// do a longest-prefix-match
		lpm, _, _ := n.lpmGet(hostIdx)

		// be aware, 0 is here a possible value for parentIdx and lpm (if not found)
		if lpm == parentIdx {
			// child is directly covered by parent
			switch kid := n.children.Items[i].(type) {
			case *node[V]: // traverse rec-descent, call with next child node,
				// next trie level, set parentIdx to 0, adjust path and depth
				path[depth&0xf] = addr
				directItems = append(directItems, kid.directItemsRec(0, path, depth+1, is4)...)

			case *leafNode[V]: // path-compressed child, stop's recursion for this child
				item := trieItem[V]{
					n:    nil,
					is4:  is4,
					cidr: kid.prefix,
					val:  kid.value,
				}
				directItems = append(directItems, item)

			case *fringeNode[V]: // path-compressed fringe, stop's recursion for this child
				item := trieItem[V]{
					n:   nil,
					is4: is4,
					// get the prefix back from trie
					cidr: cidrForFringe(path[:], depth, is4, addr),
					val:  kid.value,
				}
				directItems = append(directItems, item)
			}
		}
	}

	return directItems
}

// cmpPrefix, helper function, compare func for prefix sort,
// all cidrs are already normalized
func cmpPrefix(a, b netip.Prefix) int {
	if cmp := a.Addr().Compare(b.Addr()); cmp != 0 {
		return cmp
	}

	return cmp.Compare(a.Bits(), b.Bits())
}
