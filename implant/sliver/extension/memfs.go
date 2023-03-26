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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

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
		log.Printf("[wasm ext] memfs file: %s (%d bytes)", key, len(memFS[key]))
	}
	// {{end}}

	return WasmMemoryFS{memFS: memFS, localFS: os.DirFS(root)}
}

// WasmMemoryFS - A makeshift read only in-memory virtual file system backed by a map of names
// to bytes the key is the absolute path to the file and the bytes are the contents of the file
// empty directories are not supported.
type WasmMemoryFS struct {
	memFS   map[string][]byte
	tree    *MemFSNode
	localFS fs.FS
}

func (w WasmMemoryFS) getTree() *MemFSNode {
	if w.tree == nil {

		// {{if .Config.Debug}}
		log.Printf("[memfs] building fs tree ...")
		// {{end}}

		// Build the tree
		w.tree = &MemFSNode{fullName: "", isDir: true, Subdirs: map[string]*MemFSNode{}}
		for key := range w.memFS {
			dirPath := path.Dir(key)
			if dirPath == "." {
				dirPath = "/"
			}

			// The root path requires a bit of special handling
			segs := strings.Split(strings.TrimPrefix(dirPath, "/"), "/")
			if len(segs) == 1 && segs[0] == "" {
				segs = []string{}
			}

			// {{if .Config.Debug}}
			log.Printf("[memfs] adding dir tree segments: %s", segs)
			// {{end}}

			w.tree.Insert(segs, &MemFSNode{fullName: key, BaseName: path.Base(key), isDir: false})
		}
	}
	return w.tree
}

// Open - Open a file, the open call is either passed thru to the OS or is redirected to the WasmMemoryFS
func (w WasmMemoryFS) Open(name string) (fs.File, error) {

	name = strings.TrimPrefix(name, "/")

	// {{if .Config.Debug}}
	log.Printf("[memfs] open '%s'", name)
	// {{end}}

	if w.isMemFSPath(name) {
		name = w.memFSPath(name)

		// {{if .Config.Debug}}
		log.Printf("[memfs] in memory path >> '%s'", name)
		// {{end}}

		// Shortcut - Any exact path match is a file
		if data, ok := w.memFS[name]; ok {
			buf := bytes.NewBuffer(data)
			return MemFSNode{fullName: name, isDir: false, data: buf}, nil
		}

		// Check to see if the name is a directory
		segs := strings.Split(strings.TrimPrefix(name, "/"), "/")
		if len(segs) == 1 && segs[0] == "" {
			segs = []string{}
		}
		if w.getTree().Exists(segs) {

			// {{if .Config.Debug}}
			log.Printf("[memfs] get node of directory '%s'", name)
			// {{end}}

			dirNode := w.getTree().GetNode(segs)
			if dirNode != nil {
				return *dirNode, nil
			}
		}

		// {{if .Config.Debug}}
		log.Printf("[memfs] path '%s' not found in memory", name)
		// {{end}}
		return nil, fs.ErrPermission // Read-only for now
	}
	return w.localFS.Open(name)
}

// memFSPath - Returns a blank string for non-memfs paths, or returns the memfs path
func (w WasmMemoryFS) memFSPath(name string) string {
	// The double trim is to handle the case where the path is 'memfs' or 'memfs/'
	return strings.TrimPrefix(strings.TrimPrefix(name, "memfs"), "/")
}

// isMemFSPath - Returns true if the path is a memfs path
func (w WasmMemoryFS) isMemFSPath(name string) bool {
	// We may get passed '/memfs/foo' or 'memfs/foo' so we need to check both
	// and event 'memfs' or 'memfs/' needs to return the root path
	if strings.HasPrefix(name, "memfs/") || name == "memfs" {
		return true
	}
	return false
}

// MemFSNode - A makeshift in-memory fill system node, this object
// implements both the fs.File and fs.FileInfo interfaces.
type MemFSNode struct {
	fullName string
	BaseName string
	data     *bytes.Buffer
	isDir    bool

	parent    *MemFSNode            // Parent directory
	Subdirs   map[string]*MemFSNode // tree entires (directories only)
	FileNodes map[string]*MemFSNode // tree entires (files only)
}

