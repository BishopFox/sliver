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

import (
	"io/fs"
	"testing"
)

func TestBasicMemFSOpenFile(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test.1": {0x00},
	}}

	// Test Open File
	fi, err := wasmFS.Open("memfs/test.1")
	if err != nil {
		t.Fatal(err)
	}
	if stat, _ := fi.Stat(); stat.Name() != "memfs/test.1" {
		t.Fatalf("expected memfs/test.1, got %s", stat.Name())
	}
	defer fi.Close()
}

func TestMemFSOpenRoot(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test.1":       {0x00},
		"test.2":       {0x00},
		"a/b/c/test.3": {0x00},
	}}

	// Test Open File
	fi, err := wasmFS.Open("memfs")
	if err != nil {
		t.Fatal(err)
	}
	defer fi.Close()

	dirNode := fi.(fs.ReadDirFile)
	entries, err := dirNode.ReadDir(-1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		for entry := range entries {
			t.Logf("entry: %s", entries[entry].Name())
		}
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	di, err := dirNode.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("name: %s, is dir: %v", di.Name(), di.IsDir())
	for _, entry := range entries {
		t.Logf("entry: %s", entry.Name())
	}
}

func TestBasicMemFSOpenDir(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test/foo.1":             {0x00},
		"test/foo":               {0x00},
		"testing/foo":            {0x00},
		"a/b/c/d/e/f/test/a.txt": {0x00},
	}}

	// Test Open File
	f, err := wasmFS.Open("memfs/test")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected file, got nil and no error")
	}
	if stat, _ := f.Stat(); stat.Name() != "memfs/test" || !stat.IsDir() {
		t.Fatalf("expected 'memfs/test' dir, got '%s'", stat.Name())
	}
	defer f.Close()

	dir, err := wasmFS.Open("memfs/a/b/c/d/e/f/test")
	if err != nil {
		t.Fatal(err)
	}
	if stat, _ := dir.Stat(); stat.Name() != "memfs/a/b/c/d/e/f/test" || !stat.IsDir() {
		t.Fatalf("expected 'memfs/a/b/c/d/e/f/test' dir, got '%s'", stat.Name())
	}

	f, err = wasmFS.Open("memfs/a/b/c/d/e/f/test/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 1)
	_, err = f.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != 0x00 {
		t.Fatalf("expected 0x00, got %x", data[0])
	}
}

func TestMemFSReadFileData(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test/foo.1": {0x00},
	}}

	// Test Open File
	fi, err := wasmFS.Open("memfs/test/foo.1")
	if err != nil {
		t.Fatal(err)
	}
	if stat, _ := fi.Stat(); stat.Name() != "memfs/test/foo.1" {
		t.Fatalf("expected memfs/test/foo.1, got %s", stat.Name())
	}
	defer fi.Close()

	// Read file
	data := make([]byte, 1)
	_, err = fi.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != 0x00 {
		t.Fatalf("expected 0x00, got %x", data[0])
	}

}
