//go:build !windows

package util

import "testing"

func TestResolvePath(t *testing.T) {
	sample1 := "../../../../a/b/c"
	if ResolvePath(sample1) != "/a/b/c" {
		t.Fatalf("failed to resolve path correctly from: %s", sample1)
	}
	sample2 := "a/b/c/../../../.."
	if ResolvePath(sample2) != "/" {
		t.Fatalf("failed to resolve path correctly from: %s", sample2)
	}
}
