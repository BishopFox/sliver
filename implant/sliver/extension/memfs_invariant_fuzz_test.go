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
	"strings"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
)

func FuzzWasmMemoryFSReadOnlyInvariant(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	f.Add(bytes.Repeat([]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}, 8))
	f.Add(bytes.Repeat([]byte{0xff}, 64))

	f.Fuzz(func(t *testing.T, program []byte) {
		if len(program) > 256 {
			return
		}
		raw, err := NewWasmMemoryFS(
			map[string][]byte{
				"seed.txt":       []byte("sentinel"),
				"dir/nested.txt": []byte("nested"),
			},
			WithWasmMemoryFSReadOnly(),
		)
		if err != nil {
			t.Fatalf("NewWasmMemoryFS(read-only): %v", err)
		}
		memFS := &memFSTestFS{fs: raw}
		before, errno := memFS.Stat("seed.txt")
		memFSTestRequireErrno(t, 0, errno)

		for _, instruction := range program {
			var mutationErrno experimentalsys.Errno
			switch instruction % 10 {
			case 0:
				file, errno := memFS.OpenFile("seed.txt", experimentalsys.O_RDWR, 0)
				if file != nil {
					_ = file.Close()
				}
				mutationErrno = errno
			case 1:
				file, errno := memFS.OpenFile("new.txt", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
				if file != nil {
					_, _ = file.Write([]byte("changed"))
					_ = file.Close()
				}
				mutationErrno = errno
			case 2:
				mutationErrno = memFS.Mkdir("new-dir", 0o700)
			case 3:
				mutationErrno = memFS.Chmod("seed.txt", 0o777)
			case 4:
				mutationErrno = memFS.Rename("seed.txt", "renamed.txt")
			case 5:
				mutationErrno = memFS.Rmdir("dir")
			case 6:
				mutationErrno = memFS.Unlink("seed.txt")
			case 7:
				mutationErrno = memFS.Link("seed.txt", "hard.txt")
			case 8:
				mutationErrno = memFS.Symlink("seed.txt", "sym.txt")
			case 9:
				mutationErrno = memFS.Utimens("seed.txt", int64(instruction), int64(instruction)+1)
			}
			if mutationErrno != experimentalsys.EROFS {
				t.Fatalf("read-only instruction %d returned %v", instruction%10, mutationErrno)
			}
		}

		after, errno := memFS.Stat("seed.txt")
		memFSTestRequireErrno(t, 0, errno)
		if after != before {
			t.Fatalf("read-only operations changed metadata: before=%+v after=%+v", before, after)
		}
		if got := memFSTestReadPath(t, memFS, "seed.txt"); !bytes.Equal(got, []byte("sentinel")) {
			t.Fatalf("read-only operations changed contents: %q", got)
		}
		for _, absent := range []string{"new.txt", "new-dir", "renamed.txt", "hard.txt", "sym.txt"} {
			if _, errno := memFS.Lstat(absent); errno != experimentalsys.ENOENT {
				t.Fatalf("read-only operation created %q: %v", absent, errno)
			}
		}
	})
}

func FuzzWasmMemoryFSPathBoundaries(f *testing.F) {
	for _, seed := range []string{
		"",
		".",
		"..",
		"../host",
		"nested/../../host",
		"nul\x00byte",
		strings.Repeat("n", wasmMemoryFSMaxNameBytes),
		strings.Repeat("n", wasmMemoryFSMaxNameBytes+1),
		wasmMemoryFSEdgeValidPath(wasmMemoryFSMaxPathBytes - 1),
		wasmMemoryFSEdgeValidPath(wasmMemoryFSMaxPathBytes),
		wasmMemoryFSEdgeValidPath(wasmMemoryFSMaxPathBytes + 1),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, rawPath string) {
		if len(rawPath) > wasmMemoryFSMaxPathBytes*2 {
			return
		}
		segments, errno := normalizeWasmMemoryPath(rawPath)
		if errno == 0 {
			if len(rawPath) > wasmMemoryFSMaxPathBytes {
				t.Fatalf("accepted %d-byte path", len(rawPath))
			}
			for _, segment := range segments {
				if segment == "" || segment == "." || segment == ".." {
					t.Fatalf("accepted unsafe path segment %q in %q", segment, rawPath)
				}
				if len(segment) > wasmMemoryFSMaxNameBytes {
					t.Fatalf("accepted %d-byte path segment", len(segment))
				}
			}
		}

		raw, err := NewWasmMemoryFS(map[string][]byte{"safe.txt": []byte("sentinel")})
		if err != nil {
			t.Fatalf("NewWasmMemoryFS: %v", err)
		}
		file, _ := raw.OpenFile("memfs/"+rawPath, experimentalsys.O_RDONLY, 0)
		if file != nil {
			_ = file.Close()
		}
		if got := memFSTestReadPath(t, &memFSTestFS{fs: raw}, "safe.txt"); !bytes.Equal(got, []byte("sentinel")) {
			t.Fatalf("path %q escaped its namespace: %q", rawPath, got)
		}
	})
}
