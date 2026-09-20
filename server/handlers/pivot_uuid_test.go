package handlers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"testing"
	uuid "uuid"
)

func TestPivotUUIDBytes(t *testing.T) {
	want := uuid.NewV4()
	raw := uuidBytes(want)
	if len(raw) != len(want) {
		t.Fatalf("uuidBytes length = %d, want %d", len(raw), len(want))
	}

	got, err := uuidFromBytes(raw)
	if err != nil {
		t.Fatalf("uuidFromBytes: %v", err)
	}
	if got != want {
		t.Fatalf("uuidFromBytes = %q, want %q", got, want)
	}

	raw[0] ^= 0xff
	if got != want {
		t.Fatal("uuidFromBytes retained an alias to its input")
	}

	for _, invalidLength := range []int{0, 15, 17} {
		if _, err := uuidFromBytes(make([]byte, invalidLength)); err == nil {
			t.Fatalf("uuidFromBytes accepted %d bytes", invalidLength)
		}
	}
}
