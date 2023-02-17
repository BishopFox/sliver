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

	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
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
func NewWasmExtension(wasm []byte, args []string, extFS map[string][]byte) (*WasmExtension, error) {
	wasmExt := &WasmExtension{ctx: context.Background()}
	wasmExt.runtime = wazero.NewRuntime(wasmExt.ctx)
	wasmExt.config = wazero.NewModuleConfig().
		WithStdin(wasmExt.Stdin).
		WithStdout(wasmExt.Stdout).
		WithStderr(wasmExt.Stderr).
		WithFS(makeWasmExtFS(extFS))

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

// makeWasmExtFS - Merge the local filesystem any provided ext files
func makeWasmExtFS(extFS map[string][]byte) fs.FS {
	root := "/"
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	if vol := filepath.VolumeName(cwd); vol != "" {
		root = vol
	}
	return WasmExtFS{extFS: extFS, localFS: os.DirFS(root)}
}

// WasmExtFS - Creates an encoder.EncoderFS object from a single local directory
type WasmExtFS struct {
	extFS   map[string][]byte
	localFS fs.FS
}

func (f WasmExtFS) Open(name string) (fs.File, error) {
	if strings.HasPrefix(name, "/local/") {
		return os.Open(strings.TrimPrefix("/local", name))
	}
	if data, ok := f.extFS[name]; ok {
		return MemoryFile{key: name, data: data}, nil
	}
	return nil, nil
}

// MemoryFS - A makeshift in-memory virtual file system backed by a map of names to bytes
// the key is the absolute path to the file and the bytes are the contents of the file
// empty directories are not supported, so directories are defined as any path with a trailing
// slash that is a prefix of multiple keys. "/foo/bar" "/foo/baz" where /foo/ is a directory
type MemoryFS struct {
	extFS map[string][]byte

	tree *DirTree
}

func (m MemoryFS) Open(name string) (fs.File, error) {
	// Absolute match, so just return the buffer wrapped in a MemoryFile object
	if data, ok := m.extFS[name]; ok {
		return MemoryFile{key: name, data: data}, nil
	}

	// Check to see if the name is a directory
	if m.tree == nil {
		m.tree = &DirTree{Name: "/", Subdirs: []*DirTree{}}
		for key := range m.extFS {
			m.tree.Insert(strings.Split(path.Dir(key), "/"))
		}
	}
	if m.tree.Exists(strings.Split(name, "/")) {
		return MemoryFile{key: name, isDir: true}, nil
	}
	return nil, fs.ErrNotExist
}

// DirTree - A tree structure for representing only directories in the filesystem
type DirTree struct {
	Name    string
	Subdirs []*DirTree
}

// Exists - Should never be passed an empty slice, recursively
// calls exists on each segment until the last segment is reached
func (d *DirTree) Exists(segs []string) bool {
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

func (d *DirTree) HasSubdir(name string) bool {
	for _, subdir := range d.Subdirs {
		if subdir.Name == name {
			return true
		}
	}
	return false
}

func (d *DirTree) Insert(segs []string) {
	if len(segs) == 0 {
		return
	}
	if !d.HasSubdir(segs[0]) {
		newDir := &DirTree{Name: segs[0], Subdirs: []*DirTree{}}
		d.Subdirs = append(d.Subdirs, newDir)
		newDir.Insert(segs[1:])
	}
}

// MemoryFile - A makeshift in-memory fs.File object
type MemoryFile struct {
	key   string
	data  []byte
	isDir bool
}

func (m MemoryFile) Stat() (fs.FileInfo, error) {
	return m, nil
}

func (m MemoryFile) Read(buf []byte) (int, error) {
	n := copy(buf, m.data)
	return n, nil
}

func (m MemoryFile) Close() error {
	return nil
}

func (m MemoryFile) Name() string {
	return path.Base(m.key)
}

func (m MemoryFile) Size() int64 {
	return int64(len(m.data))
}

func (m MemoryFile) Mode() fs.FileMode {
	return 0
}

func (m MemoryFile) ModTime() time.Time {
	return time.Now()
}

func (m MemoryFile) IsDir() bool {
	return m.isDir
}

func (m MemoryFile) Sys() interface{} {
	return nil
}
