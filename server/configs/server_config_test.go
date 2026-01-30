package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServerConfigParsesYAML(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configPath := GetServerConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	data := []byte(`daemon_mode: true
daemon:
  host: "127.0.0.1"
  port: 4444
  tailscale: true
logs:
  level: 5
  grpc_unary_payloads: true
  grpc_stream_payloads: true
  tls_key_logger: true
watch_tower:
  vt_api_key: "vt"
  xforce_api_key: "xforce"
  xforce_api_password: "secret"
go_proxy: "https://proxy.example"
http_default:
  headers:
    - method: "POST"
      name: "X-Test"
      value: "abc"
      probability: 50
donut_bypass: 2
notifications:
  enabled: true
  events:
    - session-connected
  services:
    slack:
      enabled: true
      api_token: "slack-token"
      channels:
        - "C123"
  templates:
    session-connected:
      type: "text"
      template: "session.tmpl"
cc:
  linux/amd64: "/usr/bin/cc"
cxx:
  linux/amd64: "/usr/bin/c++"
`)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("failed to write yaml config: %v", err)
	}

	config := GetServerConfig()
	if !config.DaemonMode {
		t.Fatalf("expected daemon_mode true")
	}
	if config.DaemonConfig == nil || config.DaemonConfig.Port != 4444 {
		t.Fatalf("expected daemon port %d, got %v", 4444, config.DaemonConfig)
	}
	if config.Logs == nil || config.Logs.Level != 5 {
		t.Fatalf("expected logs level %d, got %v", 5, config.Logs)
	}
	if config.Logs == nil || !config.Logs.TLSKeyLogger {
		t.Fatalf("expected tls_key_logger true")
	}
	if config.Watchtower == nil || config.Watchtower.VTApiKey != "vt" {
		t.Fatalf("expected watch_tower vt_api_key %q, got %v", "vt", config.Watchtower)
	}
	if config.GoProxy != "https://proxy.example" {
		t.Fatalf("expected go_proxy %q, got %q", "https://proxy.example", config.GoProxy)
	}
	if config.HTTPDefaults == nil || len(config.HTTPDefaults.Headers) != 1 {
		t.Fatalf("expected 1 http default header, got %v", config.HTTPDefaults)
	}
	header := config.HTTPDefaults.Headers[0]
	if header.Method != "POST" || header.Name != "X-Test" || header.Value != "abc" || header.Probability != 50 {
		t.Fatalf("unexpected header values: %#v", header)
	}
	if config.DonutBypass != 2 {
		t.Fatalf("expected donut_bypass %d, got %d", 2, config.DonutBypass)
	}
	if config.Notifications == nil || !config.Notifications.Enabled {
		t.Fatalf("expected notifications enabled")
	}
	if len(config.Notifications.Events) != 1 || config.Notifications.Events[0] != "session-connected" {
		t.Fatalf("unexpected notifications events: %v", config.Notifications.Events)
	}
	if config.Notifications.Services == nil || config.Notifications.Services.Slack == nil {
		t.Fatalf("expected slack notifications config")
	}
	if !config.Notifications.Services.Slack.Enabled || config.Notifications.Services.Slack.APIToken != "slack-token" {
		t.Fatalf("unexpected slack notifications config: %#v", config.Notifications.Services.Slack)
	}
	if len(config.Notifications.Services.Slack.Channels) != 1 || config.Notifications.Services.Slack.Channels[0] != "C123" {
		t.Fatalf("unexpected slack channels: %v", config.Notifications.Services.Slack.Channels)
	}
	if config.Notifications.Templates == nil {
		t.Fatalf("expected notifications templates config")
	}
	tmpl := config.Notifications.Templates["session-connected"]
	if tmpl == nil || tmpl.Type != "text" || tmpl.Template != "session.tmpl" {
		t.Fatalf("unexpected template config: %#v", tmpl)
	}
	if config.CC["linux/amd64"] != "/usr/bin/cc" {
		t.Fatalf("expected cc override %q, got %q", "/usr/bin/cc", config.CC["linux/amd64"])
	}
	if config.CXX["linux/amd64"] != "/usr/bin/c++" {
		t.Fatalf("expected cxx override %q, got %q", "/usr/bin/c++", config.CXX["linux/amd64"])
	}
}

