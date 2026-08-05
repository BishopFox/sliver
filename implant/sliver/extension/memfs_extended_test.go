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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	wazerosys "github.com/tetratelabs/wazero/sys"
)

// memFSTestFS qualifies low-level wazero filesystem operations into the
// composite filesystem's memfs namespace. Open is intentionally not qualified:
// it also exercises the legacy io/fs passthrough API.
type memFSTestFS struct {
	fs *WasmMemoryFS
}

func (m *memFSTestFS) Open(name string) (fs.File, error) {
	return m.fs.Open(name)
}

func (m *memFSTestFS) OpenFile(name string, flag experimentalsys.Oflag, perm fs.FileMode) (experimentalsys.File, experimentalsys.Errno) {
	return m.fs.OpenFile(memFSTestPath(name), flag, perm)
}

func (m *memFSTestFS) Lstat(name string) (wazerosys.Stat_t, experimentalsys.Errno) {
	return m.fs.Lstat(memFSTestPath(name))
}

func (m *memFSTestFS) Stat(name string) (wazerosys.Stat_t, experimentalsys.Errno) {
	return m.fs.Stat(memFSTestPath(name))
}

func (m *memFSTestFS) Mkdir(name string, perm fs.FileMode) experimentalsys.Errno {
	return m.fs.Mkdir(memFSTestPath(name), perm)
}

func (m *memFSTestFS) Chmod(name string, perm fs.FileMode) experimentalsys.Errno {
	return m.fs.Chmod(memFSTestPath(name), perm)
}

func (m *memFSTestFS) Rename(from, to string) experimentalsys.Errno {
	return m.fs.Rename(memFSTestPath(from), memFSTestPath(to))
}

func (m *memFSTestFS) Rmdir(name string) experimentalsys.Errno {
	return m.fs.Rmdir(memFSTestPath(name))
}

func (m *memFSTestFS) Unlink(name string) experimentalsys.Errno {
	return m.fs.Unlink(memFSTestPath(name))
}

func (m *memFSTestFS) Link(oldName, newName string) experimentalsys.Errno {
	return m.fs.Link(memFSTestPath(oldName), memFSTestPath(newName))
}

func (m *memFSTestFS) Symlink(oldName, linkName string) experimentalsys.Errno {
	return m.fs.Symlink(oldName, memFSTestPath(linkName))
}

func (m *memFSTestFS) Readlink(name string) (string, experimentalsys.Errno) {
	return m.fs.Readlink(memFSTestPath(name))
}

func (m *memFSTestFS) Utimens(name string, atim, mtim int64) experimentalsys.Errno {
	return m.fs.Utimens(memFSTestPath(name), atim, mtim)
}

func memFSTestPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	if name == "" || name == "." {
		return "memfs"
	}
	if name == "memfs" || strings.HasPrefix(name, "memfs/") {
		return name
	}
	return "memfs/" + name
}

func memFSTestNew(t *testing.T, files map[string][]byte) *memFSTestFS {
	t.Helper()

	memFS, err := NewWasmMemoryFS(files)
	if err != nil {
		t.Fatalf("NewWasmMemoryFS failed: %v", err)
	}
	return &memFSTestFS{fs: memFS}
}

func memFSTestOpen(t *testing.T, memFS *memFSTestFS, name string, flag experimentalsys.Oflag) experimentalsys.File {
	t.Helper()

	file, errno := memFS.OpenFile(name, flag, 0o600)
	if errno != 0 {
		t.Fatalf("OpenFile(%q, %#x) failed: %v", name, flag, errno)
	}
	return file
}

func memFSTestReadAll(t *testing.T, file experimentalsys.File) []byte {
	t.Helper()

	var result []byte
	buf := make([]byte, 7)
	for {
		n, errno := file.Read(buf)
		if errno != 0 {
			t.Fatalf("Read failed: %v", errno)
		}
		result = append(result, buf[:n]...)
		if n == 0 {
			return result
		}
	}
}

func memFSTestReadPath(t *testing.T, memFS *memFSTestFS, name string) []byte {
	t.Helper()

	file := memFSTestOpen(t, memFS, name, experimentalsys.O_RDONLY)
	defer func() { _ = file.Close() }()
	return memFSTestReadAll(t, file)
}

