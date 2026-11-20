//go:build server

package assets

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"encoding/binary"
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

func TestServerArchiveExtractorsRejectDuplicatesLinksAndOverwrite(t *testing.T) {
	for _, extractor := range serverArchiveExtractors() {
		extractor := extractor
		t.Run(extractor.name, func(t *testing.T) {
			entryName := func(name string) string {
				if extractor.skipTopLevel {
					return "top/" + name
				}
				return name
			}
			withTop := func(entries ...serverArchiveTestEntry) []serverArchiveTestEntry {
				if extractor.skipTopLevel {
					return append([]serverArchiveTestEntry{{name: "top/", typeflag: tar.TypeDir}}, entries...)
				}
				return entries
			}
			t.Run("case-collision", func(t *testing.T) {
				base := t.TempDir()
				err := extractor.extract(t, base, filepath.Join(base, "destination"), withTop(
					serverArchiveTestEntry{name: entryName("file.txt"), body: "first"},
					serverArchiveTestEntry{name: entryName("FILE.txt"), body: "second"},
				))
				if err == nil {
					t.Fatal("case-colliding archive paths were accepted")
				}
			})
			t.Run("parent-case-collision", func(t *testing.T) {
				base := t.TempDir()
				err := extractor.extract(t, base, filepath.Join(base, "destination"), withTop(
					serverArchiveTestEntry{name: entryName("Dir/first.txt"), body: "first"},
					serverArchiveTestEntry{name: entryName("dir/second.txt"), body: "second"},
				))
				if err == nil {
					t.Fatal("case-colliding parent directories were accepted")
				}
			})
			if extractor.skipTopLevel {
				t.Run("duplicate-top-level", func(t *testing.T) {
					base := t.TempDir()
					err := extractor.extract(t, base, filepath.Join(base, "destination"), withTop(
						serverArchiveTestEntry{name: "top/", typeflag: tar.TypeDir},
					))
					if err == nil {
						t.Fatal("duplicate top-level directory was accepted")
					}
				})
			}
			t.Run("symlink", func(t *testing.T) {
				base := t.TempDir()
				err := extractor.extract(t, base, filepath.Join(base, "destination"), withTop(
					serverArchiveTestEntry{name: entryName("link"), typeflag: tar.TypeSymlink, linkname: "outside", body: "outside"},
				))
				if err == nil {
					t.Fatal("archive symlink was accepted")
				}
			})
			if extractor.name == "untarSkipTopLevel" {
				t.Run("hardlink", func(t *testing.T) {
					base := t.TempDir()
					err := extractor.extract(t, base, filepath.Join(base, "destination"), withTop(
						serverArchiveTestEntry{name: entryName("link"), typeflag: tar.TypeLink, linkname: "top/target"},
					))
					if err == nil {
						t.Fatal("archive hardlink was accepted")
					}
				})
			}
			t.Run("existing-file", func(t *testing.T) {
				base := t.TempDir()
				dest := filepath.Join(base, "destination")
				if err := os.Mkdir(dest, 0700); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(dest, "file.txt")
				if err := os.WriteFile(file, []byte("original"), 0600); err != nil {
					t.Fatal(err)
				}
				err := extractor.extract(t, base, dest, withTop(serverArchiveTestEntry{name: entryName("file.txt"), body: "replacement"}))
				if err == nil {
					t.Fatal("archive overwrote an existing file")
				}
				contents, err := os.ReadFile(file)
				if err != nil || string(contents) != "original" {
					t.Fatalf("existing file changed: %q, %v", contents, err)
				}
			})
		})
	}
}

func TestServerArchiveZipRejectsExcessiveExpansion(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	header := &zip.FileHeader{Name: "zeros.bin", Method: zip.Deflate}
	header.SetMode(0600)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(make([]byte, 8<<20)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := unzipBuf(archive.Bytes(), filepath.Join(t.TempDir(), "destination")); err == nil {
		t.Fatal("high expansion ZIP member was accepted")
	}
}

func TestServerArchiveZipRejectsBadCRC(t *testing.T) {
	archive := writeServerTestZip(t, []serverArchiveTestEntry{{name: "file.txt", body: "checksum-only-body"}})
	index := bytes.Index(archive, []byte("checksum-only-body"))
	if index < 0 {
		t.Fatal("test ZIP member body not found")
	}
	archive[index] ^= 0xff
	dest := filepath.Join(t.TempDir(), "destination")
	if _, err := unzipBuf(archive, dest); err == nil {
		t.Fatal("ZIP member with bad CRC was accepted")
	}
	assertServerPathDoesNotExist(t, filepath.Join(dest, "file.txt"))
}

func TestServerArchiveZipRejectsOversizeDirectory(t *testing.T) {
	archive := writeServerTestZip(t, []serverArchiveTestEntry{{name: "file.txt", body: "body"}})
	end := bytes.LastIndex(archive, []byte{'P', 'K', 5, 6})
	if end < 0 {
		t.Fatal("test ZIP directory end not found")
	}
	binary.LittleEndian.PutUint32(archive[end+12:], maxAssetArchiveCentralBytes+1)
	if _, err := unzipBuf(archive, filepath.Join(t.TempDir(), "destination")); err == nil {
		t.Fatal("ZIP with excessive central directory size was accepted")
	}
}

func TestServerArchiveTarBoundsMetadataStream(t *testing.T) {
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	if err := writer.WriteHeader(&tar.Header{Name: "top/", Mode: 0o700, Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteHeader(&tar.Header{
		Name: "top/file.txt", Mode: 0o600, Size: 1, Typeflag: tar.TypeReg,
		PAXRecords: map[string]string{"comment": strings.Repeat("x", 4096)},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "destination")
	limited := &archiveStreamLimitReader{source: bytes.NewReader(archive.Bytes()), remaining: 1024}
	if err := untarSkipTopLevel(dest, limited); err == nil || !strings.Contains(err.Error(), "stream exceeds size limit") {
		t.Fatalf("expected tar metadata stream limit error, got %v", err)
	}
	assertServerPathDoesNotExist(t, filepath.Join(dest, "file.txt"))
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
		} else if entry.typeflag == tar.TypeSymlink {
			header.SetMode(os.ModeSymlink | 0o600)
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
