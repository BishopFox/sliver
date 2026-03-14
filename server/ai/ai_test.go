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
}

func TestSafeConfigSummaryFromConfigFallsBackToConfiguredProvider(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			OpenAI:    &configs.AIProviderConfig{},
			Anthropic: &configs.AIProviderConfig{APIKey: "anthropic-key"},
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

func TestConfiguredProvidersFromConfigReportsAvailabilityOnly(t *testing.T) {
	cfg := &configs.ServerConfig{
		AI: &configs.AIConfig{
			OpenAI:    &configs.AIProviderConfig{APIKey: "openai-key"},
			Anthropic: &configs.AIProviderConfig{},
		},
	}

	providers := ConfiguredProvidersFromConfig(cfg)
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].GetName() != ProviderAnthropic || providers[0].GetConfigured() {
		t.Fatalf("expected anthropic to be present and unconfigured, got %+v", providers[0])
	}
	if providers[1].GetName() != ProviderOpenAI || !providers[1].GetConfigured() {
		t.Fatalf("expected openai to be present and configured, got %+v", providers[1])
	}
}