func memFSTestRequireErrno(t *testing.T, want, got experimentalsys.Errno) {
	t.Helper()
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestWasmMemoryFSExtendedInputIsCopied(t *testing.T) {
	contents := []byte("original")
	files := map[string][]byte{"seed.txt": contents}
	memFS := memFSTestNew(t, files)

	copy(contents, "mutated!")
	delete(files, "seed.txt")
	files["added.txt"] = []byte("late")

	if got := memFSTestReadPath(t, memFS, "seed.txt"); !bytes.Equal(got, []byte("original")) {
		t.Fatalf("constructor retained caller-owned contents: %q", got)
	}
	if _, errno := memFS.Stat("added.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("constructor retained caller-owned map: %v", errno)
	}

	file := memFSTestOpen(t, memFS, "seed.txt", experimentalsys.O_RDWR)
	if n, errno := file.Pwrite([]byte("changed"), 0); errno != 0 || n != len("changed") {
		t.Fatalf("Pwrite failed: n=%d errno=%v", n, errno)
	}
	memFSTestRequireErrno(t, 0, file.Close())
	if got := string(contents); got != "mutated!" {
		t.Fatalf("filesystem write changed caller-owned bytes: %q", got)
	}
}

func TestWasmMemoryFSExtendedOpenFlagsAndAccess(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{
		"existing.txt": []byte("existing"),
	})

	if _, errno := memFS.OpenFile("missing.txt", experimentalsys.O_RDONLY, 0); errno != experimentalsys.ENOENT {
		t.Fatalf("open missing file returned %v", errno)
	}

	readOnly := memFSTestOpen(t, memFS, "existing.txt", experimentalsys.O_RDONLY)
	if n, errno := readOnly.Write([]byte("x")); errno != experimentalsys.EBADF || n != 0 {
		t.Fatalf("write through read-only handle: n=%d errno=%v", n, errno)
	}
	memFSTestRequireErrno(t, 0, readOnly.Close())

	writeOnly := memFSTestOpen(t, memFS, "existing.txt", experimentalsys.O_WRONLY)
	if n, errno := writeOnly.Read(make([]byte, 1)); errno != experimentalsys.EBADF || n != 0 {
		t.Fatalf("read through write-only handle: n=%d errno=%v", n, errno)
	}
	memFSTestRequireErrno(t, 0, writeOnly.Close())

	created, errno := memFS.OpenFile(
		"created.txt",
		experimentalsys.O_CREAT|experimentalsys.O_EXCL|experimentalsys.O_RDWR,
		0o640,
	)
	memFSTestRequireErrno(t, 0, errno)
	if n, errno := created.Write([]byte("created")); errno != 0 || n != len("created") {
		t.Fatalf("write created file: n=%d errno=%v", n, errno)
	}
	memFSTestRequireErrno(t, 0, created.Close())

	if _, errno := memFS.OpenFile(
		"created.txt",
		experimentalsys.O_CREAT|experimentalsys.O_EXCL|experimentalsys.O_RDWR,
		0o600,
	); errno != experimentalsys.EEXIST {
		t.Fatalf("exclusive create returned %v", errno)
	}

	truncated := memFSTestOpen(t, memFS, "created.txt", experimentalsys.O_RDWR|experimentalsys.O_TRUNC)
	st, errno := truncated.Stat()
	memFSTestRequireErrno(t, 0, errno)
	if st.Size != 0 {
		t.Fatalf("O_TRUNC left size %d", st.Size)
	}
	memFSTestRequireErrno(t, 0, truncated.Close())

	if _, errno := memFS.OpenFile("existing.txt", experimentalsys.O_RDONLY|experimentalsys.O_DIRECTORY, 0); errno != experimentalsys.ENOTDIR {
		t.Fatalf("O_DIRECTORY on a file returned %v", errno)
	}
	if _, errno := memFS.OpenFile(".", experimentalsys.O_WRONLY, 0); errno != experimentalsys.EISDIR {
		t.Fatalf("write-open root directory returned %v", errno)
	}
	if _, errno := memFS.OpenFile("existing.txt", experimentalsys.Oflag(3), 0); errno != experimentalsys.EINVAL {
		t.Fatalf("invalid access mode returned %v", errno)
	}
}

