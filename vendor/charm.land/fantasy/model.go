package fantasy

import (
	"context"
	"fmt"
	"iter"
	"strings"
)

// Usage represents token usage statistics for a model call.
type Usage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	TotalTokens         int64 `json:"total_tokens"`
	ReasoningTokens     int64 `json:"reasoning_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
}

func (u Usage) String() string {
	return fmt.Sprintf("Usage{Input: %d, Output: %d, Total: %d, Reasoning: %d, CacheCreation: %d, CacheRead: %d}",
		u.InputTokens,
		u.OutputTokens,
		u.TotalTokens,
		u.ReasoningTokens,
		u.CacheCreationTokens,
		u.CacheReadTokens,
	)
}

// ResponseContent represents the content of a model response.
type ResponseContent []Content

// Text returns the text content of the response.
func (r ResponseContent) Text() string {
	for _, c := range r {
		if c.GetType() == ContentTypeText {
			if textContent, ok := AsContentType[TextContent](c); ok {
				return textContent.Text
			}
		}
	}
	return ""
}

// Reasoning returns all reasoning content parts.
func (r ResponseContent) Reasoning() []ReasoningContent {
	var reasoning []ReasoningContent
	for _, c := range r {
		if c.GetType() == ContentTypeReasoning {
			if reasoningContent, ok := AsContentType[ReasoningContent](c); ok {
				reasoning = append(reasoning, reasoningContent)
			}
		}
	}
	return reasoning
}

// ReasoningText returns all reasoning content as a concatenated string.
func (r ResponseContent) ReasoningText() string {
	var builder strings.Builder
	for _, reasoning := range r.Reasoning() {
		builder.WriteString(reasoning.Text)
	}
	return builder.String()
}

// Files returns all file content parts.
func (r ResponseContent) Files() []FileContent {
	var files []FileContent
	for _, c := range r {
		if c.GetType() == ContentTypeFile {
			if fileContent, ok := AsContentType[FileContent](c); ok {
				files = append(files, fileContent)
			}
		}
	}
	return files
}

// Sources returns all source content parts.
func (r ResponseContent) Sources() []SourceContent {
	var sources []SourceContent
	for _, c := range r {
		if c.GetType() == ContentTypeSource {
			if sourceContent, ok := AsContentType[SourceContent](c); ok {
				sources = append(sources, sourceContent)
			}
		}
	}
	return sources
}

// ToolCalls returns all tool call content parts.
func (r ResponseContent) ToolCalls() []ToolCallContent {
	var toolCalls []ToolCallContent
	for _, c := range r {
		if c.GetType() == ContentTypeToolCall {
			if toolCallContent, ok := AsContentType[ToolCallContent](c); ok {
				toolCalls = append(toolCalls, toolCallContent)
			}
		}
	}
	return toolCalls
}

// ToolResults returns all tool result content parts.
func (r ResponseContent) ToolResults() []ToolResultContent {
	var toolResults []ToolResultContent
	for _, c := range r {
		if c.GetType() == ContentTypeToolResult {
			if toolResultContent, ok := AsContentType[ToolResultContent](c); ok {
				toolResults = append(toolResults, toolResultContent)
			}
		}
	}
	return toolResults
}

// Response represents a response from a language model.
type Response struct {
	Content      ResponseContent `json:"content"`
	FinishReason FinishReason    `json:"finish_reason"`
	Usage        Usage           `json:"usage"`
	Warnings     []CallWarning   `json:"warnings"`

	// for provider specific response metadata, the key is the provider id
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// StreamPartType represents the type of a stream part.
type StreamPartType string

const (
	// StreamPartTypeWarnings represents warnings stream part type.
	StreamPartTypeWarnings StreamPartType = "warnings"
	// StreamPartTypeTextStart represents text start stream part type.
	StreamPartTypeTextStart StreamPartType = "text_start"
	// StreamPartTypeTextDelta represents text delta stream part type.
	StreamPartTypeTextDelta StreamPartType = "text_delta"
	// StreamPartTypeTextEnd represents text end stream part type.
	StreamPartTypeTextEnd StreamPartType = "text_end"

	// StreamPartTypeReasoningStart represents reasoning start stream part type.
	StreamPartTypeReasoningStart StreamPartType = "reasoning_start"
	// StreamPartTypeReasoningDelta represents reasoning delta stream part type.
	StreamPartTypeReasoningDelta StreamPartType = "reasoning_delta"
	// StreamPartTypeReasoningEnd represents reasoning end stream part type.
	StreamPartTypeReasoningEnd StreamPartType = "reasoning_end"
	// StreamPartTypeToolInputStart represents tool input start stream part type.
	StreamPartTypeToolInputStart StreamPartType = "tool_input_start"
	// StreamPartTypeToolInputDelta represents tool input delta stream part type.
	StreamPartTypeToolInputDelta StreamPartType = "tool_input_delta"
	// StreamPartTypeToolInputEnd represents tool input end stream part type.
	StreamPartTypeToolInputEnd StreamPartType = "tool_input_end"
	// StreamPartTypeToolCall represents tool call stream part type.
	StreamPartTypeToolCall StreamPartType = "tool_call"
	// StreamPartTypeToolResult represents tool result stream part type.
	StreamPartTypeToolResult StreamPartType = "tool_result"
	// StreamPartTypeSource represents source stream part type.
	StreamPartTypeSource StreamPartType = "source"
	// StreamPartTypeFinish represents finish stream part type.
	StreamPartTypeFinish StreamPartType = "finish"
	// StreamPartTypeError represents error stream part type.
	StreamPartTypeError StreamPartType = "error"
)

// StreamPart represents a part of a streaming response.
type StreamPart struct {
	Type             StreamPartType `json:"type"`
	ID               string         `json:"id"`
	ToolCallName     string         `json:"tool_call_name"`
	ToolCallInput    string         `json:"tool_call_input"`
	Delta            string         `json:"delta"`
	ProviderExecuted bool           `json:"provider_executed"`
	Usage            Usage          `json:"usage"`
	FinishReason     FinishReason   `json:"finish_reason"`
	Error            error          `json:"error"`
	Warnings         []CallWarning  `json:"warnings"`

	// Source-related fields
	SourceType SourceType `json:"source_type"`
	URL        string     `json:"url"`
	Title      string     `json:"title"`

	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// StreamResponse represents a streaming response sequence.
type StreamResponse = iter.Seq[StreamPart]

// ToolChoice represents the tool choice preference for a model call.
type ToolChoice string

const (
	// ToolChoiceNone indicates no tools should be used.
	ToolChoiceNone ToolChoice = "none"
	// ToolChoiceAuto indicates tools should be used automatically.
	ToolChoiceAuto ToolChoice = "auto"
	// ToolChoiceRequired indicates tools are required.
	ToolChoiceRequired ToolChoice = "required"
)

// SpecificToolChoice creates a tool choice for a specific tool name.
func SpecificToolChoice(name string) ToolChoice {
	return ToolChoice(name)
}

// Call represents a call to a language model.
type Call struct {
	Prompt           Prompt      `json:"prompt"`
	MaxOutputTokens  *int64      `json:"max_output_tokens"`
	Temperature      *float64    `json:"temperature"`
	TopP             *float64    `json:"top_p"`
	TopK             *int64      `json:"top_k"`
	PresencePenalty  *float64    `json:"presence_penalty"`
	FrequencyPenalty *float64    `json:"frequency_penalty"`
	Tools            []Tool      `json:"tools"`
	ToolChoice       *ToolChoice `json:"tool_choice"`

	// for provider specific options, the key is the provider id
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// CallWarningType represents the type of call warning.
type CallWarningType string

const (
	// CallWarningTypeUnsupportedSetting indicates an unsupported setting.
	CallWarningTypeUnsupportedSetting CallWarningType = "unsupported-setting"
	// CallWarningTypeUnsupportedTool indicates an unsupported tool.
	CallWarningTypeUnsupportedTool CallWarningType = "unsupported-tool"
	// CallWarningTypeOther indicates other warnings.
	CallWarningTypeOther CallWarningType = "other"
)

// CallWarning represents a warning from the model provider for this call.
// The call will proceed, but e.g. some settings might not be supported,
// which can lead to suboptimal results.
type CallWarning struct {
	Type    CallWarningType `json:"type"`
	Setting string          `json:"setting"`
	Tool    Tool            `json:"tool"`
	Details string          `json:"details"`
	Message string          `json:"message"`
}

// LanguageModel represents a language model that can generate responses and stream responses.
type LanguageModel interface {
	Generate(context.Context, Call) (*Response, error)
	Stream(context.Context, Call) (StreamResponse, error)

	GenerateObject(context.Context, ObjectCall) (*ObjectResponse, error)
	StreamObject(context.Context, ObjectCall) (ObjectStreamResponse, error)

	Provider() string
	Model() string
}
