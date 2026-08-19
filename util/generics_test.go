package util

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"sort"
	"testing"
)

func TestContains(t *testing.T) {
	ints := []int{1, 2, 3, 4}
	if !Contains(ints, 3) {
		t.Fatalf("expected slice to contain 3")
	}
	if Contains(ints, 5) {
		t.Fatalf("expected slice not to contain 5")
	}

	strs := []string{"alpha", "beta", "gamma"}
	if !Contains(strs, "beta") {
		t.Fatalf("expected slice to contain \"beta\"")
	}
	if Contains(strs, "delta") {
		t.Fatalf("expected slice not to contain \"delta\"")
	}

	// An empty slice contains nothing.
	if Contains([]string{}, "anything") {
		t.Fatalf("expected empty slice to contain nothing")
	}
}

func TestKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(m)
	if len(keys) != len(m) {
		t.Fatalf("expected %d keys, got %d", len(m), len(keys))
	}
	sort.Strings(keys)
	expected := []string{"a", "b", "c"}
	for i := range expected {
		if keys[i] != expected[i] {
			t.Fatalf("expected key %q at index %d, got %q", expected[i], i, keys[i])
		}
	}

	// The keys of an empty map is an empty (non-nil) slice.
	empty := Keys(map[int]string{})
	if len(empty) != 0 {
		t.Fatalf("expected no keys for an empty map, got %d", len(empty))
	}
}
