package fantasy

import "encoding/json"

// ProviderOptionsData is an interface for provider-specific options data.
// All implementations MUST also implement encoding/json.Marshaler and
// encoding/json.Unmarshaler interfaces to ensure proper JSON serialization
// with the provider registry system.
//
// Recommended implementation pattern using generic helpers:
//
//	// Define type constants at the top of your file
//	const TypeMyProviderOptions = "myprovider.options"
//
//	type MyProviderOptions struct {
//	    Field string `json:"field"`
//	}
//
//	// Register the type in init() - place at top of file after constants
//	func init() {
//	    fantasy.RegisterProviderType(TypeMyProviderOptions, func(data []byte) (fantasy.ProviderOptionsData, error) {
//	        var opts MyProviderOptions
//	        if err := json.Unmarshal(data, &opts); err != nil {
//	            return nil, err
//	        }
//	        return &opts, nil
//	    })
//	}
//
//	// Implement ProviderOptionsData interface
//	func (*MyProviderOptions) Options() {}
//
//	// Implement json.Marshaler using the generic helper
//	func (m MyProviderOptions) MarshalJSON() ([]byte, error) {
//	    type plain MyProviderOptions
//	    return fantasy.MarshalProviderType(TypeMyProviderOptions, plain(m))
//	}
//
//	// Implement json.Unmarshaler using the generic helper
//	// Note: Receives inner data after type routing by the registry.
//	func (m *MyProviderOptions) UnmarshalJSON(data []byte) error {
//	    type plain MyProviderOptions
//	    var p plain
//	    if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
//	        return err
//	    }
//	    *m = MyProviderOptions(p)
//	    return nil
//	}
type ProviderOptionsData interface {
	// Options is a marker method that identifies types implementing this interface.
	Options()
	json.Marshaler
	json.Unmarshaler
}

// ProviderMetadata represents additional provider-specific metadata.
// They are passed through from the provider to the AI SDK and enable
// provider-specific results that can be fully encapsulated in the provider.
//
// The outer map is keyed by the provider name, and the inner
// map is keyed by the provider-specific metadata key.
//
// Example:
//
//	{
//	  "anthropic": {
//	    "signature": "sig....."
//	  }
//	}
type ProviderMetadata map[string]ProviderOptionsData

// ProviderOptions represents additional provider-specific options.
// Options are additional input to the provider. They are passed through
// to the provider from the AI SDK and enable provider-specific functionality
// that can be fully encapsulated in the provider.
//
// This enables us to quickly ship provider-specific functionality
// without affecting the core AI SDK.
//
// The outer map is keyed by the provider name, and the inner
// map is keyed by the provider-specific option key.
//
// Example:
//
//	{
//	  "anthropic": {
//	    "cacheControl": { "type": "ephemeral" }
//	  }
//	}
type ProviderOptions map[string]ProviderOptionsData

// FinishReason represents why a language model finished generating a response.
//
// Can be one of the following:
// - `stop`: model generated stop sequence
// - `length`: model generated maximum number of tokens
// - `content-filter`: content filter violation stopped the model
// - `tool-calls`: model triggered tool calls
// - `error`: model stopped because of an error
// - `other`: model stopped for other reasons
// - `unknown`: the model has not transmitted a finish reason.
type FinishReason string

const (
	// FinishReasonStop indicates the model generated a stop sequence.
	FinishReasonStop FinishReason = "stop" // model generated stop sequence
	// FinishReasonLength indicates the model generated maximum number of tokens.
	FinishReasonLength FinishReason = "length" // model generated maximum number of tokens
	// FinishReasonContentFilter indicates content filter violation stopped the model.
	FinishReasonContentFilter FinishReason = "content-filter" // content filter violation stopped the model
	// FinishReasonToolCalls indicates the model triggered tool calls.
	FinishReasonToolCalls FinishReason = "tool-calls" // model triggered tool calls
	// FinishReasonError indicates the model stopped because of an error.
	FinishReasonError FinishReason = "error" // model stopped because of an error
	// FinishReasonOther indicates the model stopped for other reasons.
	FinishReasonOther FinishReason = "other" // model stopped for other reasons
	// FinishReasonUnknown indicates the model has not transmitted a finish reason.
	FinishReasonUnknown FinishReason = "unknown" // the model has not transmitted a finish reason
)

