package pass

import (
	"testing"

	"github.com/mmcloughlin/avo/reg"
)

func TestAllocatorSimple(t *testing.T) {
	c := reg.NewCollection()
	x, y := c.XMM(), c.YMM()

	a, err := NewAllocatorForKind(reg.KindVector)
	if err != nil {
		t.Fatal(err)
	}

	a.Add(x)
	a.Add(y)
	a.AddInterference(x, y)

	alloc, err := a.Allocate()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(alloc)

	if alloc[x] != reg.X0 || alloc[y] != reg.Y1 {
		t.Fatalf("unexpected allocation")
	}
}

func TestAllocatorImpossible(t *testing.T) {
	a, err := NewAllocatorForKind(reg.KindVector)
	if err != nil {
		t.Fatal(err)
	}

	a.AddInterference(reg.X7, reg.Z7)

	_, err = a.Allocate()
	if err == nil {
		t.Fatal("expected allocation error")
	}
}
