package configs

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

import "strings"

// AIProviderConfig - Shared AI provider configuration.
type AIProviderConfig struct {
	APIKey  string `json:"api_key" yaml:"api_key"`
	BaseURL string `json:"base_url" yaml:"base_url"`
}

// AIConfig - Server-side AI provider configuration.
type AIConfig struct {
	Provider      string            `json:"provider" yaml:"provider"`
	Model         string            `json:"model" yaml:"model"`
	ThinkingLevel string            `json:"thinking_level" yaml:"thinking_level"`
	Anthropic     *AIProviderConfig `json:"anthropic" yaml:"anthropic"`
	OpenAI        *AIProviderConfig `json:"openai" yaml:"openai"`
}

func defaultAIProviderConfig() *AIProviderConfig {
	return &AIProviderConfig{}
}

func defaultAIConfig() *AIConfig {
	return &AIConfig{
		Provider:      "",
		Model:         "",
		ThinkingLevel: "",
		Anthropic:     defaultAIProviderConfig(),
		OpenAI:        defaultAIProviderConfig(),
	}
}

func normalizeAIConfig(config *ServerConfig) {
	if config.AI == nil {
		config.AI = defaultAIConfig()
	}
	config.AI.Provider = strings.ToLower(strings.TrimSpace(config.AI.Provider))
	config.AI.Model = strings.TrimSpace(config.AI.Model)
	config.AI.ThinkingLevel = strings.ToLower(strings.TrimSpace(config.AI.ThinkingLevel))
	if config.AI.Anthropic == nil {
		config.AI.Anthropic = defaultAIProviderConfig()
	}
	config.AI.Anthropic.APIKey = strings.TrimSpace(config.AI.Anthropic.APIKey)
	config.AI.Anthropic.BaseURL = strings.TrimSpace(config.AI.Anthropic.BaseURL)
	if config.AI.OpenAI == nil {
		config.AI.OpenAI = defaultAIProviderConfig()
	}
	config.AI.OpenAI.APIKey = strings.TrimSpace(config.AI.OpenAI.APIKey)
	config.AI.OpenAI.BaseURL = strings.TrimSpace(config.AI.OpenAI.BaseURL)
}
