package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
)

const (
	defaultOpenAIModel       = "gpt-5.2"
	defaultAnthropicModel    = "claude-sonnet-4-0"
	defaultGoogleModel       = "gemini-2.5-pro"
	defaultOpenRouterModel   = "openai/gpt-5"
	defaultCompletionTimeout = 2 * time.Minute
)

var httpClient = &http.Client{Timeout: defaultCompletionTimeout}

// SetHTTPClientForTests swaps the provider HTTP client and returns a restore function.
func SetHTTPClientForTests(client *http.Client) func() {
	previous := httpClient
	if client == nil {
		client = &http.Client{Timeout: defaultCompletionTimeout}
	}
	httpClient = client
	return func() {
		httpClient = previous
	}
}

// RuntimeConfig captures the provider settings used for a single completion run.
type RuntimeConfig struct {
	Provider         string
	Model            string
	ThinkingLevel    string
	MaxOutputTokens  int64
	Temperature      *float64
	TopP             *float64
	TopK             *int64
	PresencePenalty  *float64
	FrequencyPenalty *float64

	APIKey          string
	BaseURL         string
	Headers         map[string]string
	UserAgent       string
	Organization    string
	Project         string
	Location        string
	SkipAuth        bool
	UseResponsesAPI bool
	UseBedrock      bool
}

// Completion contains the persisted assistant response details.
type Completion struct {
	Provider          string
	Model             string
	Content           string
	ProviderMessageID string
	FinishReason      string
}

type providerMessage struct {
	Role    string
	Content string
}

// ResolveRuntimeConfig derives the provider settings for a stored conversation.
func ResolveRuntimeConfig(cfg *configs.ServerConfig, conversation *clientpb.AIConversation) (*RuntimeConfig, error) {
	runtime := &RuntimeConfig{
		ThinkingLevel: normalizeThinkingLevel(cfg),
	}
	var providerConfig *configs.AIProviderConfig
	if cfg != nil && cfg.AI != nil {
		runtime.MaxOutputTokens = cfg.AI.MaxOutputTokens
		runtime.Temperature = copyOptionalFloat(cfg.AI.Temperature)
		runtime.TopP = copyOptionalFloat(cfg.AI.TopP)
		runtime.TopK = copyOptionalInt(cfg.AI.TopK)
		runtime.PresencePenalty = copyOptionalFloat(cfg.AI.PresencePenalty)
		runtime.FrequencyPenalty = copyOptionalFloat(cfg.AI.FrequencyPenalty)
	}

	if conversation != nil {
		runtime.Provider = NormalizeProviderName(conversation.GetProvider())
		runtime.Model = strings.TrimSpace(conversation.GetModel())
	}

	if runtime.Provider == "" {
		selectedProvider, selectedProviderConfig := selectedProviderConfig(cfg)
		runtime.Provider = selectedProvider
		providerConfig = selectedProviderConfig
		applyProviderRuntimeConfig(runtime, providerConfig)
	} else {
		if !IsSupportedProvider(runtime.Provider) {
			return runtime, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
		}
		providerConfig = aiProviderConfig(cfg, runtime.Provider)
		applyProviderRuntimeConfig(runtime, providerConfig)
	}

	if runtime.Model == "" && cfg != nil && cfg.AI != nil {
		runtime.Model = strings.TrimSpace(cfg.AI.Model)
	}
	if runtime.Model == "" {
		runtime.Model = defaultModelForProvider(runtime.Provider)
	}
	runtime.UseResponsesAPI = useResponsesAPI(runtime.Provider, providerConfig)

	switch {
	case runtime.Provider == "":
		return runtime, fmt.Errorf("server AI provider is not configured; run `ai-config` on the server")
	case !IsSupportedProvider(runtime.Provider):
		return runtime, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
	case !runtimeProviderConfigured(runtime):
		return runtime, errors.New(missingProviderConfigError(runtime.Provider))
	case runtime.Model == "":
		return runtime, fmt.Errorf("server AI provider %q is missing a model; update `ai.model` or choose a provider default", runtime.Provider)
	case !completionDriverAvailable(runtime):
		return runtime, missingDriverError(runtime)
	default:
		return runtime, nil
	}
}

// CompleteConversation sends the stored conversation history to the provider and returns the assistant reply.
func CompleteConversation(ctx context.Context, runtime *RuntimeConfig, conversation *clientpb.AIConversation) (*Completion, error) {
	if runtime == nil {
		return nil, fmt.Errorf("AI runtime config is required")
	}
	if conversation == nil {
		return nil, fmt.Errorf("AI conversation is required")
	}

	systemPrompt, messages, err := conversationHistory(conversation)
	if err != nil {
		return nil, err
	}

	driver, err := selectCompletionDriver(runtime)
	if err != nil {
		return nil, err
	}

	return driver.CompleteConversation(ctx, runtime, &completionRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
	})
}

