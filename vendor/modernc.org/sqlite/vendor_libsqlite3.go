// Copyright 2024 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build none
// +build none

// Tool for 1.28+ -> 1.29+. Pulls adjusted libsqlite3 code to this repo.

package main

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"modernc.org/gc/v3"
)

func fail(rc int, msg string, args ...any) {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(fmt.Sprintf(msg, args...)))
	os.Exit(rc)
}

func main() {
	for _, v := range []struct{ goos, goarch string }{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"freebsd", "amd64"},
		{"freebsd", "arm64"},
		{"linux", "386"},
		{"linux", "amd64"},
		{"linux", "arm"},
		{"linux", "arm64"},
		{"linux", "loong64"},
		{"linux", "ppc64le"},
		{"linux", "riscv64"},
		{"linux", "s390x"},
		{"windows", "amd64"},
	} {
		fmt.Printf("%s/%s\n", v.goos, v.goarch)
		base := fmt.Sprintf("ccgo_%s_%s.go", v.goos, v.goarch)
		if v.goos == "windows" {
			base = "ccgo_windows.go"
		}
		ifn := filepath.Join("..", "libsqlite3", base)
		in, err := os.ReadFile(ifn)
		if err != nil {
			fail(1, "%s\n", err)
		}

		ast, err := gc.ParseFile(ifn, in)
		if err != nil {
			fail(1, "%s\n", err)
		}

		b := bytes.NewBuffer(nil)
		s := ast.SourceFile.PackageClause.Source(true)
		s = strings.Replace(s, "package libsqlite3", "package sqlite3", 1)
		fmt.Fprintln(b, s)
		fmt.Fprint(b, ast.SourceFile.ImportDeclList.Source(true))
		taken := map[string]struct{}{}
		for n := ast.SourceFile.TopLevelDeclList; n != nil; n = n.List {
			switch x := n.TopLevelDecl.(type) {
			case *gc.TypeDeclNode:
				adn := x.TypeSpecList.TypeSpec.(*gc.AliasDeclNode)
				nm := adn.IDENT.Src()
				taken[nm] = struct{}{}
			}
		}
	loop:
		for n := ast.SourceFile.TopLevelDeclList; n != nil; n = n.List {
			switch x := n.TopLevelDecl.(type) {
			case *gc.ConstDeclNode:
				switch y := x.ConstSpec.(type) {
				case *gc.ConstSpecNode:
					if y.IDENT.Src() != "SQLITE_TRANSIENT" {
						fmt.Fprintln(b, x.Source(true))
					}
				default:
					panic(fmt.Sprintf("%v: %T %q", x.Position(), y, x.Source(false)))
				}

			case *gc.FunctionDeclNode:
				fmt.Fprintln(b, x.Source(true))
			case *gc.TypeDeclNode:
				fmt.Fprintln(b, x.Source(true))
				adn := x.TypeSpecList.TypeSpec.(*gc.AliasDeclNode)
				nm := adn.IDENT.Src()
				nm2 := nm[1:]
				if _, ok := taken[nm2]; ok {
					break
				}

				if token.IsExported(nm) {
					fmt.Fprintf(b, "\ntype %s = %s\n", nm2, nm)
				}
			case *gc.VarDeclNode:
				fmt.Fprintln(b, x.Source(true))
			default:
				fmt.Printf("%v: TODO %T\n", n.Position(), x)
				break loop
			}
		}

		b.WriteString(`
type Sqlite3_int64 = sqlite3_int64
type Sqlite3_mutex_methods = sqlite3_mutex_methods
type Sqlite3_value = sqlite3_value

type Sqlite3_index_info = sqlite3_index_info
type Sqlite3_module = sqlite3_module
type Sqlite3_vtab = sqlite3_vtab
type Sqlite3_vtab_cursor = sqlite3_vtab_cursor

`)
		base = strings.Replace(base, "ccgo_", "sqlite_", 1)
		if err := os.WriteFile(filepath.Join("lib", base), b.Bytes(), 0660); err != nil {
			fail(1, "%s\n", err)
		}
	}
}
