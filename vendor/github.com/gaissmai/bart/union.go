package bart

// unionRec recursively merges another node o into the receiver node n.
//
// All prefix and child entries from o are cloned and inserted into n.
// If a prefix already exists in n, its value is overwritten by the value from o,
// and the duplicate is counted in the return value. This count can later be used
// to update size-related metadata in the parent trie.
//
// The union handles all possible combinations of child node types (node, leaf, fringe)
// between the two nodes. Structural conflicts are resolved by creating new intermediate
// *node[V] objects and pushing both children further down the trie. Leaves and fringes
// are also recursively relocated as needed to preserve prefix semantics.
//
// The merge operation is destructive on the receiver n, but leaves the source node o unchanged.
//
// Returns the number of duplicate prefixes that were overwritten during merging.
func (n *node[V]) unionRec(cloneFn cloneFunc[V], o *node[V], depth int) (duplicates int) {
	// for all prefixes in other node do ...
	for i, oIdx := range o.prefixes.AsSlice(&[256]uint8{}) {
		// clone/copy the value from other node at idx
		clonedVal := cloneFn(o.prefixes.Items[i])

		// insert/overwrite cloned value from o into n
		if n.prefixes.InsertAt(oIdx, clonedVal) {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	for i, addr := range o.children.AsSlice(&[256]uint8{}) {
		//  12 possible combinations to union this child and other child
		//
		//  THIS,   OTHER: (always clone the other kid!)
		//  --------------
		//  NULL,   node    <-- insert node at addr
		//  NULL,   leaf    <-- insert leaf at addr
		//  NULL,   fringe  <-- insert fringe at addr

		//  node,   node    <-- union rec-descent with node
		//  node,   leaf    <-- insert leaf at depth+1
		//  node,   fringe  <-- insert fringe at depth+1

		//  leaf,   node    <-- insert new node, push this leaf down, union rec-descent
		//  leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
		//  leaf,   fringe  <-- insert new node, push this leaf and fringe down

		//  fringe, node    <-- insert new node, push this fringe down, union rec-descent
		//  fringe, leaf    <-- insert new node, push this fringe down, insert other leaf at depth+1
		//  fringe, fringe  <-- just overwrite value
		//
		// try to get child at same addr from n
		thisChild, thisExists := n.children.Get(addr)
		if !thisExists { // NULL, ... slot at addr is empty
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // NULL, node
				n.children.InsertAt(addr, otherKid.cloneRec(cloneFn))
				continue

			case *leafNode[V]: // NULL, leaf
				n.children.InsertAt(addr, otherKid.cloneLeaf(cloneFn))
				continue

			case *fringeNode[V]: // NULL, fringe
				n.children.InsertAt(addr, otherKid.cloneFringe(cloneFn))
				continue

			default:
				panic("logic error, wrong node type")
			}
		}

		switch thisKid := thisChild.(type) {
		case *node[V]: // node, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // node, node
				// both childs have node at addr, call union rec-descent on child nodes
				duplicates += thisKid.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // node, leaf
				// push this cloned leaf down, count duplicate entry
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if thisKid.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}
				continue

			case *fringeNode[V]: // node, fringe
				// push this fringe down, a fringe becomes a default route one level down
				clonedFringe := otherKid.cloneFringe(cloneFn)
				if thisKid.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}
				continue
			}

		case *leafNode[V]: // leaf, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // leaf, node
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // leaf, leaf
				// shortcut, prefixes are equal
				if thisKid.prefix == otherKid.prefix {
					thisKid.value = cloneFn(otherKid.value)
					duplicates++
					continue
				}

				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// insert at depth cloned leaf, maybe duplicate
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if nc.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // leaf, fringe
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// push this cloned fringe down, it becomes the default route
				clonedFringe := otherKid.cloneFringe(cloneFn)
				if nc.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue
			}

		case *fringeNode[V]: // fringe, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // fringe, node
				// create new node
				nc := new(node[V])

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // fringe, leaf
				// create new node
				nc := new(node[V])

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// push this cloned leaf down
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if nc.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // fringe, fringe
				thisKid.value = otherKid.cloneFringe(cloneFn).value
				duplicates++
				continue
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return duplicates
}

