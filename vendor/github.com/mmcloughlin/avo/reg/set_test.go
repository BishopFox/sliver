package reg

import "testing"

func TestSetRegisterIdentity(t *testing.T) {
	rs := []Register{
		NewVirtual(42, KindGP, B32),
		NewVirtual(43, KindGP, B32),
		NewVirtual(42, KindVector, B32),
		NewVirtual(42, KindGP, B64),
		AL, AH, CL,
		AX, R13W,
		EDX, R9L,
		RCX, R14,
		X1, X7,
		Y4, Y9,
		Z13, Z31,
	}
	s := NewEmptySet()
	for _, r := range rs {
		s.Add(r)
		s.Add(r)
	}
	if len(s) != len(rs) {
		t.Fatalf("expected set to have same size as slice: got %d expect %d", len(s), len(rs))
	}
}

func TestSetFamilyRegisters(t *testing.T) {
	fs := []*Family{GeneralPurpose, Vector}
	s := NewEmptySet()
	expect := 0
	for _, f := range fs {
		s.Update(f.Set())
		s.Add(f.Virtual(42, B64))
		expect += len(f.Registers()) + 1
	}
	if len(s) != expect {
		t.Fatalf("set size mismatch: %d expected %d", len(s), expect)
	}
}
