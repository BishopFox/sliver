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

// AIProviderConfig - Shared AI provider configuration.
type AIProviderConfig struct {
	APIKey string `json:"api_key" yaml:"api_key"`
}

// AIConfig - Server-side AI provider configuration.
type AIConfig struct {
	Anthropic *AIProviderConfig `json:"anthropic" yaml:"anthropic"`
	OpenAI    *AIProviderConfig `json:"openai" yaml:"openai"`
}

func defaultAIProviderConfig() *AIProviderConfig {
	return &AIProviderConfig{}
}

func defaultAIConfig() *AIConfig {
	return &AIConfig{
		Anthropic: defaultAIProviderConfig(),
		OpenAI:    defaultAIProviderConfig(),
	}
}

func normalizeAIConfig(config *ServerConfig) {
	if config.AI == nil {
		config.AI = defaultAIConfig()
	}
	if config.AI.Anthropic == nil {
		config.AI.Anthropic = defaultAIProviderConfig()
	}
	if config.AI.OpenAI == nil {
		config.AI.OpenAI = defaultAIProviderConfig()
	}
}
