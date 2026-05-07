package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	openai "github.com/openai/openai-go/v2"
	openairesponses "github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
)

const maxAgenticLoopIterations = 100

// AgenticToolDefinition describes a custom function tool exposed to the model.
type AgenticToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// AgenticToolExecutor executes model-requested tools for a conversation turn.
type AgenticToolExecutor interface {
	ToolDefinitions() []AgenticToolDefinition
	CallTool(context.Context, string, string) (string, error)
}

// AgenticEventSink persists and publishes non-context-window UI items.
type AgenticEventSink interface {
	ChatMessage(context.Context, AgenticChatMessage) error
	ReasoningItem(context.Context, AgenticReasoningItem) error
	ToolCallStarted(context.Context, AgenticToolCall) error
	ToolCallCompleted(context.Context, AgenticToolCallResult) error
}

// AgenticChatMessage is a provider-emitted assistant or system chat block.
type AgenticChatMessage struct {
	ItemID           string
	Role             string
	Content          string
	Status           string
	UIOnly           bool
	IncludeInContext bool
}

// AgenticReasoningItem is a completed reasoning item emitted by the provider.
type AgenticReasoningItem struct {
	ItemID           string
	Summary          string
	Content          string
	EncryptedContent string
	Status           string
}

// AgenticToolCall is a provider-emitted function tool call.
type AgenticToolCall struct {
	ItemID    string
	CallID    string
	Name      string
	Arguments string
	Status    string
}

// AgenticToolCallResult is the resolved tool call state after local execution.
type AgenticToolCallResult struct {
	AgenticToolCall
	Output string
	Error  error
}

// SupportsAgenticConversation reports whether the configured runtime can run the
// Responses API tool loop used by the Sliver AI chat.
func SupportsAgenticConversation(runtime *RuntimeConfig) bool {
	if runtime == nil || !runtime.UseResponsesAPI {
		return false
	}

	switch NormalizeProviderName(runtime.Provider) {
	case ProviderOpenAI, ProviderOpenAICompat, ProviderOpenRouter:
		return true
	default:
		return false
	}
}

