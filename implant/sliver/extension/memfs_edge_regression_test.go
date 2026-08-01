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
	"strings"
	"sync"
	"testing"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
)

func TestWasmMemoryFSEdgeRenameSamePath(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"seed.txt": []byte("seed")})

	memFSTestRequireErrno(t, 0, memFS.Rename("seed.txt", "seed.txt"))
	if got := string(memFSTestReadPath(t, memFS, "seed.txt")); got != "seed" {
		t.Fatalf("same-path rename changed contents: %q", got)
	}
	if errno := memFS.Rename("missing.txt", "missing.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("same-path rename of a missing file returned %v", errno)
	}

	readOnlyFS, err := NewWasmMemoryFS(
		map[string][]byte{"seed.txt": []byte("seed")},
		WithWasmMemoryFSReadOnly(),
	)
	if err != nil {
		t.Fatalf("NewWasmMemoryFS(read-only): %v", err)
	}
	readOnly := &memFSTestFS{fs: readOnlyFS}
	if errno := readOnly.Rename("seed.txt", "seed.txt"); errno != experimentalsys.EROFS {
		t.Fatalf("same-path rename bypassed read-only mode: %v", errno)
	}
}

func TestWasmMemoryFSEdgeCreateDirectoryFlagIsAtomic(t *testing.T) {
	memFS := memFSTestNew(t, nil)

	file, errno := memFS.OpenFile(
		"invalid",
		experimentalsys.O_CREAT|experimentalsys.O_DIRECTORY|experimentalsys.O_RDONLY,
		0o700,
	)
	if file != nil {
		_ = file.Close()
		t.Fatal("O_CREAT|O_DIRECTORY returned a file handle")
	}
	if errno != experimentalsys.EINVAL {
		t.Fatalf("O_CREAT|O_DIRECTORY returned %v", errno)
	}
	if _, errno := memFS.Lstat("invalid"); errno != experimentalsys.ENOENT {
		t.Fatalf("invalid open left a filesystem entry behind: %v", errno)
	}
}

func TestWasmMemoryFSEdgePositionedWriteOnAppendHandle(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"append.txt": []byte("a")})
	file := memFSTestOpen(t, memFS, "append.txt", experimentalsys.O_WRONLY|experimentalsys.O_APPEND)
	defer file.Close()

	if n, errno := file.Pwrite([]byte("x"), 0); errno != experimentalsys.EINVAL || n != 0 {
		t.Fatalf("Pwrite on an append handle: n=%d errno=%v", n, errno)
	}
	if got := string(memFSTestReadPath(t, memFS, "append.txt")); got != "a" {
		t.Fatalf("rejected Pwrite changed contents: %q", got)
	}
	if n, errno := file.Write([]byte("b")); errno != 0 || n != 1 {
		t.Fatalf("ordinary append after rejected Pwrite: n=%d errno=%v", n, errno)
	}
	if got := string(memFSTestReadPath(t, memFS, "append.txt")); got != "ab" {
		t.Fatalf("ordinary append produced %q", got)
	}
}

func TestWasmMemoryFSEdgeUtimensBothOmittedIsNoOp(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"time.txt": []byte("time")})
	memFSTestRequireErrno(t, 0, memFS.Utimens("time.txt", 101, 202))

	before, errno := memFS.Stat("time.txt")
	memFSTestRequireErrno(t, 0, errno)
	for range 8 {
		memFSTestRequireErrno(t, 0, memFS.Utimens(
			"time.txt",
			experimentalsys.UTIME_OMIT,
			experimentalsys.UTIME_OMIT,
		))
	}
	afterPath, errno := memFS.Stat("time.txt")
	memFSTestRequireErrno(t, 0, errno)
	if afterPath.Atim != before.Atim || afterPath.Mtim != before.Mtim || afterPath.Ctim != before.Ctim {
		t.Fatalf("path Utimens changed omitted timestamps: before=%+v after=%+v", before, afterPath)
	}

	file := memFSTestOpen(t, memFS, "time.txt", experimentalsys.O_RDONLY)
	defer file.Close()
	memFSTestRequireErrno(t, 0, file.Utimens(
		experimentalsys.UTIME_OMIT,
		experimentalsys.UTIME_OMIT,
	))
	afterHandle, errno := file.Stat()
	memFSTestRequireErrno(t, 0, errno)
	if afterHandle.Atim != before.Atim || afterHandle.Mtim != before.Mtim || afterHandle.Ctim != before.Ctim {
		t.Fatalf("handle Utimens changed omitted timestamps: before=%+v after=%+v", before, afterHandle)
	}
}

