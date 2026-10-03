package assets

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateWasmOverlaySourceHashes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	relativePath := "src/net/net_fake.go"
	content := []byte("Go source\n")
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	if err := validateWasmOverlaySourceHashes(root, map[string]string{relativePath: expected}); err != nil {
		t.Fatalf("validateWasmOverlaySourceHashes() error = %v", err)
	}

	err := validateWasmOverlaySourceHashes(root, map[string]string{relativePath: strings.Repeat("0", 64)})
	if err == nil {
		t.Fatal("validateWasmOverlaySourceHashes() succeeded with wrong hash")
	}
	for _, fragment := range []string{goVersion, relativePath, expected, strings.Repeat("0", 64)} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("validation error %q does not contain %q", err, fragment)
		}
	}

	err = validateWasmOverlaySourceHashes(root, map[string]string{"src/net/missing.go": expected})
	if err == nil || !strings.Contains(err.Error(), "missing.go") {
		t.Fatalf("missing source error = %v", err)
	}
}

func TestVerifyWasmGoWrapperArchive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		goos  string
		magic []byte
		mode  os.FileMode
	}{
		{name: "darwin", goos: "darwin", magic: []byte{0xcf, 0xfa, 0xed, 0xfe}, mode: 0o755},
		{name: "linux", goos: "linux", magic: []byte("\x7fELF"), mode: 0o755},
		{name: "windows", goos: "windows", magic: []byte("MZ\x90\x00"), mode: 0o644},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "go.zip")
			writeWrapperArchive(t, path, test.goos, test.magic, test.mode)
			if err := verifyWasmGoWrapperArchive(path, goPlatform{os: test.goos}); err != nil {
				t.Fatalf("verifyWasmGoWrapperArchive() error = %v", err)
			}
		})
	}
}

func TestVerifyWasmGoWrapperArchiveRejectsInvalidEntries(t *testing.T) {
	t.Parallel()

	t.Run("missing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "go.zip")
		archive, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(archive)
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		if err := verifyWasmGoWrapperArchive(path, goPlatform{os: "linux"}); err == nil {
			t.Fatal("missing wrapper was accepted")
		}
	})

	t.Run("not executable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "go.zip")
		writeWrapperArchive(t, path, "linux", []byte("\x7fELF"), 0o644)
		if err := verifyWasmGoWrapperArchive(path, goPlatform{os: "linux"}); err == nil {
			t.Fatal("non-executable wrapper was accepted")
		}
	})

	t.Run("wrong magic", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "go.zip")
		writeWrapperArchive(t, path, "darwin", []byte("nope"), 0o755)
		if err := verifyWasmGoWrapperArchive(path, goPlatform{os: "darwin"}); err == nil {
			t.Fatal("invalid wrapper magic was accepted")
		}
	})
}

func TestWasmGoWrapperName(t *testing.T) {
	t.Parallel()

	if got := wasmGoWrapperName("windows"); got != "sliver-wasm-go.exe" {
		t.Errorf("Windows wrapper name = %q", got)
	}
	if got := wasmGoWrapperName("linux"); got != "sliver-wasm-go" {
		t.Errorf("Linux wrapper name = %q", got)
	}
}

func TestWasmGoWrapperMode(t *testing.T) {
	t.Parallel()

	if got := wasmGoWrapperMode("windows"); got != 0o644 {
		t.Errorf("Windows wrapper mode = %v, want 0644", got)
	}
	if got := wasmGoWrapperMode("linux"); got != 0o755 {
		t.Errorf("Linux wrapper mode = %v, want 0755", got)
	}
}

func TestWasmGoWrapperArchiveAppliesTargetMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	wrapperPath := filepath.Join(root, "go", "bin", wasmGoWrapperName("darwin"))
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapperPath, []byte{0xcf, 0xfa, 0xed, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "go.zip")
	entryPath := "go/bin/" + wasmGoWrapperName("darwin")
	if err := zipDirWithModeOverrides(root, "go", archivePath, map[string]os.FileMode{
		entryPath: wasmGoWrapperMode("darwin"),
	}); err != nil {
		t.Fatal(err)
	}

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := archive.Close(); err != nil {
			t.Errorf("close wrapper archive: %v", err)
		}
	})
	for _, file := range archive.File {
		if file.Name == entryPath {
			if got := file.Mode().Perm(); got != 0o755 {
				t.Fatalf("wrapper mode = %v, want 0755", got)
			}
			if err := verifyWasmGoWrapperArchive(archivePath, goPlatform{os: "darwin"}); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("%s is missing from archive", entryPath)
}

func TestBuildWasmGoWrapper(t *testing.T) {
	goRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(goRoot, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	platform := goPlatform{os: runtime.GOOS, arch: runtime.GOARCH}
	if err := buildWasmGoWrapper(goRoot, platform); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(goRoot, "bin", wasmGoWrapperName(platform.os))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 || !validExecutableMagic(platform.os, data[:4]) {
		t.Fatalf("wrapper has invalid %s executable magic: %x", platform.os, data[:min(len(data), 4)])
	}
	if platform.os != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("wrapper mode = %v, want executable", info.Mode())
		}
	}
}

func TestOverrideEnvironment(t *testing.T) {
	t.Parallel()

	environment := overrideEnvironment(
		[]string{"GOOS=old", "goarch=old", "PATH=/bin"},
		"GOOS=wasip1",
		"GOARCH=wasm",
	)
	assertEnvironmentValue(t, environment, "GOOS", "wasip1")
	assertEnvironmentValue(t, environment, "GOARCH", "wasm")
	assertEnvironmentValue(t, environment, "PATH", "/bin")
}

func assertEnvironmentValue(t *testing.T, environment []string, name, expected string) {
	t.Helper()
	var values []string
	for _, entry := range environment {
		key, value, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, name) {
			values = append(values, value)
		}
	}
	if len(values) != 1 || values[0] != expected {
		t.Fatalf("%s values = %q, want [%q]", name, values, expected)
	}
}

func writeWrapperArchive(t *testing.T, path, goos string, content []byte, mode os.FileMode) {
	t.Helper()
	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	header := &zip.FileHeader{Name: "go/bin/" + wasmGoWrapperName(goos)}
	header.SetMode(mode)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}