// Exists - Should never be passed an empty slice, recursively
// calls exists on each segment until the last segment is reached
func (m *MemFSNode) Exists(segs []string) bool {
	// {{if .Config.Debug}}
	log.Printf("[memfs] exists -> %v", segs)
	// {{end}}

	if len(segs) == 0 {
		return true // Root node
	}
	if len(segs) == 1 {
		if m.HasSubdir(segs[0]) {
			return true
		}
		if _, ok := m.FileNodes[segs[0]]; ok {
			return true
		}
	}
	if m.HasSubdir(segs[0]) {
		return m.Subdirs[segs[0]].Exists(segs[1:])
	}
	return false
}

// GetNode - Get a node from the memfs tree
func (m *MemFSNode) GetNode(segs []string) *MemFSNode {
	if len(segs) == 0 {
		return m
	}
	if m.HasSubdir(segs[0]) {
		return m.Subdirs[segs[0]].GetNode(segs[1:])
	}
	return nil
}

// HasSubdir - Returns true if the directory has a subdir with the given name
func (m *MemFSNode) HasSubdir(name string) bool {
	_, ok := m.Subdirs[name]
	return ok
}

// Insert - Recursively inserts segments of a path into the tree
func (m *MemFSNode) Insert(segs []string, fileNode *MemFSNode) {
	if len(segs) == 0 {
		if m.FileNodes == nil {
			m.FileNodes = map[string]*MemFSNode{}
		}
		m.FileNodes[fileNode.Name()] = fileNode
		return
	}
	if !m.HasSubdir(segs[0]) {
		parentSegs := m.ParentSegs([]string{segs[0]})
		newDir := &MemFSNode{
			fullName: "/" + strings.Join(parentSegs, "/"),
			BaseName: segs[0],
			parent:   m,
			isDir:    true,
			Subdirs:  map[string]*MemFSNode{},
		}
		m.Subdirs[segs[0]] = newDir

		// {{if .Config.Debug}}
		log.Printf("[memfs] inserted dir (%s): '%s'", segs[0], newDir.fullName)
		// {{end}}

		newDir.Insert(segs[1:], fileNode)
	}
}

func (m *MemFSNode) ParentSegs(segs []string) []string {
	if m.parent == nil {
		return segs
	}
	return m.parent.ParentSegs(append([]string{m.BaseName}, segs...))
}

// Stat - Returns the MemoryNode itself since it implements FileInfo
func (m MemFSNode) Stat() (fs.FileInfo, error) {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] stat")
	// {{end}}
	return m, nil
}

func (m MemFSNode) Info() (fs.FileInfo, error) {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] info")
	// {{end}}
	return m, nil
}

// Read - Standard reader function
func (m MemFSNode) Read(buf []byte) (int, error) {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] read")
	// {{end}}
	return m.data.Read(buf)
}

// ReadDir - Read contents for directory
func (m MemFSNode) ReadDir(n int) ([]fs.DirEntry, error) {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] read dir (%d)", n)
	// {{end}}
	if m.isDir {
		entries := []fs.DirEntry{}
		for _, subdir := range m.Subdirs {
			entries = append(entries, subdir)
		}
		for _, fileNode := range m.FileNodes {
			entries = append(entries, fileNode)
		}
		if 0 <= n && n < len(entries) {
			return entries[:n], nil
		}
		return entries, nil
	}
	return nil, fs.ErrInvalid
}

// Close - No-op
func (m MemFSNode) Close() error {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] close")
	// {{end}}
	return nil
}

// Name - Returns the name of the file
func (m MemFSNode) Name() string {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] name")
	// {{end}}
	return path.Join("memfs", m.fullName)
}

// Size - Returns the size of the file
func (m MemFSNode) Size() int64 {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] size")
	// {{end}}
	return int64(m.data.Len())
}

// Mode - Returns the mode of the file
func (m MemFSNode) Mode() fs.FileMode {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] mode")
	// {{end}}
	return fs.FileMode(0444) // YOLO
}

// Type - Returns the mode of the file
func (m MemFSNode) Type() fs.FileMode {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] type")
	// {{end}}

	return fs.FileMode(0444) // YOLO
}

// ModTime - Returns the mod time of the file
func (m MemFSNode) ModTime() time.Time {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] mod time")
	// {{end}}

	return time.Now()
}

// IsDir - Returns true if the file is a directory
func (m MemFSNode) IsDir() bool {
	// {{if .Config.Debug}}
	log.Printf("[memfs node] is dir")
	// {{end}}

	return m.isDir
}

// Sys - Returns nil
func (m MemFSNode) Sys() interface{} {
	return nil
}
