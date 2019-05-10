package fnv1a

import (
	"hash/fnv"
	"testing"
	"testing/quick"
)

//go:generate go run asm.go -out fnv1a.s -stubs stub.go

func TestHash64(t *testing.T) {
	expect := func(data []byte) uint64 {
		h := fnv.New64a()
		if _, err := h.Write(data); err != nil {
			t.Fatal(err)
		}
		return h.Sum64()
	}
	if err := quick.CheckEqual(Hash64, expect, nil); err != nil {
		t.Fatal(err)
	}
}
