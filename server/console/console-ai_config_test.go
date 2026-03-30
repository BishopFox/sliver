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
			Anthropic:    &configs.AIProviderConfig{APIKey: "anthropic-key", BaseURL: "https://api.anthropic.test"},
			Google:       &configs.AIProviderConfig{},
			OpenAI:       &configs.AIProviderConfig{},
			OpenAICompat: &configs.AIProviderConfig{},
			OpenRouter:   &configs.AIProviderConfig{},
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

func TestCurrentAIConfigFormResultForProviderLoadsRequestedProviderSettings(t *testing.T) {
	serverConfig := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ai.ProviderAnthropic,
			ThinkingLevel: "medium",
			SystemPrompt:  "Stay concise.",
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
			Google:        &configs.AIProviderConfig{},
			OpenAI: &configs.AIProviderConfig{
				Models:          []string{"gpt-test", "gpt-test-mini"},
				APIKey:          "openai-key",
				BaseURL:         "https://api.openai.test",
				Organization:    "org-test",
				Project:         "proj-test",
				UseResponsesAPI: boolPtr(true),
			},
			OpenAICompat: &configs.AIProviderConfig{},
			OpenRouter:   &configs.AIProviderConfig{},
		},
	}

	result := currentAIConfigFormResultForProvider(serverConfig, ai.ProviderOpenAI)
	if result.Provider != ai.ProviderOpenAI {
		t.Fatalf("expected %q provider, got %q", ai.ProviderOpenAI, result.Provider)
	}
	if len(result.Models) != 2 || result.Models[0] != "gpt-test" || result.Models[1] != "gpt-test-mini" {
		t.Fatalf("expected provider models to load, got %#v", result.Models)
	}
	if result.SystemPrompt != "Stay concise." {
		t.Fatalf("expected shared system prompt, got %q", result.SystemPrompt)
	}
	if result.APIKey != "openai-key" {
		t.Fatalf("expected openai API key, got %q", result.APIKey)
	}
	if result.BaseURL != "https://api.openai.test" {
		t.Fatalf("expected openai base url, got %q", result.BaseURL)
	}
	if result.Organization != "org-test" {
		t.Fatalf("expected openai organization, got %q", result.Organization)
	}
	if result.Project != "proj-test" {
		t.Fatalf("expected openai project, got %q", result.Project)
	}
	if !result.UseResponsesAPI {
		t.Fatal("expected openai responses api to remain enabled")
	}
}

func TestCurrentAIConfigFormResultForProviderFallsBackToLegacySharedModel(t *testing.T) {
	serverConfig := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ai.ProviderAnthropic,
			Model:         "legacy-model",
			ThinkingLevel: "medium",
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
			OpenAI:        &configs.AIProviderConfig{APIKey: "openai-key"},
		},
	}

	result := currentAIConfigFormResultForProvider(serverConfig, ai.ProviderOpenAI)
	if len(result.Models) != 1 || result.Models[0] != "legacy-model" {
		t.Fatalf("expected legacy shared model fallback, got %#v", result.Models)
	}
}

func TestCurrentAIConfigFormResultForProviderDefaultsResponsesAPIForOpenAI(t *testing.T) {
	result := currentAIConfigFormResultForProvider(&configs.ServerConfig{}, ai.ProviderOpenAI)
	if result.Provider != ai.ProviderOpenAI {
		t.Fatalf("expected %q provider, got %q", ai.ProviderOpenAI, result.Provider)
	}
	if !result.UseResponsesAPI {
		t.Fatal("expected openai to default to responses API")
	}
}

func TestApplyAIConfigFormResultUpdatesOnlySelectedProvider(t *testing.T) {
	serverConfig := &configs.ServerConfig{
		AI: &configs.AIConfig{
			Provider:      ai.ProviderAnthropic,
			Model:         "old-model",
			ThinkingLevel: "low",
			SystemPrompt:  "Existing default",
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
			Google:        &configs.AIProviderConfig{},
			OpenAI:        &configs.AIProviderConfig{APIKey: "openai-key", BaseURL: "https://api.openai.test"},
			OpenAICompat:  &configs.AIProviderConfig{},
			OpenRouter:    &configs.AIProviderConfig{},
		},
	}

	applyAIConfigFormResult(serverConfig, &forms.AIConfigFormResult{
		Provider:        ai.ProviderOpenAI,
		Models:          []string{"gpt-test", "gpt-test-mini"},
		ThinkingLevel:   "high",
		SystemPrompt:    "Stay concise.",
		APIKey:          "new-openai-key",
		BaseURL:         "https://override.openai.test",
		Organization:    "org-test",
		Project:         "proj-test",
		UseResponsesAPI: true,
	})

	if serverConfig.AI.Provider != ai.ProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", ai.ProviderOpenAI, serverConfig.AI.Provider)
	}
	if serverConfig.AI.Model != "" {
		t.Fatalf("expected legacy model field to be cleared, got %q", serverConfig.AI.Model)
	}
	if len(serverConfig.AI.OpenAI.Models) != 2 || serverConfig.AI.OpenAI.Models[0] != "gpt-test" || serverConfig.AI.OpenAI.Models[1] != "gpt-test-mini" {
		t.Fatalf("expected models to update, got %#v", serverConfig.AI.OpenAI.Models)
	}
	if serverConfig.AI.ThinkingLevel != "high" {
		t.Fatalf("expected thinking level %q, got %q", "high", serverConfig.AI.ThinkingLevel)
	}
	if serverConfig.AI.SystemPrompt != "Stay concise." {
		t.Fatalf("expected system prompt %q, got %q", "Stay concise.", serverConfig.AI.SystemPrompt)
	}
	if serverConfig.AI.OpenAI.APIKey != "new-openai-key" {
		t.Fatalf("expected updated openai api key, got %q", serverConfig.AI.OpenAI.APIKey)
	}
	if serverConfig.AI.OpenAI.BaseURL != "https://override.openai.test" {
		t.Fatalf("expected updated openai base url, got %q", serverConfig.AI.OpenAI.BaseURL)
	}
	if serverConfig.AI.OpenAI.Organization != "org-test" {
		t.Fatalf("expected updated openai organization, got %q", serverConfig.AI.OpenAI.Organization)
	}
	if serverConfig.AI.OpenAI.Project != "proj-test" {
		t.Fatalf("expected updated openai project, got %q", serverConfig.AI.OpenAI.Project)
	}
	if serverConfig.AI.OpenAI.UseResponsesAPI == nil || !*serverConfig.AI.OpenAI.UseResponsesAPI {
		t.Fatal("expected openai responses api flag to be updated")
	}
	if serverConfig.AI.Anthropic.APIKey != "anthropic-key" {
		t.Fatalf("expected anthropic api key to remain unchanged, got %q", serverConfig.AI.Anthropic.APIKey)
	}
}
