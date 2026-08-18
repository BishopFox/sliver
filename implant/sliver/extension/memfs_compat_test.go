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
	"errors"
	"io"
	"io/fs"
	"testing"
)

func TestMemFSNodeCompatibilityTree(t *testing.T) {
	root := &MemFSNode{
		fullName: "",
		isDir:    true,
		Subdirs:  map[string]*MemFSNode{},
	}
	first := &MemFSNode{
		fullName: "docs/first.txt",
		BaseName: "first.txt",
		data:     bytes.NewBufferString("first"),
	}
	second := &MemFSNode{
		fullName: "docs/second.txt",
		BaseName: "second.txt",
		data:     bytes.NewBufferString("second"),
	}

	root.Insert([]string{"docs"}, first)
	root.Insert([]string{"docs"}, second)

	if !root.Exists(nil) || !root.Exists([]string{"docs"}) {
		t.Fatal("legacy directory tree was not populated")
	}
	if root.Exists([]string{"missing"}) {
		t.Fatal("missing legacy path unexpectedly exists")
	}

	docs := root.GetNode([]string{"docs"})
	if docs == nil {
		t.Fatal("GetNode did not find the inserted directory")
	}
	if got := docs.ParentSegs([]string{"child"}); len(got) != 2 || got[0] != "docs" || got[1] != "child" {
		t.Fatalf("unexpected parent segments: %v", got)
	}
	if len(docs.FileNodes) != 2 {
		t.Fatalf("expected two legacy file nodes, got %d", len(docs.FileNodes))
	}

	entries, err := docs.ReadDir(-1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two directory entries, got %d", len(entries))
	}
}

func TestMemFSNodeCompatibilityFileInterfaces(t *testing.T) {
	node := MemFSNode{
		fullName: "docs/file.txt",
		BaseName: "file.txt",
		data:     bytes.NewBufferString("contents"),
	}

	info, err := node.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name() != "memfs/docs/file.txt" {
		t.Fatalf("unexpected legacy name: %q", info.Name())
	}
	if info.Size() != int64(len("contents")) || info.Mode() != 0o444 || info.IsDir() {
		t.Fatalf("unexpected legacy file info: size=%d mode=%v dir=%v", info.Size(), info.Mode(), info.IsDir())
	}
	if entryInfo, err := node.Info(); err != nil || entryInfo.Name() != info.Name() {
		t.Fatalf("unexpected directory entry info: info=%v err=%v", entryInfo, err)
	}

	contents, err := io.ReadAll(node)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "contents" {
		t.Fatalf("unexpected legacy file contents: %q", contents)
	}
	if err := node.Close(); err != nil {
		t.Fatal(err)
	}
	if node.Sys() != nil || node.Type() != fs.FileMode(0o444) {
		t.Fatalf("unexpected legacy metadata: sys=%v type=%v", node.Sys(), node.Type())
	}
}

func TestMemFSNodeCompatibilityNilData(t *testing.T) {
	var node MemFSNode
	if size := node.Size(); size != 0 {
		t.Fatalf("unexpected nil-buffer size: %d", size)
	}
	if _, err := node.Read(make([]byte, 1)); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("expected invalid read error, got %v", err)
	}
}
