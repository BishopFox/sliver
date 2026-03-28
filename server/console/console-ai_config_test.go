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
			Model:         "shared-model",
			ThinkingLevel: "medium",
			SystemPrompt:  "Stay concise.",
			Anthropic:     &configs.AIProviderConfig{APIKey: "anthropic-key"},
			Google:        &configs.AIProviderConfig{},
			OpenAI: &configs.AIProviderConfig{
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
	if result.Model != "shared-model" {
		t.Fatalf("expected shared model, got %q", result.Model)
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
		Model:           "gpt-test",
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
	if serverConfig.AI.Model != "gpt-test" {
		t.Fatalf("expected model %q, got %q", "gpt-test", serverConfig.AI.Model)
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