//nolint:gocyclo // The test validates the complete positioned-I/O state sequence.
func TestWasmMemoryFSExtendedPositionedIOSeekAndTruncate(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"position.txt": []byte("abcdef")})
	file := memFSTestOpen(t, memFS, "position.txt", experimentalsys.O_RDWR)
	defer func() { _ = file.Close() }()

	if offset, errno := file.Seek(2, io.SeekStart); errno != 0 || offset != 2 {
		t.Fatalf("SeekStart: offset=%d errno=%v", offset, errno)
	}
	buf := make([]byte, 2)
	if n, errno := file.Pread(buf, 4); errno != 0 || n != 2 || string(buf) != "ef" {
		t.Fatalf("Pread: n=%d errno=%v data=%q", n, errno, buf)
	}
	if offset, errno := file.Seek(0, io.SeekCurrent); errno != 0 || offset != 2 {
		t.Fatalf("Pread changed offset: offset=%d errno=%v", offset, errno)
	}
	if n, errno := file.Pwrite([]byte("ZZ"), 0); errno != 0 || n != 2 {
		t.Fatalf("Pwrite: n=%d errno=%v", n, errno)
	}
	if offset, errno := file.Seek(0, io.SeekCurrent); errno != 0 || offset != 2 {
		t.Fatalf("Pwrite changed offset: offset=%d errno=%v", offset, errno)
	}
	if n, errno := file.Read(buf); errno != 0 || n != 2 || string(buf) != "cd" {
		t.Fatalf("Read after positioned I/O: n=%d errno=%v data=%q", n, errno, buf)
	}

	if offset, errno := file.Seek(-1, io.SeekStart); errno != experimentalsys.EINVAL || offset != 0 {
		t.Fatalf("negative SeekStart: offset=%d errno=%v", offset, errno)
	}
	if offset, errno := file.Seek(-1, io.SeekEnd); errno != 0 || offset != 5 {
		t.Fatalf("SeekEnd: offset=%d errno=%v", offset, errno)
	}
	if n, errno := file.Write([]byte("!")); errno != 0 || n != 1 {
		t.Fatalf("Write after SeekEnd: n=%d errno=%v", n, errno)
	}

	sparse, errno := memFS.OpenFile("sparse.bin", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
	memFSTestRequireErrno(t, 0, errno)
	if offset, errno := sparse.Seek(5, io.SeekStart); errno != 0 || offset != 5 {
		t.Fatalf("sparse seek: offset=%d errno=%v", offset, errno)
	}
	if n, errno := sparse.Write([]byte{'x'}); errno != 0 || n != 1 {
		t.Fatalf("sparse write: n=%d errno=%v", n, errno)
	}
	if got := memFSTestReadPath(t, memFS, "sparse.bin"); !bytes.Equal(got, []byte{0, 0, 0, 0, 0, 'x'}) {
		t.Fatalf("sparse write did not zero-fill: %v", got)
	}
	memFSTestRequireErrno(t, 0, sparse.Truncate(8))
	if got := memFSTestReadPath(t, memFS, "sparse.bin"); !bytes.Equal(got, []byte{0, 0, 0, 0, 0, 'x', 0, 0}) {
		t.Fatalf("truncate growth did not zero-fill: %v", got)
	}
	memFSTestRequireErrno(t, 0, sparse.Truncate(3))
	if got := memFSTestReadPath(t, memFS, "sparse.bin"); !bytes.Equal(got, []byte{0, 0, 0}) {
		t.Fatalf("truncate shrink failed: %v", got)
	}
	if errno := sparse.Truncate(-1); errno != experimentalsys.EINVAL {
		t.Fatalf("negative truncate returned %v", errno)
	}
	memFSTestRequireErrno(t, 0, sparse.Close())
	memFSTestRequireErrno(t, 0, sparse.Close())
	if _, errno := sparse.Read(make([]byte, 1)); errno != experimentalsys.EBADF {
		t.Fatalf("read after close returned %v", errno)
	}
}

