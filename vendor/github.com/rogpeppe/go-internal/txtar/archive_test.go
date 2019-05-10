// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txtar

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

var tests = []struct {
	name   string
	text   string
	parsed *Archive
}{
	// General test
	{
		name: "basic",
		text: `comment1
comment2
-- file1 --
File 1 text.
-- foo ---
More file 1 text.
-- file 2 --
File 2 text.
-- empty --
-- noNL --
hello world`,
		parsed: &Archive{
			Comment: []byte("comment1\ncomment2\n"),
			Files: []File{
				{"file1", []byte("File 1 text.\n-- foo ---\nMore file 1 text.\n")},
				{"file 2", []byte("File 2 text.\n")},
				{"empty", []byte{}},
				{"noNL", []byte("hello world\n")},
			},
		},
	},
	// Test CRLF input
	{
		name: "basic",
		text: "blah\r\n-- hello --\r\nhello\r\n",
		parsed: &Archive{
			Comment: []byte("blah\r\n"),
			Files: []File{
				{"hello", []byte("hello\r\n")},
			},
		},
	},
}

func Test(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Parse([]byte(tt.text))
			if !reflect.DeepEqual(a, tt.parsed) {
				t.Fatalf("Parse: wrong output:\nhave:\n%s\nwant:\n%s", shortArchive(a), shortArchive(tt.parsed))
			}
			text := Format(a)
			a = Parse(text)
			if !reflect.DeepEqual(a, tt.parsed) {
				t.Fatalf("Parse after Format: wrong output:\nhave:\n%s\nwant:\n%s", shortArchive(a), shortArchive(tt.parsed))
			}
		})
	}
}

func shortArchive(a *Archive) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "comment: %q\n", a.Comment)
	for _, f := range a.Files {
		fmt.Fprintf(&buf, "file %q: %q\n", f.Name, f.Data)
	}
	return buf.String()
}

func TestWrite(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(td)

	good := &Archive{Files: []File{File{Name: "good.txt"}}}
	if err := Write(good, td); err != nil {
		t.Fatalf("expected no error; got %v", err)
	}

	badRel := &Archive{Files: []File{File{Name: "../bad.txt"}}}
	want := `"../bad.txt": outside parent directory`
	if err := Write(badRel, td); err == nil || err.Error() != want {
		t.Fatalf("expected %v; got %v", want, err)
	}

	badAbs := &Archive{Files: []File{File{Name: "/bad.txt"}}}
	want = `"/bad.txt": outside parent directory`
	if err := Write(badAbs, td); err == nil || err.Error() != want {
		t.Fatalf("expected %v; got %v", want, err)
	}
}
