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

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/log"
	"gopkg.in/yaml.v3"
)

const (
	aiConfigFileName      = "ai.yaml"
	defaultAISystemPrompt = `You are Sliver's AI copilot for authorized security testing, detection engineering, lab work, and incident-response support in environments the operator is explicitly permitted to assess.

Your job is to help the operator make careful, high-signal decisions inside the Sliver workflow.

Operating rules:
- Assume all activity must stay within the operator's authorized scope. If a request is ambiguous, unusually risky, destructive, or inconsistent with the available context, pause, state the concern, and ask for confirmation or offer a safer alternative.
- Use the current conversation, target metadata, and Sliver context to tailor answers. If OS, privilege, transport, or session/beacon state matters, say so before recommending commands.
- Never fabricate command output, host state, credentials, files, loot, or tool results. When information is missing, say exactly what is unknown.
- Prefer the smallest next step that increases certainty. Start with low-noise, reversible discovery before collection, execution, or configuration changes.
- Highlight operational tradeoffs: stealth, telemetry, privilege requirements, target stability, cleanup burden, and chances of breaking access.
- Prefer actions that preserve operator access, host stability, and forensic defensibility. Avoid destructive, noisy, or irreversible steps unless the operator explicitly asks for them and the purpose is clear.
- Treat credentials, secrets, tokens, and loot as sensitive. Do not suggest unnecessary exposure, duplication, or broad collection.
- When troubleshooting, separate confirmed facts from hypotheses and propose the next diagnostic step rather than many speculative changes at once.
- When multiple approaches are possible, present the safest practical option first, then note faster or more aggressive alternatives only if they materially help.
- When suggesting Sliver commands, make them concrete, minimal, and easy to audit. Include assumptions and prerequisites when they matter.
- When asked to draft code, scripts, or one-liners, optimize for readability, explicitness, and minimal side effects.

Response style:
- Be concise, structured, and operator-focused.
- Prefer short checklists or step plans over long essays.
- Distinguish clearly between facts, assumptions, and recommendations.
- If the request appears outside authorized security work, refuse and redirect to safer high-level guidance.`
)

var (
	aiConfigLog = log.NamedLogger("config", "ai")
)

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
	SystemPrompt     string            `json:"system_prompt" yaml:"system_prompt"`
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

type aiConfigEnvelope struct {
	AI *AIConfig `json:"ai" yaml:"ai"`
}

// GetAIConfigPath - File path to ai.yaml.
func GetAIConfigPath() string {
	appDir := assets.GetRootAppDir()
	aiConfigPath := filepath.Join(appDir, "configs", aiConfigFileName)
	aiConfigLog.Debugf("Loading config from %s", aiConfigPath)
	return aiConfigPath
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
		SystemPrompt:     defaultAISystemPrompt,
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

// Save - Save AI config file to disk.
func (c *AIConfig) Save() error {
	config := normalizeAIConfig(c)
	configPath := GetAIConfigPath()
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		aiConfigLog.Debugf("Creating config dir %s", configDir)
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			return err
		}
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	aiConfigLog.Infof("Saving config to %s", configPath)
	err = os.WriteFile(configPath, data, 0600)
	if err != nil {
		aiConfigLog.Errorf("Failed to write config %s", err)
	}
	return err
}

// GetAIConfig - Get AI config value from ai.yaml.
func GetAIConfig() *AIConfig {
	return getAIConfig(nil)
}

func getAIConfig(migratedConfig *AIConfig) *AIConfig {
	configPath := GetAIConfigPath()
	config := defaultAIConfig()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			aiConfigLog.Errorf("Failed to read config file %s", err)
			return config
		}
		err = yaml.Unmarshal(data, config)
		if err != nil {
			aiConfigLog.Errorf("Failed to parse config file %s", err)
			return config
		}
	} else if migratedConfig != nil {
		config = migratedConfig
		aiConfigLog.Infof("Migrating embedded AI config to %s", configPath)
	} else {
		aiConfigLog.Warnf("Config file does not exist, using defaults")
	}

	config = normalizeAIConfig(config)

	err := config.Save() // This updates the config with any missing fields
	if err != nil {
		aiConfigLog.Errorf("Failed to save default config %s", err)
		return config
	}
	return config
}

func aiConfigFromYAML(data []byte) *AIConfig {
	envelope := &aiConfigEnvelope{}
	if err := yaml.Unmarshal(data, envelope); err != nil {
		aiConfigLog.Errorf("Failed to parse embedded AI config %s", err)
		return nil
	}
	if envelope.AI == nil {
		return nil
	}
	return normalizeAIConfig(envelope.AI)
}

func aiConfigFromJSON(data []byte) *AIConfig {
	envelope := &aiConfigEnvelope{}
	if err := json.Unmarshal(data, envelope); err != nil {
		aiConfigLog.Errorf("Failed to parse embedded AI config %s", err)
		return nil
	}
	if envelope.AI == nil {
		return nil
	}
	return normalizeAIConfig(envelope.AI)
}

func normalizeAIConfig(config *AIConfig) *AIConfig {
	if config == nil {
		config = defaultAIConfig()
	}
	config.Provider = strings.ToLower(strings.TrimSpace(config.Provider))
	config.Model = strings.TrimSpace(config.Model)
	config.ThinkingLevel = strings.ToLower(strings.TrimSpace(config.ThinkingLevel))
	config.SystemPrompt = strings.TrimSpace(config.SystemPrompt)
	if config.MaxOutputTokens < 0 {
		config.MaxOutputTokens = 0
	}
	config.Temperature = normalizeOptionalFloat(config.Temperature)
	config.TopP = normalizeOptionalFloat(config.TopP)
	config.TopK = normalizeOptionalInt(config.TopK)
	config.PresencePenalty = normalizeOptionalFloat(config.PresencePenalty)
	config.FrequencyPenalty = normalizeOptionalFloat(config.FrequencyPenalty)
	if config.Anthropic == nil {
		config.Anthropic = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.Anthropic)
	if config.Google == nil {
		config.Google = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.Google)
	if config.OpenAI == nil {
		config.OpenAI = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.OpenAI)
	if config.OpenAICompat == nil {
		config.OpenAICompat = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.OpenAICompat)
	if config.OpenRouter == nil {
		config.OpenRouter = defaultAIProviderConfig()
	}
	normalizeAIProviderConfig(config.OpenRouter)
	return config
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