func TestWasmMemoryFSExtendedAppend(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"append.txt": []byte("a")})
	file := memFSTestOpen(t, memFS, "append.txt", experimentalsys.O_WRONLY|experimentalsys.O_APPEND)

	if offset, errno := file.Seek(0, io.SeekStart); errno != 0 || offset != 0 {
		t.Fatalf("seek append handle: offset=%d errno=%v", offset, errno)
	}
	if n, errno := file.Write([]byte("b")); errno != 0 || n != 1 {
		t.Fatalf("append write: n=%d errno=%v", n, errno)
	}
	if got := string(memFSTestReadPath(t, memFS, "append.txt")); got != "ab" {
		t.Fatalf("append ignored end of file: %q", got)
	}

	memFSTestRequireErrno(t, 0, file.SetAppend(false))
	if file.IsAppend() {
		t.Fatal("SetAppend(false) did not clear append mode")
	}
	if offset, errno := file.Seek(0, io.SeekStart); errno != 0 || offset != 0 {
		t.Fatalf("seek non-append handle: offset=%d errno=%v", offset, errno)
	}
	if n, errno := file.Write([]byte("z")); errno != 0 || n != 1 {
		t.Fatalf("non-append write: n=%d errno=%v", n, errno)
	}
	if got := string(memFSTestReadPath(t, memFS, "append.txt")); got != "zb" {
		t.Fatalf("disabled append still appended: %q", got)
	}
	memFSTestRequireErrno(t, 0, file.Close())
}

//nolint:gocyclo // The test covers directory iteration and related metadata as one scenario.
func TestWasmMemoryFSExtendedDirectoriesAndMetadata(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{
		"a/one.txt":       []byte("one"),
		"a/two.txt":       []byte("two"),
		"a/sub/three.txt": []byte("three"),
	})

	root, errno := memFS.Stat(".")
	memFSTestRequireErrno(t, 0, errno)
	if root.Mode&fs.ModeDir == 0 || root.Ino == 0 {
		t.Fatalf("invalid root stat: %+v", root)
	}
	dirStat, errno := memFS.Stat("a")
	memFSTestRequireErrno(t, 0, errno)
	if dirStat.Mode&fs.ModeDir == 0 || dirStat.Ino == 0 {
		t.Fatalf("invalid directory stat: %+v", dirStat)
	}

	memFSTestRequireErrno(t, 0, memFS.Mkdir("empty", 0o750))
	if errno := memFS.Mkdir("empty", 0o750); errno != experimentalsys.EEXIST {
		t.Fatalf("duplicate Mkdir returned %v", errno)
	}
	if errno := memFS.Mkdir("a/one.txt/child", 0o700); errno != experimentalsys.ENOTDIR {
		t.Fatalf("Mkdir below file returned %v", errno)
	}

	dir := memFSTestOpen(t, memFS, "a", experimentalsys.O_RDONLY|experimentalsys.O_DIRECTORY)
	defer func() { _ = dir.Close() }()
	var names []string
	for {
		entries, errno := dir.Readdir(1)
		memFSTestRequireErrno(t, 0, errno)
		if len(entries) == 0 {
			break
		}
		if len(entries) != 1 {
			t.Fatalf("Readdir(1) returned %d entries", len(entries))
		}
		names = append(names, entries[0].Name)
	}
	sort.Strings(names)
	if want := []string{"one.txt", "sub", "two.txt"}; !equalStrings(names, want) {
		t.Fatalf("directory entries: got %v want %v", names, want)
	}
	if offset, errno := dir.Seek(0, io.SeekStart); errno != 0 || offset != 0 {
		t.Fatalf("rewind directory: offset=%d errno=%v", offset, errno)
	}
	entries, errno := dir.Readdir(-1)
	memFSTestRequireErrno(t, 0, errno)
	if len(entries) != 3 {
		t.Fatalf("Readdir(-1) after rewind returned %d entries", len(entries))
	}
	if _, errno := dir.Seek(1, io.SeekStart); errno != experimentalsys.EINVAL {
		t.Fatalf("nonzero directory seek returned %v", errno)
	}

	file := memFSTestOpen(t, memFS, "a/one.txt", experimentalsys.O_RDWR)
	before, errno := file.Stat()
	memFSTestRequireErrno(t, 0, errno)
	if got := memFSTestReadAll(t, file); string(got) != "one" {
		t.Fatalf("read file: %q", got)
	}
	after, errno := file.Stat()
	memFSTestRequireErrno(t, 0, errno)
	if before.Size != after.Size || after.Size != 3 {
		t.Fatalf("read changed reported size: before=%d after=%d", before.Size, after.Size)
	}
	memFSTestRequireErrno(t, 0, file.Sync())
	memFSTestRequireErrno(t, 0, file.Datasync())

	memFSTestRequireErrno(t, 0, memFS.Chmod("a/one.txt", 0o601))
	st, errno := memFS.Stat("a/one.txt")
	memFSTestRequireErrno(t, 0, errno)
	if st.Mode.Perm() != 0o601 {
		t.Fatalf("Chmod mode = %o", st.Mode.Perm())
	}

	memFSTestRequireErrno(t, 0, memFS.Utimens("a/one.txt", 111, 222))
	st, errno = memFS.Stat("a/one.txt")
	memFSTestRequireErrno(t, 0, errno)
	if st.Atim != 111 || st.Mtim != 222 {
		t.Fatalf("Utimens times: atim=%d mtim=%d", st.Atim, st.Mtim)
	}
	memFSTestRequireErrno(t, 0, memFS.Utimens("a/one.txt", experimentalsys.UTIME_OMIT, 333))
	st, errno = memFS.Stat("a/one.txt")
	memFSTestRequireErrno(t, 0, errno)
	if st.Atim != 111 || st.Mtim != 333 {
		t.Fatalf("UTIME_OMIT times: atim=%d mtim=%d", st.Atim, st.Mtim)
	}
	memFSTestRequireErrno(t, 0, file.Close())
}

