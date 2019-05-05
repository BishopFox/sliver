package reg

import "testing"

func TestSpecSize(t *testing.T) {
	cases := []struct {
		Spec Spec
		Size uint
	}{
		{S0, 0},
		{S8L, 1},
		{S8H, 1},
		{S16, 2},
		{S32, 4},
		{S64, 8},
		{S128, 16},
		{S256, 32},
		{S512, 64},
	}
	for _, c := range cases {
		if c.Spec.Size() != c.Size {
			t.Errorf("%v.Size() = %d; expect = %d", c.Spec, c.Spec.Size(), c.Size)
		}
	}
}

func TestToVirtual(t *testing.T) {
	v := GeneralPurpose.Virtual(42, B32)
	if ToVirtual(v) != v {
		t.Errorf("ToVirtual(v) != v for virtual register")
	}
	if ToVirtual(ECX) != nil {
		t.Errorf("ToVirtual should be nil for physical registers")
	}
}

func TestToPhysical(t *testing.T) {
	v := GeneralPurpose.Virtual(42, B32)
	if ToPhysical(v) != nil {
		t.Errorf("ToPhysical should be nil for virtual registers")
	}
	if ToPhysical(ECX) != ECX {
		t.Errorf("ToPhysical(p) != p for physical register")
	}
}

func TestAreConflicting(t *testing.T) {
	cases := []struct {
		X, Y   Physical
		Expect bool
	}{
		{ECX, X3, false},
		{AL, AH, false},
		{AL, AX, true},
		{AL, BX, false},
		{X3, Y4, false},
		{X3, Y3, true},
		{Y3, Z4, false},
		{Y3, Z3, true},
	}
	for _, c := range cases {
		if AreConflicting(c.X, c.Y) != c.Expect {
			t.Errorf("AreConflicting(%s, %s) != %v", c.X, c.Y, c.Expect)
		}
	}
}

func TestFamilyLookup(t *testing.T) {
	cases := []struct {
		Family *Family
		ID     PID
		Spec   Spec
		Expect Physical
	}{
		{GeneralPurpose, 0, S8, AL},
		{GeneralPurpose, 1, S8L, CL},
		{GeneralPurpose, 2, S8H, DH},
		{GeneralPurpose, 3, S16, BX},
		{GeneralPurpose, 9, S32, R9L},
		{GeneralPurpose, 13, S64, R13},
		{GeneralPurpose, 13, S512, nil},
		{GeneralPurpose, 133, S64, nil},
		{Vector, 1, S128, X1},
		{Vector, 13, S256, Y13},
		{Vector, 27, S512, Z27},
		{Vector, 1, S16, nil},
		{Vector, 299, S256, nil},
	}
	for _, c := range cases {
		got := c.Family.Lookup(c.ID, c.Spec)
		if got != c.Expect {
			t.Errorf("pid=%v spec=%v: lookup got %v expect %v", c.ID, c.Spec, got, c.Expect)
		}
	}
}

func TestPhysicalAs(t *testing.T) {
	cases := []struct {
		Register Physical
		Spec     Spec
		Expect   Physical
	}{
		{DX, S8L, DL},
		{DX, S8H, DH},
		{DX, S8, DL},
		{DX, S16, DX},
		{DX, S32, EDX},
		{DX, S64, RDX},
		{DX, S256, nil},
	}
	for _, c := range cases {
		got := c.Register.as(c.Spec)
		if got != c.Expect {
			t.Errorf("%s.as(%v) = %v; expect %v", c.Register.Asm(), c.Spec, got, c.Expect)
		}
	}
}

func TestVirtualAs(t *testing.T) {
	cases := []struct {
		Virtual  Register
		Physical Physical
		Match    bool
	}{
		{GeneralPurpose.Virtual(0, B8), CL, true},
		{GeneralPurpose.Virtual(0, B8), CH, true},
		{GeneralPurpose.Virtual(0, B32).as(S8L), CL, true},
		{GeneralPurpose.Virtual(0, B32).as(S8L), CH, false},
		{GeneralPurpose.Virtual(0, B16).as(S32), R9L, true},
		{GeneralPurpose.Virtual(0, B16).as(S32), R9, false},
	}
	for _, c := range cases {
		if c.Virtual.(Virtual).SatisfiedBy(c.Physical) != c.Match {
			t.Errorf("%s.SatisfiedBy(%v) != %v", c.Virtual.Asm(), c.Physical, c.Match)
		}
	}
}
