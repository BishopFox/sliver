package configs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestConfigSaveReportsWriteFailure(t *testing.T) {
	tests := []struct {
		name string
		path func() string
		save func() error
	}{
		{
			name: "database",
			path: GetDatabaseConfigPath,
			save: func() error { return (&DatabaseConfig{Dialect: Sqlite}).Save() },
		},
		{
			name: "server",
			path: GetServerConfigPath,
			save: func() error { return (&ServerConfig{}).Save() },
		},
		{
			name: "crack",
			path: getCrackConfigPath,
			save: func() error { return SaveCrackConfig(&clientpb.CrackConfig{}) },
		},
		{
			name: "ai",
			path: GetAIConfigPath,
			save: func() error { return (&AIConfig{}).Save() },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
			path := test.path()
			if err := os.MkdirAll(path, 0700); err != nil {
				t.Fatalf("create directory at config path: %v", err)
			}
			if err := test.save(); err == nil {
				t.Fatal("save succeeded despite config path being a directory")
			}
		})
	}
}

func TestAtomicWriteConfigPreservesPreviousFileOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	previous := []byte("value: previous\n")
	if err := os.WriteFile(path, previous, 0644); err != nil {
		t.Fatal(err)
	}

	oldRename := renameConfigFile
	t.Cleanup(func() { renameConfigFile = oldRename })
	injectedErr := errors.New("injected rename failure")
	renameConfigFile = func(_, newPath string) error {
		if newPath != path {
			t.Fatalf("rename targeted %q; want %q", newPath, path)
		}
		return injectedErr
	}
	if err := atomicWriteConfig(path, []byte("value: replacement\n")); !errors.Is(err, injectedErr) {
		t.Fatalf("save error = %v; want injected rename error", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, previous) {
		t.Fatalf("failed save changed previous config: %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("failed save left temporary files: %v", entries)
	}

	renameConfigFile = oldRename
	replacement := []byte("value: replacement\n")
	if err := atomicWriteConfig(path, replacement); err != nil {
		t.Fatalf("save after rename recovery: %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, replacement) {
		t.Fatalf("saved config = %q; want %q", got, replacement)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("saved config mode = %04o; want 0600", got)
	}
}

func TestAtomicWriteConfigPreservesExistingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yaml")
	link := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(target, []byte("old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(target), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	want := []byte("new\n")
	if err := atomicWriteConfig(link, want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("save replaced existing symlink")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("symlink target = %q; want %q", got, want)
	}
}

func TestLegacyConfigNotRetiredAfterSaveFailure(t *testing.T) {
	tests := []struct {
		name       string
		path       func() string
		legacyPath func() string
		legacyJSON []byte
		load       func(*testing.T)
	}{
		{
			name:       "database",
			path:       GetDatabaseConfigPath,
			legacyPath: getDatabaseLegacyConfigPath,
			legacyJSON: []byte(`{"dialect":"postgresql","database":"legacy-db","max_idle_conns":1,"max_open_conns":1}`),
			load: func(t *testing.T) {
				if got := GetDatabaseConfig().Database; got != "legacy-db" {
					t.Fatalf("loaded database = %q; want legacy-db", got)
				}
			},
		},
		{
			name:       "server",
			path:       GetServerConfigPath,
			legacyPath: getServerLegacyConfigPath,
			legacyJSON: []byte(`{"daemon":{"host":"127.0.0.1","port":31337},"logs":{"level":3},"go_proxy":"http://legacy.example"}`),
			load: func(t *testing.T) {
				if got := GetServerConfig().GoProxy; got != "http://legacy.example" {
					t.Fatalf("loaded proxy = %q; want legacy value", got)
				}
			},
		},
		{
			name:       "crack",
			path:       getCrackConfigPath,
			legacyPath: getCrackLegacyConfigPath,
			legacyJSON: []byte(`{"AutoFire":true,"MaxFileSize":4096,"ChunkSize":2048,"MaxDiskUsage":8192}`),
			load: func(t *testing.T) {
				config, err := LoadCrackConfig()
				if err == nil {
					t.Fatal("load did not report failed save")
				}
				if config.MaxFileSize != 4096 {
					t.Fatalf("loaded max file size = %d; want 4096", config.MaxFileSize)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SLIVER_ROOT_DIR", t.TempDir())
			path := test.path()
			legacyPath := test.legacyPath()
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(legacyPath, test.legacyJSON, 0600); err != nil {
				t.Fatal(err)
			}

			oldRename := renameConfigFile
			t.Cleanup(func() { renameConfigFile = oldRename })
			injectedErr := errors.New("injected config rename failure")
			attempted := false
			renameConfigFile = func(oldPath, newPath string) error {
				if newPath == path {
					attempted = true
					return injectedErr
				}
				return oldRename(oldPath, newPath)
			}
			test.load(t)
			if !attempted {
				t.Fatal("config save did not attempt to replace YAML")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed save created YAML config: %v", err)
			}
			got, err := os.ReadFile(legacyPath)
			if err != nil {
				t.Fatalf("failed save removed legacy config: %v", err)
			}
			if !bytes.Equal(got, test.legacyJSON) {
				t.Fatalf("failed save changed legacy config: %q", got)
			}
			if _, err := os.Stat(legacyBackupPath(legacyPath)); !os.IsNotExist(err) {
				t.Fatalf("failed save retired legacy config: %v", err)
			}
		})
	}
}
