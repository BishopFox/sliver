package textflag

import "testing"

func TestAttributes(t *testing.T) {
	if !attrtest() {
		t.Fail()
	}
}
