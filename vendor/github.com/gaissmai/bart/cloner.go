package bart

// Cloner is an interface that enables deep cloning of values of type V.
// If a value implements Cloner[V], Table methods such as InsertPersist, UpdatePersist,
// DeletePersist, Clone, and Union will use its Clone method to perform deep copies.
type Cloner[V any] interface {
	Clone() V
}

// cloneFunc is a type definition for a function that takes a value of type V
// and returns the (possibly cloned) value of type V.
type cloneFunc[V any] func(V) V

// cloneFnFactory returns a cloneFunc.
// If V implements Cloner[V], the returned function should perform
// a deep copy using Clone(), otherwise it returns nil.
func cloneFnFactory[V any]() cloneFunc[V] {
	var zero V
	// you can't assert directly on a type parameter
	if _, ok := any(zero).(Cloner[V]); ok {
		return cloneVal[V]
	}
	return nil
}

// cloneVal invokes the Clone method to deeply copy val.
// Assumes that val implements Cloner[V].
func cloneVal[V any](val V) V {
	// you can't assert directly on a type parameter
	return any(val).(Cloner[V]).Clone()
}

// copyVal just copies the value.
func copyVal[V any](val V) V {
	return val
}

// cloneLeaf creates and returns a copy of the leafNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
// The prefix field is always copied as is.
func (l *leafNode[V]) cloneLeaf(cloneFn cloneFunc[V]) *leafNode[V] {
	if cloneFn == nil {
		return &leafNode[V]{prefix: l.prefix, value: l.value}
	}
	return &leafNode[V]{prefix: l.prefix, value: cloneFn(l.value)}
}

// cloneFringe creates and returns a copy of the fringeNode receiver.
// If cloneFn is nil, the value is copied directly without modification.
// Otherwise, cloneFn is applied to the value for deep cloning.
func (l *fringeNode[V]) cloneFringe(cloneFn cloneFunc[V]) *fringeNode[V] {
	if cloneFn == nil {
		return &fringeNode[V]{value: l.value}
	}
	return &fringeNode[V]{value: cloneFn(l.value)}
}

// cloneFlat returns a shallow copy of the current node[V], optionally performing deep copies of values.
//
// If cloneFn is nil, the stored values in prefixes are copied directly without modification.
// Otherwise, cloneFn is applied to each stored value for deep cloning.
// Child nodes are cloned shallowly: leafNode and fringeNode children are cloned via their clone methods,
// but child nodes of type *node[V] (subnodes) are assigned as-is without recursive cloning.
// This method does not recursively clone descendants beyond the immediate children.
//
// Note: The returned node is a new instance with copied slices but only shallow copies of nested nodes,
// except for leafNode and fringeNode children which are cloned according to cloneFn.
func (n *node[V]) cloneFlat(cloneFn cloneFunc[V]) *node[V] {
	if n == nil {
		return nil
	}

	c := new(node[V])
	if n.isEmpty() {
		return c
	}

	// copy ...
	c.prefixes = *(n.prefixes.Copy())

	// ... and clone the values
	if cloneFn != nil {
		for i, v := range c.prefixes.Items {
			c.prefixes.Items[i] = cloneFn(v)
		}
	}

	// copy ...
	c.children = *(n.children.Copy())

	// Iterate over children to flat clone leaf/fringe nodes;
	// for *node[V] children, keep shallow references (no recursive clone)
	for i, anyKid := range c.children.Items {
		switch kid := anyKid.(type) {
		case *node[V]:
			// Shallow copy
		case *leafNode[V]:
			// Clone leaf nodes, applying cloneFn as needed
			c.children.Items[i] = kid.cloneLeaf(cloneFn)
		case *fringeNode[V]:
			// Clone fringe nodes, applying cloneFn as needed
			c.children.Items[i] = kid.cloneFringe(cloneFn)
		default:
			panic("logic error, wrong node type")
		}
	}

	return c
}

// cloneRec performs a recursive deep copy of the node[V] and all its descendants.
//
// If cloneFn is nil, the stored values are copied directly without modification.
// Otherwise cloneFn is applied to each stored value for deep cloning.
//
// This method first creates a shallow clone of the current node using cloneFlat,
// applying cloneFn to values as described there. Then it recursively clones all
// child nodes of type *node[V], performing a full deep clone down the subtree.
//
// Child nodes of type *leafNode[V] and *fringeNode[V] are already cloned
// by cloneFlat.
//
// Returns a new instance of node[V] which is a complete deep clone of the
// receiver node with all descendants.
func (n *node[V]) cloneRec(cloneFn cloneFunc[V]) *node[V] {
	if n == nil {
		return nil
	}

	// Perform a flat clone of the current node.
	c := n.cloneFlat(cloneFn)

	// Recursively clone all child nodes of type *node[V]
	for i, kidAny := range c.children.Items {
		if kid, ok := kidAny.(*node[V]); ok {
			c.children.Items[i] = kid.cloneRec(cloneFn)
		}
	}

	return c
}
