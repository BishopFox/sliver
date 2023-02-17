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
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// WasmExtension - Wasm extension
type WasmExtension struct {
	ctx     context.Context
	mod     wazero.CompiledModule
	config  wazero.ModuleConfig
	runtime wazero.Runtime
	closer  api.Closer

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Execute - Execute the Wasm module
func (w *WasmExtension) Execute(args []string) error {
	args = append([]string{"wasi"}, args...)
	conf := w.config.WithArgs(args...)
	if _, err := w.runtime.InstantiateModule(w.ctx, w.mod, conf); err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(os.Stderr, "exit_code: %d\n", exitErr.ExitCode())
		} else if !ok {
			return err
		}
	}
	return nil
}

// Close - Close the Wasm module
func (w *WasmExtension) Close() error {
	return w.closer.Close(w.ctx)
}

// NewWasmExtension - Create a new Wasm extension
func NewWasmExtension(stdin io.Reader, stdout io.Writer, stderr io.Writer, wasm []byte, args []string, memFS map[string][]byte) (*WasmExtension, error) {
	wasmExt := &WasmExtension{
		ctx:    context.Background(),
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	wasmExt.runtime = wazero.NewRuntime(wasmExt.ctx)
	wasmExt.config = wazero.NewModuleConfig().
		WithStdin(wasmExt.Stdin).
		WithStdout(wasmExt.Stdout).
		WithStderr(wasmExt.Stderr).
		WithFS(makeWasmMemFS(memFS))

	var err error
	wasmExt.closer, err = wasi_snapshot_preview1.Instantiate(wasmExt.ctx, wasmExt.runtime)
	if err != nil {
		return nil, err
	}
	wasmExt.mod, err = wasmExt.runtime.CompileModule(wasmExt.ctx, wasm)
	if err != nil {
		return nil, err
	}
	return wasmExt, nil
}

type MemoryFile struct {
}

func (m MemoryFile) Stat() (fs.FileInfo, error) {
}

// makeWasmMemFS - Merge the local filesystem any provided ext files
func makeWasmMemFS(memFS map[string][]byte) fs.FS {
	root := "/"
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	if vol := filepath.VolumeName(cwd); vol != "" {
		root = vol
	}
	return WasmMemoryFS{memFS: memFS, localFS: os.DirFS(root)}
}

// WasmMemoryFS - A makeshift in-memory virtual file system backed by a map of names to bytes
// the key is the absolute path to the file and the bytes are the contents of the file
// empty directories are not supported, so directories are defined as any path with a trailing
// slash that is a prefix of multiple keys. "/foo/bar" "/foo/baz" where /foo/ is a directory
type WasmMemoryFS struct {
	memFS   map[string][]byte
	tree    *MemoryFSDirTree
	localFS fs.FS
}

func (w WasmMemoryFS) Open(name string) (fs.File, error) {
	name = path.Clean(name)
	if strings.HasPrefix(name, "/memfs/") {
		name = strings.TrimPrefix(name, "/memfs")
		if data, ok := w.memFS[name]; ok {
			return MemoryFSNode{key: name, isDir: false, data: data}, nil
		}

		// Check to see if the name is a directory
		if w.tree == nil {
			w.tree = &MemoryFSDirTree{Name: "/", Subdirs: []*MemoryFSDirTree{}}
			for key := range w.memFS {
				segs := strings.Split(strings.TrimPrefix(path.Dir(key), "/"), "/")
				w.tree.Insert(segs)
			}
		}
		if w.tree.Exists(strings.Split(strings.TrimPrefix(name, "/"), "/")) {
			return MemoryFSNode{key: name, isDir: true, data: []byte{}}, nil
		}
		return nil, os.ErrNotExist
	}
	cwd, _ := os.Getwd()
	return os.Open(filepath.Join(filepath.VolumeName(cwd), name))
}

// MemoryFSDirTree - A tree structure for representing only directories in the filesystem
type MemoryFSDirTree struct {
	Name    string
	Subdirs []*MemoryFSDirTree
}

// Exists - Should never be passed an empty slice, recursively
// calls exists on each segment until the last segment is reached
func (d *MemoryFSDirTree) Exists(segs []string) bool {
	if len(segs) <= 1 {
		return d.HasSubdir(segs[0])
	}
	for _, subdir := range d.Subdirs {
		if subdir.Name == segs[0] {
			return subdir.Exists(segs[1:])
		}
	}
	return false
}

// HasSubdir - Returns true if the directory has a subdir with the given name
func (d *MemoryFSDirTree) HasSubdir(name string) bool {
	for _, subdir := range d.Subdirs {
		if subdir.Name == name {
			return true
		}
	}
	return false
}

// Insert - Recursively inserts segments of a path into the tree
func (d *MemoryFSDirTree) Insert(segs []string) {
	if len(segs) == 0 {
		return
	}
	if !d.HasSubdir(segs[0]) {
		newDir := &MemoryFSDirTree{Name: segs[0], Subdirs: []*MemoryFSDirTree{}}
		d.Subdirs = append(d.Subdirs, newDir)
		newDir.Insert(segs[1:])
	}
}

// MemoryFSNode - A makeshift in-memory fs.File object
type MemoryFSNode struct {
	key   string
	data  []byte
	isDir bool
}

// Stat - Returns the MemoryNode itself since it implements FileInfo
func (m MemoryFSNode) Stat() (fs.FileInfo, error) {
	return m, nil
}

// Read - Standard reader function
func (m MemoryFSNode) Read(buf []byte) (int, error) {
	n := copy(buf, m.data)
	return n, nil
}

// Close - No-op
func (m MemoryFSNode) Close() error {
	return nil
}

// Name - Returns the name of the file
func (m MemoryFSNode) Name() string {
	return path.Base(m.key)
}

// Size - Returns the size of the file
func (m MemoryFSNode) Size() int64 {
	return int64(len(m.data))
}

// Mode - Returns the mode of the file
func (m MemoryFSNode) Mode() fs.FileMode {
	return 0777 // YOLO
}

// ModTime - Returns the mod time of the file
func (m MemoryFSNode) ModTime() time.Time {
	return time.Now()
}

// IsDir - Returns true if the file is a directory
func (m MemoryFSNode) IsDir() bool {
	return m.isDir
}

// Sys - Returns nil
func (m MemoryFSNode) Sys() interface{} {
	return nil
}
