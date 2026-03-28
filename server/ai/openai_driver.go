package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	openairesponses "github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
)

const (
	openAIDriverName         = "openai-go"
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
)

type openAIDriver struct{}

type openAIClient struct {
	chat      openai.ChatCompletionService
	responses openairesponses.ResponseService
}

func newOpenAIDriver() sdkDriver {
	return &openAIDriver{}
}

func (d *openAIDriver) Name() string {
	return openAIDriverName
}

func (d *openAIDriver) Supports(runtime *RuntimeConfig) bool {
	if runtime == nil {
		return false
	}

	switch NormalizeProviderName(runtime.Provider) {
	case ProviderOpenAI, ProviderOpenAICompat, ProviderOpenRouter:
		return true
	default:
		return false
	}
}

func (d *openAIDriver) CompleteConversation(ctx context.Context, runtime *RuntimeConfig, request *completionRequest) (*Completion, error) {
	if runtime == nil {
		return nil, fmt.Errorf("AI runtime config is required")
	}
	if request == nil {
		return nil, fmt.Errorf("AI completion request is required")
	}

	client := newOpenAIClient(openAIRequestOptions(runtime)...)
	if runtime.UseResponsesAPI {
		return d.completeResponses(ctx, client, runtime, request)
	}
	return d.completeChat(ctx, client, runtime, request)
}

func newOpenAIClient(opts ...option.RequestOption) *openAIClient {
	return &openAIClient{
		chat:      openai.NewChatCompletionService(opts...),
		responses: openairesponses.NewResponseService(opts...),
	}
}

func openAIRequestOptions(runtime *RuntimeConfig) []option.RequestOption {
	opts := []option.RequestOption{
		option.WithHTTPClient(httpClient),
		option.WithRequestTimeout(defaultCompletionTimeout),
		option.WithMaxRetries(0),
	}

	if runtime != nil && NormalizeProviderName(runtime.Provider) == ProviderOpenAI {
		opts = append(opts, option.WithEnvironmentProduction())
	}
	if baseURL := openAIBaseURL(runtime); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if runtime != nil {
		if runtime.APIKey != "" {
			opts = append(opts, option.WithAPIKey(runtime.APIKey))
		}
		if runtime.Organization != "" {
			opts = append(opts, option.WithOrganization(runtime.Organization))
		}
		if runtime.Project != "" {
			opts = append(opts, option.WithProject(runtime.Project))
		}
		for key, value := range providerHeaders(runtime) {
			opts = append(opts, option.WithHeader(key, value))
		}
		if runtime.TopK != nil && NormalizeProviderName(runtime.Provider) != ProviderOpenAI {
			opts = append(opts, option.WithJSONSet("top_k", *runtime.TopK))
		}
	}

	return opts
}

func openAIBaseURL(runtime *RuntimeConfig) string {
	if runtime == nil {
		return ""
	}

	if runtime.BaseURL != "" {
		return runtime.BaseURL
	}
	if NormalizeProviderName(runtime.Provider) == ProviderOpenRouter {
		return defaultOpenRouterBaseURL
	}
	return ""
}

func (d *openAIDriver) completeResponses(ctx context.Context, client *openAIClient, runtime *RuntimeConfig, request *completionRequest) (*Completion, error) {
	params := openairesponses.ResponseNewParams{
		Input: openairesponses.ResponseNewParamsInputUnion{
			OfInputItemList: buildResponseInput(request.SystemPrompt, request.Messages),
		},
		Model: shared.ResponsesModel(runtime.Model),
	}
	if runtime.MaxOutputTokens > 0 {
		params.MaxOutputTokens = openai.Int(runtime.MaxOutputTokens)
	}
	if runtime.Temperature != nil {
		params.Temperature = openai.Float(*runtime.Temperature)
	}
	if runtime.TopP != nil {
		params.TopP = openai.Float(*runtime.TopP)
	}
	if reasoning, ok := responseReasoningParam(runtime); ok {
		params.Reasoning = reasoning
	}

	response, err := client.responses.New(ctx, params)
	if err != nil {
		return nil, formatOpenAIError(runtime.Provider, err)
	}

	content := strings.TrimSpace(response.OutputText())
	if content == "" {
		return nil, fmt.Errorf("%s response did not include assistant text", runtime.Provider)
	}

	return &Completion{
		Provider:          runtime.Provider,
		Model:             fallbackString(response.Model, runtime.Model),
		Content:           content,
		ProviderMessageID: strings.TrimSpace(response.ID),
		FinishReason:      responseFinishReason(response),
	}, nil
}

