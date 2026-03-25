package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"charm.land/fantasy"
	fantasyanthropic "charm.land/fantasy/providers/anthropic"
	fantasygoogle "charm.land/fantasy/providers/google"
	fantasyopenai "charm.land/fantasy/providers/openai"
	fantasyopenaicompat "charm.land/fantasy/providers/openaicompat"
	fantasyopenrouter "charm.land/fantasy/providers/openrouter"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
)

const (
	defaultOpenAIModel       = "gpt-5.2"
	defaultAnthropicModel    = "claude-sonnet-4-0"
	defaultGoogleModel       = "gemini-2.5-pro"
	defaultOpenRouterModel   = "openai/gpt-5"
	defaultCompletionTimeout = 2 * time.Minute

	anthropicThinkingBudgetLow = 1024
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

	provider, err := newFantasyProvider(runtime)
	if err != nil {
		return nil, err
	}

	model, err := provider.LanguageModel(ctx, runtime.Model)
	if err != nil {
		return nil, formatFantasyError(runtime.Provider, err)
	}

	response, err := model.Generate(ctx, buildFantasyCall(runtime, systemPrompt, messages))
	if err != nil {
		return nil, formatFantasyError(runtime.Provider, err)
	}

	content := extractFantasyResponseText(response)
	if content == "" {
		return nil, fmt.Errorf("%s response did not include assistant text", runtime.Provider)
	}

	finishReason := strings.TrimSpace(string(response.FinishReason))
	if finishReason == "" || finishReason == string(fantasy.FinishReasonUnknown) {
		finishReason = "completed"
	}

	return &Completion{
		Provider:          fallbackString(model.Provider(), runtime.Provider),
		Model:             fallbackString(model.Model(), runtime.Model),
		Content:           content,
		ProviderMessageID: extractProviderMessageID(response),
		FinishReason:      finishReason,
	}, nil
}

func newFantasyProvider(runtime *RuntimeConfig) (fantasy.Provider, error) {
	if runtime == nil {
		return nil, fmt.Errorf("AI runtime config is required")
	}

	headers := providerHeaders(runtime)

	switch runtime.Provider {
	case ProviderAnthropic:
		opts := []fantasyanthropic.Option{
			fantasyanthropic.WithName(ProviderAnthropic),
			fantasyanthropic.WithHTTPClient(httpClient),
		}
		if runtime.APIKey != "" {
			opts = append(opts, fantasyanthropic.WithAPIKey(runtime.APIKey))
		}
		if runtime.BaseURL != "" {
			opts = append(opts, fantasyanthropic.WithBaseURL(runtime.BaseURL))
		}
		if len(headers) > 0 {
			opts = append(opts, fantasyanthropic.WithHeaders(headers))
		}
		if runtime.Project != "" && runtime.Location != "" {
			opts = append(opts, fantasyanthropic.WithVertex(runtime.Project, runtime.Location))
		}
		if runtime.SkipAuth {
			opts = append(opts, fantasyanthropic.WithSkipAuth(true))
		}
		if runtime.UseBedrock {
			opts = append(opts, fantasyanthropic.WithBedrock())
		}
		return fantasyanthropic.New(opts...)
	case ProviderGoogle:
		opts := []fantasygoogle.Option{
			fantasygoogle.WithName(ProviderGoogle),
			fantasygoogle.WithHTTPClient(httpClient),
		}
		if runtime.BaseURL != "" {
			opts = append(opts, fantasygoogle.WithBaseURL(runtime.BaseURL))
		}
		if len(headers) > 0 {
			opts = append(opts, fantasygoogle.WithHeaders(headers))
		}
		if runtime.APIKey != "" {
			opts = append(opts, fantasygoogle.WithGeminiAPIKey(runtime.APIKey))
		} else if runtime.Project != "" && runtime.Location != "" {
			opts = append(opts, fantasygoogle.WithVertex(runtime.Project, runtime.Location))
		}
		if runtime.SkipAuth {
			opts = append(opts, fantasygoogle.WithSkipAuth(true))
		}
		return fantasygoogle.New(opts...)
	case ProviderOpenAI:
		opts := []fantasyopenai.Option{
			fantasyopenai.WithName(ProviderOpenAI),
			fantasyopenai.WithHTTPClient(httpClient),
		}
		if runtime.APIKey != "" {
			opts = append(opts, fantasyopenai.WithAPIKey(runtime.APIKey))
		}
		if runtime.BaseURL != "" {
			opts = append(opts, fantasyopenai.WithBaseURL(runtime.BaseURL))
		}
		if len(headers) > 0 {
			opts = append(opts, fantasyopenai.WithHeaders(headers))
		}
		if runtime.Organization != "" {
			opts = append(opts, fantasyopenai.WithOrganization(runtime.Organization))
		}
		if runtime.Project != "" {
			opts = append(opts, fantasyopenai.WithProject(runtime.Project))
		}
		if runtime.UseResponsesAPI {
			opts = append(opts, fantasyopenai.WithUseResponsesAPI())
		}
		return fantasyopenai.New(opts...)
	case ProviderOpenAICompat:
		opts := []fantasyopenaicompat.Option{
			fantasyopenaicompat.WithName(ProviderOpenAICompat),
			fantasyopenaicompat.WithHTTPClient(httpClient),
		}
		if runtime.APIKey != "" {
			opts = append(opts, fantasyopenaicompat.WithAPIKey(runtime.APIKey))
		}
		if runtime.BaseURL != "" {
			opts = append(opts, fantasyopenaicompat.WithBaseURL(runtime.BaseURL))
		}
		if len(headers) > 0 {
			opts = append(opts, fantasyopenaicompat.WithHeaders(headers))
		}
		if runtime.UseResponsesAPI {
			opts = append(opts, fantasyopenaicompat.WithUseResponsesAPI())
		}
		return fantasyopenaicompat.New(opts...)
	case ProviderOpenRouter:
		opts := []fantasyopenrouter.Option{
			fantasyopenrouter.WithName(ProviderOpenRouter),
			fantasyopenrouter.WithHTTPClient(httpClient),
		}
		if runtime.APIKey != "" {
			opts = append(opts, fantasyopenrouter.WithAPIKey(runtime.APIKey))
		}
		if len(headers) > 0 {
			opts = append(opts, fantasyopenrouter.WithHeaders(headers))
		}
		return fantasyopenrouter.New(opts...)
	default:
		return nil, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
	}
}

