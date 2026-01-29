package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDatabaseConfigParsesYAML(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configPath := GetDatabaseConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	data := []byte(`dialect: mysql
database: "sliver"
username: "tester"
password: "secret"
host: "127.0.0.1"
port: 3306
params:
  charset: "utf8mb4"
pragmas: {}
max_idle_conns: 7
max_open_conns: 15
log_level: "info"
`)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("failed to write yaml config: %v", err)
	}

	config := GetDatabaseConfig()
	if config.Dialect != MySQL {
		t.Fatalf("expected dialect %q, got %q", MySQL, config.Dialect)
	}
	if config.Database != "sliver" {
		t.Fatalf("expected database %q, got %q", "sliver", config.Database)
	}
	if config.Username != "tester" {
		t.Fatalf("expected username %q, got %q", "tester", config.Username)
	}
	if config.Password != "secret" {
		t.Fatalf("expected password %q, got %q", "secret", config.Password)
	}
	if config.Host != "127.0.0.1" {
		t.Fatalf("expected host %q, got %q", "127.0.0.1", config.Host)
	}
	if config.Port != 3306 {
		t.Fatalf("expected port %d, got %d", 3306, config.Port)
	}
	if config.Params["charset"] != "utf8mb4" {
		t.Fatalf("expected charset param %q, got %q", "utf8mb4", config.Params["charset"])
	}
	if config.MaxIdleConns != 7 {
		t.Fatalf("expected max_idle_conns %d, got %d", 7, config.MaxIdleConns)
	}
	if config.MaxOpenConns != 15 {
		t.Fatalf("expected max_open_conns %d, got %d", 15, config.MaxOpenConns)
	}
	if config.LogLevel != "info" {
		t.Fatalf("expected log_level %q, got %q", "info", config.LogLevel)
	}
}

func TestDatabaseConfigWritesDefault(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	config := GetDatabaseConfig()
	if config.Dialect != Sqlite {
		t.Fatalf("expected default dialect %q, got %q", Sqlite, config.Dialect)
	}
	if _, err := os.Stat(GetDatabaseConfigPath()); err != nil {
		t.Fatalf("expected default config file to exist: %v", err)
	}
}

func TestDatabaseConfigMigratesLegacyJSON(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configDir := filepath.Dir(GetDatabaseConfigPath())
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	legacyPath := filepath.Join(configDir, databaseLegacyConfigFileName)

	legacy := &DatabaseConfig{
		Dialect:      Postgres,
		Database:     "sliver",
		Username:     "legacy",
		Password:     "pass",
		Host:         "db.internal",
		Port:         5432,
		Params:       map[string]string{"sslmode": "disable"},
		Pragmas:      map[string]string{},
		MaxIdleConns: 4,
		MaxOpenConns: 9,
		LogLevel:     "debug",
	}
	data, err := json.MarshalIndent(legacy, "", "    ")
	if err != nil {
		t.Fatalf("failed to marshal legacy config: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0600); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	config := GetDatabaseConfig()
	if config.Dialect != Postgres {
		t.Fatalf("expected dialect %q, got %q", Postgres, config.Dialect)
	}
	if config.Database != "sliver" {
		t.Fatalf("expected database %q, got %q", "sliver", config.Database)
	}
	if config.Params["sslmode"] != "disable" {
		t.Fatalf("expected sslmode param %q, got %q", "disable", config.Params["sslmode"])
	}
	if _, err := os.Stat(GetDatabaseConfigPath()); err != nil {
		t.Fatalf("expected migrated yaml config file to exist: %v", err)
	}
}