func (d *openAIDriver) completeChat(ctx context.Context, client *openAIClient, runtime *RuntimeConfig, request *completionRequest) (*Completion, error) {
	params := openai.ChatCompletionNewParams{
		Messages: buildChatMessages(request.SystemPrompt, request.Messages),
		Model:    shared.ChatModel(runtime.Model),
	}
	if runtime.MaxOutputTokens > 0 {
		if NormalizeProviderName(runtime.Provider) == ProviderOpenAI {
			params.MaxCompletionTokens = openai.Int(runtime.MaxOutputTokens)
		} else {
			params.MaxTokens = openai.Int(runtime.MaxOutputTokens)
		}
	}
	if runtime.Temperature != nil {
		params.Temperature = openai.Float(*runtime.Temperature)
	}
	if runtime.TopP != nil {
		params.TopP = openai.Float(*runtime.TopP)
	}
	if runtime.PresencePenalty != nil {
		params.PresencePenalty = openai.Float(*runtime.PresencePenalty)
	}
	if runtime.FrequencyPenalty != nil {
		params.FrequencyPenalty = openai.Float(*runtime.FrequencyPenalty)
	}
	if effort, ok := reasoningEffortForThinkingLevel(runtime.ThinkingLevel); ok {
		params.ReasoningEffort = effort
	}

	response, err := client.chat.New(ctx, params)
	if err != nil {
		return nil, formatOpenAIError(runtime.Provider, err)
	}

	content, finishReason := extractChatCompletionResult(response)
	if content == "" {
		return nil, fmt.Errorf("%s response did not include assistant text", runtime.Provider)
	}

	return &Completion{
		Provider:          runtime.Provider,
		Model:             fallbackString(response.Model, runtime.Model),
		Content:           content,
		ProviderMessageID: strings.TrimSpace(response.ID),
		FinishReason:      finishReason,
	}, nil
}

func buildResponseInput(systemPrompt string, messages []providerMessage) openairesponses.ResponseInputParam {
	input := make(openairesponses.ResponseInputParam, 0, len(messages)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		input = append(input, openairesponses.ResponseInputItemParamOfMessage(strings.TrimSpace(systemPrompt), openairesponses.EasyInputMessageRoleSystem))
	}
	for _, message := range messages {
		input = append(input, openairesponses.ResponseInputItemParamOfMessage(message.Content, responseMessageRole(message.Role)))
	}
	return input
}

func buildChatMessages(systemPrompt string, messages []providerMessage) []openai.ChatCompletionMessageParamUnion {
	chatMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		chatMessages = append(chatMessages, openai.SystemMessage(strings.TrimSpace(systemPrompt)))
	}
	for _, message := range messages {
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "assistant":
			chatMessages = append(chatMessages, openai.AssistantMessage(message.Content))
		default:
			chatMessages = append(chatMessages, openai.UserMessage(message.Content))
		}
	}
	return chatMessages
}

func responseMessageRole(role string) openairesponses.EasyInputMessageRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return openairesponses.EasyInputMessageRoleAssistant
	default:
		return openairesponses.EasyInputMessageRoleUser
	}
}

func extractChatCompletionResult(response *openai.ChatCompletion) (string, string) {
	if response == nil || len(response.Choices) == 0 {
		return "", ""
	}

	for _, choice := range response.Choices {
		content := strings.TrimSpace(choice.Message.Content)
		if content == "" {
			continue
		}
		finishReason := strings.TrimSpace(choice.FinishReason)
		if finishReason == "" {
			finishReason = "completed"
		}
		return content, finishReason
	}

	return "", ""
}

func responseFinishReason(response *openairesponses.Response) string {
	if response == nil {
		return ""
	}

	if response.Status == openairesponses.ResponseStatusIncomplete && strings.TrimSpace(response.IncompleteDetails.Reason) != "" {
		return strings.TrimSpace(response.IncompleteDetails.Reason)
	}

	finishReason := strings.TrimSpace(string(response.Status))
	if finishReason == "" {
		return "completed"
	}
	return finishReason
}

func reasoningEffortForThinkingLevel(thinkingLevel string) (shared.ReasoningEffort, bool) {
	switch strings.ToLower(strings.TrimSpace(thinkingLevel)) {
	case "low":
		return shared.ReasoningEffortLow, true
	case "medium":
		return shared.ReasoningEffortMedium, true
	case "high":
		return shared.ReasoningEffortHigh, true
	case "xhigh":
		return shared.ReasoningEffort("xhigh"), true
	default:
		return "", false
	}
}

func responseReasoningParam(runtime *RuntimeConfig) (shared.ReasoningParam, bool) {
	effort, ok := reasoningEffortForThinkingLevel(runtime.ThinkingLevel)
	if !ok {
		return shared.ReasoningParam{}, false
	}

	reasoning := shared.ReasoningParam{Effort: effort}
	switch NormalizeProviderName(runtime.Provider) {
	case ProviderOpenAI, ProviderOpenRouter:
		// Native OpenAI-style Responses backends can return a reasoning summary
		// for UI-only transcript blocks. Use "auto" because summary modes are
		// model-specific; hard-coding "concise" can suppress summaries on models
		// that only expose a different summarizer.
		reasoning.Summary = shared.ReasoningSummaryAuto
	}

	return reasoning, true
}

func formatOpenAIError(provider string, err error) error {
	if err == nil {
		return nil
	}

	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		message := strings.TrimSpace(apiErr.Message)
		if message == "" {
			message = strings.TrimSpace(apiErr.RawJSON())
		}
		if message == "" {
			message = strings.TrimSpace(err.Error())
		}
		if apiErr.StatusCode > 0 {
			return fmt.Errorf("%s API request failed with HTTP %d: %s", provider, apiErr.StatusCode, truncateForError(message))
		}
		if message != "" {
			return fmt.Errorf("%s API request failed: %s", provider, truncateForError(message))
		}
	}

	return err
}
