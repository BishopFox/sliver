package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeImportConfig(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportCommandReportsInvalidConfig(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	path := writeImportConfig(t, "missing-operator.cfg", `{"lhost":"server"}`)

	cmd := importCmd()
	err := cmd.RunE(cmd, []string{path})
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
		t.Fatalf("import error = %v, want error identifying %q", err, path)
	}
}

func TestImportCommandReportsConfigWriteFailure(t *testing.T) {
	clientRoot := t.TempDir()
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", clientRoot)
	path := writeImportConfig(t, "alice.cfg", `{"operator":"alice","lhost":"server"}`)

	configDir := filepath.Join(clientRoot, "configs")
	if err := os.MkdirAll(filepath.Join(configDir, "alice_server.cfg"), 0o700); err != nil {
		t.Fatal(err)
	}

	cmd := importCmd()
	err := cmd.RunE(cmd, []string{path})
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
		t.Fatalf("import error = %v, want write error identifying %q", err, path)
	}
}

func TestImportCommandStopsAfterFirstFailure(t *testing.T) {
	clientRoot := t.TempDir()
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", clientRoot)
	invalidPath := writeImportConfig(t, "invalid.cfg", `{"lhost":"server"}`)
	validPath := writeImportConfig(t, "valid.cfg", `{"operator":"alice","lhost":"server"}`)

	cmd := importCmd()
	err := cmd.RunE(cmd, []string{invalidPath, validPath})
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(invalidPath)) {
		t.Fatalf("import error = %v, want first file %q", err, invalidPath)
	}
	if _, err := os.Stat(filepath.Join(clientRoot, "configs", "alice_server.cfg")); !os.IsNotExist(err) {
		t.Fatalf("later config was imported after first failure: stat error = %v", err)
	}
}

func TestImportCommandRequiresPath(t *testing.T) {
	cmd := importCmd()
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("import without a path succeeded")
	}
}
