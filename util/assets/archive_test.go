package assets

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "go.tar.gz")
	writeTarGzArchive(t, archivePath, &tar.Header{
		Name:     "go/bin/tool",
		Mode:     0o755,
		Typeflag: tar.TypeReg,
	}, []byte("tool"))

	destDir := filepath.Join(root, "dest")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(archivePath, destDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "go", "bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "tool" {
		t.Fatalf("extracted data = %q, want %q", data, "tool")
	}
}

func TestExtractTarGzRejectsUnsafeEntries(t *testing.T) {
	tests := []struct {
		name   string
		header *tar.Header
	}{
		{
			name: "parent traversal",
			header: &tar.Header{
				Name:     "../escape",
				Mode:     0o600,
				Typeflag: tar.TypeReg,
			},
		},
		{
			name: "symlink",
			header: &tar.Header{
				Name:     "go/link",
				Linkname: "../../escape",
				Mode:     0o777,
				Typeflag: tar.TypeSymlink,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			archivePath := filepath.Join(root, "bad.tar.gz")
			writeTarGzArchive(t, archivePath, test.header, nil)

			destDir := filepath.Join(root, "dest")
			if err := os.MkdirAll(destDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := extractTarGz(archivePath, destDir); err == nil {
				t.Fatal("extractTarGz() accepted an unsafe archive entry")
			}
			if _, err := os.Lstat(filepath.Join(root, "escape")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("escape path exists or could not be checked: %v", err)
			}
		})
	}
}

func TestExtractZip(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "go.zip")
	writeZipArchive(t, archivePath, "go/bin/tool.exe", []byte("tool"))

	destDir := filepath.Join(root, "dest")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(archivePath, destDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "go", "bin", "tool.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "tool" {
		t.Fatalf("extracted data = %q, want %q", data, "tool")
	}
}

func TestExtractZipRejectsParentTraversal(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "bad.zip")
	writeZipArchive(t, archivePath, "../escape", []byte("escape"))

	destDir := filepath.Join(root, "dest")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(archivePath, destDir); err == nil {
		t.Fatal("extractZip() accepted a parent traversal entry")
	}
	if _, err := os.Stat(filepath.Join(root, "escape")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("escape path exists or could not be checked: %v", err)
	}
}

func writeTarGzArchive(t *testing.T, path string, header *tar.Header, contents []byte) {
	t.Helper()

	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(archive)
	tarWriter := tar.NewWriter(gzipWriter)
	header.Size = int64(len(contents))
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if len(contents) > 0 {
		if _, err := tarWriter.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeZipArchive(t *testing.T, path, name string, contents []byte) {
	t.Helper()

	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipWriter := zip.NewWriter(archive)
	entry, err := zipWriter.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}