func buildFantasyCall(runtime *RuntimeConfig, systemPrompt string, messages []providerMessage) fantasy.Call {
	prompt := make(fantasy.Prompt, 0, len(messages)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		prompt = append(prompt, fantasy.Message{
			Role: fantasy.MessageRoleSystem,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: strings.TrimSpace(systemPrompt)},
			},
		})
	}
	for _, message := range messages {
		prompt = append(prompt, fantasy.Message{
			Role: toFantasyMessageRole(message.Role),
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: message.Content},
			},
		})
	}

	call := fantasy.Call{
		Prompt:           prompt,
		ProviderOptions:  fantasyProviderOptions(runtime),
		MaxOutputTokens:  optionalRuntimeInt(runtime.MaxOutputTokens),
		Temperature:      copyOptionalFloat(runtime.Temperature),
		TopP:             copyOptionalFloat(runtime.TopP),
		TopK:             copyOptionalInt(runtime.TopK),
		PresencePenalty:  copyOptionalFloat(runtime.PresencePenalty),
		FrequencyPenalty: copyOptionalFloat(runtime.FrequencyPenalty),
	}
	return call
}

func fantasyProviderOptions(runtime *RuntimeConfig) fantasy.ProviderOptions {
	if runtime == nil {
		return nil
	}

	switch runtime.Provider {
	case ProviderAnthropic:
		if budget := anthropicThinkingBudget(runtime.ThinkingLevel); budget > 0 {
			return fantasyanthropic.NewProviderOptions(&fantasyanthropic.ProviderOptions{
				Thinking: &fantasyanthropic.ThinkingProviderOption{BudgetTokens: budget},
			})
		}
	case ProviderGoogle:
		if thinkingConfig := googleThinkingConfig(runtime.ThinkingLevel); thinkingConfig != nil {
			return fantasy.ProviderOptions{
				fantasygoogle.Name: &fantasygoogle.ProviderOptions{
					ThinkingConfig: thinkingConfig,
				},
			}
		}
	case ProviderOpenAI:
		if effort := openAIReasoningEffort(runtime.ThinkingLevel); effort != nil {
			if runtime.UseResponsesAPI && fantasyopenai.IsResponsesModel(runtime.Model) {
				return fantasyopenai.NewResponsesProviderOptions(&fantasyopenai.ResponsesProviderOptions{
					ReasoningEffort: effort,
				})
			}
			return fantasyopenai.NewProviderOptions(&fantasyopenai.ProviderOptions{
				ReasoningEffort: effort,
			})
		}
	case ProviderOpenAICompat:
		if effort := openAIReasoningEffort(runtime.ThinkingLevel); effort != nil {
			return fantasyopenaicompat.NewProviderOptions(&fantasyopenaicompat.ProviderOptions{
				ReasoningEffort: effort,
			})
		}
	case ProviderOpenRouter:
		if reasoning := openRouterReasoningOptions(runtime.ThinkingLevel); reasoning != nil {
			return fantasyopenrouter.NewProviderOptions(&fantasyopenrouter.ProviderOptions{
				Reasoning: reasoning,
			})
		}
	}

	return nil
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

func openAIReasoningEffort(thinkingLevel string) *fantasyopenai.ReasoningEffort {
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low":
		return fantasyopenai.ReasoningEffortOption(fantasyopenai.ReasoningEffortLow)
	case "medium":
		return fantasyopenai.ReasoningEffortOption(fantasyopenai.ReasoningEffortMedium)
	case "high":
		return fantasyopenai.ReasoningEffortOption(fantasyopenai.ReasoningEffortHigh)
	default:
		return nil
	}
}

