package console

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

	"github.com/bishopfox/sliver/server/ai"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/console/forms"
	"github.com/spf13/cobra"
)

func aiConfigCmd(_ *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Printf(Warn + "This command does not accept arguments.\n")
		return
	}

	serverConfig := configs.GetServerConfig()
	if serverConfig == nil {
		fmt.Printf(Warn + "Failed to load server configuration.\n")
		return
	}

	result := currentAIConfigFormResult(serverConfig)
	if err := forms.AIConfig(result); err != nil {
		if errors.Is(err, forms.ErrUserAborted) {
			return
		}
		fmt.Printf(Warn+"AI configuration form failed: %v\n", err)
		return
	}

	applyAIConfigFormResult(serverConfig, result)
	if err := serverConfig.Save(); err != nil {
		fmt.Printf(Warn+"Failed to save AI configuration: %v\n", err)
		return
	}

	fmt.Printf(Info+"Saved AI configuration to %s\n", configs.GetServerConfigPath())
	fmt.Printf(
		Info+"Provider=%s, model=%s, thinking=%s, api_key=%s\n",
		formatAIValue(serverConfig.AI.Provider, ai.ProviderOpenAI),
		formatAIValue(serverConfig.AI.Model, "provider default"),
		formatAIValue(serverConfig.AI.ThinkingLevel, "provider default"),
		apiKeyStatus(aiProviderConfig(serverConfig.AI, serverConfig.AI.Provider)),
	)
	fmt.Printf(Info + "Run ai-config again to configure credentials for a different provider.\n")
}

func currentAIConfigFormResult(serverConfig *configs.ServerConfig) *forms.AIConfigFormResult {
	if serverConfig == nil {
		return &forms.AIConfigFormResult{Provider: ai.ProviderOpenAI}
	}

	provider := selectedAIProvider(serverConfig.AI)
	providerConfig := aiProviderConfig(serverConfig.AI, provider)

	result := &forms.AIConfigFormResult{
		Provider: provider,
	}
	if serverConfig.AI != nil {
		result.Model = strings.TrimSpace(serverConfig.AI.Model)
		result.ThinkingLevel = strings.TrimSpace(serverConfig.AI.ThinkingLevel)
	}
	if providerConfig != nil {
		result.APIKey = strings.TrimSpace(providerConfig.APIKey)
		result.BaseURL = strings.TrimSpace(providerConfig.BaseURL)
	}
	return result
}

func applyAIConfigFormResult(serverConfig *configs.ServerConfig, result *forms.AIConfigFormResult) {
	if serverConfig == nil || result == nil {
		return
	}
	if serverConfig.AI == nil {
		serverConfig.AI = &configs.AIConfig{}
	}

	provider := ai.NormalizeProviderName(result.Provider)
	serverConfig.AI.Provider = provider
	serverConfig.AI.Model = strings.TrimSpace(result.Model)
	serverConfig.AI.ThinkingLevel = strings.ToLower(strings.TrimSpace(result.ThinkingLevel))

	providerConfig := aiProviderConfig(serverConfig.AI, provider)
	if providerConfig == nil {
		return
	}
	providerConfig.APIKey = strings.TrimSpace(result.APIKey)
	providerConfig.BaseURL = strings.TrimSpace(result.BaseURL)
}

func selectedAIProvider(aiConfig *configs.AIConfig) string {
	if aiConfig != nil && ai.IsSupportedProvider(aiConfig.Provider) {
		return ai.NormalizeProviderName(aiConfig.Provider)
	}

	for _, provider := range ai.SupportedProviders() {
		if providerConfig := aiProviderConfig(aiConfig, provider); providerConfig != nil && strings.TrimSpace(providerConfig.APIKey) != "" {
			return provider
		}
	}

	return ai.ProviderOpenAI
}

func aiProviderConfig(aiConfig *configs.AIConfig, provider string) *configs.AIProviderConfig {
	if aiConfig == nil {
		return nil
	}

	switch ai.NormalizeProviderName(provider) {
	case ai.ProviderAnthropic:
		if aiConfig.Anthropic == nil {
			aiConfig.Anthropic = &configs.AIProviderConfig{}
		}
		return aiConfig.Anthropic
	case ai.ProviderOpenAI:
		if aiConfig.OpenAI == nil {
			aiConfig.OpenAI = &configs.AIProviderConfig{}
		}
		return aiConfig.OpenAI
	default:
		return nil
	}
}

func formatAIValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func apiKeyStatus(providerConfig *configs.AIProviderConfig) string {
	if providerConfig == nil || strings.TrimSpace(providerConfig.APIKey) == "" {
		return "not set"
	}
	return "configured"
}
