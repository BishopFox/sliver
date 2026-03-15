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
	APIKey          string            `json:"api_key" yaml:"api_key"`
	BaseURL         string            `json:"base_url" yaml:"base_url"`
	Headers         map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	UserAgent       string            `json:"user_agent,omitempty" yaml:"user_agent,omitempty"`
	Organization    string            `json:"organization,omitempty" yaml:"organization,omitempty"`
	Project         string            `json:"project,omitempty" yaml:"project,omitempty"`
	Location        string            `json:"location,omitempty" yaml:"location,omitempty"`
	SkipAuth        bool              `json:"skip_auth,omitempty" yaml:"skip_auth,omitempty"`
	UseResponsesAPI *bool             `json:"use_responses_api,omitempty" yaml:"use_responses_api,omitempty"`
	UseBedrock      bool              `json:"use_bedrock,omitempty" yaml:"use_bedrock,omitempty"`
}

// AIConfig - Server-side AI provider configuration.
type AIConfig struct {
	Provider         string            `json:"provider" yaml:"provider"`
	Model            string            `json:"model" yaml:"model"`
	ThinkingLevel    string            `json:"thinking_level" yaml:"thinking_level"`
	MaxOutputTokens  int64             `json:"max_output_tokens,omitempty" yaml:"max_output_tokens,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	TopP             *float64          `json:"top_p,omitempty" yaml:"top_p,omitempty"`
	TopK             *int64            `json:"top_k,omitempty" yaml:"top_k,omitempty"`
	PresencePenalty  *float64          `json:"presence_penalty,omitempty" yaml:"presence_penalty,omitempty"`
	FrequencyPenalty *float64          `json:"frequency_penalty,omitempty" yaml:"frequency_penalty,omitempty"`
	Anthropic        *AIProviderConfig `json:"anthropic" yaml:"anthropic"`
	Google           *AIProviderConfig `json:"google" yaml:"google"`
	OpenAI           *AIProviderConfig `json:"openai" yaml:"openai"`
	OpenAICompat     *AIProviderConfig `json:"openai_compat" yaml:"openai_compat"`
	OpenRouter       *AIProviderConfig `json:"openrouter" yaml:"openrouter"`
}

func defaultAIProviderConfig() *AIProviderConfig {
	return &AIProviderConfig{
		Headers: map[string]string{},
	}
}

func defaultAIConfig() *AIConfig {
	return &AIConfig{
		Provider:         "",
		Model:            "",
		ThinkingLevel:    "",
		MaxOutputTokens:  0,
		Temperature:      nil,
		TopP:             nil,
		TopK:             nil,
		PresencePenalty:  nil,
		FrequencyPenalty: nil,
		Anthropic:        defaultAIProviderConfig(),
		Google:           defaultAIProviderConfig(),
		OpenAI:           defaultAIProviderConfig(),
		OpenAICompat:     defaultAIProviderConfig(),
		OpenRouter:       defaultAIProviderConfig(),
	}
}

func normalizeAIConfig(config *ServerConfig) {
	if config.AI == nil {
		config.AI = defaultAIConfig()
	}
	config.AI.Provider = strings.ToLower(strings.TrimSpace(config.AI.Provider))
	config.AI.Model = strings.TrimSpace(config.AI.Model)
	config.AI.ThinkingLevel = strings.ToLower(strings.TrimSpace(config.AI.ThinkingLevel))
	if config.AI.MaxOutputTokens < 0 {
		config.AI.MaxOutputTokens = 0
	}
	config.AI.Temperature = normalizeOptionalFloat(config.AI.Temperature)
	config.AI.TopP = normalizeOptionalFloat(config.AI.TopP)
	config.AI.TopK = normalizeOptionalInt(config.AI.TopK)
	config.AI.PresencePenalty = normalizeOptionalFloat(config.AI.PresencePenalty)
	config.AI.FrequencyPenalty = normalizeOptionalFloat(config.AI.FrequencyPenalty)
	if config.AI.Anthropic == nil {
		config.AI.Anthropic = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.AI.Anthropic)
	if config.AI.Google == nil {
		config.AI.Google = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.AI.Google)
	if config.AI.OpenAI == nil {
		config.AI.OpenAI = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.AI.OpenAI)
	if config.AI.OpenAICompat == nil {
		config.AI.OpenAICompat = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.AI.OpenAICompat)
	if config.AI.OpenRouter == nil {
		config.AI.OpenRouter = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.AI.OpenRouter)
}

func normalizeAIProviderConfig(provider *AIProviderConfig) {
	if provider == nil {
		return
	}
	provider.APIKey = strings.TrimSpace(provider.APIKey)
	provider.BaseURL = strings.TrimSpace(provider.BaseURL)
	provider.UserAgent = strings.TrimSpace(provider.UserAgent)
	provider.Organization = strings.TrimSpace(provider.Organization)
	provider.Project = strings.TrimSpace(provider.Project)
	provider.Location = strings.TrimSpace(provider.Location)
	provider.UseResponsesAPI = normalizeOptionalBool(provider.UseResponsesAPI)
	provider.Headers = normalizeStringMap(provider.Headers)
}

func normalizeOptionalFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}

func normalizeOptionalInt(value *int64) *int64 {
	if value == nil {
		return nil
	}
	if *value < 0 {
		zero := int64(0)
		return &zero
	}
	normalized := *value
	return &normalized
}

func normalizeOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}

func normalizeStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return map[string]string{}
	}
	return normalized
}
