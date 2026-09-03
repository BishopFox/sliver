//go:build server

package assets

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type serverArchiveTestEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

type serverArchiveExtractor struct {
	name         string
	skipTopLevel bool
	extract      func(*testing.T, string, string, []serverArchiveTestEntry) error
}

func serverArchiveExtractors() []serverArchiveExtractor {
	return []serverArchiveExtractor{
		{
			name: "unzipBuf",
			extract: func(t *testing.T, _, dest string, entries []serverArchiveTestEntry) error {
				_, err := unzipBuf(writeServerTestZip(t, entries), dest)
				return err
			},
		},
		{
			name: "unzip",
			extract: func(t *testing.T, base, dest string, entries []serverArchiveTestEntry) error {
				archivePath := filepath.Join(base, "archive.zip")
				if err := os.WriteFile(archivePath, writeServerTestZip(t, entries), 0o600); err != nil {
					t.Fatal(err)
				}
				_, err := unzip(archivePath, dest)
				return err
			},
		},
		{
			name:         "unzipSkipTopLevel",
			skipTopLevel: true,
			extract: func(t *testing.T, _, dest string, entries []serverArchiveTestEntry) error {
				archiveBytes := writeServerTestZip(t, entries)
				reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
				if err != nil {
					t.Fatal(err)
				}
				return unzipSkipTopLevel(dest, reader)
			},
		},
		{
			name:         "untarSkipTopLevel",
			skipTopLevel: true,
			extract: func(t *testing.T, _, dest string, entries []serverArchiveTestEntry) error {
				return untarSkipTopLevel(dest, bytes.NewReader(writeServerTestTar(t, entries)))
			},
		},
	}
}

func TestServerArchiveExtractors(t *testing.T) {
	for _, extractor := range serverArchiveExtractors() {
		extractor := extractor
		t.Run(extractor.name, func(t *testing.T) {
			base := t.TempDir()
			dest := filepath.Join(base, "destination")
			entries := []serverArchiveTestEntry{
				{name: "directory/", typeflag: tar.TypeDir},
				{name: "directory/file.txt", body: "contents"},
			}
			if extractor.skipTopLevel {
				entries = []serverArchiveTestEntry{
					{name: "top/", typeflag: tar.TypeDir},
					{name: "top/directory/", typeflag: tar.TypeDir},
					{name: "top/directory/file.txt", body: "contents"},
				}
			}

			if err := extractor.extract(t, base, dest, entries); err != nil {
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

func TestServerArchiveExtractorsRejectUnsafePaths(t *testing.T) {
	for _, extractor := range serverArchiveExtractors() {
		extractor := extractor
		t.Run(extractor.name, func(t *testing.T) {
			tests := []struct {
				name        string
				entryName   func(string, bool) string
				outsidePath func(string) string
			}{
				{
					name: "dot-dot",
					entryName: func(_ string, skip bool) string {
						if skip {
							return "top/../../escaped.txt"
						}
						return "../escaped.txt"
					},
					outsidePath: func(base string) string { return filepath.Join(base, "escaped.txt") },
				},
				{
					name: "absolute",
					entryName: func(base string, _ bool) string {
						return filepath.ToSlash(filepath.Join(base, "absolute.txt"))
					},
					outsidePath: func(base string) string { return filepath.Join(base, "absolute.txt") },
				},
				{
					name: "sibling-prefix",
					entryName: func(_ string, skip bool) string {
						if skip {
							return "top-sibling/escaped.txt"
						}
						return "../destination-sibling/escaped.txt"
					},
					outsidePath: func(base string) string {
						return filepath.Join(base, "destination-sibling", "escaped.txt")
					},
				},
			}

			for _, test := range tests {
				test := test
				t.Run(test.name, func(t *testing.T) {
					base := t.TempDir()
					dest := filepath.Join(base, "destination")
					entries := []serverArchiveTestEntry{{
						name: test.entryName(base, extractor.skipTopLevel),
						body: "escape",
					}}
					if extractor.skipTopLevel {
						entries = append([]serverArchiveTestEntry{{name: "top/", typeflag: tar.TypeDir}}, entries...)
					}

					if err := extractor.extract(t, base, dest, entries); err == nil {
						t.Fatal("extract archive succeeded with unsafe path")
					}
					assertServerPathDoesNotExist(t, test.outsidePath(base))
				})
			}
		})
	}
}

func TestServerArchiveExtractorsRejectPreexistingSymlinkEscape(t *testing.T) {
	for _, extractor := range serverArchiveExtractors() {
		extractor := extractor
		t.Run(extractor.name, func(t *testing.T) {
			base := t.TempDir()
			dest := filepath.Join(base, "destination")
			outside := filepath.Join(base, "outside")
			if err := os.MkdirAll(dest, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(dest, "link")); err != nil {
				t.Skipf("symlinks are unavailable: %v", err)
			}
			entryName := "link/escaped.txt"
			entries := []serverArchiveTestEntry{{name: entryName, body: "escape"}}
			if extractor.skipTopLevel {
				entries = []serverArchiveTestEntry{
					{name: "top/", typeflag: tar.TypeDir},
					{name: "top/" + entryName, body: "escape"},
				}
			}

			if err := extractor.extract(t, base, dest, entries); err == nil {
				t.Fatal("extract archive succeeded through symlink outside destination")
			}
			assertServerPathDoesNotExist(t, filepath.Join(outside, "escaped.txt"))
		})
	}
}

func TestServerArchivePathsUsePlatformPaths(t *testing.T) {
	got, err := archiveEntryPath("directory/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("directory", "file.txt"); got != want {
		t.Fatalf("archiveEntryPath() = %q, want %q", got, want)
	}
	if _, err := archivePathBelowTopLevel("top-sibling/file.txt", "top"); err == nil {
		t.Fatal("archivePathBelowTopLevel accepted a sibling-prefix path")
	}

	if runtime.GOOS == "windows" {
		for _, name := range []string{"C:/escaped.txt", `directory\escaped.txt`} {
			if _, err := archiveEntryPath(name); err == nil {
				t.Errorf("archiveEntryPath(%q) succeeded on Windows", name)
			}
		}
	}
}

func writeServerTestZip(t *testing.T, entries []serverArchiveTestEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		if entry.typeflag == tar.TypeDir || strings.HasSuffix(entry.name, "/") {
			header.SetMode(os.ModeDir | 0o700)
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
	return archive.Bytes()
}

func writeServerTestTar(t *testing.T, entries []serverArchiveTestEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
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
			header.Mode = 0o700
			header.Size = 0
		}
		if typeflag == tar.TypeSymlink {
			header.Size = 0
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if typeflag == tar.TypeReg {
			if _, err := writer.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func assertServerPathDoesNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("path %q exists after rejected extraction (error: %v)", path, err)
	}
}