func TestWasmMemoryFSEdgeResourceQuotas(t *testing.T) {
	t.Run("entries are bounded and reclaimed", func(t *testing.T) {
		raw, err := NewWasmMemoryFS(nil, wasmMemoryFSEdgeLimits(32, 2, 8))
		if err != nil {
			t.Fatalf("NewWasmMemoryFS: %v", err)
		}
		memFS := &memFSTestFS{fs: raw}

		memFSTestRequireErrno(t, 0, memFS.Mkdir("dir", 0o700))
		file, errno := memFS.OpenFile("dir/file", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
		memFSTestRequireErrno(t, 0, errno)
		memFSTestRequireErrno(t, 0, file.Close())
		if errno := memFS.Mkdir("overflow", 0o700); errno != experimentalsys.ERANGE {
			t.Fatalf("entry quota returned %v", errno)
		}
		if _, errno := memFS.Stat("overflow"); errno != experimentalsys.ENOENT {
			t.Fatalf("entry quota left a partial entry: %v", errno)
		}
		memFSTestRequireErrno(t, 0, memFS.Unlink("dir/file"))
		memFSTestRequireErrno(t, 0, memFS.Mkdir("dir/reused", 0o700))
	})

	t.Run("implicit directories count during initialization", func(t *testing.T) {
		if _, err := NewWasmMemoryFS(
			map[string][]byte{"dir/file": []byte("x")},
			wasmMemoryFSEdgeLimits(32, 1, 8),
		); err == nil {
			t.Fatal("constructor accepted a tree larger than the entry quota")
		}
	})

	t.Run("open handles are bounded and reclaimed", func(t *testing.T) {
		raw, err := NewWasmMemoryFS(
			map[string][]byte{"seed.txt": []byte("seed")},
			wasmMemoryFSEdgeLimits(32, 8, 1),
		)
		if err != nil {
			t.Fatalf("NewWasmMemoryFS: %v", err)
		}
		legacy, err := raw.Open("memfs/seed.txt")
		if err != nil {
			t.Fatalf("legacy Open: %v", err)
		}
		if file, errno := raw.OpenFile("memfs/seed.txt", experimentalsys.O_RDONLY, 0); errno != experimentalsys.ERANGE {
			if file != nil {
				_ = file.Close()
			}
			t.Fatalf("open-handle quota returned %v", errno)
		}
		if err := legacy.Close(); err != nil {
			t.Fatalf("legacy Close: %v", err)
		}
		file, errno := raw.OpenFile("memfs/seed.txt", experimentalsys.O_RDONLY, 0)
		memFSTestRequireErrno(t, 0, errno)
		memFSTestRequireErrno(t, 0, file.Close())
	})

	t.Run("unlinked open data remains charged", func(t *testing.T) {
		raw, err := NewWasmMemoryFS(nil, wasmMemoryFSEdgeLimits(4, 8, 8))
		if err != nil {
			t.Fatalf("NewWasmMemoryFS: %v", err)
		}
		memFS := &memFSTestFS{fs: raw}

		first, errno := memFS.OpenFile("first", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
		memFSTestRequireErrno(t, 0, errno)
		if n, errno := first.Write([]byte("1234")); errno != 0 || n != 4 {
			t.Fatalf("fill byte quota: n=%d errno=%v", n, errno)
		}
		if n, errno := first.Pwrite([]byte("5"), 4); errno != experimentalsys.ERANGE || n != 0 {
			t.Fatalf("byte quota extension: n=%d errno=%v", n, errno)
		}
		memFSTestRequireErrno(t, 0, memFS.Unlink("first"))

		second, errno := memFS.OpenFile("second", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
		memFSTestRequireErrno(t, 0, errno)
		if n, errno := second.Write([]byte("x")); errno != experimentalsys.ERANGE || n != 0 {
			t.Fatalf("unlinked open data was not charged: n=%d errno=%v", n, errno)
		}
		memFSTestRequireErrno(t, 0, first.Close())
		if n, errno := second.Write([]byte("abcd")); errno != 0 || n != 4 {
			t.Fatalf("released byte quota was not reusable: n=%d errno=%v", n, errno)
		}
		memFSTestRequireErrno(t, 0, second.Close())
	})
}

func TestWasmMemoryFSEdgeConcurrentChmodPreservesNodeTypes(t *testing.T) {
	memFS := memFSTestNew(t, map[string][]byte{"file": []byte("data")})
	memFSTestRequireErrno(t, 0, memFS.Mkdir("dir", 0o700))
	file := memFSTestOpen(t, memFS, "file", experimentalsys.O_RDONLY)
	directory := memFSTestOpen(t, memFS, "dir", experimentalsys.O_RDONLY|experimentalsys.O_DIRECTORY)
	defer file.Close()
	defer directory.Close()

	const iterations = 1000
	errCh := make(chan error, 4)
	var wait sync.WaitGroup
	wait.Add(4)
	go func() {
		defer wait.Done()
		for index := 0; index < iterations; index++ {
			perm := fs.FileMode(0o600 | index&0o77)
			if errno := memFS.Chmod("file", perm); errno != 0 {
				errCh <- fmt.Errorf("chmod file: %w", errno)
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		for index := 0; index < iterations; index++ {
			perm := fs.FileMode(0o700 | index&0o77)
			if errno := memFS.Chmod("dir", perm); errno != 0 {
				errCh <- fmt.Errorf("chmod directory: %w", errno)
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		for range iterations {
			isDirectory, errno := file.IsDir()
			if errno != 0 || isDirectory {
				errCh <- fmt.Errorf("regular file type changed: isDir=%t errno=%w", isDirectory, errno)
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		for range iterations {
			isDirectory, errno := directory.IsDir()
			if errno != 0 || !isDirectory {
				errCh <- fmt.Errorf("directory type changed: isDir=%t errno=%w", isDirectory, errno)
				return
			}
		}
	}()
	wait.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestWasmMemoryFSEdgeCreateThroughDanglingSymlink(t *testing.T) {
	memFS := memFSTestNew(t, nil)
	memFSTestRequireErrno(t, 0, memFS.Mkdir("dir", 0o700))
	memFSTestRequireErrno(t, 0, memFS.Symlink("target.txt", "dir/link.txt"))

	file, errno := memFS.OpenFile("dir/link.txt", experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o640)
	memFSTestRequireErrno(t, 0, errno)
	if n, errno := file.Write([]byte("created")); errno != 0 || n != len("created") {
		t.Fatalf("write through dangling symlink: n=%d errno=%v", n, errno)
	}
	memFSTestRequireErrno(t, 0, file.Close())
	if got := string(memFSTestReadPath(t, memFS, "dir/target.txt")); got != "created" {
		t.Fatalf("symlink target contents: %q", got)
	}
	if target, errno := memFS.Readlink("dir/link.txt"); errno != 0 || target != "target.txt" {
		t.Fatalf("create replaced symlink: target=%q errno=%v", target, errno)
	}

	memFSTestRequireErrno(t, 0, memFS.Symlink("missing.txt", "dir/exclusive.txt"))
	if file, errno := memFS.OpenFile(
		"dir/exclusive.txt",
		experimentalsys.O_CREAT|experimentalsys.O_EXCL|experimentalsys.O_NOFOLLOW|experimentalsys.O_RDWR,
		0o600,
	); errno != experimentalsys.EEXIST {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("exclusive create of a symlink returned %v", errno)
	}
	if _, errno := memFS.Stat("dir/missing.txt"); errno != experimentalsys.ENOENT {
		t.Fatalf("exclusive create changed dangling target: %v", errno)
	}
}

func TestWasmMemoryFSEdgeSymlinkTargetBounds(t *testing.T) {
	memFS := memFSTestNew(t, nil)
	componentLimit := strings.Repeat("a", wasmMemoryFSMaxNameBytes)
	memFSTestRequireErrno(t, 0, memFS.Symlink(componentLimit, "component-limit"))
	if errno := memFS.Symlink(componentLimit+"a", "component-overflow"); errno != experimentalsys.ENAMETOOLONG {
		t.Fatalf("overlong symlink component returned %v", errno)
	}

	pathLimit := wasmMemoryFSEdgeValidPath(wasmMemoryFSMaxPathBytes)
	if len(pathLimit) != wasmMemoryFSMaxPathBytes {
		t.Fatalf("path-limit helper produced %d bytes", len(pathLimit))
	}
	memFSTestRequireErrno(t, 0, memFS.Symlink(pathLimit, "path-limit"))
	if target, errno := memFS.Readlink("path-limit"); errno != 0 || target != pathLimit {
		t.Fatalf("path-limit symlink: target length=%d errno=%v", len(target), errno)
	}
	pathOverflow := wasmMemoryFSEdgeValidPath(wasmMemoryFSMaxPathBytes + 1)
	if errno := memFS.Symlink(pathOverflow, "path-overflow"); errno != experimentalsys.ENAMETOOLONG {
		t.Fatalf("overlong symlink target returned %v", errno)
	}
	for _, rejected := range []string{"component-overflow", "path-overflow"} {
		if _, errno := memFS.Lstat(rejected); errno != experimentalsys.ENOENT {
			t.Fatalf("rejected symlink %q left an entry: %v", rejected, errno)
		}
	}
}

func TestWasmMemoryFSEdgeRawPassthroughIsReadOnlyAndIsolated(t *testing.T) {
	raw, err := NewWasmMemoryFS(map[string][]byte{"seed.txt": []byte("memory")})
	if err != nil {
		t.Fatalf("NewWasmMemoryFS: %v", err)
	}
	hostPath := filepath.Join(t.TempDir(), "host.txt")
	if err := os.WriteFile(hostPath, []byte("host-sentinel"), 0o600); err != nil {
		t.Fatalf("write host fixture: %v", err)
	}
	hostGuestPath := memFSTestHostPath(hostPath)

	hostFile, errno := raw.OpenFile(hostGuestPath, experimentalsys.O_RDONLY, 0)
	memFSTestRequireErrno(t, 0, errno)
	if got := string(memFSTestReadAll(t, hostFile)); got != "host-sentinel" {
		t.Fatalf("raw passthrough contents: %q", got)
	}
	memFSTestRequireErrno(t, 0, hostFile.Close())
	if stat, errno := raw.Stat(hostGuestPath); errno != 0 || stat.Size != int64(len("host-sentinel")) {
		t.Fatalf("raw passthrough Stat: stat=%+v errno=%v", stat, errno)
	}

	if writable, errno := raw.OpenFile(hostGuestPath, experimentalsys.O_WRONLY|experimentalsys.O_TRUNC, 0o600); errno == 0 {
		if n, writeErrno := writable.Write([]byte("changed")); writeErrno == 0 || n != 0 {
			_ = writable.Close()
			t.Fatalf("raw passthrough write: n=%d errno=%v", n, writeErrno)
		}
		_ = writable.Close()
	}
	createdPath := filepath.Join(filepath.Dir(hostPath), "created.txt")
	createdGuestPath := memFSTestHostPath(createdPath)
	if created, errno := raw.OpenFile(createdGuestPath, experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600); errno == 0 {
		_, _ = created.Write([]byte("created"))
		_ = created.Close()
	}
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("raw passthrough created a host file: %v", err)
	}

	for _, traversal := range []string{
		"memfs/../" + hostGuestPath,
		"./memfs/../" + hostGuestPath,
		"/memfs/../../" + hostGuestPath,
	} {
		file, errno := raw.OpenFile(traversal, experimentalsys.O_RDONLY, 0)
		if file != nil {
			_ = file.Close()
		}
		if errno != experimentalsys.EINVAL {
			t.Fatalf("raw traversal %q returned %v", traversal, errno)
		}
	}
	if contents, err := os.ReadFile(hostPath); err != nil || !bytes.Equal(contents, []byte("host-sentinel")) {
		t.Fatalf("raw passthrough changed host fixture: contents=%q err=%v", contents, err)
	}
	if got := string(memFSTestReadPath(t, &memFSTestFS{fs: raw}, "seed.txt")); got != "memory" {
		t.Fatalf("raw passthrough changed memfs: %q", got)
	}
}

func TestWasmMemoryFSEdgeNamespaceParsingIsBounded(t *testing.T) {
	const slashCount = 1 << 20
	name := strings.Repeat("/", slashCount) + "memfs/file.txt"
	subpath, memoryPath := splitWasmMemoryPath(name)
	if !memoryPath || subpath != "file.txt" {
		t.Fatalf("large namespace path: memory=%t subpath=%q", memoryPath, subpath)
	}

	name = strings.Repeat("./", slashCount/2) + "memfs/file.txt"
	subpath, memoryPath = splitWasmMemoryPath(name)
	if !memoryPath || subpath != "file.txt" {
		t.Fatalf("large dotted namespace path: memory=%t subpath=%q", memoryPath, subpath)
	}
}

func TestWasmMemoryFSEdgeConcurrentLazyPointerInitialization(t *testing.T) {
	raw := &WasmMemoryFS{memFS: map[string][]byte{"seed.txt": []byte("seed")}}

	const workers = 32
	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			<-start
			file, errno := raw.OpenFile("memfs/seed.txt", experimentalsys.O_RDONLY, 0)
			if errno != 0 {
				errCh <- fmt.Errorf("OpenFile: %w", errno)
				return
			}
			contents := make([]byte, len("seed"))
			n, errno := file.Read(contents)
			if errno != 0 || n != len(contents) {
				_ = file.Close()
				errCh <- fmt.Errorf("Read: n=%d errno=%w", n, errno)
				return
			}
			if errno := file.Close(); errno != 0 {
				errCh <- fmt.Errorf("Close: %w", errno)
				return
			}
			if !bytes.Equal(contents, []byte("seed")) {
				errCh <- fmt.Errorf("contents: %q", contents)
			}
		}()
	}
	close(start)
	wait.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestWasmMemoryFSEdgeConcurrentMixedInterfaces(t *testing.T) {
	testCases := map[string]func(*testing.T) *WasmMemoryFS{
		"constructor": func(t *testing.T) *WasmMemoryFS {
			raw, err := NewWasmMemoryFS(map[string][]byte{"seed.txt": []byte("seed")})
			if err != nil {
				t.Fatalf("NewWasmMemoryFS: %v", err)
			}
			return raw
		},
		"preinitialized legacy literal": func(t *testing.T) *WasmMemoryFS {
			raw := &WasmMemoryFS{memFS: map[string][]byte{"seed.txt": []byte("seed")}}
			if _, errno := raw.Stat("memfs/seed.txt"); errno != 0 {
				t.Fatalf("initialize legacy literal: %v", errno)
			}
			return raw
		},
	}

	for name, newFS := range testCases {
		t.Run(name, func(t *testing.T) {
			raw := newFS(t)
			const workers = 32
			errCh := make(chan error, workers)
			var wait sync.WaitGroup
			wait.Add(workers)
			for worker := range workers {
				go func() {
					defer wait.Done()
					if worker%2 == 0 {
						file, err := raw.Open("memfs/seed.txt")
						if err != nil {
							errCh <- fmt.Errorf("legacy Open: %w", err)
							return
						}
						contents, err := io.ReadAll(file)
						closeErr := file.Close()
						if err != nil {
							errCh <- fmt.Errorf("legacy Read: %w", err)
							return
						}
						if closeErr != nil {
							errCh <- fmt.Errorf("legacy Close: %w", closeErr)
							return
						}
						if !bytes.Equal(contents, []byte("seed")) {
							errCh <- fmt.Errorf("legacy contents: %q", contents)
						}
						return
					}

					file, errno := raw.OpenFile("memfs/seed.txt", experimentalsys.O_RDONLY, 0)
					if errno != 0 {
						errCh <- fmt.Errorf("OpenFile: %w", errno)
						return
					}
					contents := make([]byte, len("seed"))
					n, errno := file.Read(contents)
					if errno != 0 || n != len(contents) {
						_ = file.Close()
						errCh <- fmt.Errorf("Read: n=%d errno=%w", n, errno)
						return
					}
					if errno := file.Close(); errno != 0 {
						errCh <- fmt.Errorf("Close: %w", errno)
						return
					}
					if !bytes.Equal(contents, []byte("seed")) {
						errCh <- fmt.Errorf("contents: %q", contents)
					}
				}()
			}
			wait.Wait()
			close(errCh)
			for err := range errCh {
				t.Error(err)
			}
		})
	}
}

func wasmMemoryFSEdgeLimits(maxBytes, maxEntries, maxOpenFiles int64) WasmMemoryFSOption {
	return func(config *wasmMemoryFSConfig) {
		config.maxBytes = maxBytes
		config.maxEntries = maxEntries
		config.maxOpenFiles = maxOpenFiles
	}
}

func wasmMemoryFSEdgeValidPath(length int) string {
	if length <= 0 {
		return ""
	}
	segments := make([]string, 0, length/(wasmMemoryFSMaxNameBytes+1)+1)
	remaining := length
	for remaining > wasmMemoryFSMaxNameBytes {
		segmentLength := wasmMemoryFSMaxNameBytes
		if remaining-segmentLength-1 == 0 {
			segmentLength--
		}
		segments = append(segments, strings.Repeat("a", segmentLength))
		remaining -= segmentLength + 1
	}
	segments = append(segments, strings.Repeat("b", remaining))
	return strings.Join(segments, "/")
}
