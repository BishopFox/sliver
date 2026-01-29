package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestCrackConfigParsesYAML(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configPath := getCrackConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	data := []byte(`AutoFire: false
MaxFileSize: 2048
ChunkSize: 1024
MaxDiskUsage: 4096
`)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("failed to write yaml config: %v", err)
	}

	config, err := LoadCrackConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.AutoFire {
		t.Fatalf("expected AutoFire false")
	}
	if config.MaxFileSize != 2048 {
		t.Fatalf("expected MaxFileSize %d, got %d", 2048, config.MaxFileSize)
	}
	if config.ChunkSize != 1024 {
		t.Fatalf("expected ChunkSize %d, got %d", 1024, config.ChunkSize)
	}
	if config.MaxDiskUsage != 4096 {
		t.Fatalf("expected MaxDiskUsage %d, got %d", 4096, config.MaxDiskUsage)
	}
}

func TestCrackConfigWritesDefault(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	config, err := LoadCrackConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config == nil {
		t.Fatalf("expected default crack config")
	}
	if _, err := os.Stat(getCrackConfigPath()); err != nil {
		t.Fatalf("expected default config file to exist: %v", err)
	}
}

func TestCrackConfigMigratesLegacyJSON(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configDir := filepath.Dir(getCrackConfigPath())
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	legacyPath := filepath.Join(configDir, crackLegacyConfigFileName)

	legacy := &clientpb.CrackConfig{
		AutoFire:     false,
		MaxFileSize:  4096,
		ChunkSize:    2048,
		MaxDiskUsage: 8192,
	}
	data, err := json.MarshalIndent(legacy, "", "    ")
	if err != nil {
		t.Fatalf("failed to marshal legacy config: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0600); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	config, err := LoadCrackConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if config.MaxFileSize != 4096 {
		t.Fatalf("expected MaxFileSize %d, got %d", 4096, config.MaxFileSize)
	}
	if _, err := os.Stat(getCrackConfigPath()); err != nil {
		t.Fatalf("expected migrated yaml config file to exist: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy config to be renamed: %v", err)
	}
	if _, err := os.Stat(legacyBackupPath(legacyPath)); err != nil {
		t.Fatalf("expected legacy backup file to exist: %v", err)
	}
}
