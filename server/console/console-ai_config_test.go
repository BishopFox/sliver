package console

import (
	"testing"

	"github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/console/forms"
)

func TestCurrentAIConfigFormResultUsesConfiguredProviderFallback(t *testing.T) {
	serverConfig := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Anthropic: &configs.AIProviderConfig{APIKey: "anthropic-key", BaseURL: "https://api.anthropic.test"},
			OpenAI:    &configs.AIProviderConfig{},
		},
	}

	result := currentAIConfigFormResult(serverConfig)
	if result.Provider != ai.ProviderAnthropic {
		t.Fatalf("expected %q provider, got %q", ai.ProviderAnthropic, result.Provider)
	}
	if result.APIKey != "anthropic-key" {
		t.Fatalf("expected anthropic API key, got %q", result.APIKey)
	}
	if result.BaseURL != "https://api.anthropic.test" {
		t.Fatalf("expected anthropic base url, got %q", result.BaseURL)
	}
}

func TestApplyAIConfigFormResultUpdatesOnlySelectedProvider(t *testing.T) {
	serverConfig := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ai.ProviderAnthropic,
			Model:         "old-model",
			ThinkingLevel: "low",
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
			OpenAI:        &configs.AIProviderConfig{APIKey: "openai-key", BaseURL: "https://api.openai.test"},
		},
	}

	applyAIConfigFormResult(serverConfig, &forms.AIConfigFormResult{
		Provider:      ai.ProviderOpenAI,
		Model:         "gpt-test",
		ThinkingLevel: "high",
		APIKey:        "new-openai-key",
		BaseURL:       "https://override.openai.test",
	})

	if serverConfig.AI.Provider != ai.ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", ai.ProviderOpenAI, serverConfig.AI.Provider)
	}
	if serverConfig.AI.Model != "gpt-test" {
		t.Fatalf("expected model %q, got %q", "gpt-test", serverConfig.AI.Model)
	}
	if serverConfig.AI.ThinkingLevel != "high" {
		t.Fatalf("expected thinking level %q, got %q", "high", serverConfig.AI.ThinkingLevel)
	}
	if serverConfig.AI.OpenAI.APIKey != "new-openai-key" {
		t.Fatalf("expected updated openai api key, got %q", serverConfig.AI.OpenAI.APIKey)
	}
	if serverConfig.AI.OpenAI.BaseURL != "https://override.openai.test" {
		t.Fatalf("expected updated openai base url, got %q", serverConfig.AI.OpenAI.BaseURL)
	}
	if serverConfig.AI.Anthropic.APIKey != "anthropic-key" {
		t.Fatalf("expected anthropic api key to remain unchanged, got %q", serverConfig.AI.Anthropic.APIKey)
	}
}
