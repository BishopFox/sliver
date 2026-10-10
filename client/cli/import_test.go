package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	if got := commandExitCode(err); got != 1 {
		t.Fatalf("invalid config exit code = %d, want 1", got)
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
	if got := commandExitCode(err); got != 1 {
		t.Fatalf("config write failure exit code = %d, want 1", got)
	}
	if _, ok := errors.AsType[*os.PathError](err); !ok {
		t.Fatalf("config write failure does not wrap the filesystem error: %v", err)
	}
}

func TestImportCommandReadFailureExitCode(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "missing.cfg")

	cmd := importCmd()
	err := cmd.RunE(cmd, []string{path})
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
		t.Fatalf("import error = %v, want read error identifying %q", err, path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("import error does not wrap the missing file error: %v", err)
	}
	if got := commandExitCode(fmt.Errorf("command failed: %w", err)); got != 3 {
		t.Fatalf("wrapped read failure exit code = %d, want 3", got)
	}
}

func TestImportCommandParseFailureExitCode(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	path := writeImportConfig(t, "malformed.cfg", `{"operator":`)

	cmd := importCmd()
	err := cmd.RunE(cmd, []string{path})
	if err == nil || !strings.Contains(err.Error(), strconv.Quote(path)) {
		t.Fatalf("import error = %v, want parse error identifying %q", err, path)
	}
	if _, ok := errors.AsType[*json.SyntaxError](err); !ok {
		t.Fatalf("import error does not wrap the JSON syntax error: %v", err)
	}
	if got := commandExitCode(err); got != 3 {
		t.Fatalf("parse failure exit code = %d, want 3", got)
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

func TestImportCommandMissingPathPreservesSuccess(t *testing.T) {
	cmd := importCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("import without a path failed: %v", err)
	}
	if got := commandExitCode(err); got != 0 {
		t.Fatalf("missing path exit code = %d, want 0", got)
	}
	if got, want := output.String(), "Missing config file path, see --help"; got != want {
		t.Fatalf("missing path output = %q, want %q", got, want)
	}
}
