package generate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildScriptWorkspace(t *testing.T, manifests ...string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "magefiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range manifests {
		if err := os.WriteFile(filepath.Join(root, "inputs", name), []byte(`{"version":1}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "inputs", "sliver-material.json"), []byte("private material sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestWriteRetainedBuildScript(t *testing.T) {
	for _, tc := range []struct {
		name, hostOS, script string
	}{
		{name: "unix", hostOS: "darwin", script: "build.sh"},
		{name: "windows", hostOS: "windows", script: "build.ps1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := buildScriptWorkspace(t, "go-build-123456.json")
			if err := writeRetainedBuildScript(root, tc.hostOS); err != nil {
				t.Fatal(err)
			}
			scriptPath := filepath.Join(root, tc.script)
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"mage", "-d", "-w", "build", "magefiles", "go-build-123456.json"} {
				if !strings.Contains(string(content), want) {
					t.Fatalf("%s does not contain %q", tc.script, want)
				}
			}
			if strings.Contains(string(content), "private material sentinel") {
				t.Fatal("human build script contains private material")
			}
			assertRetainedBuildScriptMode(t, scriptPath, tc.script)
			other := "build.sh"
			if tc.script == other {
				other = "build.ps1"
			}
			if _, err := os.Lstat(filepath.Join(root, other)); !os.IsNotExist(err) {
				t.Fatalf("unexpected second build script: %v", err)
			}
		})
	}
}

func assertRetainedBuildScriptMode(t *testing.T, path, script string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("script is not a regular file: %v", info.Mode())
	}
	if runtime.GOOS == "windows" {
		return
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 || mode&0o400 == 0 {
		t.Fatalf("script permissions %o expose or hide it", mode)
	}
	if script == "build.sh" && mode&0o100 == 0 {
		t.Fatalf("Unix build script is not executable: %o", mode)
	}
}

func TestWriteRetainedBuildScriptRejectsInvalidManifest(t *testing.T) {
	for _, tc := range []struct {
		name      string
		manifests []string
	}{
		{name: "missing"},
		{name: "ambiguous", manifests: []string{"go-build-1.json", "go-build-2.json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := buildScriptWorkspace(t, tc.manifests...)
			if err := writeRetainedBuildScript(root, "darwin"); err == nil {
				t.Fatal("invalid manifest selection was accepted")
			}
			if _, err := os.Lstat(filepath.Join(root, "build.sh")); !os.IsNotExist(err) {
				t.Fatalf("invalid manifest produced a script: %v", err)
			}
		})
	}
	t.Run("symlink", func(t *testing.T) {
		root := buildScriptWorkspace(t)
		outside := filepath.Join(t.TempDir(), "outside.json")
		if err := os.WriteFile(outside, []byte(`{"version":1}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(root, "inputs", "go-build-1.json")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if err := writeRetainedBuildScript(root, "darwin"); err == nil {
			t.Fatal("symlink manifest was accepted")
		}
		if _, err := os.Lstat(filepath.Join(root, "build.sh")); !os.IsNotExist(err) {
			t.Fatalf("symlink manifest produced a script: %v", err)
		}
	})
}

func TestWriteRetainedBuildScriptDoesNotOverwrite(t *testing.T) {
	root := buildScriptWorkspace(t, "go-build-1.json")
	path := filepath.Join(root, "build.sh")
	want := []byte("keep existing script")
	if err := os.WriteFile(path, want, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeRetainedBuildScript(root, "darwin"); err == nil {
		t.Fatal("existing script was overwritten without an error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("existing script changed: %q", got)
	}
}
