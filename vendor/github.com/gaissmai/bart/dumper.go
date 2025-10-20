// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type nodeType byte

const (
	nullNode nodeType = iota // empty node
	fullNode                 // prefixes and children or path-compressed prefixes
	halfNode                 // no prefixes, only children and path-compressed prefixes
	pathNode                 // only children, no prefix nor path-compressed prefixes
	stopNode                 // no children, only prefixes or path-compressed prefixes
)

// ##################################################
//  useful during development, debugging and testing
// ##################################################

// dumpString is just a wrapper for dump.
func (t *Table[V]) dumpString() string {
	w := new(strings.Builder)
	t.dump(w)

	return w.String()
}

// dump the table structure and all the nodes to w.
func (t *Table[V]) dump(w io.Writer) {
	if t == nil {
		return
	}

	if t.size4 > 0 {
		stats := t.root4.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv4: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d),",
			t.size4, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)
		t.root4.dumpRec(w, stridePath{}, 0, true)
	}

	if t.size6 > 0 {
		stats := t.root6.nodeStatsRec()
		fmt.Fprintln(w)
		fmt.Fprintf(w, "### IPv6: size(%d), nodes(%d), pfxs(%d), leaves(%d), fringes(%d),",
			t.size6, stats.nodes, stats.pfxs, stats.leaves, stats.fringes)
		t.root6.dumpRec(w, stridePath{}, 0, false)
	}
}

// dumpRec, rec-descent the trie.
func (n *node[V]) dumpRec(w io.Writer, path stridePath, depth int, is4 bool) {
	// dump this node
	n.dump(w, path, depth, is4)

	// the node may have childs, rec-descent down
	for i, addr := range n.children.Bits() {
		path[depth&15] = addr

		if child, ok := n.children.Items[i].(*node[V]); ok {
			child.dumpRec(w, path, depth+1, is4)
		}
	}
}

// dump the node to w.
func (n *node[V]) dump(w io.Writer, path stridePath, depth int, is4 bool) {
	bits := depth * strideLen
	indent := strings.Repeat(".", depth)

	// node type with depth and octet path and bits.
	fmt.Fprintf(w, "\n%s[%s] depth:  %d path: [%s] / %d\n",
		indent, n.hasType(), depth, ipStridePath(path, depth, is4), bits)

	if nPfxCount := n.prefixes.Len(); nPfxCount != 0 {
		// no heap allocs
		allIndices := n.prefixes.Bits()

		// print the baseIndices for this node.
		fmt.Fprintf(w, "%sindexs(#%d): %s\n", indent, nPfxCount, n.prefixes.String())

		// print the prefixes for this node
		fmt.Fprintf(w, "%sprefxs(#%d):", indent, nPfxCount)

		for _, idx := range allIndices {
			pfx := cidrFromPath(path, depth, is4, idx)
			fmt.Fprintf(w, " %s", pfx)
		}

		fmt.Fprintln(w)

		// skip values if the payload is the empty struct
		if _, ok := any(n.prefixes.Items[0]).(struct{}); !ok {

			// print the values for this node
			fmt.Fprintf(w, "%svalues(#%d):", indent, nPfxCount)

			for _, val := range n.prefixes.Items {
				fmt.Fprintf(w, " %#v", val)
			}

			fmt.Fprintln(w)
		}
	}

	if n.children.Len() != 0 {

		childAddrs := make([]uint8, 0, maxItems)
		leafAddrs := make([]uint8, 0, maxItems)
		fringeAddrs := make([]uint8, 0, maxItems)

		// the node has recursive child nodes or path-compressed leaves
		for i, addr := range n.children.Bits() {
			switch n.children.Items[i].(type) {
			case *node[V]:
				childAddrs = append(childAddrs, addr)
				continue

			case *fringeNode[V]:
				fringeAddrs = append(fringeAddrs, addr)

			case *leafNode[V]:
				leafAddrs = append(leafAddrs, addr)

			default:
				panic("logic error, wrong node type")
			}
		}

		// print the children for this node.
		fmt.Fprintf(w, "%soctets(#%d): %s\n", indent, n.children.Len(), n.children.String())

		if leafCount := len(leafAddrs); leafCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sleaves(#%d):", indent, leafCount)

			for _, addr := range leafAddrs {
				k := n.children.MustGet(addr)
				pc := k.(*leafNode[V])

				// Lite: val is the empty struct, don't print it
				switch any(pc.value).(type) {
				case struct{}:
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), pc.prefix)
				default:
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), pc.prefix, pc.value)
				}
			}

			fmt.Fprintln(w)
		}

		if fringeCount := len(fringeAddrs); fringeCount > 0 {
			// print the pathcomp prefixes for this node
			fmt.Fprintf(w, "%sfringe(#%d):", indent, fringeCount)

			for _, addr := range fringeAddrs {
				fringePfx := cidrForFringe(path[:], depth, is4, addr)

				k := n.children.MustGet(addr)
				pc := k.(*fringeNode[V])

				// Lite: val is the empty struct, don't print it
				switch any(pc.value).(type) {
				case struct{}:
					fmt.Fprintf(w, " %s:{%s}", addrFmt(addr, is4), fringePfx)
				default:
					fmt.Fprintf(w, " %s:{%s, %v}", addrFmt(addr, is4), fringePfx, pc.value)
				}
			}

			fmt.Fprintln(w)
		}

		if childCount := len(childAddrs); childCount > 0 {
			// print the next child
			fmt.Fprintf(w, "%schilds(#%d):", indent, childCount)

			for _, addr := range childAddrs {
				fmt.Fprintf(w, " %s", addrFmt(addr, is4))
			}

			fmt.Fprintln(w)
		}

	}
}

