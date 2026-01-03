package goname

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameImportBasic(t *testing.T) {
	fixture := filepath.Join("testdata", "rename-import", "basic")
	tmp := t.TempDir()
	if err := copyDir(fixture, tmp); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	result, err := RenameImport(tmp, "foo/bar", "acme/corp")
	if err != nil {
		t.Fatalf("RenameImport: %v", err)
	}

	if result.OldImportPrefix != "foo/bar" {
		t.Fatalf("old prefix mismatch: %s", result.OldImportPrefix)
	}
	if result.NewImportPrefix != "acme/corp" {
		t.Fatalf("new prefix mismatch: %s", result.NewImportPrefix)
	}
	if result.FilesUpdated != 2 {
		t.Fatalf("files updated mismatch: %d", result.FilesUpdated)
	}
	if result.ImportsUpdated != 4 {
		t.Fatalf("imports updated mismatch: %d", result.ImportsUpdated)
	}

	mainImports := importPaths(t, filepath.Join(tmp, "main.go"))
	for _, want := range []string{"acme/corp", "acme/corp/baz", "acme/corp/qux", "foo/barista", "other/pkg"} {
		if !mainImports[want] {
			t.Fatalf("main.go missing import: %s", want)
		}
	}
	if mainImports["foo/bar"] || mainImports["foo/bar/baz"] || mainImports["foo/bar/qux"] {
		t.Fatalf("main.go still references old prefix")
	}

	barImports := importPaths(t, filepath.Join(tmp, "foo", "bar", "bar.go"))
	if !barImports["acme/corp/baz"] {
		t.Fatalf("bar.go import not updated")
	}

	vendorImports := importPaths(t, filepath.Join(tmp, "vendor", "foo", "bar", "vendor.go"))
	if !vendorImports["foo/bar/baz"] {
		t.Fatalf("vendor import should remain unchanged")
	}

	hiddenImports := importPaths(t, filepath.Join(tmp, ".hidden", "hidden.go"))
	if !hiddenImports["foo/bar/baz"] {
		t.Fatalf("hidden import should remain unchanged")
	}

	mainBytes, err := os.ReadFile(filepath.Join(tmp, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainBytes), "const example = \"foo/bar/baz\"") {
		t.Fatalf("string literal should remain unchanged")
	}
}

func TestRenameImportNoChanges(t *testing.T) {
	fixture := filepath.Join("testdata", "rename-import", "basic")
	tmp := t.TempDir()
	if err := copyDir(fixture, tmp); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}

	_, err := RenameImport(tmp, "missing/prefix", "acme/corp")
	if err == nil {
		t.Fatalf("expected error for missing prefix")
	}
}

func TestRenameImportInvalidArgs(t *testing.T) {
	_, err := RenameImport(t.TempDir(), "", "acme/corp")
	if err == nil {
		t.Fatalf("expected error for empty old prefix")
	}
	_, err = RenameImport(t.TempDir(), "foo/bar", "")
	if err == nil {
		t.Fatalf("expected error for empty new prefix")
	}
}