// CompleteConversationAgentic runs a Codex-style Responses API loop that emits
// reasoning and tool-call items between the user prompt and the final assistant
// reply. The returned completion is the assistant message that should be stored
// in the conversation context window.
func CompleteConversationAgentic(
	ctx context.Context,
	runtime *RuntimeConfig,
	conversation *clientpb.AIConversation,
	tools AgenticToolExecutor,
	sink AgenticEventSink,
) (*Completion, error) {
	if runtime == nil {
		return nil, fmt.Errorf("AI runtime config is required")
	}
	if conversation == nil {
		return nil, fmt.Errorf("AI conversation is required")
	}
	if !SupportsAgenticConversation(runtime) {
		return nil, fmt.Errorf("AI provider %q does not support the agentic Responses API loop", runtime.Provider)
	}

	systemPrompt, messages, err := conversationHistory(conversation)
	if err != nil {
		return nil, err
	}

	client := newOpenAIClient(openAIRequestOptions(runtime)...)
	currentInput := buildResponseInput(systemPrompt, messages)
	toolParams := appendOpenAIResponseTools(runtime, buildAgenticToolParams(tools))
	previousResponseID := ""
	var maxUsage *ContextWindowUsage

	for step := 0; step < maxAgenticLoopIterations; step++ {
		params := openairesponses.ResponseNewParams{
			Input: openairesponses.ResponseNewParamsInputUnion{
				OfInputItemList: currentInput,
			},
			Model: shared.ResponsesModel(runtime.Model),
		}
		if previousResponseID != "" {
			params.PreviousResponseID = openai.String(previousResponseID)
		}
		if len(toolParams) > 0 {
			params.Tools = toolParams
			params.ParallelToolCalls = openai.Bool(false)
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

		maxUsage = mergeContextWindowUsage(maxUsage, resolveContextWindowUsage(
			ctx,
			runtime,
			fallbackString(response.Model, runtime.Model),
			response.Usage.InputTokens,
			response.Usage.OutputTokens,
			response.Usage.TotalTokens,
		))
		previousResponseID = strings.TrimSpace(response.ID)
		nextInput := make(openairesponses.ResponseInputParam, 0)
		finalTexts := make([]string, 0, len(response.Output))
		chatItems := make([]AgenticChatMessage, 0)

		for _, item := range response.Output {
			switch output := item.AsAny().(type) {
			case openairesponses.ResponseReasoningItem:
				if sink != nil {
					if err := sink.ReasoningItem(ctx, AgenticReasoningItem{
						ItemID:           strings.TrimSpace(output.ID),
						Summary:          responseReasoningSummary(output),
						Content:          responseReasoningContent(output),
						EncryptedContent: strings.TrimSpace(output.EncryptedContent),
						Status:           strings.TrimSpace(string(output.Status)),
					}); err != nil {
						return nil, err
					}
				}

			case openairesponses.ResponseFunctionToolCall:
				toolCall := AgenticToolCall{
					ItemID:    strings.TrimSpace(output.ID),
					CallID:    strings.TrimSpace(output.CallID),
					Name:      strings.TrimSpace(output.Name),
					Arguments: strings.TrimSpace(output.Arguments),
					Status:    strings.TrimSpace(string(output.Status)),
				}
				if sink != nil {
					if err := sink.ToolCallStarted(ctx, toolCall); err != nil {
						return nil, err
					}
				}

				toolOutput, toolErr := agenticToolOutput(ctx, tools, toolCall)
				if sink != nil {
					if err := sink.ToolCallCompleted(ctx, AgenticToolCallResult{
						AgenticToolCall: toolCall,
						Output:          toolOutput,
						Error:           toolErr,
					}); err != nil {
						return nil, err
					}
				}

				nextInput = append(nextInput, openairesponses.ResponseInputItemParamOfFunctionCallOutput(toolCall.CallID, toolOutput))

			case openairesponses.ResponseOutputMessage:
				if text := strings.TrimSpace(responseOutputMessageText(output)); text != "" {
					chatItems = append(chatItems, AgenticChatMessage{
						ItemID:           strings.TrimSpace(output.ID),
						Role:             strings.TrimSpace(string(output.Role)),
						Content:          text,
						Status:           strings.TrimSpace(string(output.Status)),
						IncludeInContext: true,
					})
					finalTexts = append(finalTexts, text)
				}
			}
		}

		if len(nextInput) > 0 {
			if sink != nil {
				for _, chatItem := range chatItems {
					chatItem.UIOnly = true
					chatItem.IncludeInContext = false
					if err := sink.ChatMessage(ctx, chatItem); err != nil {
						return nil, err
					}
				}
			}
			currentInput = nextInput
			continue
		}

		finalText := joinTextBlocks(finalTexts)
		if finalText == "" {
			finalText = strings.TrimSpace(response.OutputText())
		}
		if finalText == "" {
			return nil, fmt.Errorf("%s response did not include assistant text", runtime.Provider)
		}

		return &Completion{
			Provider:           runtime.Provider,
			Model:              fallbackString(response.Model, runtime.Model),
			Content:            finalText,
			ProviderMessageID:  strings.TrimSpace(response.ID),
			FinishReason:       responseFinishReason(response),
			ContextWindowUsage: maxUsage,
		}, nil
	}

	return nil, fmt.Errorf("AI agent loop exceeded %d provider steps", maxAgenticLoopIterations)
}

func buildAgenticToolParams(tools AgenticToolExecutor) []openairesponses.ToolUnionParam {
	if tools == nil {
		return nil
	}

	definitions := tools.ToolDefinitions()
	if len(definitions) == 0 {
		return nil
	}

	params := make([]openairesponses.ToolUnionParam, 0, len(definitions))
	for _, definition := range definitions {
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			continue
		}
		tool := openairesponses.ToolParamOfFunction(name, normalizeOpenAIStrictSchema(definition.Parameters), true)
		if description := strings.TrimSpace(definition.Description); description != "" && tool.OfFunction != nil {
			tool.OfFunction.Description = openai.String(description)
		}
		params = append(params, tool)
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func normalizeOpenAIStrictSchema(schema map[string]any) map[string]any {
	if len(schema) == 0 {
		return map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"required":             []string{},
			"additionalProperties": false,
		}
	}

	cloned, ok := cloneJSONValue(schema).(map[string]any)
	if !ok || cloned == nil {
		return schema
	}

	normalizeOpenAIStrictSchemaNode(cloned)
	return cloned
}

func normalizeOpenAIStrictSchemaNode(node map[string]any) {
	if node == nil {
		return
	}

	if properties, ok := schemaProperties(node); ok {
		required := schemaRequiredSet(node["required"])
		names := make([]string, 0, len(properties))
		for name, rawProperty := range properties {
			names = append(names, name)

			property, ok := rawProperty.(map[string]any)
			if !ok || property == nil {
				continue
			}
			if !required[name] {
				makeJSONSchemaNullable(property)
			}
			normalizeOpenAIStrictSchemaNode(property)
		}
		sort.Strings(names)
		node["required"] = names
		if _, exists := node["additionalProperties"]; !exists {
			node["additionalProperties"] = false
		}
	}

	if items, ok := node["items"].(map[string]any); ok {
		normalizeOpenAIStrictSchemaNode(items)
	}

	for _, key := range []string{"anyOf", "allOf", "oneOf"} {
		variants, ok := node[key].([]any)
		if !ok {
			continue
		}
		for _, rawVariant := range variants {
			variant, ok := rawVariant.(map[string]any)
			if !ok || variant == nil {
				continue
			}
			normalizeOpenAIStrictSchemaNode(variant)
		}
	}
}

func schemaProperties(node map[string]any) (map[string]any, bool) {
	if node == nil {
		return nil, false
	}

	rawProperties, ok := node["properties"]
	if !ok {
		return nil, false
	}

	properties, ok := rawProperties.(map[string]any)
	if !ok {
		return nil, false
	}
	return properties, true
}

func schemaRequiredSet(raw any) map[string]bool {
	required := map[string]bool{}
	switch values := raw.(type) {
	case []string:
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				required[value] = true
			}
		}
	case []any:
		for _, rawValue := range values {
			value, ok := rawValue.(string)
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			if value != "" {
				required[value] = true
			}
		}
	}
	return required
}

