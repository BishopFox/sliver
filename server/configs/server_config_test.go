package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	aiPath := GetAIConfigPath()
	aiData := []byte(`provider: "openai"
thinking_level: "high"
max_output_tokens: 2048
temperature: 0.25
anthropic:
  api_key: "anthropic-key"
  base_url: "https://api.anthropic.example"
  use_bedrock: true
openai:
  models:
    - "gpt-test"
    - "gpt-mini-test"
  api_key: "openai-key"
  base_url: "https://api.openai.example"
  organization: "org-test"
  project: "proj-test"
  use_responses_api: true
google:
  api_key: "google-key"
  project: "vertex-project"
  location: "us-central1"
  skip_auth: true
openai_compat:
  base_url: "http://127.0.0.1:8080/v1"
openrouter:
  api_key: "openrouter-key"
  user_agent: "sliver-test/1.0"
`)
	if err := os.WriteFile(aiPath, aiData, 0600); err != nil {
		t.Fatalf("failed to write ai yaml config: %v", err)
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
	if config.AI == nil || config.AI.Anthropic == nil || config.AI.Anthropic.APIKey != "anthropic-key" {
		t.Fatalf("expected ai anthropic api key %q, got %#v", "anthropic-key", config.AI)
	}
	if config.AI == nil || config.AI.Anthropic == nil || config.AI.Anthropic.BaseURL != "https://api.anthropic.example" {
		t.Fatalf("expected ai anthropic base url %q, got %#v", "https://api.anthropic.example", config.AI)
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.APIKey != "openai-key" {
		t.Fatalf("expected ai openai api key %q, got %#v", "openai-key", config.AI)
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.BaseURL != "https://api.openai.example" {
		t.Fatalf("expected ai openai base url %q, got %#v", "https://api.openai.example", config.AI)
	}
	if config.AI == nil || config.AI.Provider != "openai" {
		t.Fatalf("expected ai provider %q, got %#v", "openai", config.AI)
	}
	if config.AI == nil || config.AI.Model != "" {
		t.Fatalf("expected legacy ai model field to remain empty, got %#v", config.AI)
	}
	if config.AI == nil || config.AI.OpenAI == nil || len(config.AI.OpenAI.Models) != 2 || config.AI.OpenAI.Models[0] != "gpt-test" || config.AI.OpenAI.Models[1] != "gpt-mini-test" {
		t.Fatalf("expected ai openai models to load, got %#v", config.AI)
	}
	if config.AI == nil || config.AI.ThinkingLevel != "high" {
		t.Fatalf("expected ai thinking level %q, got %#v", "high", config.AI)
	}
	if config.AI == nil || config.AI.MaxOutputTokens != 2048 {
		t.Fatalf("expected ai max_output_tokens %d, got %#v", 2048, config.AI)
	}
	if config.AI == nil || config.AI.Temperature == nil || *config.AI.Temperature != 0.25 {
		t.Fatalf("expected ai temperature %v, got %#v", 0.25, config.AI)
	}
	if config.AI == nil || config.AI.Anthropic == nil || !config.AI.Anthropic.UseBedrock {
		t.Fatalf("expected anthropic use_bedrock to load")
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.Organization != "org-test" {
		t.Fatalf("expected openai organization %q, got %#v", "org-test", config.AI)
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.Project != "proj-test" {
		t.Fatalf("expected openai project %q, got %#v", "proj-test", config.AI)
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.UseResponsesAPI == nil || !*config.AI.OpenAI.UseResponsesAPI {
		t.Fatalf("expected openai use_responses_api to load")
	}
	if config.AI == nil || config.AI.Google == nil || config.AI.Google.APIKey != "google-key" {
		t.Fatalf("expected google api key %q, got %#v", "google-key", config.AI)
	}
	if config.AI == nil || config.AI.Google == nil || config.AI.Google.Project != "vertex-project" || config.AI.Google.Location != "us-central1" || !config.AI.Google.SkipAuth {
		t.Fatalf("expected google vertex config to load, got %#v", config.AI)
	}
	if config.AI == nil || config.AI.OpenAICompat == nil || config.AI.OpenAICompat.BaseURL != "http://127.0.0.1:8080/v1" {
		t.Fatalf("expected openai_compat base url %q, got %#v", "http://127.0.0.1:8080/v1", config.AI)
	}
	if config.AI == nil || config.AI.OpenRouter == nil || config.AI.OpenRouter.APIKey != "openrouter-key" || config.AI.OpenRouter.UserAgent != "sliver-test/1.0" {
		t.Fatalf("expected openrouter config to load, got %#v", config.AI)
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
	if config.AI == nil || config.AI.Anthropic == nil || config.AI.Google == nil || config.AI.OpenAI == nil || config.AI.OpenAICompat == nil || config.AI.OpenRouter == nil {
		t.Fatalf("expected default ai config")
	}
	if config.AI.Provider != "" || config.AI.Model != "" || config.AI.ThinkingLevel != "" {
		t.Fatalf("expected empty default ai selections, got %#v", config.AI)
	}
	if config.AI.SystemPrompt != defaultAISystemPrompt {
		t.Fatalf("expected default ai system prompt %q, got %q", defaultAISystemPrompt, config.AI.SystemPrompt)
	}
	if _, err := os.Stat(GetServerConfigPath()); err != nil {
		t.Fatalf("expected default config file to exist: %v", err)
	}
	if _, err := os.Stat(GetAIConfigPath()); err != nil {
		t.Fatalf("expected default ai config file to exist: %v", err)
	}
	aiData, err := os.ReadFile(GetAIConfigPath())
	if err != nil {
		t.Fatalf("failed to read default ai config: %v", err)
	}
	if strings.Contains(string(aiData), "\nmodel:") || strings.HasPrefix(string(aiData), "model:") {
		t.Fatalf("expected default ai config to omit legacy model field, got %q", string(aiData))
	}
	data, err := os.ReadFile(GetServerConfigPath())
	if err != nil {
		t.Fatalf("failed to read default server config: %v", err)
	}
	if strings.Contains(string(data), "\nai:") || strings.HasPrefix(string(data), "ai:") {
		t.Fatalf("expected default server config to exclude ai block, got %q", string(data))
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
		AI           *AIConfig           `json:"ai"`
		Watchtower   *WatchTowerConfig   `json:"watch_tower"`
		GoProxy      string              `json:"go_proxy"`
		HTTPDefaults *legacyHTTPDefaults `json:"http_default"`
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
		AI: &AIConfig{
			Provider:        "anthropic",
			Model:           "claude-test",
			ThinkingLevel:   "medium",
			MaxOutputTokens: 1024,
			Anthropic:       &AIProviderConfig{APIKey: "anthropic-legacy", BaseURL: "https://legacy.anthropic.example", UseBedrock: true},
			Google:          &AIProviderConfig{Project: "legacy-project", Location: "europe-west1"},
			OpenAI:          &AIProviderConfig{APIKey: "openai-legacy", BaseURL: "https://legacy.openai.example", UseResponsesAPI: boolPtr(true)},
			OpenAICompat:    &AIProviderConfig{BaseURL: "http://127.0.0.1:9000/v1"},
			OpenRouter:      &AIProviderConfig{APIKey: "openrouter-legacy"},
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
		CC:  map[string]string{"linux/amd64": "/usr/bin/cc"},
		CXX: map[string]string{"linux/amd64": "/usr/bin/c++"},
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
	if config.AI == nil || config.AI.Anthropic == nil || config.AI.Anthropic.APIKey != "anthropic-legacy" {
		t.Fatalf("expected legacy ai anthropic config to load")
	}
	if config.AI == nil || config.AI.Anthropic == nil || config.AI.Anthropic.BaseURL != "https://legacy.anthropic.example" {
		t.Fatalf("expected legacy ai anthropic base url to load")
	}
	if config.AI == nil || config.AI.Anthropic == nil || !config.AI.Anthropic.UseBedrock {
		t.Fatalf("expected legacy ai anthropic bedrock flag to load")
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.APIKey != "openai-legacy" {
		t.Fatalf("expected legacy ai openai config to load")
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.BaseURL != "https://legacy.openai.example" {
		t.Fatalf("expected legacy ai openai base url to load")
	}
	if config.AI == nil || config.AI.OpenAI == nil || config.AI.OpenAI.UseResponsesAPI == nil || !*config.AI.OpenAI.UseResponsesAPI {
		t.Fatalf("expected legacy ai openai responses flag to load")
	}
	if config.AI == nil || config.AI.Google == nil || config.AI.Google.Project != "legacy-project" || config.AI.Google.Location != "europe-west1" {
		t.Fatalf("expected legacy ai google config to load")
	}
	if config.AI == nil || config.AI.OpenAICompat == nil || config.AI.OpenAICompat.BaseURL != "http://127.0.0.1:9000/v1" {
		t.Fatalf("expected legacy ai openai_compat config to load")
	}
	if config.AI == nil || config.AI.OpenRouter == nil || config.AI.OpenRouter.APIKey != "openrouter-legacy" {
		t.Fatalf("expected legacy ai openrouter config to load")
	}
	if config.AI == nil || config.AI.Provider != "anthropic" || config.AI.ThinkingLevel != "medium" || config.AI.MaxOutputTokens != 1024 {
		t.Fatalf("expected legacy ai selections to load, got %#v", config.AI)
	}
	if config.AI == nil || config.AI.Model != "" {
		t.Fatalf("expected legacy ai model field to be cleared, got %#v", config.AI)
	}
	if config.AI == nil || config.AI.Anthropic == nil || len(config.AI.Anthropic.Models) != 1 || config.AI.Anthropic.Models[0] != "claude-test" {
		t.Fatalf("expected legacy model to migrate into anthropic models, got %#v", config.AI)
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
	if _, err := os.Stat(GetAIConfigPath()); err != nil {
		t.Fatalf("expected migrated ai yaml config file to exist: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy config to be renamed: %v", err)
	}
	if _, err := os.Stat(legacyBackupPath(legacyPath)); err != nil {
		t.Fatalf("expected legacy backup file to exist: %v", err)
	}
	serverData, err := os.ReadFile(GetServerConfigPath())
	if err != nil {
		t.Fatalf("failed to read migrated server config: %v", err)
	}
	if strings.Contains(string(serverData), "\nai:") || strings.HasPrefix(string(serverData), "ai:") {
		t.Fatalf("expected migrated server config to exclude ai block, got %q", string(serverData))
	}
}

func boolPtr(value bool) *bool {
	return &value
}
