package ai

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
)

const (
	ProviderAnthropic    = "anthropic"
	ProviderGoogle       = "google"
	ProviderOpenAI       = "openai"
	ProviderOpenAICompat = "openai-compat"
	ProviderOpenRouter   = "openrouter"
)

var (
	// ErrUnsupportedProvider - Returned when an AI provider is unknown.
	ErrUnsupportedProvider = errors.New("unsupported AI provider")
)

// Provider - Provider metadata derived from server configuration.
type Provider struct {
	Name   string
	Config *configs.AIProviderConfig
}

// SupportedProviders - Returns the currently supported server-side AI providers.
func SupportedProviders() []string {
	return []string{
		ProviderAnthropic,
		ProviderGoogle,
		ProviderOpenAI,
		ProviderOpenAICompat,
		ProviderOpenRouter,
	}
}

// NormalizeProviderName - Convert user supplied provider names to canonical identifiers.
func NormalizeProviderName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "gemini":
		return ProviderGoogle
	case "openai_compat", "openaicompat", "openai-compatible":
		return ProviderOpenAICompat
	default:
		return normalized
	}
}

// IsSupportedProvider - Indicates whether the provider is supported by the server scaffolding.
func IsSupportedProvider(name string) bool {
	switch NormalizeProviderName(name) {
	case ProviderAnthropic, ProviderGoogle, ProviderOpenAI, ProviderOpenAICompat, ProviderOpenRouter:
		return true
	default:
		return false
	}
}

// ProviderFromConfig - Resolve a configured provider by name.
func ProviderFromConfig(name string) (*Provider, error) {
	cfg := configs.GetServerConfig()
	if cfg == nil || cfg.AI == nil {
		return nil, ErrUnsupportedProvider
	}

	switch NormalizeProviderName(name) {
	case ProviderAnthropic:
		return &Provider{Name: ProviderAnthropic, Config: cfg.AI.Anthropic}, nil
	case ProviderGoogle:
		return &Provider{Name: ProviderGoogle, Config: cfg.AI.Google}, nil
	case ProviderOpenAI:
		return &Provider{Name: ProviderOpenAI, Config: cfg.AI.OpenAI}, nil
	case ProviderOpenAICompat:
		return &Provider{Name: ProviderOpenAICompat, Config: cfg.AI.OpenAICompat}, nil
	case ProviderOpenRouter:
		return &Provider{Name: ProviderOpenRouter, Config: cfg.AI.OpenRouter}, nil
	default:
		return nil, ErrUnsupportedProvider
	}
}

// ConfiguredProviders - Return provider availability without exposing configured secrets.
func ConfiguredProviders() []*clientpb.AIProviderConfig {
	return ConfiguredProvidersFromConfig(configs.GetServerConfig())
}

// ConfiguredProvidersFromConfig - Return provider availability for a specific config.
func ConfiguredProvidersFromConfig(cfg *configs.ServerConfig) []*clientpb.AIProviderConfig {
	providers := make([]*clientpb.AIProviderConfig, 0, len(SupportedProviders()))
	for _, name := range SupportedProviders() {
		providerConfig := aiProviderConfig(cfg, name)
		providers = append(providers, &clientpb.AIProviderConfig{
			Name:       name,
			Configured: isProviderConfigured(name, providerConfig),
		})
	}
	return providers
}

// SafeConfigSummary - Return the client-safe AI configuration summary.
func SafeConfigSummary() *clientpb.AIConfigSummary {
	return SafeConfigSummaryFromConfig(configs.GetServerConfig())
}

