package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsParsesYAML(t *testing.T) {
	t.Setenv(envVarName, t.TempDir())

	rootDir, _ := filepath.Abs(GetRootAppDir())
	settingsPath := filepath.Join(rootDir, settingsFileName)

	data := []byte(`tables: "Compact"
autoadult: true
beacon_autoresults: false
small_term_width: 120
always_overflow: true
vim_mode: true
user_connect: true
console_logs: false
prompt: "basic"
prompt_template: "{{.Host}}"
`)
	if err := os.WriteFile(settingsPath, data, 0o600); err != nil {
		t.Fatalf("failed to write yaml settings: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if settings.TableStyle != "Compact" {
		t.Fatalf("expected TableStyle %q, got %q", "Compact", settings.TableStyle)
	}
	if !settings.AutoAdult {
		t.Fatalf("expected AutoAdult true")
	}
	if settings.BeaconAutoResults {
		t.Fatalf("expected BeaconAutoResults false")
	}
	if settings.SmallTermWidth != 120 {
		t.Fatalf("expected SmallTermWidth %d, got %d", 120, settings.SmallTermWidth)
	}
	if !settings.AlwaysOverflow {
		t.Fatalf("expected AlwaysOverflow true")
	}
	if !settings.VimMode {
		t.Fatalf("expected VimMode true")
	}
	if !settings.UserConnect {
		t.Fatalf("expected UserConnect true")
	}
	if settings.ConsoleLogs {
		t.Fatalf("expected ConsoleLogs false")
	}
	if settings.PromptStyle != PromptStyleBasic {
		t.Fatalf("expected PromptStyle %q, got %q", PromptStyleBasic, settings.PromptStyle)
	}
	if settings.PromptTemplate != "{{.Host}}" {
		t.Fatalf("expected PromptTemplate %q, got %q", "{{.Host}}", settings.PromptTemplate)
	}
}

func TestLoadSettingsMigratesLegacyJSON(t *testing.T) {
	t.Setenv(envVarName, t.TempDir())

	rootDir, _ := filepath.Abs(GetRootAppDir())
	legacyPath := filepath.Join(rootDir, settingsLegacyFileName)

	legacy := &ClientSettings{
		TableStyle:        "Legacy",
		AutoAdult:         true,
		BeaconAutoResults: false,
		SmallTermWidth:    200,
		AlwaysOverflow:    true,
		VimMode:           true,
		UserConnect:       true,
		ConsoleLogs:       false,
		PromptStyle:       PromptStyleOperatorHost,
		PromptTemplate:    "{{.Operator}}@{{.Host}} sliver > ",
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal legacy settings: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0o600); err != nil {
		t.Fatalf("failed to write legacy settings: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if settings.TableStyle != "Legacy" {
		t.Fatalf("expected TableStyle %q, got %q", "Legacy", settings.TableStyle)
	}
	if settings.PromptStyle != PromptStyleOperatorHost {
		t.Fatalf("expected PromptStyle %q, got %q", PromptStyleOperatorHost, settings.PromptStyle)
	}
	if settings.PromptTemplate != "{{.Operator}}@{{.Host}} sliver > " {
		t.Fatalf("expected PromptTemplate %q, got %q", "{{.Operator}}@{{.Host}} sliver > ", settings.PromptTemplate)
	}
	if _, err := os.Stat(filepath.Join(rootDir, settingsFileName)); err != nil {
		t.Fatalf("expected yaml settings to exist: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy settings to be renamed: %v", err)
	}
	if _, err := os.Stat(legacyBackupPath(legacyPath)); err != nil {
		t.Fatalf("expected legacy backup to exist: %v", err)
	}
}

func TestLoadSettingsWritesDefault(t *testing.T) {
	t.Setenv(envVarName, t.TempDir())

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if settings.TableStyle != "SliverDefault" {
		t.Fatalf("expected default TableStyle %q, got %q", "SliverDefault", settings.TableStyle)
	}
	if settings.PromptStyle != PromptStyleHost {
		t.Fatalf("expected default PromptStyle %q, got %q", PromptStyleHost, settings.PromptStyle)
	}
	if settings.PromptTemplate != DefaultPromptTemplate {
		t.Fatalf("expected default PromptTemplate %q, got %q", DefaultPromptTemplate, settings.PromptTemplate)
	}
	rootDir, _ := filepath.Abs(GetRootAppDir())
	if _, err := os.Stat(filepath.Join(rootDir, settingsFileName)); err != nil {
		t.Fatalf("expected default settings to be written: %v", err)
	}
}
