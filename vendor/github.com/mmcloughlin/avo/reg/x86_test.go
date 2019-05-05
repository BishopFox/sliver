package reg

import "testing"

func TestAsMethods(t *testing.T) {
	cases := [][2]Register{
		{RAX.As8(), AL},
		{ECX.As8L(), CL},
		{EBX.As8H(), BH},
		{R9B.As16(), R9W},
		{DH.As32(), EDX},
		{R14L.As64(), R14},
		{X2.AsX(), X2},
		{X4.AsY(), Y4},
		{X9.AsZ(), Z9},
		{Y2.AsX(), X2},
		{Y4.AsY(), Y4},
		{Y9.AsZ(), Z9},
		{Z2.AsX(), X2},
		{Z4.AsY(), Y4},
		{Z9.AsZ(), Z9},
	}
	for _, c := range cases {
		if c[0] != c[1] {
			t.FailNow()
		}
	}
}
