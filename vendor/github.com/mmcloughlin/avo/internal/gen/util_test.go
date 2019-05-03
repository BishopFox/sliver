package gen

import (
	"reflect"
	"testing"
)

func TestCross(t *testing.T) {
	x := [][]string{
		{"a", "b", "c"},
		{"1", "2"},
		{"!", "?"},
	}
	expect := [][]string{
		{"a", "1", "!"},
		{"a", "1", "?"},
		{"a", "2", "!"},
		{"a", "2", "?"},
		{"b", "1", "!"},
		{"b", "1", "?"},
		{"b", "2", "!"},
		{"b", "2", "?"},
		{"c", "1", "!"},
		{"c", "1", "?"},
		{"c", "2", "!"},
		{"c", "2", "?"},
	}
	got := cross(x)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("bad cross product")
	}
}

func TestCrossSimple(t *testing.T) {
	x := [][]string{
		{"a", "b"},
	}
	expect := [][]string{
		{"a"},
		{"b"},
	}
	got := cross(x)
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("bad cross product")
	}
}
