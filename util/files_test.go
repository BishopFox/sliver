package util

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
)

func TestByteCountBinary(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TiB"},
		{1024 * 1024 * 1024 * 1024 * 1024, "1.0 PiB"},
		{1024 * 1024 * 1024 * 1024 * 1024 * 1024, "1.0 EiB"},
	}
	for _, test := range tests {
		if got := ByteCountBinary(test.bytes); got != test.expected {
			t.Errorf("ByteCountBinary(%d) = %q, want %q", test.bytes, got, test.expected)
		}
	}
}

func TestDeflateBuf(t *testing.T) {
	// Highly compressible input so we can also assert the output is smaller.
	original := bytes.Repeat([]byte("sliver adversary emulation framework\n"), 512)

	compressed := DeflateBuf(original)
	if len(compressed) == 0 {
		t.Fatalf("DeflateBuf returned an empty buffer")
	}
	if len(compressed) >= len(original) {
		t.Fatalf("DeflateBuf did not compress: %d >= %d", len(compressed), len(original))
	}

	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()
	inflated, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to inflate DeflateBuf output: %v", err)
	}
	if !bytes.Equal(inflated, original) {
		t.Fatalf("round-trip mismatch: inflated data does not equal the original")
	}
}

func TestDeflateBufEmpty(t *testing.T) {
	compressed := DeflateBuf([]byte{})
	reader := flate.NewReader(bytes.NewReader(compressed))
	defer reader.Close()
	inflated, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to inflate empty DeflateBuf output: %v", err)
	}
	if len(inflated) != 0 {
		t.Fatalf("expected empty inflated output, got %d bytes", len(inflated))
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	content := []byte("the quick brown fox jumps over the lazy dog")

	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile returned an error: %v", err)
	}

	copied, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if !bytes.Equal(copied, content) {
		t.Fatalf("copied content mismatch: got %q, want %q", copied, content)
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "does-not-exist.bin")
	dst := filepath.Join(dir, "dst.bin")

	if err := CopyFile(src, dst); err == nil {
		t.Fatalf("expected an error copying a nonexistent source file")
	}
}

func TestReadFileFromTarGz(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.tar.gz")
	wantName := "nested/data.txt"
	wantContent := []byte("payload contents inside the archive")

	writeTarGz(t, archivePath, map[string][]byte{
		wantName:           wantContent,
		"nested/other.txt": []byte("some other file"),
	})

	got, err := ReadFileFromTarGz(archivePath, wantName)
	if err != nil {
		t.Fatalf("ReadFileFromTarGz returned an error: %v", err)
	}
	if !bytes.Equal(got, wantContent) {
		t.Fatalf("content mismatch: got %q, want %q", got, wantContent)
	}
}

func TestReadFileFromTarGzMissingEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.tar.gz")
	writeTarGz(t, archivePath, map[string][]byte{"present.txt": []byte("here")})

	got, err := ReadFileFromTarGz(archivePath, "absent.txt")
	if err != nil {
		t.Fatalf("ReadFileFromTarGz returned an error for a missing entry: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a missing entry, got %q", got)
	}
}

func TestReadFileFromTarGzMissingArchive(t *testing.T) {
	if _, err := ReadFileFromTarGz(filepath.Join(t.TempDir(), "nope.tar.gz"), "any"); err == nil {
		t.Fatalf("expected an error opening a nonexistent archive")
	}
}

func TestChmodR(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode bits are not meaningful on Windows")
	}

	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("failed to create sub directory: %v", err)
	}
	filePath := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	const filePerm = os.FileMode(0o600)
	const dirPerm = os.FileMode(0o700)
	if err := ChmodR(root, filePerm, dirPerm); err != nil {
		t.Fatalf("ChmodR returned an error: %v", err)
	}

	for _, dir := range []string{root, subDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat %s: %v", dir, err)
		}
		if info.Mode().Perm() != dirPerm {
			t.Fatalf("directory %s has perm %o, want %o", dir, info.Mode().Perm(), dirPerm)
		}
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != filePerm {
		t.Fatalf("file has perm %o, want %o", info.Mode().Perm(), filePerm)
	}
}

// writeTarGz builds a gzip-compressed tar archive at path containing the given
// name -> content entries.
func writeTarGz(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	for name, content := range entries {
		header := &tar.Header{
			Name:     name,
			Mode:     0o600,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("failed to write tar body: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
}