// hasType returns the nodeType.
func (n *node[V]) hasType() nodeType {
	s := n.nodeStats()

	switch {
	case s.pfxs == 0 && s.childs == 0:
		return nullNode
	case s.nodes == 0:
		return stopNode
	case (s.leaves > 0 || s.fringes > 0) && s.nodes > 0 && s.pfxs == 0:
		return halfNode
	case (s.pfxs > 0 || s.leaves > 0 || s.fringes > 0) && s.nodes > 0:
		return fullNode
	case (s.pfxs == 0 && s.leaves == 0 && s.fringes == 0) && s.nodes > 0:
		return pathNode
	default:
		panic(fmt.Sprintf("UNREACHABLE: pfx: %d, chld: %d, node: %d, leaf: %d, fringe: %d",
			s.pfxs, s.childs, s.nodes, s.leaves, s.fringes))
	}
}

// addrFmt, different format strings for IPv4 and IPv6, decimal versus hex.
func addrFmt(addr byte, is4 bool) string {
	if is4 {
		return fmt.Sprintf("%d", addr)
	}

	return fmt.Sprintf("0x%02x", addr)
}

// ip stride path, different formats for IPv4 and IPv6, dotted decimal or hex.
//
//	127.0.0
//	2001:0d
func ipStridePath(path stridePath, depth int, is4 bool) string {
	buf := new(strings.Builder)

	if is4 {
		for i, b := range path[:depth] {
			if i != 0 {
				buf.WriteString(".")
			}

			buf.WriteString(strconv.Itoa(int(b)))
		}

		return buf.String()
	}

	for i, b := range path[:depth] {
		if i != 0 && i%2 == 0 {
			buf.WriteString(":")
		}

		fmt.Fprintf(buf, "%02x", b)
	}

	return buf.String()
}

// String implements Stringer for nodeType.
func (nt nodeType) String() string {
	switch nt {
	case nullNode:
		return "NULL"
	case fullNode:
		return "FULL"
	case halfNode:
		return "HALF"
	case pathNode:
		return "PATH"
	case stopNode:
		return "STOP"
	default:
		return "unreachable"
	}
}

// stats, only used for dump, tests and benchmarks
type stats struct {
	pfxs    int
	childs  int
	nodes   int
	leaves  int
	fringes int
}

// node statistics for this single node
func (n *node[V]) nodeStats() stats {
	var s stats

	s.pfxs = n.prefixes.Len()
	s.childs = n.children.Len()

	for i := range n.children.Bits() {
		switch n.children.Items[i].(type) {
		case *node[V]:
			s.nodes++

		case *fringeNode[V]:
			s.fringes++

		case *leafNode[V]:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}

// nodeStatsRec, calculate the number of pfxs, nodes and leaves under n, rec-descent.
func (n *node[V]) nodeStatsRec() stats {
	var s stats
	if n == nil || n.isEmpty() {
		return s
	}

	s.pfxs = n.prefixes.Len()
	s.childs = n.children.Len()
	s.nodes = 1 // this node
	s.leaves = 0
	s.fringes = 0

	for _, kidAny := range n.children.Items {
		switch kid := kidAny.(type) {
		case *node[V]:
			// rec-descent
			rs := kid.nodeStatsRec()

			s.pfxs += rs.pfxs
			s.childs += rs.childs
			s.nodes += rs.nodes
			s.leaves += rs.leaves
			s.fringes += rs.fringes

		case *fringeNode[V]:
			s.fringes++

		case *leafNode[V]:
			s.leaves++

		default:
			panic("logic error, wrong node type")
		}
	}

	return s
}