func TestWasmMemoryFSExtendedRenameUnlinkAndRmdir(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{
		"src/file.txt": []byte("source"),
		"dst/old.txt":  []byte("old"),
	})

	open := memFSTestOpen(t, memFS, "src/file.txt", experimentalsys.O_RDONLY)
	memFSTestRequireErrno(t, 0, memFS.Rename("src/file.txt", "dst/moved.txt"))
	if _, errno := memFS.Stat("src/file.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("renamed source still exists: %v", errno)
	}
	if got := string(memFSTestReadPath(t, memFS, "dst/moved.txt")); got != "source" {
		t.Fatalf("renamed contents: %q", got)
	}
	if got := string(memFSTestReadAll(t, open)); got != "source" {
		t.Fatalf("open handle did not survive rename: %q", got)
	}

	memFSTestRequireErrno(t, 0, memFS.Rename("dst/moved.txt", "dst/old.txt"))
	if got := string(memFSTestReadPath(t, memFS, "dst/old.txt")); got != "source" {
		t.Fatalf("rename-overwrite contents: %q", got)
	}
	if errno := memFS.Rmdir("dst"); errno != experimentalsys.ENOTEMPTY {
		t.Fatalf("Rmdir nonempty directory returned %v", errno)
	}
	if errno := memFS.Unlink("dst"); errno != experimentalsys.EISDIR {
		t.Fatalf("Unlink directory returned %v", errno)
	}
	if errno := memFS.Rmdir("dst/old.txt"); errno != experimentalsys.ENOTDIR {
		t.Fatalf("Rmdir file returned %v", errno)
	}

	unlinked := memFSTestOpen(t, memFS, "dst/old.txt", experimentalsys.O_RDONLY)
	memFSTestRequireErrno(t, 0, memFS.Unlink("dst/old.txt"))
	if _, errno := memFS.Stat("dst/old.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("unlinked path still exists: %v", errno)
	}
	if got := string(memFSTestReadAll(t, unlinked)); got != "source" {
		t.Fatalf("open handle did not survive unlink: %q", got)
	}
	memFSTestRequireErrno(t, 0, unlinked.Close())
	memFSTestRequireErrno(t, 0, memFS.Rmdir("dst"))

	memFSTestRequireErrno(t, 0, memFS.Mkdir("cycle", 0o700))
	memFSTestRequireErrno(t, 0, memFS.Mkdir("cycle/child", 0o700))
	if errno := memFS.Rename("cycle", "cycle/child/again"); errno != experimentalsys.EINVAL {
		t.Fatalf("rename directory into descendant returned %v", errno)
	}
	if errno := memFS.Rmdir("."); errno != experimentalsys.EINVAL && errno != experimentalsys.EPERM {
		t.Fatalf("remove root returned %v", errno)
	}
	memFSTestRequireErrno(t, 0, open.Close())
}

