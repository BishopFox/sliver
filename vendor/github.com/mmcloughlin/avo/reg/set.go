package reg

// Set is a set of registers.
type Set map[Register]bool

// NewEmptySet builds an empty register set.
func NewEmptySet() Set {
	return Set{}
}

// NewSetFromSlice forms a set from the given register list.
func NewSetFromSlice(rs []Register) Set {
	s := NewEmptySet()
	for _, r := range rs {
		s.Add(r)
	}
	return s
}

// Clone returns a copy of s.
func (s Set) Clone() Set {
	c := NewEmptySet()
	for r := range s {
		c.Add(r)
	}
	return c
}

// Add r to s.
func (s Set) Add(r Register) {
	s[r] = true
}

// Discard removes r from s, if present.
func (s Set) Discard(r Register) {
	delete(s, r)
}

// Update adds every register in t to s.
func (s Set) Update(t Set) {
	for r := range t {
		s.Add(r)
	}
}

// Difference returns the set of registers in s but not t.
func (s Set) Difference(t Set) Set {
	d := s.Clone()
	d.DifferenceUpdate(t)
	return d
}

// DifferenceUpdate removes every element of t from s.
func (s Set) DifferenceUpdate(t Set) {
	for r := range t {
		s.Discard(r)
	}
}

// Equals returns true if s and t contain the same registers.
func (s Set) Equals(t Set) bool {
	if len(s) != len(t) {
		return false
	}
	for r := range s {
		if _, found := t[r]; !found {
			return false
		}
	}
	return true
}

// OfKind returns the set of elements of s with kind k.
func (s Set) OfKind(k Kind) Set {
	t := NewEmptySet()
	for r := range s {
		if r.Kind() == k {
			t.Add(r)
		}
	}
	return t
}
