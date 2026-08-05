package shell

import "testing"

func TestReplaceControlChar(t *testing.T) {
	controlChars := []uint8{1, 2, 3}
	original, ok := replaceControlChar(controlChars, 1, shellEscapeByte)
	if !ok {
		t.Fatal("replaceControlChar() unexpectedly rejected a valid index")
	}
	if original != 2 {
		t.Fatalf("replaceControlChar() original = %d, want 2", original)
	}
	if controlChars[1] != shellEscapeByte {
		t.Fatalf("replaceControlChar() value = %d, want %d", controlChars[1], shellEscapeByte)
	}

	if _, ok := replaceControlChar(controlChars, len(controlChars), shellEscapeByte); ok {
		t.Fatal("replaceControlChar() accepted an out-of-range index")
	}
}