func conversationHistory(conversation *clientpb.AIConversation) (string, []providerMessage, error) {
	if conversation == nil {
		return "", nil, fmt.Errorf("AI conversation is required")
	}

	systemPrompt := strings.TrimSpace(conversation.GetSystemPrompt())
	messages := make([]providerMessage, 0, len(conversation.GetMessages()))
	for _, message := range conversation.GetMessages() {
		if message == nil {
			continue
		}
		if message.GetVisibility() == clientpb.AIConversationMessageVisibility_AI_MESSAGE_VISIBILITY_UI_ONLY {
			continue
		}
		if message.GetKind() != clientpb.AIConversationMessageKind_AI_MESSAGE_KIND_CHAT {
			continue
		}

		role := strings.ToLower(strings.TrimSpace(message.GetRole()))
		content := strings.TrimSpace(message.GetContent())
		if content == "" {
			continue
		}

		switch role {
		case "user", "assistant":
			messages = append(messages, providerMessage{
				Role:    role,
				Content: content,
			})
		case "system":
			if systemPrompt == "" {
				systemPrompt = content
			}
		default:
			continue
		}
	}

	if len(messages) == 0 {
		return systemPrompt, nil, fmt.Errorf("AI conversation has no user or assistant messages")
	}
	if messages[len(messages)-1].Role != "user" {
		return systemPrompt, nil, fmt.Errorf("AI conversation is not awaiting an assistant response")
	}

	return systemPrompt, messages, nil
}

func defaultModelForProvider(provider string) string {
	switch NormalizeProviderName(provider) {
	case ProviderOpenAI:
		return defaultOpenAIModel
	case ProviderAnthropic:
		return defaultAnthropicModel
	case ProviderGoogle:
		return defaultGoogleModel
	case ProviderOpenRouter:
		return defaultOpenRouterModel
	default:
		return ""
	}
}

func completionDriverAvailable(runtime *RuntimeConfig) bool {
	_, err := selectCompletionDriver(runtime)
	return err == nil
}

func missingDriverError(runtime *RuntimeConfig) error {
	if runtime == nil {
		return fmt.Errorf("AI runtime config is required")
	}
	_, err := selectCompletionDriver(runtime)
	if err != nil {
		return err
	}
	return nil
}

func normalizeThinkingLevel(cfg *configs.ServerConfig) string {
	if cfg == nil || cfg.AI == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(cfg.AI.ThinkingLevel)) {
	case "low", "medium", "high", "disabled":
		return strings.ToLower(strings.TrimSpace(cfg.AI.ThinkingLevel))
	default:
		return ""
	}
}

func applyProviderRuntimeConfig(runtime *RuntimeConfig, providerConfig *configs.AIProviderConfig) {
	if runtime == nil || providerConfig == nil {
		return
	}
	runtime.APIKey = strings.TrimSpace(providerConfig.APIKey)
	runtime.BaseURL = strings.TrimSpace(providerConfig.BaseURL)
	runtime.Headers = copyStringMap(providerConfig.Headers)
	runtime.UserAgent = strings.TrimSpace(providerConfig.UserAgent)
	runtime.Organization = strings.TrimSpace(providerConfig.Organization)
	runtime.Project = strings.TrimSpace(providerConfig.Project)
	runtime.Location = strings.TrimSpace(providerConfig.Location)
	runtime.SkipAuth = providerConfig.SkipAuth
	if providerConfig.UseResponsesAPI != nil {
		runtime.UseResponsesAPI = *providerConfig.UseResponsesAPI
	}
	runtime.UseBedrock = providerConfig.UseBedrock
}

func useResponsesAPI(provider string, providerConfig *configs.AIProviderConfig) bool {
	if providerConfig != nil && providerConfig.UseResponsesAPI != nil {
		return *providerConfig.UseResponsesAPI
	}
	return NormalizeProviderName(provider) == ProviderOpenAI
}

func providerHeaders(runtime *RuntimeConfig) map[string]string {
	if runtime == nil {
		return nil
	}

	if len(runtime.Headers) == 0 && runtime.UserAgent == "" {
		return nil
	}

	headers := make(map[string]string, len(runtime.Headers)+1)
	for key, value := range runtime.Headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		headers[key] = value
	}
	if runtime.UserAgent != "" {
		headers["User-Agent"] = runtime.UserAgent
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func runtimeProviderConfigured(runtime *RuntimeConfig) bool {
	if runtime == nil {
		return false
	}
	switch runtime.Provider {
	case ProviderAnthropic:
		return runtime.APIKey != "" ||
			runtime.UseBedrock ||
			(runtime.Project != "" && runtime.Location != "")
	case ProviderGoogle:
		return runtime.APIKey != "" ||
			(runtime.Project != "" && runtime.Location != "")
	case ProviderOpenAI:
		return runtime.APIKey != ""
	case ProviderOpenAICompat:
		return runtime.BaseURL != ""
	case ProviderOpenRouter:
		return runtime.APIKey != ""
	default:
		return false
	}
}

func copyOptionalFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}

func copyOptionalInt(value *int64) *int64 {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		copied[key] = value
	}
	if len(copied) == 0 {
		return nil
	}
	return copied
}

func joinTextBlocks(blocks []string) string {
	var filtered []string
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		filtered = append(filtered, block)
	}
	return strings.Join(filtered, "\n\n")
}

func truncateForError(value string) string {
	const maxLen = 512
	value = strings.TrimSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return strings.TrimSpace(value[:maxLen]) + "..."
}

func fallbackString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