func anthropicThinkingBudget(thinkingLevel string) int64 {
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low":
		return anthropicThinkingBudgetLow
	case "medium":
		return anthropicThinkingBudgetLow * 2
	case "high":
		return anthropicThinkingBudgetLow * 4
	default:
		return 0
	}
}

func googleThinkingConfig(thinkingLevel string) *fantasygoogle.ThinkingConfig {
	var budget int64
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low":
		budget = 1024
	case "medium":
		budget = 2048
	case "high":
		budget = 4096
	default:
		return nil
	}

	return &fantasygoogle.ThinkingConfig{
		ThinkingBudget: fantasy.Opt(budget),
	}
}

func openRouterReasoningOptions(thinkingLevel string) *fantasyopenrouter.ReasoningOptions {
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low":
		return &fantasyopenrouter.ReasoningOptions{
			Enabled: fantasy.Opt(true),
			Effort:  fantasyopenrouter.ReasoningEffortOption(fantasyopenrouter.ReasoningEffortLow),
		}
	case "medium":
		return &fantasyopenrouter.ReasoningOptions{
			Enabled: fantasy.Opt(true),
			Effort:  fantasyopenrouter.ReasoningEffortOption(fantasyopenrouter.ReasoningEffortMedium),
		}
	case "high":
		return &fantasyopenrouter.ReasoningOptions{
			Enabled: fantasy.Opt(true),
			Effort:  fantasyopenrouter.ReasoningEffortOption(fantasyopenrouter.ReasoningEffortHigh),
		}
	default:
		return nil
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

func extractFantasyResponseText(response *fantasy.Response) string {
	if response == nil {
		return ""
	}
	blocks := make([]string, 0, len(response.Content))
	for _, content := range response.Content {
		if content == nil || content.GetType() != fantasy.ContentTypeText {
			continue
		}
		textContent, ok := fantasy.AsContentType[fantasy.TextContent](content)
		if !ok {
			continue
		}
		if strings.TrimSpace(textContent.Text) == "" {
			continue
		}
		blocks = append(blocks, textContent.Text)
	}
	return joinTextBlocks(blocks)
}

func extractProviderMessageID(_ *fantasy.Response) string {
	// Fantasy's unified response model does not currently expose a stable top-level
	// provider message ID for plain text generations.
	return ""
}

func formatFantasyError(provider string, err error) error {
	if err == nil {
		return nil
	}

	var providerErr *fantasy.ProviderError
	if errors.As(err, &providerErr) {
		message := strings.TrimSpace(providerErr.Error())
		if providerErr.StatusCode > 0 {
			if message == "" {
				message = http.StatusText(providerErr.StatusCode)
			}
			return fmt.Errorf("%s API request failed with HTTP %d: %s", provider, providerErr.StatusCode, truncateForError(message))
		}
		if message != "" {
			return fmt.Errorf("%s API request failed: %s", provider, truncateForError(message))
		}
	}

	return err
}

func toFantasyMessageRole(role string) fantasy.MessageRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return fantasy.MessageRoleAssistant
	default:
		return fantasy.MessageRoleUser
	}
}

func optionalRuntimeInt(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return fantasy.Opt(value)
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
