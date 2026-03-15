package docs

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"embed"
	"io/fs"
	"path"
	"strings"
)

const markdownDir = "sliver-docs/pages/docs/md"

var (
	// rawFS stores the embedded markdown sources for the Sliver docs site.
	//go:embed sliver-docs/pages/docs/md/*.md
	rawFS embed.FS

	// FS provides access to the embedded markdown docs rooted at the md directory.
	FS = mustSubFS(rawFS, markdownDir)
)

// Doc is a single embedded markdown document.
type Doc struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Docs mirrors the JSON shape used by the docs site.
type Docs struct {
	Docs []Doc `json:"docs"`
}

// All returns every embedded markdown document.
func All() (Docs, error) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		return Docs{}, err
	}

	docs := make([]Doc, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".md" {
			continue
		}

		content, err := fs.ReadFile(FS, entry.Name())
		if err != nil {
			return Docs{}, err
		}

		docs = append(docs, Doc{
			Name:    strings.TrimSuffix(entry.Name(), path.Ext(entry.Name())),
			Content: string(content),
		})
	}
	return Docs{Docs: docs}, nil
}

// Read returns the embedded markdown for a doc name, with or without the .md suffix.
func Read(name string) (string, error) {
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}

	content, err := fs.ReadFile(FS, name)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func mustSubFS(embedded fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(embedded, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
