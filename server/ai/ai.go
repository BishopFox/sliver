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
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
)

var (
	// ErrUnsupportedProvider - Returned when an AI provider is unknown.
	ErrUnsupportedProvider = errors.New("unsupported AI provider")
)

// Provider - Provider metadata derived from server configuration.
type Provider struct {
	Name   string
	APIKey string
}

// SupportedProviders - Returns the currently supported server-side AI providers.
func SupportedProviders() []string {
	return []string{ProviderAnthropic, ProviderOpenAI}
}

// NormalizeProviderName - Convert user supplied provider names to canonical identifiers.
func NormalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IsSupportedProvider - Indicates whether the provider is supported by the server scaffolding.
func IsSupportedProvider(name string) bool {
	switch NormalizeProviderName(name) {
	case ProviderAnthropic, ProviderOpenAI:
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
		if cfg.AI.Anthropic == nil {
			return &Provider{Name: ProviderAnthropic}, nil
		}
		return &Provider{Name: ProviderAnthropic, APIKey: cfg.AI.Anthropic.APIKey}, nil
	case ProviderOpenAI:
		if cfg.AI.OpenAI == nil {
			return &Provider{Name: ProviderOpenAI}, nil
		}
		return &Provider{Name: ProviderOpenAI, APIKey: cfg.AI.OpenAI.APIKey}, nil
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
			Configured: providerConfig != nil && strings.TrimSpace(providerConfig.APIKey) != "",
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
		summary.Model = strings.TrimSpace(cfg.AI.Model)
		summary.ThinkingLevel = strings.TrimSpace(cfg.AI.ThinkingLevel)
	}

	provider, providerConfig := selectedProviderConfig(cfg)
	summary.Provider = provider

	switch {
	case cfg == nil || cfg.AI == nil:
		summary.Error = "server AI is not configured; run `ai-config` on the server"
	case provider == "":
		summary.Error = "server AI is missing a configured provider; run `ai-config` on the server"
	case providerConfig == nil || strings.TrimSpace(providerConfig.APIKey) == "":
		summary.Error = fmt.Sprintf("server AI provider %q is missing an API key; run `ai-config` on the server", provider)
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
		if providerConfig != nil && strings.TrimSpace(providerConfig.APIKey) != "" {
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
	case ProviderOpenAI:
		return cfg.AI.OpenAI
	default:
		return nil
	}
}
