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
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
)

func FuzzWasmMemoryFSPaths(f *testing.F) {
	for _, seed := range []string{
		"",
		"file",
		"nested/file",
		".",
		"..",
		"../safe.txt",
		"nested/../../safe.txt",
		"/absolute",
		"double//slash",
		"trailing/",
		"nul\x00byte",
		"unicode-世界",
		`windows\path`,
		memFSFuzzStringOfLength('a', 255),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 256 {
			return
		}

		memFS := memFSTestNew(t, map[string][]byte{"safe.txt": []byte("sentinel")})
		memFSTestRequireErrno(t, 0, memFS.Mkdir("scratch", 0o700))
		candidate := "scratch/" + raw

		if file, _ := memFS.OpenFile(candidate, experimentalsys.O_RDONLY, 0); file != nil {
			_ = file.Close()
		}
		if file, _ := memFS.OpenFile(candidate, experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600); file != nil {
			_, _ = file.Write([]byte("fuzz"))
			_ = file.Close()
		}
		_, _ = memFS.Stat(candidate)
		_, _ = memFS.Lstat(candidate)
		_ = memFS.Chmod(candidate, 0o640)
		_ = memFS.Utimens(candidate, 1, 2)
		_ = memFS.Rename(candidate, "scratch/renamed")
		_ = memFS.Unlink(candidate)
		_ = memFS.Rmdir(candidate)
		_ = memFS.Mkdir(candidate, 0o700)
		_ = memFS.Link(candidate, "scratch/hard")
		_ = memFS.Symlink(candidate, "scratch/sym")
		_, _ = memFS.Readlink(candidate)

		if file, err := memFS.Open("memfs/" + candidate); err == nil {
			_, _ = io.ReadAll(file)
			_ = file.Close()
		}

		if got := memFSTestReadPath(t, memFS, "safe.txt"); !bytes.Equal(got, []byte("sentinel")) {
			t.Fatalf("path %q escaped its namespace and changed sentinel: %q", raw, got)
		}
	})
}

//nolint:gocyclo // The fuzz bytecode intentionally dispatches every filesystem operation.
func FuzzWasmMemoryFSOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 3, 'a', 'b', 2, 4, 0, 0, 4, 8, 0, 0})
	f.Add([]byte{1, 12, 'x', 'y', 5, 12, 4, 0, 3, 0xff, 0, 0})
	f.Add(bytes.Repeat([]byte{0, 2, 'z', '!'}, 16))

	f.Fuzz(func(t *testing.T, program []byte) {
		if len(program) > 256 {
			return
		}

		memFS := memFSTestNew(t, map[string][]byte{"file.bin": nil})
		file := memFSTestOpen(t, memFS, "file.bin", experimentalsys.O_RDWR)
		defer func() { _ = file.Close() }()

		model := []byte(nil)
		var offset int64
		for pc := 0; pc+3 < len(program); pc += 4 {
			opcode := program[pc] % 8
			arg := program[pc+1]
			payload := append([]byte(nil), program[pc+2:pc+4]...)

			switch opcode {
			case 0: // sequential write
				payload = payload[:int(arg)%3]
				n, errno := file.Write(payload)
				memFSTestRequireErrno(t, 0, errno)
				if n != len(payload) {
					t.Fatalf("Write count=%d want=%d", n, len(payload))
				}
				model = memFSFuzzWriteAt(model, offset, payload)
				offset += int64(len(payload))

			case 1: // positioned write
				writeOffset := int64(arg % 64)
				n, errno := file.Pwrite(payload, writeOffset)
				memFSTestRequireErrno(t, 0, errno)
				if n != len(payload) {
					t.Fatalf("Pwrite count=%d want=%d", n, len(payload))
				}
				model = memFSFuzzWriteAt(model, writeOffset, payload)

			case 2: // absolute seek
				want := int64(arg % 64)
				got, errno := file.Seek(want, io.SeekStart)
				memFSTestRequireErrno(t, 0, errno)
				if got != want {
					t.Fatalf("SeekStart=%d want=%d", got, want)
				}
				offset = want

			case 3: // relative seek, including invalid negative results
				delta := int64(int8(arg) % 16)
				want := offset + delta
				got, errno := file.Seek(delta, io.SeekCurrent)
				if want < 0 {
					memFSTestRequireErrno(t, experimentalsys.EINVAL, errno)
					continue
				}
				memFSTestRequireErrno(t, 0, errno)
				if got != want {
					t.Fatalf("SeekCurrent=%d want=%d", got, want)
				}
				offset = want

			case 4: // truncate
				size := int(arg % 64)
				memFSTestRequireErrno(t, 0, file.Truncate(int64(size)))
				if size <= len(model) {
					model = model[:size]
				} else {
					model = append(model, make([]byte, size-len(model))...)
				}

			case 5: // sequential read
				readLen := int(arg % 8)
				buf := make([]byte, readLen)
				n, errno := file.Read(buf)
				memFSTestRequireErrno(t, 0, errno)
				want := memFSFuzzReadAt(model, offset, readLen)
				if n != len(want) || !bytes.Equal(buf[:n], want) {
					t.Fatalf("Read at %d: n=%d data=%v want=%v", offset, n, buf[:n], want)
				}
				offset += int64(n)

			case 6: // positioned read
				readOffset := int64(arg % 64)
				buf := make([]byte, 2)
				n, errno := file.Pread(buf, readOffset)
				memFSTestRequireErrno(t, 0, errno)
				want := memFSFuzzReadAt(model, readOffset, len(buf))
				if n != len(want) || !bytes.Equal(buf[:n], want) {
					t.Fatalf("Pread at %d: n=%d data=%v want=%v", readOffset, n, buf[:n], want)
				}

			case 7: // close and reopen resets the sequential offset
				memFSTestRequireErrno(t, 0, file.Close())
				file = memFSTestOpen(t, memFS, "file.bin", experimentalsys.O_RDWR)
				offset = 0
			}

			st, errno := file.Stat()
			memFSTestRequireErrno(t, 0, errno)
			if st.Size != int64(len(model)) {
				t.Fatalf("size=%d want=%d after opcode %d", st.Size, len(model), opcode)
			}
		}

		if got := memFSTestReadPath(t, memFS, "file.bin"); !bytes.Equal(got, model) {
			t.Fatalf("final contents=%v want=%v", got, model)
		}
	})
}

func memFSFuzzWriteAt(model []byte, offset int64, contents []byte) []byte {
	if len(contents) == 0 {
		return model
	}
	end := int(offset) + len(contents)
	if end > len(model) {
		model = append(model, make([]byte, end-len(model))...)
	}
	copy(model[int(offset):], contents)
	return model
}

func memFSFuzzReadAt(model []byte, offset int64, length int) []byte {
	if offset >= int64(len(model)) || length == 0 {
		return nil
	}
	end := int(offset) + length
	if end > len(model) {
		end = len(model)
	}
	return model[int(offset):end]
}

func memFSFuzzStringOfLength(ch byte, count int) string {
	return string(bytes.Repeat([]byte{ch}, count))
}