func makeJSONSchemaNullable(node map[string]any) {
	if node == nil {
		return
	}

	switch value := node["type"].(type) {
	case string:
		value = strings.TrimSpace(value)
		if value != "" && value != "null" {
			node["type"] = []string{value, "null"}
		}
	case []string:
		if !stringSliceContains(value, "null") {
			node["type"] = append(append([]string(nil), value...), "null")
		}
	case []any:
		if !anyStringSliceContains(value, "null") {
			node["type"] = append(append([]any(nil), value...), "null")
		}
	default:
		if _, ok := node["anyOf"]; ok {
			appendSchemaVariant(node, "anyOf", map[string]any{"type": "null"})
		} else if _, ok := node["properties"]; ok {
			node["type"] = []string{"object", "null"}
		} else if _, ok := node["items"]; ok {
			node["type"] = []string{"array", "null"}
		}
	}

	if enum, ok := nullableEnum(node["enum"]); ok {
		node["enum"] = enum
	}
}

func appendSchemaVariant(node map[string]any, key string, variant map[string]any) {
	if node == nil || variant == nil {
		return
	}

	rawVariants, ok := node[key].([]any)
	if !ok {
		node[key] = []any{variant}
		return
	}
	for _, rawVariant := range rawVariants {
		existing, ok := rawVariant.(map[string]any)
		if !ok || existing == nil {
			continue
		}
		if len(existing) == len(variant) {
			matched := true
			for variantKey, variantValue := range variant {
				if existing[variantKey] != variantValue {
					matched = false
					break
				}
			}
			if matched {
				return
			}
		}
	}
	node[key] = append(rawVariants, variant)
}

func nullableEnum(raw any) (any, bool) {
	switch values := raw.(type) {
	case []any:
		for _, value := range values {
			if value == nil {
				return values, true
			}
		}
		return append(append([]any(nil), values...), nil), true
	case []string:
		enum := make([]any, 0, len(values)+1)
		for _, value := range values {
			enum = append(enum, value)
		}
		enum = append(enum, nil)
		return enum, true
	default:
		return nil, false
	}
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func anyStringSliceContains(values []any, needle string) bool {
	for _, rawValue := range values {
		value, ok := rawValue.(string)
		if ok && strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nested := range typed {
			cloned[key] = cloneJSONValue(nested)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for idx, nested := range typed {
			cloned[idx] = cloneJSONValue(nested)
		}
		return cloned
	case []string:
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return typed
	}
}

func responseOutputMessageText(message openairesponses.ResponseOutputMessage) string {
	parts := make([]string, 0, len(message.Content))
	for _, content := range message.Content {
		switch part := content.AsAny().(type) {
		case openairesponses.ResponseOutputText:
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		case openairesponses.ResponseOutputRefusal:
			if refusal := strings.TrimSpace(part.Refusal); refusal != "" {
				parts = append(parts, refusal)
			}
		}
	}
	return joinTextBlocks(parts)
}

func responseReasoningSummary(item openairesponses.ResponseReasoningItem) string {
	summaries := make([]string, 0, len(item.Summary))
	for _, summary := range item.Summary {
		if text := strings.TrimSpace(summary.Text); text != "" {
			summaries = append(summaries, text)
		}
	}
	return joinTextBlocks(summaries)
}

func responseReasoningContent(item openairesponses.ResponseReasoningItem) string {
	content := make([]string, 0, len(item.Content))
	for _, block := range item.Content {
		if text := strings.TrimSpace(block.Text); text != "" {
			content = append(content, text)
		}
	}
	return joinTextBlocks(content)
}

func agenticToolOutput(ctx context.Context, tools AgenticToolExecutor, toolCall AgenticToolCall) (string, error) {
	if tools == nil {
		err := fmt.Errorf("tool executor is not configured")
		return agenticToolErrorJSON(err), err
	}

	output, err := tools.CallTool(ctx, toolCall.Name, toolCall.Arguments)
	if err != nil {
		return agenticToolErrorJSON(err), err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		empty, marshalErr := json.Marshal(map[string]any{"ok": true})
		if marshalErr != nil {
			return `{"ok":true}`, nil
		}
		return string(empty), nil
	}
	return output, nil
}

func agenticToolErrorJSON(err error) string {
	payload := map[string]any{
		"ok":    false,
		"error": "tool call failed",
	}
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		payload["error"] = truncateForError(err.Error())
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return `{"ok":false,"error":"tool call failed"}`
	}
	return string(data)
}
