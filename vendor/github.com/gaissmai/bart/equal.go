package bart

import (
	"reflect"
)

// Equaler is a generic interface for types that can decide their own
// equality logic. It can be used to override the potentially expensive
// default comparison with [reflect.DeepEqual].
type Equaler[V any] interface {
	Equal(other V) bool
}

// equal compares two values of type V for equality.
// If V implements Equaler[V], that custom equality method is used.
// Otherwise, [reflect.DeepEqual] is used as a fallback.
func equal[V any](v1, v2 V) bool {
	// you can't assert directly on a type parameter
	if v1, ok := any(v1).(Equaler[V]); ok {
		return v1.Equal(v2)
	}
	// fallback
	return reflect.DeepEqual(v1, v2)
}

// equalRec compares two nodes recursively.
// It checks equality of children/prefixes via bitsets, and recursively
// descends into sub-nodes or compares leaf/fringe node values.
func (n *node[V]) equalRec(o *node[V]) bool {
	if n.prefixes.BitSet256 != o.prefixes.BitSet256 {
		return false
	}

	if n.children.BitSet256 != o.children.BitSet256 {
		return false
	}

	for i, nVal := range n.prefixes.Items {
		if !equal(nVal, o.prefixes.Items[i]) {
			return false
		}
	}

	for i, nKid := range n.children.Items {
		oKid := o.children.Items[i]

		switch nKid := nKid.(type) {
		case *node[V]:
			// oKid must also be a node
			oKid, ok := oKid.(*node[V])
			if !ok {
				return false
			}

			// compare rec-descent
			if !nKid.equalRec(oKid) {
				return false
			}

		case *leafNode[V]:
			// oKid must also be a leaf
			oKid, ok := oKid.(*leafNode[V])
			if !ok {
				return false
			}

			// compare prefixes
			if nKid.prefix != oKid.prefix {
				return false
			}

			// compare values
			if !equal(nKid.value, oKid.value) {
				return false
			}

		case *fringeNode[V]:
			// oKid must also be a fringe
			oKid, ok := oKid.(*fringeNode[V])
			if !ok {
				return false
			}

			// compare values
			if !equal(nKid.value, oKid.value) {
				return false
			}

		default:
			panic("logic error, wrong node type")
		}
	}

	return true
}
