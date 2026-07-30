package generate

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestZipFilesIncludesArchiveHeaders(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"implant.a": "archive",
		"implant.h": "cgo header",
		"main.h":    "sliver header",
	}
	var paths []string
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		paths = append(paths, path)
	}

	zipPath := filepath.Join(dir, "implant.zip")
	if err := zipFiles(zipPath, paths...); err != nil {
		t.Fatalf("zipFiles() error: %v", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	got := map[string]string{}
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zipped file %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read zipped file %s: %v", file.Name, err)
		}
		got[file.Name] = string(data)
	}

	if len(got) != len(files) {
		t.Fatalf("zip contained %d files, want %d", len(got), len(files))
	}
	for name, contents := range files {
		if got[name] != contents {
			t.Fatalf("zipped %s = %q, want %q", name, got[name], contents)
		}
	}
}
