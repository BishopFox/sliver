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
	"bytes"
	"io"
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
	stat, err := fi.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name() != "memfs/test.1" {
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
	wantEntries := map[string]bool{"memfs/test.1": false, "memfs/test.2": false, "memfs/a": true}
	if len(entries) != len(wantEntries) {
		t.Fatalf("expected %d entries, got %d", len(wantEntries), len(entries))
	}
	for _, entry := range entries {
		wantDir, ok := wantEntries[entry.Name()]
		if !ok || entry.IsDir() != wantDir {
			t.Fatalf("unexpected entry %q (directory=%v)", entry.Name(), entry.IsDir())
		}
		delete(wantEntries, entry.Name())
	}

	di, err := dirNode.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if di.Name() != "memfs" || !di.IsDir() {
		t.Fatalf("expected memfs root directory, got %q (directory=%v)", di.Name(), di.IsDir())
	}
}

func TestBasicMemFSOpenDir(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test/foo.1":             {0x00},
		"test/foo":               {0x00},
		"testing/foo":            {0x00},
		"a/b/c/d/e/f/test/a.txt": {0x31, 0x00, 0x7f},
	}}

	// Test Open File
	f, err := wasmFS.Open("memfs/test")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected file, got nil and no error")
	}
	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name() != "memfs/test" || !stat.IsDir() {
		t.Fatalf("expected 'memfs/test' dir, got '%s'", stat.Name())
	}
	defer f.Close()

	dir, err := wasmFS.Open("memfs/a/b/c/d/e/f/test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dir.Close(); err != nil {
			t.Errorf("close directory: %v", err)
		}
	}()
	stat, err = dir.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name() != "memfs/a/b/c/d/e/f/test" || !stat.IsDir() {
		t.Fatalf("expected 'memfs/a/b/c/d/e/f/test' dir, got '%s'", stat.Name())
	}

	f, err = wasmFS.Open("memfs/a/b/c/d/e/f/test/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close file: %v", err)
		}
	}()
	assertMemFSFileContents(t, f, []byte{0x31, 0x00, 0x7f})
}

func TestMemFSReadFileData(t *testing.T) {
	wasmFS := WasmMemoryFS{memFS: map[string][]byte{
		"test/foo.1": {0x41, 0x00, 0xff, 0x42},
	}}

	// Test Open File
	fi, err := wasmFS.Open("memfs/test/foo.1")
	if err != nil {
		t.Fatal(err)
	}
	stat, err := fi.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.Name() != "memfs/test/foo.1" {
		t.Fatalf("expected memfs/test/foo.1, got %s", stat.Name())
	}
	defer fi.Close()

	assertMemFSFileContents(t, fi, []byte{0x41, 0x00, 0xff, 0x42})
}

func assertMemFSFileContents(t *testing.T, file fs.File, want []byte) {
	t.Helper()
	// Read in chunks to verify byte counts and advancement of the file offset.
	for offset := 0; offset < len(want); {
		end := min(offset+2, len(want))
		data := make([]byte, end-offset)
		n, err := file.Read(data)
		if err != nil || n != len(data) || !bytes.Equal(data, want[offset:end]) {
			t.Fatalf("Read at %d = (%x, %d, %v), want (%x, %d, nil)", offset, data, n, err, want[offset:end], len(data))
		}
		offset = end
	}
	if n, err := file.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Fatalf("Read after file contents = (%d, %v), want (0, EOF)", n, err)
	}
}
