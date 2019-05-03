package geohash

import "testing"

//go:generate go run asm.go -out geohash.s -stubs stub.go

func TestEncodeIntMountEverest(t *testing.T) {
	if EncodeInt(27.988056, 86.925278) != 0xceb7f254240fd612 {
		t.Fail()
	}
}
