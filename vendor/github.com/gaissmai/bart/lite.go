// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

package bart

import (
	"net/netip"
)

// Lite is just a convenience wrapper for Table, instantiated with an
// empty struct as payload. Lite is ideal for simple IP ACLs
// (access-control-lists) with plain true/false results without a payload.
//
// Lite delegates or adapts all methods to the embedded table.
// The following methods are pointless without a payload:
//   - Lookup (use Contains)
//   - Get (use Exists)
//   - Modify
//   - ModifyPersist
type Lite struct {
	Table[struct{}]
}

// Exists returns true if the prefix exists in the table.
// It's an adapter to [Table.Get].
func (l *Lite) Exists(pfx netip.Prefix) bool {
	_, ok := l.Get(pfx)
	return ok
}

// Contains is a wrapper for the underlying table.
func (l *Lite) Contains(ip netip.Addr) bool {
	return l.Table.Contains(ip)
}

// Insert is an adapter for the underlying table.
func (l *Lite) Insert(pfx netip.Prefix) {
	l.Table.Insert(pfx, struct{}{})
}

// InsertPersist is an adapter for the underlying table.
func (l *Lite) InsertPersist(pfx netip.Prefix) *Lite {
	tbl := l.Table.InsertPersist(pfx, struct{}{})
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}
}

// Delete is a wrapper for the underlying table.
func (l *Lite) Delete(pfx netip.Prefix) bool {
	_, found := l.Table.Delete(pfx)
	return found
}

// DeletePersist is an adapter for the underlying table.
func (l *Lite) DeletePersist(pfx netip.Prefix) (*Lite, bool) {
	tbl, _, found := l.Table.DeletePersist(pfx)
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}, found
}

// WalkPersist is an adapter for the underlying table.
func (l *Lite) WalkPersist(fn func(*Lite, netip.Prefix) (*Lite, bool)) *Lite {
	//nolint:govet // shallow copy of Table is here by intention
	pl := &Lite{l.Table}

	var proceed bool
	for pfx := range l.All() {
		if pl, proceed = fn(pl, pfx); !proceed {
			break
		}
	}
	return pl
}

// Clone is an adapter for the underlying table.
func (l *Lite) Clone() *Lite {
	tbl := l.Table.Clone()
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}
}

// Union is an adapter for the underlying table.
func (l *Lite) Union(o *Lite) {
	l.Table.Union(&o.Table)
}

// UnionPersist is an adapter for the underlying table.
func (l *Lite) UnionPersist(o *Lite) *Lite {
	tbl := l.Table.UnionPersist(&o.Table)
	//nolint:govet // copy of *tbl is here by intention
	return &Lite{*tbl}
}

// Overlaps4 is an adapter for the underlying table.
func (l *Lite) Overlaps4(o *Lite) bool {
	return l.Table.Overlaps4(&o.Table)
}

// Overlaps6 is an adapter for the underlying table.
func (l *Lite) Overlaps6(o *Lite) bool {
	return l.Table.Overlaps6(&o.Table)
}

// Overlaps is an adapter for the underlying table.
func (l *Lite) Overlaps(o *Lite) bool {
	return l.Table.Overlaps(&o.Table)
}

// Equal is an adapter for the underlying table.
func (l *Lite) Equal(o *Lite) bool {
	return l.Table.Equal(&o.Table)
}
