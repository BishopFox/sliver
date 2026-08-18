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
	"path"
	"strings"
	"time"
)

var (
	_ fs.File        = MemFSNode{}
	_ fs.FileInfo    = MemFSNode{}
	_ fs.DirEntry    = MemFSNode{}
	_ fs.ReadDirFile = MemFSNode{}
)

// MemFSNode is the node type exposed by the original read-only memory
// filesystem implementation. It is retained so code which used that API
// continues to compile; new code should use NewWasmMemoryFS and the standard
// io/fs or wazero experimental/sys interfaces instead.
//
// Deprecated: use NewWasmMemoryFS.
type MemFSNode struct {
	fullName string
	BaseName string
	data     *bytes.Buffer
	isDir    bool

	parent    *MemFSNode
	Subdirs   map[string]*MemFSNode
	FileNodes map[string]*MemFSNode
}

// Exists reports whether segs identify an entry in this legacy tree.
//
// Deprecated: use fs.Stat with a WasmMemoryFS.
func (m *MemFSNode) Exists(segs []string) bool {
	if m == nil {
		return false
	}
	if len(segs) == 0 {
		return true
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

// GetNode returns the legacy directory node identified by segs.
//
// Deprecated: use fs.Stat with a WasmMemoryFS.
func (m *MemFSNode) GetNode(segs []string) *MemFSNode {
	if m == nil {
		return nil
	}
	if len(segs) == 0 {
		return m
	}
	if m.HasSubdir(segs[0]) {
		return m.Subdirs[segs[0]].GetNode(segs[1:])
	}
	return nil
}

// HasSubdir reports whether name is an immediate child directory.
//
// Deprecated: use fs.Stat with a WasmMemoryFS.
func (m *MemFSNode) HasSubdir(name string) bool {
	if m == nil {
		return false
	}
	_, ok := m.Subdirs[name]
	return ok
}

// Insert adds fileNode below the directory path in segs. Its map layout and
// generated names intentionally match the original MemFSNode implementation.
//
// Deprecated: create files through WasmMemoryFS.OpenFile.
func (m *MemFSNode) Insert(segs []string, fileNode *MemFSNode) {
	if m == nil || fileNode == nil {
		return
	}
	if len(segs) == 0 {
		if m.FileNodes == nil {
			m.FileNodes = map[string]*MemFSNode{}
		}
		m.FileNodes[fileNode.Name()] = fileNode
		return
	}
	if m.Subdirs == nil {
		m.Subdirs = map[string]*MemFSNode{}
	}
	if !m.HasSubdir(segs[0]) {
		parentSegs := m.ParentSegs([]string{segs[0]})
		m.Subdirs[segs[0]] = &MemFSNode{
			fullName: "/" + strings.Join(parentSegs, "/"),
			BaseName: segs[0],
			parent:   m,
			isDir:    true,
			Subdirs:  map[string]*MemFSNode{},
		}
	}
	m.Subdirs[segs[0]].Insert(segs[1:], fileNode)
}

// ParentSegs prepends the path segments of this node's ancestors.
//
// Deprecated: use path manipulation with a WasmMemoryFS path.
func (m *MemFSNode) ParentSegs(segs []string) []string {
	if m == nil || m.parent == nil {
		return segs
	}
	return m.parent.ParentSegs(append([]string{m.BaseName}, segs...))
}

// Stat returns this legacy node as its file information.
func (m MemFSNode) Stat() (fs.FileInfo, error) {
	return m, nil
}

// Info returns this legacy node as its file information.
func (m MemFSNode) Info() (fs.FileInfo, error) {
	return m, nil
}

// Read reads from the legacy node's in-memory buffer.
func (m MemFSNode) Read(buffer []byte) (int, error) {
	if m.data == nil {
		return 0, fs.ErrInvalid
	}
	return m.data.Read(buffer)
}

// ReadDir returns this legacy directory's immediate children.
func (m MemFSNode) ReadDir(count int) ([]fs.DirEntry, error) {
	if !m.isDir {
		return nil, fs.ErrInvalid
	}
	entries := make([]fs.DirEntry, 0, len(m.Subdirs)+len(m.FileNodes))
	for _, subdir := range m.Subdirs {
		entries = append(entries, subdir)
	}
	for _, fileNode := range m.FileNodes {
		entries = append(entries, fileNode)
	}
	if 0 <= count && count < len(entries) {
		return entries[:count], nil
	}
	return entries, nil
}

// Close is a no-op retained for fs.File compatibility.
func (m MemFSNode) Close() error {
	return nil
}

// Name returns the original memfs-qualified name of this node.
func (m MemFSNode) Name() string {
	return path.Join("memfs", m.fullName)
}

// Size returns the unread bytes remaining in this legacy node's buffer.
func (m MemFSNode) Size() int64 {
	if m.data == nil {
		return 0
	}
	return int64(m.data.Len())
}

// Mode returns the mode exposed by the original MemFSNode API.
func (m MemFSNode) Mode() fs.FileMode {
	return 0o444
}

// Type returns the type value exposed by the original MemFSNode API.
func (m MemFSNode) Type() fs.FileMode {
	return 0o444
}

// ModTime returns the current time, matching the original MemFSNode API.
func (m MemFSNode) ModTime() time.Time {
	return time.Now()
}

// IsDir reports whether this legacy node represents a directory.
func (m MemFSNode) IsDir() bool {
	return m.isDir
}

// Sys returns no system-specific file information.
func (m MemFSNode) Sys() any {
	return nil
}
