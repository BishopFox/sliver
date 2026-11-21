package workspace

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWorkspaceCopiesLiteralSourceAndPrivateInputs(t *testing.T) {
	appDir := t.TempDir()
	first, err := New(appDir, "linux", "amd64", "example")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, first)
	second, err := New(appDir, "linux", "amd64", "example")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, second)
	if first.Root == second.Root {
		t.Fatal("concurrent builds received the same workspace")
	}
	if !strings.HasPrefix(first.Root, filepath.Join(appDir, "slivers", "linux", "amd64", "example")) {
		t.Fatalf("unexpected workspace root %q", first.Root)
	}

	source := fstest.MapFS{
		"main.go":           &fstest.MapFile{Data: []byte(`package main // {{ .Value }}`)},
		"nested/config.txt": &fstest.MapFile{Data: []byte("contents")},
	}
	if err := first.CopySource(source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(first.SourceDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `package main // {{ .Value }}` {
		t.Fatalf("source was processed during copy: %q", data)
	}
	if err := first.CopyMagefiles(fstest.MapFS{"magefile.go": &fstest.MapFile{Data: []byte("package main")}}); err != nil {
		t.Fatal(err)
	}
	checkPrivateInput(t, first)
	if err := first.Remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.Root); !os.IsNotExist(err) {
		t.Fatalf("workspace remains after cleanup: %v", err)
	}
	if _, err := os.Stat(second.Root); err != nil {
		t.Fatalf("cleanup removed another build: %v", err)
	}
}

