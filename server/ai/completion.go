package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/configs"
)

const (
	defaultOpenAIBaseURL       = "https://api.openai.com/v1"
	defaultOpenAIModel         = "gpt-5.2"
	openAIResponsesPath        = "/responses"
	defaultAnthropicBaseURL    = "https://api.anthropic.com"
	defaultAnthropicModel      = "claude-sonnet-4-0"
	anthropicMessagesPath      = "/v1/messages"
	anthropicAPIVersion        = "2023-06-01"
	defaultCompletionTimeout   = 2 * time.Minute
	defaultAnthropicMaxTokens  = 8192
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
	Provider      string
	APIKey        string
	BaseURL       string
	Model         string
	ThinkingLevel string
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

type openAIRequest struct {
	Model     string           `json:"model"`
	Input     []openAIMessage  `json:"input"`
	Reasoning *openAIReasoning `json:"reasoning,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIReasoning struct {
	Effort string `json:"effort,omitempty"`
}

type openAIResponse struct {
	ID         string                 `json:"id"`
	Model      string                 `json:"model"`
	Status     string                 `json:"status"`
	OutputText string                 `json:"output_text"`
	Output     []openAIResponseOutput `json:"output"`
}

type openAIResponseOutput struct {
	ID      string                   `json:"id"`
	Type    string                   `json:"type"`
	Role    string                   `json:"role"`
	Status  string                   `json:"status"`
	Content []map[string]interface{} `json:"content"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Thinking  *anthropicThinking `json:"thinking,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type anthropicResponse struct {
	ID         string                   `json:"id"`
	Model      string                   `json:"model"`
	StopReason string                   `json:"stop_reason"`
	Content    []map[string]interface{} `json:"content"`
}

type anthropicErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ResolveRuntimeConfig derives the provider settings for a stored conversation.
func ResolveRuntimeConfig(cfg *configs.ServerConfig, conversation *clientpb.AIConversation) (*RuntimeConfig, error) {
	runtime := &RuntimeConfig{
		ThinkingLevel: normalizeThinkingLevel(cfg),
	}

	if conversation != nil {
		runtime.Provider = NormalizeProviderName(conversation.GetProvider())
		runtime.Model = strings.TrimSpace(conversation.GetModel())
	}

	if runtime.Provider == "" {
		selectedProvider, providerConfig := selectedProviderConfig(cfg)
		runtime.Provider = selectedProvider
		if providerConfig != nil {
			runtime.APIKey = strings.TrimSpace(providerConfig.APIKey)
			runtime.BaseURL = strings.TrimSpace(providerConfig.BaseURL)
		}
	} else {
		if !IsSupportedProvider(runtime.Provider) {
			return runtime, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
		}
		providerConfig := aiProviderConfig(cfg, runtime.Provider)
		if providerConfig != nil {
			runtime.APIKey = strings.TrimSpace(providerConfig.APIKey)
			runtime.BaseURL = strings.TrimSpace(providerConfig.BaseURL)
		}
	}

	if runtime.Model == "" && cfg != nil && cfg.AI != nil {
		runtime.Model = strings.TrimSpace(cfg.AI.Model)
	}
	if runtime.Model == "" {
		runtime.Model = defaultModelForProvider(runtime.Provider)
	}

	switch {
	case runtime.Provider == "":
		return runtime, fmt.Errorf("server AI provider is not configured; run `ai-config` on the server")
	case !IsSupportedProvider(runtime.Provider):
		return runtime, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
	case runtime.APIKey == "":
		return runtime, fmt.Errorf("server AI provider %q is missing an API key; run `ai-config` on the server", runtime.Provider)
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

	switch runtime.Provider {
	case ProviderOpenAI:
		return completeWithOpenAI(ctx, runtime, systemPrompt, messages)
	case ProviderAnthropic:
		return completeWithAnthropic(ctx, runtime, systemPrompt, messages)
	default:
		return nil, fmt.Errorf("unsupported AI provider %q", runtime.Provider)
	}
}

func completeWithOpenAI(ctx context.Context, runtime *RuntimeConfig, systemPrompt string, messages []providerMessage) (*Completion, error) {
	endpoint, err := resolveProviderEndpoint(runtime.BaseURL, defaultOpenAIBaseURL, openAIResponsesPath)
	if err != nil {
		return nil, err
	}

	input := make([]openAIMessage, 0, len(messages)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		input = append(input, openAIMessage{Role: "system", Content: systemPrompt})
	}
	for _, message := range messages {
		input = append(input, openAIMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	reqBody := &openAIRequest{
		Model: runtime.Model,
		Input: input,
	}
	if effort := openAIReasoningEffort(runtime.ThinkingLevel); effort != "" {
		reqBody.Reasoning = &openAIReasoning{Effort: effort}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+runtime.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, providerHTTPError("openai", resp.StatusCode, body, parseOpenAIError)
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}

	content := strings.TrimSpace(parsed.OutputText)
	if content == "" {
		content = joinTextBlocks(extractOpenAIOutputText(parsed.Output))
	}
	if content == "" {
		return nil, fmt.Errorf("openai response did not include assistant text")
	}

	finishReason := strings.TrimSpace(parsed.Status)
	if finishReason == "" {
		finishReason = "completed"
	}

	return &Completion{
		Provider:          runtime.Provider,
		Model:             fallbackString(strings.TrimSpace(parsed.Model), runtime.Model),
		Content:           content,
		ProviderMessageID: strings.TrimSpace(parsed.ID),
		FinishReason:      finishReason,
	}, nil
}

func completeWithAnthropic(ctx context.Context, runtime *RuntimeConfig, systemPrompt string, messages []providerMessage) (*Completion, error) {
	endpoint, err := resolveProviderEndpoint(runtime.BaseURL, defaultAnthropicBaseURL, anthropicMessagesPath)
	if err != nil {
		return nil, err
	}

	reqBody := &anthropicRequest{
		Model:     runtime.Model,
		System:    strings.TrimSpace(systemPrompt),
		MaxTokens: defaultAnthropicMaxTokens,
		Messages:  make([]anthropicMessage, 0, len(messages)),
	}
	for _, message := range messages {
		reqBody.Messages = append(reqBody.Messages, anthropicMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	if budget := anthropicThinkingBudget(runtime.ThinkingLevel); budget > 0 {
		reqBody.Thinking = &anthropicThinking{
			Type:         "enabled",
			BudgetTokens: budget,
		}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", runtime.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, providerHTTPError("anthropic", resp.StatusCode, body, parseAnthropicError)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	content := joinTextBlocks(extractTypedTextBlocks(parsed.Content, "text"))
	if content == "" {
		return nil, fmt.Errorf("anthropic response did not include assistant text")
	}

	finishReason := strings.TrimSpace(parsed.StopReason)
	if finishReason == "" {
		finishReason = "completed"
	}

	return &Completion{
		Provider:          runtime.Provider,
		Model:             fallbackString(strings.TrimSpace(parsed.Model), runtime.Model),
		Content:           content,
		ProviderMessageID: strings.TrimSpace(parsed.ID),
		FinishReason:      finishReason,
	}, nil
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

func openAIReasoningEffort(thinkingLevel string) string {
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(thinkingLevel))
	default:
		return ""
	}
}

func anthropicThinkingBudget(thinkingLevel string) int {
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

func resolveProviderEndpoint(baseURL string, defaultBase string, requiredPath string) (string, error) {
	rawURL := strings.TrimSpace(baseURL)
	if rawURL == "" {
		rawURL = defaultBase
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid AI provider base URL %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid AI provider base URL %q", rawURL)
	}

	if !strings.HasSuffix(parsed.Path, requiredPath) {
		if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
			parsed.Path = requiredPath
		} else {
			parsed.Path = strings.TrimRight(parsed.Path, "/") + requiredPath
		}
	}
	parsed.Fragment = ""

	return parsed.String(), nil
}

func extractOpenAIOutputText(output []openAIResponseOutput) []string {
	var blocks []string
	for _, item := range output {
		if strings.ToLower(strings.TrimSpace(item.Type)) != "message" {
			continue
		}
		blocks = append(blocks, extractTypedTextBlocks(item.Content, "output_text", "text")...)
	}
	return blocks
}

func extractTypedTextBlocks(items []map[string]interface{}, allowedTypes ...string) []string {
	if len(items) == 0 {
		return nil
	}

	allowed := map[string]struct{}{}
	for _, allowedType := range allowedTypes {
		allowed[strings.ToLower(strings.TrimSpace(allowedType))] = struct{}{}
	}

	blocks := make([]string, 0, len(items))
	for _, item := range items {
		itemType := strings.ToLower(strings.TrimSpace(asString(item["type"])))
		if len(allowed) > 0 {
			if _, ok := allowed[itemType]; !ok {
				continue
			}
		}

		text := strings.TrimSpace(asString(item["text"]))
		if text == "" {
			if nested, ok := item["text"].(map[string]interface{}); ok {
				text = strings.TrimSpace(asString(nested["value"]))
			}
		}
		if text == "" {
			continue
		}
		blocks = append(blocks, text)
	}
	return blocks
}

func providerHTTPError(provider string, statusCode int, body []byte, parser func([]byte) string) error {
	message := strings.TrimSpace(parser(body))
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return fmt.Errorf("%s API request failed with HTTP %d: %s", provider, statusCode, truncateForError(message))
}

func parseOpenAIError(body []byte) string {
	var parsed openAIErrorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return parsed.Error.Message
}

func parseAnthropicError(body []byte) string {
	var parsed anthropicErrorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return parsed.Error.Message
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

func asString(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}

func fallbackString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}