//nolint:gocyclo // Hard-link and symbolic-link compatibility are paired coverage.
func TestWasmMemoryFSExtendedLinks(t *testing.T) {
	t.Run("hard link", func(t *testing.T) {
		memFS := memFSTestNew(t, map[string][]byte{"target.txt": []byte("target")})
		if errno := memFS.Link("target.txt", "hard.txt"); errno == experimentalsys.ENOSYS {
			t.Skip("hard links are intentionally unsupported")
		} else {
			memFSTestRequireErrno(t, 0, errno)
		}
		target, errno := memFS.Stat("target.txt")
		memFSTestRequireErrno(t, 0, errno)
		hard, errno := memFS.Stat("hard.txt")
		memFSTestRequireErrno(t, 0, errno)
		if target.Ino == 0 || target.Ino != hard.Ino || target.Nlink < 2 || hard.Nlink < 2 {
			t.Fatalf("hard-link metadata target=%+v hard=%+v", target, hard)
		}
		file := memFSTestOpen(t, memFS, "hard.txt", experimentalsys.O_RDWR)
		if n, errno := file.Pwrite([]byte("X"), 0); errno != 0 || n != 1 {
			t.Fatalf("write hard link: n=%d errno=%v", n, errno)
		}
		memFSTestRequireErrno(t, 0, file.Close())
		if got := string(memFSTestReadPath(t, memFS, "target.txt")); got != "Xarget" {
			t.Fatalf("hard link did not share contents: %q", got)
		}
		memFSTestRequireErrno(t, 0, memFS.Unlink("target.txt"))
		if got := string(memFSTestReadPath(t, memFS, "hard.txt")); got != "Xarget" {
			t.Fatalf("hard link did not survive unlink: %q", got)
		}
	})

	t.Run("symbolic link", func(t *testing.T) {
		memFS := memFSTestNew(t, map[string][]byte{"target.txt": []byte("target")})
		if errno := memFS.Symlink("target.txt", "sym.txt"); errno == experimentalsys.ENOSYS {
			t.Skip("symbolic links are intentionally unsupported")
		} else {
			memFSTestRequireErrno(t, 0, errno)
		}
		target, errno := memFS.Readlink("sym.txt")
		memFSTestRequireErrno(t, 0, errno)
		if target != "target.txt" {
			t.Fatalf("Readlink = %q", target)
		}
		lst, errno := memFS.Lstat("sym.txt")
		memFSTestRequireErrno(t, 0, errno)
		if lst.Mode&fs.ModeSymlink == 0 {
			t.Fatalf("Lstat mode = %v", lst.Mode)
		}
		st, errno := memFS.Stat("sym.txt")
		memFSTestRequireErrno(t, 0, errno)
		if st.Mode&fs.ModeSymlink != 0 || st.Size != int64(len("target")) {
			t.Fatalf("Stat followed symlink incorrectly: %+v", st)
		}
		if got := string(memFSTestReadPath(t, memFS, "sym.txt")); got != "target" {
			t.Fatalf("open symlink: %q", got)
		}
		memFSTestRequireErrno(t, 0, memFS.Unlink("sym.txt"))
		if got := string(memFSTestReadPath(t, memFS, "target.txt")); got != "target" {
			t.Fatalf("unlink symlink changed target: %q", got)
		}
	})
}

