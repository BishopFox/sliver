package opcodescsv

import "testing"

func TestBuildOrderMap(t *testing.T) {
	is, err := ReadFile("testdata/x86.v0.2.csv")
	if err != nil {
		t.Fatal(err)
	}

	orders := BuildOrderMap(is)

	for opcode, order := range orders {
		if order == UnknownOrder {
			t.Errorf("%s has unknown order", opcode)
		}
	}
}
