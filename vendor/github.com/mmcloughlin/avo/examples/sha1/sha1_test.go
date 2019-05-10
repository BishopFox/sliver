package sha1

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out sha1.s -stubs stub.go

func TestVectors(t *testing.T) {
	cases := []struct {
		Data      string
		HexDigest string
	}{
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"The quick brown fox jumps over the lazy dog", "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"},
		{"The quick brown fox jumps over the lazy cog", "de9f2c7fd25e1b3afad3e85a0bd17d9b100db4b3"},
	}
	for _, c := range cases {
		digest := Sum([]byte(c.Data))
		got := hex.EncodeToString(digest[:])
		if got != c.HexDigest {
			t.Errorf("Sum(%#v) = %s; expect %s", c.Data, got, c.HexDigest)
		}
	}
}

func TestCmp(t *testing.T) {
	if err := quick.CheckEqual(Sum, sha1.Sum, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLengths(t *testing.T) {
	data := make([]byte, BlockSize)
	for n := 0; n <= BlockSize; n++ {
		got := Sum(data[:n])
		expect := sha1.Sum(data[:n])
		if !bytes.Equal(got[:], expect[:]) {
			t.Errorf("failed on length %d", n)
		}
	}
}
