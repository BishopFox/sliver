package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLocationPaths(t *testing.T) {
	root := t.TempDir()

	newDirectory := filepath.Join(root, "new", "payloads")
	path, err := saveLocation(newDirectory+"/", "stage.bin", nil)
	if err != nil {
		t.Fatalf("new directory: %v", err)
	}
	if want := filepath.Join(newDirectory, "stage.bin"); path != want {
		t.Fatalf("new directory path = %q, want %q", path, want)
	}
	if info, err := os.Stat(newDirectory); err != nil || !info.IsDir() {
		t.Fatalf("new directory not created: info=%v err=%v", info, err)
	}
	nativeDirectory := filepath.Join(root, "native")
	path, err = saveLocation(nativeDirectory+string(os.PathSeparator), "native.bin", nil)
	if err != nil || path != filepath.Join(nativeDirectory, "native.bin") {
		t.Fatalf("native separator directory path = %q, err = %v", path, err)
	}

	explicitFile := filepath.Join(root, "missing", "parent", "payload.bin")
	path, err = saveLocation(explicitFile, "ignored.bin", nil)
	if err != nil {
		t.Fatalf("missing parent: %v", err)
	}
	if path != explicitFile {
		t.Fatalf("explicit file path = %q, want %q", path, explicitFile)
	}
	if info, err := os.Stat(filepath.Dir(explicitFile)); err != nil || !info.IsDir() {
		t.Fatalf("parent not created: info=%v err=%v", info, err)
	}

	if _, err := saveLocation(filepath.Join(root, "invalid\x00name"), "ignored.bin", nil); err == nil {
		t.Fatal("non-NotExist stat error accepted")
	}
}

func TestSaveLocationRejectsUnsafeArtifactName(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"../escape", `..\escape`, "CON", "payload\nnext"} {
		if _, err := saveLocation(directory, name, nil); err == nil {
			t.Fatalf("unsafe artifact filename %q was accepted", name)
		}
	}
}
