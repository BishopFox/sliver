package goname

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestRenameModule(t *testing.T) {
	fixture := filepath.Join("testdata", "basic")
	tmp := t.TempDir()
	if err := copyDir(fixture, tmp); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	result, err := RenameModule(tmp, "example.net/new")
	if err != nil {
		t.Fatalf("RenameModule: %v", err)
	}

	if result.OldModule != "example.com/old" {
		t.Fatalf("old module mismatch: %s", result.OldModule)
	}
	if result.NewModule != "example.net/new" {
		t.Fatalf("new module mismatch: %s", result.NewModule)
	}
	if !result.GoModUpdated {
		t.Fatalf("expected go.mod update")
	}
	if result.FilesUpdated != 2 {
		t.Fatalf("files updated mismatch: %d", result.FilesUpdated)
	}
	if result.ImportsUpdated != 3 {
		t.Fatalf("imports updated mismatch: %d", result.ImportsUpdated)
	}

	modBytes, err := os.ReadFile(filepath.Join(tmp, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	modFile, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	if modFile.Module == nil || modFile.Module.Mod.Path != "example.net/new" {
		t.Fatalf("module path not updated: %#v", modFile.Module)
	}

	mainImports := importPaths(t, filepath.Join(tmp, "main.go"))
	for _, want := range []string{"example.net/new", "example.net/new/pkg/foo", "example.com/oldish/pkg"} {
		if !mainImports[want] {
			t.Fatalf("main.go missing import: %s", want)
		}
	}
	if mainImports["example.com/old"] || mainImports["example.com/old/pkg/foo"] {
		t.Fatalf("main.go still references old module")
	}

	fooImports := importPaths(t, filepath.Join(tmp, "pkg", "foo", "foo.go"))
	if !fooImports["example.net/new/pkg/bar"] {
		t.Fatalf("foo.go import not updated")
	}

	vendorImports := importPaths(t, filepath.Join(tmp, "vendor", "example.com", "old", "vendor.go"))
	if !vendorImports["example.com/old/pkg/foo"] {
		t.Fatalf("vendor import should remain unchanged")
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmp, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainBytes), "const example = \"example.com/old/pkg/foo\"") {
		t.Fatalf("string literal should remain unchanged")
	}
}

func TestRenameModuleMissingGoMod(t *testing.T) {
	_, err := RenameModule(t.TempDir(), "example.com/new")
	if err == nil {
		t.Fatalf("expected error for missing go.mod")
	}
}

func importPaths(t *testing.T, path string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	imports := make(map[string]bool)
	for _, imp := range file.Imports {
		pathValue, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("unquote import in %s: %v", path, err)
		}
		imports[pathValue] = true
	}
	return imports
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