// unionRecPersist is similar to unionRec but performs an immutable union of nodes.
func (n *node[V]) unionRecPersist(cloneFn cloneFunc[V], o *node[V], depth int) (duplicates int) {
	// for all prefixes in other node do ...
	for i, oIdx := range o.prefixes.AsSlice(&[256]uint8{}) {
		// clone/copy the value from other node
		clonedVal := cloneFn(o.prefixes.Items[i])

		// insert/overwrite cloned value from o into n
		if exists := n.prefixes.InsertAt(oIdx, clonedVal); exists {
			// this prefix is duplicate in n and o
			duplicates++
		}
	}

	// for all child addrs in other node do ...
	for i, addr := range o.children.AsSlice(&[256]uint8{}) {
		//  12 possible combinations to union this child and other child
		//
		//  THIS,   OTHER: (always clone the other kid!)
		//  --------------
		//  NULL,   node    <-- insert node at addr
		//  NULL,   leaf    <-- insert leaf at addr
		//  NULL,   fringe  <-- insert fringe at addr

		//  node,   node    <-- CLONE: union rec-descent with node
		//  node,   leaf    <-- CLONE: insert leaf at depth+1
		//  node,   fringe  <-- CLONE: insert fringe at depth+1

		//  leaf,   node    <-- insert new node, push this leaf down, union rec-descent
		//  leaf,   leaf    <-- insert new node, push both leaves down (!first check equality)
		//  leaf,   fringe  <-- insert new node, push this leaf and fringe down

		//  fringe, node    <-- insert new node, push this fringe down, union rec-descent
		//  fringe, leaf    <-- insert new node, push this fringe down, insert other leaf at depth+1
		//  fringe, fringe  <-- just overwrite value
		//
		// try to get child at same addr from n
		thisChild, thisExists := n.children.Get(addr)
		if !thisExists { // NULL, ... slot at addr is empty
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // NULL, node
				n.children.InsertAt(addr, otherKid.cloneRec(cloneFn))
				continue

			case *leafNode[V]: // NULL, leaf
				n.children.InsertAt(addr, otherKid.cloneLeaf(cloneFn))
				continue

			case *fringeNode[V]: // NULL, fringe
				n.children.InsertAt(addr, otherKid.cloneFringe(cloneFn))
				continue

			default:
				panic("logic error, wrong node type")
			}
		}

		switch thisKid := thisChild.(type) {
		case *node[V]: // node, ...
			// CLONE the node

			// thisKid points now to cloned kid
			thisKid = thisKid.cloneFlat(cloneFn)

			// replace kid with cloned thisKid
			n.children.InsertAt(addr, thisKid)

			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // node, node
				// both childs have node at addr, call union rec-descent on child nodes
				duplicates += thisKid.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // node, leaf
				// push this cloned leaf down, count duplicate entry
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if thisKid.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}
				continue

			case *fringeNode[V]: // node, fringe
				// push this fringe down, a fringe becomes a default route one level down
				clonedFringe := otherKid.cloneFringe(cloneFn)
				if thisKid.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}
				continue
			}

		case *leafNode[V]: // leaf, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // leaf, node
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // leaf, leaf
				// shortcut, prefixes are equal
				if thisKid.prefix == otherKid.prefix {
					thisKid.value = cloneFn(otherKid.value)
					duplicates++
					continue
				}

				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// insert at depth cloned leaf, maybe duplicate
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if nc.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // leaf, fringe
				// create new node
				nc := new(node[V])

				// push this leaf down
				nc.insertAtDepth(thisKid.prefix, thisKid.value, depth+1)

				// push this cloned fringe down, it becomes the default route
				clonedFringe := otherKid.cloneFringe(cloneFn)
				if nc.prefixes.InsertAt(1, clonedFringe.value) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue
			}

		case *fringeNode[V]: // fringe, ...
			switch otherKid := o.children.Items[i].(type) {
			case *node[V]: // fringe, node
				// create new node
				nc := new(node[V])

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)

				// unionRec this new node with other kid node
				duplicates += nc.unionRec(cloneFn, otherKid.cloneRec(cloneFn), depth+1)
				continue

			case *leafNode[V]: // fringe, leaf
				// create new node
				nc := new(node[V])

				// push this fringe down, it becomes the default route
				nc.prefixes.InsertAt(1, thisKid.value)

				// push this cloned leaf down
				clonedLeaf := otherKid.cloneLeaf(cloneFn)
				if nc.insertAtDepth(clonedLeaf.prefix, clonedLeaf.value, depth+1) {
					duplicates++
				}

				// insert the new node at current addr
				n.children.InsertAt(addr, nc)
				continue

			case *fringeNode[V]: // fringe, fringe
				thisKid.value = otherKid.cloneFringe(cloneFn).value
				duplicates++
				continue
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return duplicates
}