// SafeConfigSummaryFromConfig - Return the client-safe AI configuration summary for a specific config.
func SafeConfigSummaryFromConfig(cfg *configs.ServerConfig) *clientpb.AIConfigSummary {
	summary := &clientpb.AIConfigSummary{}
	if cfg != nil && cfg.AI != nil {
		summary.ThinkingLevel = strings.TrimSpace(cfg.AI.ThinkingLevel)
		summary.SystemPrompt = strings.TrimSpace(cfg.AI.SystemPrompt)
	}

	provider, providerConfig := selectedProviderConfig(cfg)
	summary.Provider = provider
	summary.Model = configuredDefaultModel(provider, selectedAIConfig(cfg), providerConfig)

	switch {
	case cfg == nil || cfg.AI == nil:
		summary.Error = "server AI is not configured; run `ai-config` on the server"
	case provider == "":
		summary.Error = "server AI is missing a configured provider; run `ai-config` on the server"
	case !isProviderConfigured(provider, providerConfig):
		summary.Error = missingProviderConfigError(provider)
	default:
		summary.Valid = true
	}

	return summary
}

func selectedProviderConfig(cfg *configs.ServerConfig) (string, *configs.AIProviderConfig) {
	if cfg == nil || cfg.AI == nil {
		return "", nil
	}

	explicitProvider := NormalizeProviderName(cfg.AI.Provider)
	if IsSupportedProvider(explicitProvider) {
		return explicitProvider, aiProviderConfig(cfg, explicitProvider)
	}

	for _, provider := range SupportedProviders() {
		providerConfig := aiProviderConfig(cfg, provider)
		if isProviderConfigured(provider, providerConfig) {
			return provider, providerConfig
		}
	}

	return "", nil
}

func aiProviderConfig(cfg *configs.ServerConfig, provider string) *configs.AIProviderConfig {
	if cfg == nil || cfg.AI == nil {
		return nil
	}

	switch NormalizeProviderName(provider) {
	case ProviderAnthropic:
		return cfg.AI.Anthropic
	case ProviderGoogle:
		return cfg.AI.Google
	case ProviderOpenAI:
		return cfg.AI.OpenAI
	case ProviderOpenAICompat:
		return cfg.AI.OpenAICompat
	case ProviderOpenRouter:
		return cfg.AI.OpenRouter
	default:
		return nil
	}
}

func isProviderConfigured(provider string, providerConfig *configs.AIProviderConfig) bool {
	if providerConfig == nil {
		return false
	}

	switch NormalizeProviderName(provider) {
	case ProviderAnthropic:
		return strings.TrimSpace(providerConfig.APIKey) != "" ||
			providerConfig.UseBedrock ||
			(strings.TrimSpace(providerConfig.Project) != "" && strings.TrimSpace(providerConfig.Location) != "")
	case ProviderGoogle:
		return strings.TrimSpace(providerConfig.APIKey) != "" ||
			(strings.TrimSpace(providerConfig.Project) != "" && strings.TrimSpace(providerConfig.Location) != "")
	case ProviderOpenAI:
		return strings.TrimSpace(providerConfig.APIKey) != ""
	case ProviderOpenAICompat:
		return strings.TrimSpace(providerConfig.BaseURL) != ""
	case ProviderOpenRouter:
		return strings.TrimSpace(providerConfig.APIKey) != ""
	default:
		return false
	}
}

func missingProviderConfigError(provider string) string {
	switch NormalizeProviderName(provider) {
	case ProviderAnthropic:
		return "server AI provider \"anthropic\" needs an API key, Bedrock mode, or a Vertex project/location; run `ai-config` on the server"
	case ProviderGoogle:
		return "server AI provider \"google\" needs a Gemini API key or a Vertex project/location; run `ai-config` on the server"
	case ProviderOpenAI:
		return "server AI provider \"openai\" is missing an API key; run `ai-config` on the server"
	case ProviderOpenAICompat:
		return "server AI provider \"openai-compat\" is missing a base URL; run `ai-config` on the server"
	case ProviderOpenRouter:
		return "server AI provider \"openrouter\" is missing an API key; run `ai-config` on the server"
	default:
		return fmt.Sprintf("server AI provider %q is not fully configured; run `ai-config` on the server", provider)
	}
}
