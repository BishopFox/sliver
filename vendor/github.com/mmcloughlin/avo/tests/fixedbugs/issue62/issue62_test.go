package issue62

import "testing"

//go:generate go run asm.go -out issue62.s

func TestPrivate(t *testing.T) {
	private()
}