func TestServerConfigWritesDefault(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	config := GetServerConfig()
	if config.DaemonConfig == nil {
		t.Fatalf("expected default daemon config")
	}
	if _, err := os.Stat(GetServerConfigPath()); err != nil {
		t.Fatalf("expected default config file to exist: %v", err)
	}
}

func TestServerConfigMigratesLegacyJSON(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configDir := filepath.Dir(GetServerConfigPath())
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	legacyPath := filepath.Join(configDir, serverLegacyConfigFileName)

	type legacyHeader struct {
		ID                    string  `json:"id"`
		HttpC2ServerConfigID  *string `json:"httpc2serverconfigid"`
		HttpC2ImplantConfigID *string `json:"httpc2implantconfigid"`
		Method                string  `json:"method"`
		Name                  string  `json:"name"`
		Value                 string  `json:"value"`
		Probability           int32   `json:"probability"`
	}
	type legacyHTTPDefaults struct {
		Headers []legacyHeader `json:"headers"`
	}
	type legacyServerConfig struct {
		DaemonMode   bool                `json:"daemon_mode"`
		DaemonConfig *DaemonConfig       `json:"daemon"`
		Logs         *LogConfig          `json:"logs"`
		Watchtower   *WatchTowerConfig   `json:"watch_tower"`
		GoProxy      string              `json:"go_proxy"`
		HTTPDefaults *legacyHTTPDefaults `json:"http_default"`
		DonutBypass  int                 `json:"donut_bypass"`
		CC           map[string]string   `json:"cc"`
		CXX          map[string]string   `json:"cxx"`
	}
	legacy := &legacyServerConfig{
		DaemonMode: true,
		DaemonConfig: &DaemonConfig{
			Host:      "127.0.0.1",
			Port:      31338,
			Tailscale: false,
		},
		Logs: &LogConfig{
			Level:              4,
			GRPCUnaryPayloads:  true,
			GRPCStreamPayloads: true,
			TLSKeyLogger:       true,
		},
		Watchtower: &WatchTowerConfig{
			VTApiKey:          "vt-legacy",
			XForceApiKey:      "xforce-legacy",
			XForceApiPassword: "secret",
		},
		GoProxy: "https://proxy.legacy",
		HTTPDefaults: &legacyHTTPDefaults{
			Headers: []legacyHeader{
				{
					ID:          "00000000-0000-0000-0000-000000000000",
					Method:      "GET",
					Name:        "Cache-Control",
					Value:       "no-cache",
					Probability: 42,
				},
			},
		},
		DonutBypass: 2,
		CC:          map[string]string{"linux/amd64": "/usr/bin/cc"},
		CXX:         map[string]string{"linux/amd64": "/usr/bin/c++"},
	}
	data, err := json.MarshalIndent(legacy, "", "    ")
	if err != nil {
		t.Fatalf("failed to marshal legacy config: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0600); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	config := GetServerConfig()
	if !config.DaemonMode || config.DaemonConfig == nil || config.DaemonConfig.Port != 31338 {
		t.Fatalf("expected legacy daemon config to load")
	}
	if config.Watchtower == nil || config.Watchtower.VTApiKey != "vt-legacy" {
		t.Fatalf("expected legacy watch_tower to load")
	}
	if config.GoProxy != "https://proxy.legacy" {
		t.Fatalf("expected legacy go_proxy %q, got %q", "https://proxy.legacy", config.GoProxy)
	}
	if config.HTTPDefaults == nil || len(config.HTTPDefaults.Headers) != 1 {
		t.Fatalf("expected legacy http defaults to load")
	}
	if _, err := os.Stat(GetServerConfigPath()); err != nil {
		t.Fatalf("expected migrated yaml config file to exist: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy config to be renamed: %v", err)
	}
	if _, err := os.Stat(legacyBackupPath(legacyPath)); err != nil {
		t.Fatalf("expected legacy backup file to exist: %v", err)
	}
}
