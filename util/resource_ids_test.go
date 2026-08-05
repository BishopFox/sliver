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
	"reflect"
	"testing"
)

func TestRemoveElement(t *testing.T) {
	tests := []struct {
		name     string
		slice    []uint64
		value    uint64
		expected []uint64
	}{
		{
			name:     "removes every occurrence",
			slice:    []uint64{1, 2, 3, 2, 4, 2},
			value:    2,
			expected: []uint64{1, 3, 4},
		},
		{
			name:     "value not present leaves slice unchanged",
			slice:    []uint64{1, 2, 3},
			value:    9,
			expected: []uint64{1, 2, 3},
		},
		{
			name:     "single matching element yields empty slice",
			slice:    []uint64{7},
			value:    7,
			expected: []uint64{},
		},
		{
			name:     "empty input yields empty slice",
			slice:    []uint64{},
			value:    1,
			expected: []uint64{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := RemoveElement(test.slice, test.value)
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("RemoveElement(%v, %d) = %v, want %v", test.slice, test.value, got, test.expected)
			}
		})
	}
}

func TestRemoveElementDoesNotMutateInput(t *testing.T) {
	original := []uint64{1, 2, 3, 2}
	snapshot := append([]uint64{}, original...)
	RemoveElement(original, 2)
	if !reflect.DeepEqual(original, snapshot) {
		t.Fatalf("RemoveElement mutated its input: got %v, want %v", original, snapshot)
	}
}
