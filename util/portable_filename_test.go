package util

import "testing"

func TestValidPortableFileName(t *testing.T) {
	for _, name := range []string{"implant", "implant.exe", "hello world.zip", "héllo.bin", ".hidden"} {
		if !ValidPortableFileName(name) {
			t.Errorf("valid artifact filename %q was rejected", name)
		}
	}
	for _, name := range []string{
		"", ".", "..", "CON", "con.exe", "LPT9.bin", "a/b", `a\b`, "a:b", "a?b",
		"bad\nname", "bad\u2028name", "trailing.", " leading", "trailing ", string([]byte{'b', 0xff}),
	} {
		if ValidPortableFileName(name) {
			t.Errorf("unsafe artifact filename %q was accepted", name)
		}
	}
}
