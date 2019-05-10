package pass

import (
	"errors"
	"math"

	"github.com/mmcloughlin/avo/reg"
)

// edge is an edge of the interference graph, indicating that registers X and Y
// must be in non-conflicting registers.
type edge struct {
	X, Y reg.Register
}

// Allocator is a graph-coloring register allocator.
type Allocator struct {
	registers  []reg.Physical
	allocation reg.Allocation
	edges      []*edge
	possible   map[reg.Virtual][]reg.Physical
	vidtopid   map[reg.VID]reg.PID
}

// NewAllocator builds an allocator for the given physical registers.
func NewAllocator(rs []reg.Physical) (*Allocator, error) {
	if len(rs) == 0 {
		return nil, errors.New("no registers")
	}
	return &Allocator{
		registers:  rs,
		allocation: reg.NewEmptyAllocation(),
		possible:   map[reg.Virtual][]reg.Physical{},
		vidtopid:   map[reg.VID]reg.PID{},
	}, nil
}

// NewAllocatorForKind builds an allocator for the given kind of registers.
func NewAllocatorForKind(k reg.Kind) (*Allocator, error) {
	f := reg.FamilyOfKind(k)
	if f == nil {
		return nil, errors.New("unknown register family")
	}
	return NewAllocator(f.Registers())
}

// AddInterferenceSet records that r interferes with every register in s. Convenience wrapper around AddInterference.
func (a *Allocator) AddInterferenceSet(r reg.Register, s reg.Set) {
	for y := range s {
		a.AddInterference(r, y)
	}
}

// AddInterference records that x and y must be assigned to non-conflicting physical registers.
func (a *Allocator) AddInterference(x, y reg.Register) {
	a.Add(x)
	a.Add(y)
	a.edges = append(a.edges, &edge{X: x, Y: y})
}

// Add adds a register to be allocated. Does nothing if the register has already been added.
func (a *Allocator) Add(r reg.Register) {
	v, ok := r.(reg.Virtual)
	if !ok {
		return
	}
	if _, found := a.possible[v]; found {
		return
	}
	a.possible[v] = a.possibleregisters(v)
}

// Allocate allocates physical registers.
func (a *Allocator) Allocate() (reg.Allocation, error) {
	for {
		if err := a.update(); err != nil {
			return nil, err
		}

		if a.remaining() == 0 {
			break
		}

		v := a.mostrestricted()
		if err := a.alloc(v); err != nil {
			return nil, err
		}
	}
	return a.allocation, nil
}

// update possible allocations based on edges.
func (a *Allocator) update() error {
	for v := range a.possible {
		pid, found := a.vidtopid[v.VirtualID()]
		if !found {
			continue
		}
		a.possible[v] = filterregisters(a.possible[v], func(r reg.Physical) bool {
			return r.PhysicalID() == pid
		})
	}

	var rem []*edge
	for _, e := range a.edges {
		e.X, e.Y = a.allocation.LookupDefault(e.X), a.allocation.LookupDefault(e.Y)

		px, py := reg.ToPhysical(e.X), reg.ToPhysical(e.Y)
		vx, vy := reg.ToVirtual(e.X), reg.ToVirtual(e.Y)

		switch {
		case vx != nil && vy != nil:
			rem = append(rem, e)
			continue
		case px != nil && py != nil:
			if reg.AreConflicting(px, py) {
				return errors.New("impossible register allocation")
			}
		case px != nil && vy != nil:
			a.discardconflicting(vy, px)
		case vx != nil && py != nil:
			a.discardconflicting(vx, py)
		default:
			panic("unreachable")
		}
	}
	a.edges = rem

	return nil
}

// mostrestricted returns the virtual register with the least possibilities.
func (a *Allocator) mostrestricted() reg.Virtual {
	n := int(math.MaxInt32)
	var v reg.Virtual
	for r, p := range a.possible {
		if len(p) < n || (len(p) == n && v != nil && r.VirtualID() < v.VirtualID()) {
			n = len(p)
			v = r
		}
	}
	return v
}

// discardconflicting removes registers from vs possible list that conflict with p.
func (a *Allocator) discardconflicting(v reg.Virtual, p reg.Physical) {
	a.possible[v] = filterregisters(a.possible[v], func(r reg.Physical) bool {
		if pid, found := a.vidtopid[v.VirtualID()]; found && pid == p.PhysicalID() {
			return true
		}
		return !reg.AreConflicting(r, p)
	})
}

// alloc attempts to allocate a register to v.
func (a *Allocator) alloc(v reg.Virtual) error {
	ps := a.possible[v]
	if len(ps) == 0 {
		return errors.New("failed to allocate registers")
	}
	p := ps[0]
	a.allocation[v] = p
	delete(a.possible, v)
	a.vidtopid[v.VirtualID()] = p.PhysicalID()
	return nil
}

// remaining returns the number of unallocated registers.
func (a *Allocator) remaining() int {
	return len(a.possible)
}

// possibleregisters returns all allocate-able registers for the given virtual.
func (a *Allocator) possibleregisters(v reg.Virtual) []reg.Physical {
	return filterregisters(a.registers, func(r reg.Physical) bool {
		return v.SatisfiedBy(r) && (r.Info()&reg.Restricted) == 0
	})
}

func filterregisters(in []reg.Physical, predicate func(reg.Physical) bool) []reg.Physical {
	var rs []reg.Physical
	for _, r := range in {
		if predicate(r) {
			rs = append(rs, r)
		}
	}
	return rs
}
