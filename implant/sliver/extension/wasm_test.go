package extension

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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

import "testing"

var (
	extFS = map[string][]byte{
		"/test.1": {0x00},
	}
)

func TestWasmFS1(t *testing.T) {
	wasmFS := WasmMemFS{memFS: extFS}

	// Test ReadDir
	fi, err := wasmFS.Open("/test.1")
	if err != nil {
		t.Fatal(err)
	}
	if stat, _ := fi.Stat(); stat.Name() != "test.1" {
		t.Fatalf("expected test.1, got %s", stat.Name())
	}
	defer fi.Close()
}
