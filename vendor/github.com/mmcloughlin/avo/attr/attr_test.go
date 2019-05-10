package attr

import "testing"

func TestAttributeAsm(t *testing.T) {
	cases := []struct {
		Attribute Attribute
		Expect    string
	}{
		{0, "0"},
		{32768, "32768"},
		{1, "NOPROF"},
		{DUPOK, "DUPOK"},
		{RODATA | NOSPLIT, "NOSPLIT|RODATA"},
		{WRAPPER | 16384 | NOPTR, "NOPTR|WRAPPER|16384"},
		{NEEDCTXT + NOFRAME + TLSBSS, "NEEDCTXT|TLSBSS|NOFRAME"},
		{REFLECTMETHOD, "1024"}, // REFLECTMETHOD special case due to https://golang.org/issue/29487
	}
	for _, c := range cases {
		got := c.Attribute.Asm()
		if got != c.Expect {
			t.Errorf("Attribute(%d).Asm() = %#v; expect %#v", c.Attribute, got, c.Expect)
		}
	}
}

func TestAttributeContainsTextFlags(t *testing.T) {
	cases := []struct {
		Attribute Attribute
		Expect    bool
	}{
		{0, false},
		{32768, false},
		{1, true},
		{DUPOK, true},
		{WRAPPER | 16384 | NOPTR, true},
	}
	for _, c := range cases {
		if c.Attribute.ContainsTextFlags() != c.Expect {
			t.Errorf("%s: ContainsTextFlags() expected %#v", c.Attribute.Asm(), c.Expect)
		}
	}
}
