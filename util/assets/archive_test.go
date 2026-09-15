package assets

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type archiveTestEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

type archiveTestFormat struct {
	name      string
	extension string
	write     func(*testing.T, string, []archiveTestEntry)
	extract   func(string, string) error
}

func archiveTestFormats() []archiveTestFormat {
	return []archiveTestFormat{
		{name: "zip", extension: ".zip", write: writeTestZip, extract: extractZip},
		{name: "tar.gz", extension: ".tar.gz", write: writeTestTarGz, extract: extractTarGz},
	}
}

func TestExtractArchive(t *testing.T) {
	for _, format := range archiveTestFormats() {
		format := format
		t.Run(format.name, func(t *testing.T) {
			base := t.TempDir()
			archivePath := filepath.Join(base, "archive"+format.extension)
			dest := filepath.Join(base, "dest")
			format.write(t, archivePath, []archiveTestEntry{
				{name: "directory/", typeflag: tar.TypeDir},
				{name: "directory/file.txt", body: "contents"},
			})

			if err := format.extract(archivePath, dest); err != nil {
				t.Fatalf("extract archive: %v", err)
			}
			contents, err := os.ReadFile(filepath.Join(dest, "directory", "file.txt"))
			if err != nil {
				t.Fatalf("read extracted file: %v", err)
			}
			if string(contents) != "contents" {
				t.Fatalf("extracted contents = %q, want %q", contents, "contents")
			}
		})
	}
}

func TestExtractArchiveRejectsUnsafePaths(t *testing.T) {
	for _, format := range archiveTestFormats() {
		format := format
		t.Run(format.name, func(t *testing.T) {
			tests := []struct {
				name      string
				entryName func(string) string
				outside   func(string) string
			}{
				{
					name:      "dot-dot",
					entryName: func(string) string { return "../escaped.txt" },
					outside:   func(base string) string { return filepath.Join(base, "escaped.txt") },
				},
				{
					name: "absolute",
					entryName: func(base string) string {
						return filepath.ToSlash(filepath.Join(base, "absolute.txt"))
					},
					outside: func(base string) string { return filepath.Join(base, "absolute.txt") },
				},
				{
					name:      "sibling-prefix",
					entryName: func(string) string { return "../destination-sibling/escaped.txt" },
					outside: func(base string) string {
						return filepath.Join(base, "destination-sibling", "escaped.txt")
					},
				},
			}

			for _, test := range tests {
				test := test
				t.Run(test.name, func(t *testing.T) {
					base := t.TempDir()
					archivePath := filepath.Join(base, "archive"+format.extension)
					dest := filepath.Join(base, "destination")
					format.write(t, archivePath, []archiveTestEntry{{
						name: test.entryName(base),
						body: "escape",
					}})

					if err := format.extract(archivePath, dest); err == nil {
						t.Fatal("extract archive succeeded with unsafe path")
					}
					assertPathDoesNotExist(t, test.outside(base))
				})
			}
		})
	}
}

func TestExtractArchiveRejectsPreexistingSymlinkEscape(t *testing.T) {
	for _, format := range archiveTestFormats() {
		format := format
		t.Run(format.name, func(t *testing.T) {
			base := t.TempDir()
			archivePath := filepath.Join(base, "archive"+format.extension)
			dest := filepath.Join(base, "destination")
			outside := filepath.Join(base, "outside")
			if err := os.MkdirAll(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(dest, "link")); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}
			format.write(t, archivePath, []archiveTestEntry{{name: "link/escaped.txt", body: "escape"}})

			if err := format.extract(archivePath, dest); err == nil {
				t.Fatal("extract archive succeeded through symlink outside destination")
			}
			assertPathDoesNotExist(t, filepath.Join(outside, "escaped.txt"))
		})
	}
}

func TestExtractTarGzRejectsArchiveSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	archivePath := filepath.Join(base, "archive.tar.gz")
	dest := filepath.Join(base, "destination")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestTarGz(t, archivePath, []archiveTestEntry{
		{name: "link", typeflag: tar.TypeSymlink, linkname: "../outside"},
		{name: "link/escaped.txt", body: "escape"},
	})

	if err := extractTarGz(archivePath, dest); err == nil {
		t.Fatal("extract tar.gz succeeded with escaping symlink target")
	}
	assertPathDoesNotExist(t, filepath.Join(outside, "escaped.txt"))
}

func TestArchiveEntryPathUsesPlatformPaths(t *testing.T) {
	got, err := archiveEntryPath("directory/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("directory", "file.txt"); got != want {
		t.Fatalf("archiveEntryPath() = %q, want %q", got, want)
	}

	if runtime.GOOS == "windows" {
		for _, name := range []string{"C:/escaped.txt", `directory\escaped.txt`} {
			if _, err := archiveEntryPath(name); err == nil {
				t.Errorf("archiveEntryPath(%q) succeeded on Windows", name)
			}
		}
	}
}

func writeTestZip(t *testing.T, archivePath string, entries []archiveTestEntry) {
	t.Helper()
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archiveFile)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		if entry.typeflag == tar.TypeDir || strings.HasSuffix(entry.name, "/") {
			header.SetMode(os.ModeDir | 0o755)
		} else {
			header.SetMode(0o600)
		}
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entryWriter.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestTarGz(t *testing.T, archivePath string, entries []archiveTestEntry) {
	t.Helper()
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(archiveFile)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o600,
			Size:     int64(len(entry.body)),
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		if typeflag == tar.TypeDir {
			header.Mode = 0o755
			header.Size = 0
		}
		if typeflag == tar.TypeSymlink {
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("path %q exists after rejected extraction (error: %v)", path, err)
	}
}