func TestWasmMemoryFSExtendedReadOnly(t *testing.T) {
	rawMemFS, err := NewWasmMemoryFS(
		map[string][]byte{"seed.txt": []byte("seed")},
		WithWasmMemoryFSReadOnly(),
	)
	if err != nil {
		t.Fatalf("NewWasmMemoryFS(read-only) failed: %v", err)
	}
	memFS := &memFSTestFS{fs: rawMemFS}
	if got := string(memFSTestReadPath(t, memFS, "seed.txt")); got != "seed" {
		t.Fatalf("read-only read: %q", got)
	}

	mutations := []struct {
		name string
		do   func() experimentalsys.Errno
	}{
		{"open existing writable", func() experimentalsys.Errno {
			_, errno := memFS.OpenFile("seed.txt", experimentalsys.O_RDWR, 0)
			return errno
		}},
		{"create", func() experimentalsys.Errno {
			_, errno := memFS.OpenFile("new.txt", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
			return errno
		}},
		{"mkdir", func() experimentalsys.Errno { return memFS.Mkdir("dir", 0o700) }},
		{"chmod", func() experimentalsys.Errno { return memFS.Chmod("seed.txt", 0o600) }},
		{"rename", func() experimentalsys.Errno { return memFS.Rename("seed.txt", "renamed.txt") }},
		{"rmdir", func() experimentalsys.Errno { return memFS.Rmdir(".") }},
		{"unlink", func() experimentalsys.Errno { return memFS.Unlink("seed.txt") }},
		{"link", func() experimentalsys.Errno { return memFS.Link("seed.txt", "hard.txt") }},
		{"symlink", func() experimentalsys.Errno { return memFS.Symlink("seed.txt", "sym.txt") }},
		{"utimens", func() experimentalsys.Errno { return memFS.Utimens("seed.txt", 1, 2) }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if errno := mutation.do(); errno != experimentalsys.EROFS {
				t.Fatalf("read-only mutation returned %v", errno)
			}
		})
	}

	file := memFSTestOpen(t, memFS, "seed.txt", experimentalsys.O_RDONLY)
	if _, errno := file.Write([]byte("x")); errno != experimentalsys.EBADF {
		t.Fatalf("write through read-only handle returned %v", errno)
	}
	if _, errno := file.Pwrite([]byte("x"), 0); errno != experimentalsys.EBADF {
		t.Fatalf("pwrite through read-only handle returned %v", errno)
	}
	if errno := file.Truncate(0); errno != experimentalsys.EBADF {
		t.Fatalf("truncate through read-only handle returned %v", errno)
	}
	if errno := file.Utimens(1, 2); errno != experimentalsys.EBADF && errno != experimentalsys.EROFS {
		t.Fatalf("utimens through read-only handle returned %v", errno)
	}
	memFSTestRequireErrno(t, 0, file.Close())

	if got := string(memFSTestReadPath(t, memFS, "seed.txt")); got != "seed" {
		t.Fatalf("read-only mutation changed contents: %q", got)
	}
	if _, errno := memFS.Stat("new.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("read-only create changed namespace: %v", errno)
	}
}