func checkPrivateInput(t *testing.T, work *Workspace) {
	t.Helper()
	materialPath, err := work.WriteInput("material.json", []byte(`{"secret":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(materialPath)
	if err != nil {
		t.Fatal(err)
	}
	// Windows maps file attributes to FileMode without POSIX permission bits.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("material mode = %o", info.Mode().Perm())
	}
	if _, err := work.WriteInput("material.json", []byte("replacement")); err == nil {
		t.Fatal("duplicate input file overwrote secret material")
	}
}

func cleanWorkspace(t *testing.T, work *Workspace) {
	t.Helper()
	t.Cleanup(func() {
		if err := work.Remove(); err != nil {
			t.Errorf("remove workspace: %v", err)
		}
	})
}

func TestWorkspaceCopyRejectsPortablePathCollisions(t *testing.T) {
	for _, source := range []fstest.MapFS{
		{"File.go": &fstest.MapFile{Data: []byte("a")}, "file.go": &fstest.MapFile{Data: []byte("b")}},
		{"caf\u00e9/a.go": &fstest.MapFile{Data: []byte("a")}, "cafe\u0301/b.go": &fstest.MapFile{Data: []byte("b")}},
		{"Stra\u00dfe/a.go": &fstest.MapFile{Data: []byte("a")}, "STRASSE/b.go": &fstest.MapFile{Data: []byte("b")}},
	} {
		work, err := New(t.TempDir(), "linux", "amd64", "collision")
		if err != nil {
			t.Fatal(err)
		}
		if err := work.CopySource(source); err == nil {
			t.Errorf("copied source with platform-colliding paths: %#v", source)
		}
		if err := work.Remove(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorkspaceRejectsUnsafeNamesAndSourceLinks(t *testing.T) {
	appDir := t.TempDir()
	for _, name := range []string{"..", "../escape", "C:drive", "name/other"} {
		if _, err := New(appDir, "linux", "amd64", name); err == nil {
			t.Errorf("accepted unsafe name %q", name)
		}
	}
	for _, name := range []string{".hidden", "name.", "CON"} {
		portable, err := New(appDir, "linux", "amd64", name)
		if err != nil {
			t.Errorf("rejected legacy name %q: %v", name, err)
			continue
		}
		if !strings.Contains(portable.Root, string(filepath.Separator)+"name-") {
			t.Errorf("legacy name %q did not get a portable directory: %q", name, portable.Root)
		}
		_ = portable.Remove()
	}
	workspace, err := New(appDir, "linux", "amd64", "safe")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, workspace)
	if err := workspace.CopySource(fstest.MapFS{"link": &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("target")}}); err == nil {
		t.Fatal("copied a source symlink")
	}
	if _, err := workspace.WriteInput("../outside", []byte("secret")); err == nil {
		t.Fatal("accepted a traversing input name")
	}
}

func TestWorkspaceRejectsReplacedSourceDirectory(t *testing.T) {
	workspace, err := New(t.TempDir(), "linux", "amd64", "safe")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, workspace)
	outside := t.TempDir()
	if err := os.Remove(workspace.SourceDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, workspace.SourceDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := workspace.CopySource(fstest.MapFS{"main.go": &fstest.MapFile{Data: []byte("source")}}); err == nil {
		t.Fatal("copied source through a replaced destination")
	}
	if _, err := os.Stat(filepath.Join(outside, "main.go")); !os.IsNotExist(err) {
		t.Fatalf("source escaped its workspace: %v", err)
	}
}

func TestRemoveOutputWorkspaceOnlyRemovesMatchingBuild(t *testing.T) {
	appDir := t.TempDir()
	first, err := New(appDir, "linux", "amd64", "example")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, first)
	second, err := New(appDir, "linux", "amd64", "example")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, second)
	output := filepath.Join(first.BinDir, "example.elf")
	if err := os.WriteFile(output, []byte("artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		filepath.Join(first.SourceDir, "example.elf"),
		filepath.Join(first.BinDir, "..", "src", "example.elf"),
		filepath.Join(appDir, "builds", "example.elf"),
	} {
		if err := RemoveOutputWorkspace(appDir, "linux", "amd64", "example", bad); err == nil {
			t.Errorf("accepted unrelated output path %q", bad)
		}
	}
	if err := RemoveOutputWorkspace(appDir, "linux", "amd64", "other", output); err == nil {
		t.Fatal("accepted wrong implant name")
	}
	if _, err := os.Stat(first.Root); err != nil {
		t.Fatalf("invalid cleanup removed workspace: %v", err)
	}
	if err := RemoveOutputWorkspace(appDir, "linux", "amd64", "example", output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.Root); !os.IsNotExist(err) {
		t.Fatalf("matching workspace was not removed: %v", err)
	}
	if _, err := os.Stat(second.Root); err != nil {
		t.Fatalf("cleanup removed another build: %v", err)
	}
}

type zipEntry struct {
	name string
	mode fs.FileMode
	data string
}

func writeZIP(t *testing.T, entries []zipEntry) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "bundle.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		out, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(entry.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestSourceZIPExtractionAcceptsFilesAndRejectsUnsafeEntries(t *testing.T) {
	appDir := t.TempDir()
	valid, err := New(appDir, "linux", "amd64", "valid")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, valid)
	archivePath := writeZIP(t, []zipEntry{{"main.go", 0o600, "package main // {{ .Value }}"}, {"nested/file", 0o600, "bytes"}})
	if err := valid.ExtractSourceArchive(archivePath, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(valid.SourceDir, "main.go"))
	if err != nil || !bytes.Contains(data, []byte("{{ .Value }}")) {
		t.Fatalf("extracted source changed template text: %q, %v", data, err)
	}

	cases := []struct {
		name    string
		entries []zipEntry
		limits  ArchiveLimits
	}{
		{"traversal", []zipEntry{{"../outside", 0o600, "bad"}}, ArchiveLimits{}},
		{"windows path", []zipEntry{{`C:\outside`, 0o600, "bad"}}, ArchiveLimits{}},
		{"symlink", []zipEntry{{"link", fs.ModeSymlink | 0o777, "../outside"}}, ArchiveLimits{}},
		{"duplicate", []zipEntry{{"file", 0o600, "a"}, {"file", 0o600, "b"}}, ArchiveLimits{}},
		{"case collision", []zipEntry{{"File", 0o600, "a"}, {"file", 0o600, "b"}}, ArchiveLimits{}},
		{"parent case collision", []zipEntry{{"Dir/a", 0o600, "a"}, {"dir/b", 0o600, "b"}}, ArchiveLimits{}},
		{"implicit directory replaced by file", []zipEntry{{"dir/a", 0o600, "a"}, {"dir", 0o600, "b"}}, ArchiveLimits{}},
		{"file used as directory", []zipEntry{{"dir", 0o600, "a"}, {"dir/b", 0o600, "b"}}, ArchiveLimits{}},
		{"unicode collision", []zipEntry{{"caf\u00e9/a", 0o600, "a"}, {"cafe\u0301/b", 0o600, "b"}}, ArchiveLimits{}},
		{"case fold collision", []zipEntry{{"Stra\u00dfe/a", 0o600, "a"}, {"STRASSE/b", 0o600, "b"}}, ArchiveLimits{}},
		{"file limit", []zipEntry{{"file", 0o600, strings.Repeat("x", 64)}}, ArchiveLimits{MaxFileBytes: 32}},
		{"entry limit", []zipEntry{{"one", 0o600, "a"}, {"two", 0o600, "b"}}, ArchiveLimits{MaxEntries: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, err := New(appDir, "linux", "amd64", "reject")
			if err != nil {
				t.Fatal(err)
			}
			cleanWorkspace(t, workspace)
			archivePath := writeZIP(t, tc.entries)
			if err := workspace.ExtractSourceArchive(archivePath, tc.limits); err == nil {
				t.Fatal("unsafe ZIP was accepted")
			}
		})
	}
	if _, err := os.Stat(filepath.Join(appDir, "slivers", "linux", "amd64", "outside")); !os.IsNotExist(err) {
		t.Fatalf("archive escaped its destination: %v", err)
	}
}

func TestSourceZIPRejectsForgedCentralDirectoryCountsBeforeParsing(t *testing.T) {
	for _, count := range []uint16{0, 60000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			archivePath := writeZIP(t, []zipEntry{{"main.go", 0o600, "package main"}})
			archive, err := os.ReadFile(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			end := archive[len(archive)-22:]
			binary.LittleEndian.PutUint16(end[8:10], count)
			binary.LittleEndian.PutUint16(end[10:12], count)
			if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			if err := precheckZIPDirectory(file, int64(len(archive)), defaultMaxEntries); err == nil {
				t.Fatal("forged central directory count passed precheck")
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			workspace, err := New(t.TempDir(), "linux", "amd64", "forged")
			if err != nil {
				t.Fatal(err)
			}
			cleanWorkspace(t, workspace)
			if err := workspace.ExtractSourceArchive(archivePath, ArchiveLimits{}); err == nil {
				t.Fatal("forged central directory count was extracted")
			}
		})
	}
}

func TestSourceZIPRejectsEncryptedAndUnsupportedMethods(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		patch func(local, central []byte)
	}{
		{"encrypted", func(local, central []byte) {
			binary.LittleEndian.PutUint16(local[6:8], binary.LittleEndian.Uint16(local[6:8])|1)
			binary.LittleEndian.PutUint16(central[8:10], binary.LittleEndian.Uint16(central[8:10])|1)
		}},
		{"unsupported method", func(local, central []byte) {
			binary.LittleEndian.PutUint16(local[8:10], 99)
			binary.LittleEndian.PutUint16(central[10:12], 99)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			archivePath := writeZIP(t, []zipEntry{{"file", 0o600, "bytes"}})
			archive, err := os.ReadFile(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			local := bytes.Index(archive, []byte("PK\x03\x04"))
			central := bytes.Index(archive, []byte("PK\x01\x02"))
			if local < 0 || central < 0 {
				t.Fatal("missing ZIP headers")
			}
			testCase.patch(archive[local:], archive[central:])
			if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
				t.Fatal(err)
			}
			workspace, err := New(t.TempDir(), "linux", "amd64", "reject")
			if err != nil {
				t.Fatal(err)
			}
			cleanWorkspace(t, workspace)
			if err := workspace.ExtractSourceArchive(archivePath, ArchiveLimits{}); err == nil ||
				!strings.Contains(err.Error(), "unsupported ZIP entry") {
				t.Fatalf("expected unsupported ZIP entry error, got %v", err)
			}
		})
	}
}

func writeTAR(t *testing.T, compressed bool, entries []tar.Header) string {
	t.Helper()
	suffix := ".tar"
	if compressed {
		suffix = ".tar.gz"
	}
	archivePath := filepath.Join(t.TempDir(), "bundle"+suffix)
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	var writer *tar.Writer
	var gz *gzip.Writer
	if compressed {
		gz = gzip.NewWriter(file)
		writer = tar.NewWriter(gz)
	} else {
		writer = tar.NewWriter(file)
	}
	for _, entry := range entries {
		if err := writer.WriteHeader(&entry); err != nil {
			t.Fatal(err)
		}
		if entry.Typeflag == tar.TypeReg {
			if _, err := writer.Write(bytes.Repeat([]byte("x"), int(entry.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestTARExtractionRejectsLinksTraversalAndOversize(t *testing.T) {
	appDir := t.TempDir()
	cases := []struct {
		name       string
		compressed bool
		header     tar.Header
		limits     ArchiveLimits
	}{
		{"traversal", false, tar.Header{Name: "../outside", Typeflag: tar.TypeReg, Size: 1}, ArchiveLimits{}},
		{"symlink", false, tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "target"}, ArchiveLimits{}},
		{"hardlink", true, tar.Header{Name: "link", Typeflag: tar.TypeLink, Linkname: "target"}, ArchiveLimits{}},
		{"oversize", true, tar.Header{Name: "file", Typeflag: tar.TypeReg, Size: 64}, ArchiveLimits{MaxFileBytes: 32}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspace, err := New(appDir, "linux", "amd64", "reject")
			if err != nil {
				t.Fatal(err)
			}
			cleanWorkspace(t, workspace)
			archivePath := writeTAR(t, tc.compressed, []tar.Header{tc.header})
			if err := workspace.ExtractSourceArchive(archivePath, tc.limits); err == nil {
				t.Fatal("unsafe TAR was accepted")
			}
		})
	}
	valid, err := New(appDir, "linux", "amd64", "valid")
	if err != nil {
		t.Fatal(err)
	}
	cleanWorkspace(t, valid)
	archivePath := writeTAR(t, true, []tar.Header{{Name: "file", Typeflag: tar.TypeReg, Size: 4, Mode: 0o600}})
	if err := valid.ExtractSourceArchive(archivePath, ArchiveLimits{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(valid.SourceDir, "file"))
	if err != nil || string(data) != "xxxx" {
		t.Fatalf("unexpected TAR extraction: %q, %v", data, err)
	}
}

func TestTARExtractionRejectsConflictingPaths(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		compressed bool
		entries    []tar.Header
	}{
		{"parent case collision", true, []tar.Header{
			{Name: "Dir/a", Typeflag: tar.TypeReg, Size: 1, Mode: 0o600},
			{Name: "dir/b", Typeflag: tar.TypeReg, Size: 1, Mode: 0o600},
		}},
		{"file used as directory", false, []tar.Header{
			{Name: "dir", Typeflag: tar.TypeReg, Size: 1, Mode: 0o600},
			{Name: "dir/b", Typeflag: tar.TypeReg, Size: 1, Mode: 0o600},
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			archivePath := writeTAR(t, testCase.compressed, testCase.entries)
			workspace, err := New(t.TempDir(), "linux", "amd64", "reject")
			if err != nil {
				t.Fatal(err)
			}
			cleanWorkspace(t, workspace)
			if err := workspace.ExtractSourceArchive(archivePath, ArchiveLimits{}); err == nil {
				t.Fatal("conflicting TAR paths were extracted")
			}
		})
	}
}
