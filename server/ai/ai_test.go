package ai

import (
	"testing"

	"github.com/bishopfox/sliver/server/configs"
)

func TestSafeConfigSummaryFromConfigUsesExplicitConfiguredProvider(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ProviderOpenAI,
			Model:         "gpt-test",
			ThinkingLevel: "high",
			SystemPrompt:  "Stay concise.",
			OpenAI:        &configs.AIProviderConfig{APIKey: "openai-key"},
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
		},
	}

	summary := SafeConfigSummaryFromConfig(cfg)
	if !summary.GetValid() {
		t.Fatalf("expected config to be valid, got error %q", summary.GetError())
	}
	if summary.GetProvider() != ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", ProviderOpenAI, summary.GetProvider())
	}
	if summary.GetModel() != "gpt-test" {
		t.Fatalf("expected model %q, got %q", "gpt-test", summary.GetModel())
	}
	if summary.GetThinkingLevel() != "high" {
		t.Fatalf("expected thinking level %q, got %q", "high", summary.GetThinkingLevel())
	}
	if summary.GetSystemPrompt() != "Stay concise." {
		t.Fatalf("expected system prompt %q, got %q", "Stay concise.", summary.GetSystemPrompt())
	}
}

func TestSafeConfigSummaryFromConfigFallsBackToConfiguredProvider(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			OpenAI:       &configs.AIProviderConfig{},
			OpenAICompat: &configs.AIProviderConfig{},
			OpenRouter:   &configs.AIProviderConfig{},
			Google:       &configs.AIProviderConfig{},
			Anthropic:    &configs.AIProviderConfig{APIKey: "anthropic-key"},
		},
	}

	summary := SafeConfigSummaryFromConfig(cfg)
	if !summary.GetValid() {
		t.Fatalf("expected fallback config to be valid, got error %q", summary.GetError())
	}
	if summary.GetProvider() != ProviderAnthropic {
		t.Fatalf("expected provider fallback %q, got %q", ProviderAnthropic, summary.GetProvider())
	}
}

func TestSafeConfigSummaryFromConfigRejectsExplicitProviderWithoutAPIKey(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:  ProviderOpenAI,
			OpenAI:    &configs.AIProviderConfig{},
			Anthropic: &configs.AIProviderConfig{APIKey: "anthropic-key"},
		},
	}

	summary := SafeConfigSummaryFromConfig(cfg)
	if summary.GetValid() {
		t.Fatal("expected config to be invalid")
	}
	if summary.GetProvider() != ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", ProviderOpenAI, summary.GetProvider())
	}
	expected := "server AI provider \"openai\" is missing an API key; run `ai-config` on the server"
	if summary.GetError() != expected {
		t.Fatalf("expected error %q, got %q", expected, summary.GetError())
	}
}

func TestSafeConfigSummaryFromConfigAcceptsGoogleVertexConfiguration(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderGoogle,
			Google: &configs.AIProviderConfig{
				Project:  "vertex-project",
				Location: "us-central1",
			},
		},
	}

	summary := SafeConfigSummaryFromConfig(cfg)
	if !summary.GetValid() {
		t.Fatalf("expected google vertex config to be valid, got error %q", summary.GetError())
	}
	if summary.GetProvider() != ProviderGoogle {
		t.Fatalf("expected provider %q, got %q", ProviderGoogle, summary.GetProvider())
	}
}

func TestSafeConfigSummaryFromConfigAcceptsOpenAICompatBaseURL(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider: ProviderOpenAICompat,
			OpenAICompat: &configs.AIProviderConfig{
				BaseURL: "http://127.0.0.1:11434/v1",
			},
		},
	}

	summary := SafeConfigSummaryFromConfig(cfg)
	if !summary.GetValid() {
		t.Fatalf("expected openai-compat config to be valid, got error %q", summary.GetError())
	}
	if summary.GetProvider() != ProviderOpenAICompat {
		t.Fatalf("expected provider %q, got %q", ProviderOpenAICompat, summary.GetProvider())
	}
}

func TestConfiguredProvidersFromConfigReportsAvailabilityOnly(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Anthropic:    &configs.AIProviderConfig{},
			Google:       &configs.AIProviderConfig{Project: "vertex-project", Location: "us-central1"},
			OpenAI:       &configs.AIProviderConfig{APIKey: "openai-key"},
			OpenAICompat: &configs.AIProviderConfig{BaseURL: "http://127.0.0.1:8080/v1"},
			OpenRouter:   &configs.AIProviderConfig{},
		},
	}

	providers := ConfiguredProvidersFromConfig(cfg)
	if len(providers) != len(SupportedProviders()) {
		t.Fatalf("expected %d providers, got %d", len(SupportedProviders()), len(providers))
	}
	if providers[0].GetName() != ProviderAnthropic || providers[0].GetConfigured() {
		t.Fatalf("expected anthropic to be present and unconfigured, got %+v", providers[0])
	}
	if providers[1].GetName() != ProviderGoogle || !providers[1].GetConfigured() {
		t.Fatalf("expected google to be present and configured, got %+v", providers[1])
	}
	if providers[2].GetName() != ProviderOpenAI || !providers[2].GetConfigured() {
		t.Fatalf("expected openai to be present and configured, got %+v", providers[2])
	}
	if providers[3].GetName() != ProviderOpenAICompat || !providers[3].GetConfigured() {
		t.Fatalf("expected openai-compat to be present and configured, got %+v", providers[3])
	}
	if providers[4].GetName() != ProviderOpenRouter || providers[4].GetConfigured() {
		t.Fatalf("expected openrouter to be present and unconfigured, got %+v", providers[4])
	}
}
