package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIConfigParsesYAML(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	configPath := GetAIConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	data := []byte(`provider: "openai"
model: "gpt-test"
thinking_level: "high"
system_prompt: "Stay concise."
max_output_tokens: 2048
temperature: 0.25
anthropic:
  api_key: "anthropic-key"
  base_url: "https://api.anthropic.example"
  use_bedrock: true
openai:
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
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("failed to write yaml config: %v", err)
	}

	config := GetAIConfig()
	if config.Provider != "openai" {
		t.Fatalf("expected provider %q, got %q", "openai", config.Provider)
	}
	if config.Model != "gpt-test" {
		t.Fatalf("expected model %q, got %q", "gpt-test", config.Model)
	}
	if config.ThinkingLevel != "high" {
		t.Fatalf("expected thinking level %q, got %q", "high", config.ThinkingLevel)
	}
	if config.SystemPrompt != "Stay concise." {
		t.Fatalf("expected system prompt %q, got %q", "Stay concise.", config.SystemPrompt)
	}
	if config.MaxOutputTokens != 2048 {
		t.Fatalf("expected max_output_tokens %d, got %d", 2048, config.MaxOutputTokens)
	}
	if config.Temperature == nil || *config.Temperature != 0.25 {
		t.Fatalf("expected temperature %v, got %#v", 0.25, config.Temperature)
	}
	if config.Anthropic == nil || config.Anthropic.APIKey != "anthropic-key" || !config.Anthropic.UseBedrock {
		t.Fatalf("expected anthropic config to load, got %#v", config.Anthropic)
	}
	if config.OpenAI == nil || config.OpenAI.APIKey != "openai-key" || config.OpenAI.Organization != "org-test" || config.OpenAI.Project != "proj-test" {
		t.Fatalf("expected openai config to load, got %#v", config.OpenAI)
	}
	if config.OpenAI == nil || config.OpenAI.UseResponsesAPI == nil || !*config.OpenAI.UseResponsesAPI {
		t.Fatalf("expected openai responses api flag to load")
	}
	if config.Google == nil || config.Google.APIKey != "google-key" || config.Google.Project != "vertex-project" || config.Google.Location != "us-central1" || !config.Google.SkipAuth {
		t.Fatalf("expected google config to load, got %#v", config.Google)
	}
	if config.OpenAICompat == nil || config.OpenAICompat.BaseURL != "http://127.0.0.1:8080/v1" {
		t.Fatalf("expected openai_compat config to load, got %#v", config.OpenAICompat)
	}
	if config.OpenRouter == nil || config.OpenRouter.APIKey != "openrouter-key" || config.OpenRouter.UserAgent != "sliver-test/1.0" {
		t.Fatalf("expected openrouter config to load, got %#v", config.OpenRouter)
	}
}

func TestAIConfigWritesDefault(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	config := GetAIConfig()
	if config.Provider != "" || config.Model != "" || config.ThinkingLevel != "" {
		t.Fatalf("expected empty default ai selections, got %#v", config)
	}
	if config.SystemPrompt != defaultAISystemPrompt {
		t.Fatalf("expected default system prompt %q, got %q", defaultAISystemPrompt, config.SystemPrompt)
	}
	if config.Anthropic == nil || config.Google == nil || config.OpenAI == nil || config.OpenAICompat == nil || config.OpenRouter == nil {
		t.Fatalf("expected default provider configs to exist, got %#v", config)
	}
	if _, err := os.Stat(GetAIConfigPath()); err != nil {
		t.Fatalf("expected default ai config file to exist: %v", err)
	}
	data, err := os.ReadFile(GetAIConfigPath())
	if err != nil {
		t.Fatalf("failed to read default ai config: %v", err)
	}
	if !strings.Contains(string(data), "system_prompt:") {
		t.Fatalf("expected default ai config to include system_prompt field, got %q", string(data))
	}
	if !strings.Contains(string(data), "authorized security testing") {
		t.Fatalf("expected default ai config to include the default system prompt, got %q", string(data))
	}
}

func TestServerConfigMigratesEmbeddedAIConfigFromYAML(t *testing.T) {
	t.Setenv("SLIVER_ROOT_DIR", t.TempDir())

	serverConfigPath := GetServerConfigPath()
	if err := os.MkdirAll(filepath.Dir(serverConfigPath), 0700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	data := []byte(`daemon_mode: true
daemon:
  host: "127.0.0.1"
  port: 4444
ai:
  provider: "openai"
  model: "gpt-test"
  system_prompt: "Stay concise."
  openai:
    api_key: "openai-key"
`)
	if err := os.WriteFile(serverConfigPath, data, 0600); err != nil {
		t.Fatalf("failed to write yaml config: %v", err)
	}

	config := GetServerConfig()
	if !config.DaemonMode || config.DaemonConfig == nil || config.DaemonConfig.Port != 4444 {
		t.Fatalf("expected daemon settings to load, got %#v", config.DaemonConfig)
	}
	if config.AI == nil || config.AI.Provider != "openai" || config.AI.Model != "gpt-test" || config.AI.SystemPrompt != "Stay concise." {
		t.Fatalf("expected embedded AI config to migrate, got %#v", config.AI)
	}
	if config.AI.OpenAI == nil || config.AI.OpenAI.APIKey != "openai-key" {
		t.Fatalf("expected embedded openai api key to migrate, got %#v", config.AI.OpenAI)
	}
	if _, err := os.Stat(GetAIConfigPath()); err != nil {
		t.Fatalf("expected migrated ai config file to exist: %v", err)
	}

	serverData, err := os.ReadFile(GetServerConfigPath())
	if err != nil {
		t.Fatalf("failed to read migrated server config: %v", err)
	}
	if strings.Contains(string(serverData), "\nai:") || strings.HasPrefix(string(serverData), "ai:") {
		t.Fatalf("expected migrated server config to drop embedded ai block, got %q", string(serverData))
	}
}