//nolint:gocyclo // The legacy API and traversal checks form one compatibility scenario.
func TestWasmMemoryFSExtendedLegacyOpenAndPassthroughSafety(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"nested/seed.txt": []byte("memory")})
	for _, name := range []string{"memfs/nested/seed.txt", "/memfs/nested/seed.txt"} {
		file, err := memFS.Open(name)
		if err != nil {
			t.Fatalf("legacy Open(%q): %v", name, err)
		}
		contents, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil || string(contents) != "memory" {
			t.Fatalf("legacy Open(%q): contents=%q err=%v", name, contents, err)
		}
	}

	hostDir := t.TempDir()
	hostPath := filepath.Join(hostDir, "host.txt")
	if err := os.WriteFile(hostPath, []byte("host-sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	guestPath := memFSTestHostPath(hostPath)
	hostFile, err := memFS.Open(guestPath)
	if err != nil {
		t.Fatalf("passthrough Open(%q): %v", guestPath, err)
	}
	contents, err := io.ReadAll(hostFile)
	if err != nil {
		t.Fatalf("read passthrough: %v", err)
	}
	if string(contents) != "host-sentinel" {
		t.Fatalf("passthrough contents: %q", contents)
	}
	if writer, ok := hostFile.(io.Writer); ok {
		if _, err := writer.Write([]byte("changed")); err == nil {
			t.Fatal("passthrough file unexpectedly allowed writes")
		}
	}
	_ = hostFile.Close()
	if contents, err := os.ReadFile(hostPath); err != nil || string(contents) != "host-sentinel" {
		t.Fatalf("passthrough write changed host: contents=%q err=%v", contents, err)
	}

	for _, traversal := range []string{
		"memfs/../" + guestPath,
		"./memfs/../" + guestPath,
		"/memfs/../../" + guestPath,
		"memfs/nested/../../../" + guestPath,
	} {
		file, err := memFS.Open(traversal)
		if err == nil {
			_ = file.Close()
			t.Fatalf("virtual traversal unexpectedly opened %q", traversal)
		}
	}
	if _, errno := memFS.OpenFile("memfs/../escape.txt", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600); errno != experimentalsys.EINVAL && errno != experimentalsys.EPERM {
		t.Fatalf("low-level traversal returned %v", errno)
	}
	if _, errno := memFS.fs.OpenFile("./memfs/../escape.txt", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600); errno != experimentalsys.EINVAL && errno != experimentalsys.EPERM {
		t.Fatalf("dot-prefixed low-level traversal returned %v", errno)
	}
	if contents, err := os.ReadFile(hostPath); err != nil || string(contents) != "host-sentinel" {
		t.Fatalf("traversal changed host: contents=%q err=%v", contents, err)
	}
}

//nolint:gocyclo // The concurrency test validates append and positioned-write invariants together.
func TestWasmMemoryFSExtendedConcurrentAccess(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"append.log": nil, "blocks.bin": nil})

	const (
		writers = 16
		writes  = 32
	)
	record := []byte("record\n")
	errCh := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			file, errno := memFS.OpenFile("append.log", experimentalsys.O_WRONLY|experimentalsys.O_APPEND, 0)
			if errno != 0 {
				errCh <- errno
				return
			}
			defer func() { _ = file.Close() }()
			for j := 0; j < writes; j++ {
				if n, errno := file.Write(record); errno != 0 || n != len(record) {
					errCh <- fmt.Errorf("append n=%d errno=%v", n, errno)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	appended := memFSTestReadPath(t, memFS, "append.log")
	if want := writers * writes; bytes.Count(appended, record) != want || len(appended) != want*len(record) {
		t.Fatalf("concurrent append lost data: records=%d bytes=%d", bytes.Count(appended, record), len(appended))
	}

	const blockSize = 32
	blocks := memFSTestOpen(t, memFS, "blocks.bin", experimentalsys.O_RDWR)
	memFSTestRequireErrno(t, 0, blocks.Truncate(writers*blockSize))
	memFSTestRequireErrno(t, 0, blocks.Close())
	errCh = make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			file, errno := memFS.OpenFile("blocks.bin", experimentalsys.O_RDWR, 0)
			if errno != 0 {
				errCh <- errno
				return
			}
			defer func() { _ = file.Close() }()
			block := bytes.Repeat([]byte{byte(index + 1)}, blockSize)
			if n, errno := file.Pwrite(block, int64(index*blockSize)); errno != 0 || n != len(block) {
				errCh <- fmt.Errorf("pwrite n=%d errno=%v", n, errno)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	contents := memFSTestReadPath(t, memFS, "blocks.bin")
	for i := 0; i < writers; i++ {
		want := bytes.Repeat([]byte{byte(i + 1)}, blockSize)
		got := contents[i*blockSize : (i+1)*blockSize]
		if !bytes.Equal(got, want) {
			t.Fatalf("block %d was corrupted: %v", i, got)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func memFSTestHostPath(name string) string {
	if volume := filepath.VolumeName(name); volume != "" {
		name = strings.TrimPrefix(name, volume)
	}
	return strings.TrimPrefix(filepath.ToSlash(name), "/")
}