// Prompt represents a list of messages for the language model.
type Prompt []Message

// MessageRole represents the role of a message.
type MessageRole string

const (
	// MessageRoleSystem represents a system message.
	MessageRoleSystem MessageRole = "system"
	// MessageRoleUser represents a user message.
	MessageRoleUser MessageRole = "user"
	// MessageRoleAssistant represents an assistant message.
	MessageRoleAssistant MessageRole = "assistant"
	// MessageRoleTool represents a tool message.
	MessageRoleTool MessageRole = "tool"
)

// Message represents a message in a prompt.
type Message struct {
	Role            MessageRole     `json:"role"`
	Content         []MessagePart   `json:"content"`
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// AsContentType converts a Content interface to a specific content type.
func AsContentType[T Content](content Content) (T, bool) {
	var zero T
	if content == nil {
		return zero, false
	}
	switch v := any(content).(type) {
	case T:
		return v, true
	case *T:
		return *v, true
	default:
		return zero, false
	}
}

// AsMessagePart converts a MessagePart interface to a specific message part type.
func AsMessagePart[T MessagePart](content MessagePart) (T, bool) {
	var zero T
	if content == nil {
		return zero, false
	}
	switch v := any(content).(type) {
	case T:
		return v, true
	case *T:
		return *v, true
	default:
		return zero, false
	}
}

// MessagePart represents a part of a message content.
type MessagePart interface {
	GetType() ContentType
	Options() ProviderOptions
}

// TextPart represents text content in a message.
type TextPart struct {
	Text            string          `json:"text"`
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// GetType returns the type of the text part.
func (t TextPart) GetType() ContentType {
	return ContentTypeText
}

// Options returns the provider options for the text part.
func (t TextPart) Options() ProviderOptions {
	return t.ProviderOptions
}

// ReasoningPart represents reasoning content in a message.
type ReasoningPart struct {
	Text            string          `json:"text"`
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// GetType returns the type of the reasoning part.
func (r ReasoningPart) GetType() ContentType {
	return ContentTypeReasoning
}

// Options returns the provider options for the reasoning part.
func (r ReasoningPart) Options() ProviderOptions {
	return r.ProviderOptions
}

// FilePart represents file content in a message.
type FilePart struct {
	Filename        string          `json:"filename"`
	Data            []byte          `json:"data"`
	MediaType       string          `json:"media_type"`
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// GetType returns the type of the file part.
func (f FilePart) GetType() ContentType {
	return ContentTypeFile
}

// Options returns the provider options for the file part.
func (f FilePart) Options() ProviderOptions {
	return f.ProviderOptions
}

// ToolCallPart represents a tool call in a message.
type ToolCallPart struct {
	ToolCallID       string          `json:"tool_call_id"`
	ToolName         string          `json:"tool_name"`
	Input            string          `json:"input"` // the json string
	ProviderExecuted bool            `json:"provider_executed"`
	ProviderOptions  ProviderOptions `json:"provider_options"`
}

// GetType returns the type of the tool call part.
func (t ToolCallPart) GetType() ContentType {
	return ContentTypeToolCall
}

// Options returns the provider options for the tool call part.
func (t ToolCallPart) Options() ProviderOptions {
	return t.ProviderOptions
}

// ToolResultPart represents a tool result in a message.
type ToolResultPart struct {
	ToolCallID      string                  `json:"tool_call_id"`
	Output          ToolResultOutputContent `json:"output"`
	ProviderOptions ProviderOptions         `json:"provider_options"`
}

// GetType returns the type of the tool result part.
func (t ToolResultPart) GetType() ContentType {
	return ContentTypeToolResult
}

// Options returns the provider options for the tool result part.
func (t ToolResultPart) Options() ProviderOptions {
	return t.ProviderOptions
}

// ToolResultContentType represents the type of tool result output.
type ToolResultContentType string

const (
	// ToolResultContentTypeText represents text output.
	ToolResultContentTypeText ToolResultContentType = "text"
	// ToolResultContentTypeError represents error text output.
	ToolResultContentTypeError ToolResultContentType = "error"
	// ToolResultContentTypeMedia represents content output.
	ToolResultContentTypeMedia ToolResultContentType = "media"
)

// ToolResultOutputContent represents the output content of a tool result.
type ToolResultOutputContent interface {
	GetType() ToolResultContentType
}

// ToolResultOutputContentText represents text output content of a tool result.
type ToolResultOutputContentText struct {
	Text string `json:"text"`
}

// GetType returns the type of the tool result output content text.
func (t ToolResultOutputContentText) GetType() ToolResultContentType {
	return ToolResultContentTypeText
}

// ToolResultOutputContentError represents error output content of a tool result.
type ToolResultOutputContentError struct {
	Error error `json:"error"`
}

// GetType returns the type of the tool result output content error.
func (t ToolResultOutputContentError) GetType() ToolResultContentType {
	return ToolResultContentTypeError
}

// ToolResultOutputContentMedia represents media output content of a tool result.
type ToolResultOutputContentMedia struct {
	Data      string `json:"data"`           // for media type (base64)
	MediaType string `json:"media_type"`     // for media type
	Text      string `json:"text,omitempty"` // optional text content accompanying the media
}

// GetType returns the type of the tool result output content media.
func (t ToolResultOutputContentMedia) GetType() ToolResultContentType {
	return ToolResultContentTypeMedia
}

// AsToolResultOutputType converts a ToolResultOutputContent interface to a specific type.
func AsToolResultOutputType[T ToolResultOutputContent](content ToolResultOutputContent) (T, bool) {
	var zero T
	if content == nil {
		return zero, false
	}
	switch v := any(content).(type) {
	case T:
		return v, true
	case *T:
		return *v, true
	default:
		return zero, false
	}
}

// ContentType represents the type of content.
type ContentType string

const (
	// ContentTypeText represents text content.
	ContentTypeText ContentType = "text"
	// ContentTypeReasoning represents reasoning content.
	ContentTypeReasoning ContentType = "reasoning"
	// ContentTypeFile represents file content.
	ContentTypeFile ContentType = "file"
	// ContentTypeSource represents source content.
	ContentTypeSource ContentType = "source"
	// ContentTypeToolCall represents a tool call.
	ContentTypeToolCall ContentType = "tool-call"
	// ContentTypeToolResult represents a tool result.
	ContentTypeToolResult ContentType = "tool-result"
)

// Content represents generated content from the model.
type Content interface {
	GetType() ContentType
}

// TextContent represents text that the model has generated.
type TextContent struct {
	// The text content.
	Text             string           `json:"text"`
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// GetType returns the type of the text content.
func (t TextContent) GetType() ContentType {
	return ContentTypeText
}

// ReasoningContent represents reasoning that the model has generated.
type ReasoningContent struct {
	Text             string           `json:"text"`
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// GetType returns the type of the reasoning content.
func (r ReasoningContent) GetType() ContentType {
	return ContentTypeReasoning
}

// FileContent represents a file that has been generated by the model.
// Generated files as base64 encoded strings or binary data.
// The files should be returned without any unnecessary conversion.
type FileContent struct {
	// The IANA media type of the file, e.g. `image/png` or `audio/mp3`.
	// @see https://www.iana.org/assignments/media-types/media-types.xhtml
	MediaType string `json:"media_type"`
	// Generated file data as binary data.
	Data             []byte           `json:"data"`
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// GetType returns the type of the file content.
func (f FileContent) GetType() ContentType {
	return ContentTypeFile
}

// SourceType represents the type of source.
type SourceType string

const (
	// SourceTypeURL represents a URL source.
	SourceTypeURL SourceType = "url"
	// SourceTypeDocument represents a document source.
	SourceTypeDocument SourceType = "document"
)

// SourceContent represents a source that has been used as input to generate the response.
type SourceContent struct {
	SourceType       SourceType       `json:"source_type"` // "url" or "document"
	ID               string           `json:"id"`
	URL              string           `json:"url"` // for URL sources
	Title            string           `json:"title"`
	MediaType        string           `json:"media_type"` // for document sources (IANA media type)
	Filename         string           `json:"filename"`   // for document sources
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// GetType returns the type of the source content.
func (s SourceContent) GetType() ContentType {
	return ContentTypeSource
}

// ToolCallContent represents tool calls that the model has generated.
type ToolCallContent struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	// Stringified JSON object with the tool call arguments.
	// Must match the parameters schema of the tool.
	Input string `json:"input"`
	// Whether the tool call will be executed by the provider.
	// If this flag is not set or is false, the tool call will be executed by the client.
	ProviderExecuted bool `json:"provider_executed"`
	// Additional provider-specific metadata for the tool call.
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
	// Whether this tool call is invalid (failed validation/parsing)
	Invalid bool `json:"invalid,omitempty"`
	// Error that occurred during validation/parsing (only set if Invalid is true)
	ValidationError error `json:"validation_error,omitempty"`
}

// GetType returns the type of the tool call content.
func (t ToolCallContent) GetType() ContentType {
	return ContentTypeToolCall
}

// ToolResultContent represents result of a tool call that has been executed by the provider.
type ToolResultContent struct {
	// The ID of the tool call that this result is associated with.
	ToolCallID string `json:"tool_call_id"`
	// Name of the tool that generated this result.
	ToolName string `json:"tool_name"`
	// Result of the tool call. This is a JSON-serializable object.
	Result         ToolResultOutputContent `json:"result"`
	ClientMetadata string                  `json:"client_metadata"` // Metadata from the client that executed the tool
	// Whether the tool result was generated by the provider.
	// If this flag is set to true, the tool result was generated by the provider.
	// If this flag is not set or is false, the tool result was generated by the client.
	ProviderExecuted bool `json:"provider_executed"`
	// Additional provider-specific metadata for the tool result.
	ProviderMetadata ProviderMetadata `json:"provider_metadata"`
}

// GetType returns the type of the tool result content.
func (t ToolResultContent) GetType() ContentType {
	return ContentTypeToolResult
}

// ToolType represents the type of tool.
type ToolType string

const (
	// ToolTypeFunction represents a function tool.
	ToolTypeFunction ToolType = "function"
	// ToolTypeProviderDefined represents a provider-defined tool.
	ToolTypeProviderDefined ToolType = "provider-defined"
)

// Tool represents a tool that can be used by the model.
//
// Note: this is **not** the user-facing tool definition. The AI SDK methods will
// map the user-facing tool definitions to this format.
type Tool interface {
	GetType() ToolType
	GetName() string
}

// FunctionTool represents a function tool.
//
// A tool has a name, a description, and a set of parameters.
type FunctionTool struct {
	// Name of the tool. Unique within this model call.
	Name string `json:"name"`
	// Description of the tool. The language model uses this to understand the
	// tool's purpose and to provide better completion suggestions.
	Description string `json:"description"`
	// InputSchema - the parameters that the tool expects. The language model uses this to
	// understand the tool's input requirements and to provide matching suggestions.
	InputSchema map[string]any `json:"input_schema"` // JSON Schema
	// ProviderOptions are provider-specific options for the tool.
	ProviderOptions ProviderOptions `json:"provider_options"`
}

// GetType returns the type of the function tool.
func (f FunctionTool) GetType() ToolType {
	return ToolTypeFunction
}

// GetName returns the name of the function tool.
func (f FunctionTool) GetName() string {
	return f.Name
}

// ProviderDefinedTool represents the configuration of a tool that is defined by the provider.
type ProviderDefinedTool struct {
	// ID of the tool. Should follow the format `<provider-name>.<unique-tool-name>`.
	ID string `json:"id"`
	// Name of the tool that the user must use in the tool set.
	Name string `json:"name"`
	// Args for configuring the tool. Must match the expected arguments defined by the provider for this tool.
	Args map[string]any `json:"args"`
}

// GetType returns the type of the provider-defined tool.
func (p ProviderDefinedTool) GetType() ToolType {
	return ToolTypeProviderDefined
}

// GetName returns the name of the provider-defined tool.
func (p ProviderDefinedTool) GetName() string {
	return p.Name
}

// NewUserMessage creates a new user message with the given prompt and optional files.
func NewUserMessage(prompt string, files ...FilePart) Message {
	content := make([]MessagePart, 0, len(files)+1)
	content = append(content, TextPart{Text: prompt})

	for _, f := range files {
		content = append(content, f)
	}

	return Message{
		Role:    MessageRoleUser,
		Content: content,
	}
}

// NewSystemMessage creates a new system message with the given prompts.
func NewSystemMessage(prompt ...string) Message {
	content := make([]MessagePart, 0, len(prompt))
	for _, p := range prompt {
		content = append(content, TextPart{Text: p})
	}

	return Message{
		Role:    MessageRoleSystem,
		Content: content,
	}
}
