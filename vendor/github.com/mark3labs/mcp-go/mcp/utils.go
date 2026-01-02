package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cast"
)

// ClientRequest types
var (
	_ ClientRequest = (*PingRequest)(nil)
	_ ClientRequest = (*InitializeRequest)(nil)
	_ ClientRequest = (*CompleteRequest)(nil)
	_ ClientRequest = (*SetLevelRequest)(nil)
	_ ClientRequest = (*GetPromptRequest)(nil)
	_ ClientRequest = (*ListPromptsRequest)(nil)
	_ ClientRequest = (*ListResourcesRequest)(nil)
	_ ClientRequest = (*ReadResourceRequest)(nil)
	_ ClientRequest = (*SubscribeRequest)(nil)
	_ ClientRequest = (*UnsubscribeRequest)(nil)
	_ ClientRequest = (*CallToolRequest)(nil)
	_ ClientRequest = (*ListToolsRequest)(nil)
)

// ClientNotification types
var (
	_ ClientNotification = (*CancelledNotification)(nil)
	_ ClientNotification = (*ProgressNotification)(nil)
	_ ClientNotification = (*InitializedNotification)(nil)
	_ ClientNotification = (*RootsListChangedNotification)(nil)
)

// ClientResult types
var (
	_ ClientResult = (*EmptyResult)(nil)
	_ ClientResult = (*CreateMessageResult)(nil)
	_ ClientResult = (*ListRootsResult)(nil)
)

// ServerRequest types
var (
	_ ServerRequest = (*PingRequest)(nil)
	_ ServerRequest = (*CreateMessageRequest)(nil)
	_ ServerRequest = (*ListRootsRequest)(nil)
)

// ServerNotification types
var (
	_ ServerNotification = (*CancelledNotification)(nil)
	_ ServerNotification = (*ProgressNotification)(nil)
	_ ServerNotification = (*LoggingMessageNotification)(nil)
	_ ServerNotification = (*ResourceUpdatedNotification)(nil)
	_ ServerNotification = (*ResourceListChangedNotification)(nil)
	_ ServerNotification = (*ToolListChangedNotification)(nil)
	_ ServerNotification = (*PromptListChangedNotification)(nil)
)

// ServerResult types
var (
	_ ServerResult = (*EmptyResult)(nil)
	_ ServerResult = (*InitializeResult)(nil)
	_ ServerResult = (*CompleteResult)(nil)
	_ ServerResult = (*GetPromptResult)(nil)
	_ ServerResult = (*ListPromptsResult)(nil)
	_ ServerResult = (*ListResourcesResult)(nil)
	_ ServerResult = (*ReadResourceResult)(nil)
	_ ServerResult = (*CallToolResult)(nil)
	_ ServerResult = (*ListToolsResult)(nil)
)

// Helper functions for type assertions

// asType attempts to cast the given interface to the given type
func asType[T any](content any) (*T, bool) {
	tc, ok := content.(T)
	if !ok {
		return nil, false
	}
	return &tc, true
}

// AsTextContent attempts to cast the given interface to TextContent
func AsTextContent(content any) (*TextContent, bool) {
	return asType[TextContent](content)
}

// AsImageContent attempts to cast the given interface to ImageContent
func AsImageContent(content any) (*ImageContent, bool) {
	return asType[ImageContent](content)
}

// AsAudioContent attempts to cast the given interface to AudioContent
func AsAudioContent(content any) (*AudioContent, bool) {
	return asType[AudioContent](content)
}

// AsEmbeddedResource attempts to cast the given interface to EmbeddedResource
func AsEmbeddedResource(content any) (*EmbeddedResource, bool) {
	return asType[EmbeddedResource](content)
}

// AsTextResourceContents attempts to cast the given interface to TextResourceContents
func AsTextResourceContents(content any) (*TextResourceContents, bool) {
	return asType[TextResourceContents](content)
}

// AsBlobResourceContents attempts to cast the given interface to BlobResourceContents
func AsBlobResourceContents(content any) (*BlobResourceContents, bool) {
	return asType[BlobResourceContents](content)
}

// Helper function for JSON-RPC

// NewJSONRPCResponse creates a new JSONRPCResponse with the given id and result.
// NOTE: This function expects a Result struct, but JSONRPCResponse.Result is typed as `any`.
// The Result struct wraps the actual result data with optional metadata.
// For direct result assignment, use NewJSONRPCResultResponse instead.
func NewJSONRPCResponse(id RequestId, result Result) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCResultResponse creates a new JSONRPCResponse with the given id and result.
// This function accepts any type for the result, matching the JSONRPCResponse.Result field type.
func NewJSONRPCResultResponse(id RequestId, result any) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCErrorDetails creates a new JSONRPCErrorDetails with the given code, message, and data.
func NewJSONRPCErrorDetails(code int, message string, data any) JSONRPCErrorDetails {
	return JSONRPCErrorDetails{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewJSONRPCError creates a new JSONRPCResponse with the given id, code, and message
func NewJSONRPCError(
	id RequestId,
	code int,
	message string,
	data any,
) JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error:   NewJSONRPCErrorDetails(code, message, data),
	}
}

// NewProgressNotification
// Helper function for creating a progress notification
func NewProgressNotification(
	token ProgressToken,
	progress float64,
	total *float64,
	message *string,
) ProgressNotification {
	notification := ProgressNotification{
		Notification: Notification{
			Method: "notifications/progress",
		},
		Params: struct {
			ProgressToken ProgressToken `json:"progressToken"`
			Progress      float64       `json:"progress"`
			Total         float64       `json:"total,omitempty"`
			Message       string        `json:"message,omitempty"`
		}{
			ProgressToken: token,
			Progress:      progress,
		},
	}
	if total != nil {
		notification.Params.Total = *total
	}
	if message != nil {
		notification.Params.Message = *message
	}
	return notification
}

// NewLoggingMessageNotification
// Helper function for creating a logging message notification
func NewLoggingMessageNotification(
	level LoggingLevel,
	logger string,
	data any,
) LoggingMessageNotification {
	return LoggingMessageNotification{
		Notification: Notification{
			Method: "notifications/message",
		},
		Params: struct {
			Level  LoggingLevel `json:"level"`
			Logger string       `json:"logger,omitempty"`
			Data   any          `json:"data"`
		}{
			Level:  level,
			Logger: logger,
			Data:   data,
		},
	}
}

// NewPromptMessage
// Helper function to create a new PromptMessage
func NewPromptMessage(role Role, content Content) PromptMessage {
	return PromptMessage{
		Role:    role,
		Content: content,
	}
}

// NewTextContent
// Helper function to create a new TextContent
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: ContentTypeText,
		Text: text,
	}
}

// NewImageContent
// Helper function to create a new ImageContent
func NewImageContent(data, mimeType string) ImageContent {
	return ImageContent{
		Type:     ContentTypeImage,
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create a new AudioContent
func NewAudioContent(data, mimeType string) AudioContent {
	return AudioContent{
		Type:     ContentTypeAudio,
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create a new ResourceLink
func NewResourceLink(uri, name, description, mimeType string) ResourceLink {
	return ResourceLink{
		Type:        ContentTypeLink,
		URI:         uri,
		Name:        name,
		Description: description,
		MIMEType:    mimeType,
	}
}

// Helper function to create a new EmbeddedResource
func NewEmbeddedResource(resource ResourceContents) EmbeddedResource {
	return EmbeddedResource{
		Type:     ContentTypeResource,
		Resource: resource,
	}
}

// NewToolResultText creates a new CallToolResult with a text content
func NewToolResultText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
		},
	}
}

// NewToolResultJSON creates a new CallToolResult with a JSON content.
func NewToolResultJSON[T any](data T) (*CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal JSON: %w", err)
	}

	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: string(b),
			},
		},
		StructuredContent: data,
	}, nil
}

// NewToolResultStructured creates a new CallToolResult with structured content.
// It includes both the structured content and a text representation for backward compatibility.
func NewToolResultStructured(structured any, fallbackText string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: fallbackText,
			},
		},
		StructuredContent: structured,
	}
}

// NewToolResultStructuredOnly creates a new CallToolResult with structured
// content and creates a JSON string fallback for backwards compatibility.
// This is useful when you want to provide structured data without any specific text fallback.
func NewToolResultStructuredOnly(structured any) *CallToolResult {
	var fallbackText string
	// Convert to JSON string for backward compatibility
	jsonBytes, err := json.Marshal(structured)
	if err != nil {
		fallbackText = fmt.Sprintf("Error serializing structured content: %v", err)
	} else {
		fallbackText = string(jsonBytes)
	}

	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: fallbackText,
			},
		},
		StructuredContent: structured,
	}
}

// NewToolResultImage creates a new CallToolResult with both text and image content
func NewToolResultImage(text, imageData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
			ImageContent{
				Type:     ContentTypeImage,
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
	}
}

// NewToolResultAudio creates a new CallToolResult with both text and audio content
func NewToolResultAudio(text, imageData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
			AudioContent{
				Type:     ContentTypeAudio,
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
	}
}

// NewToolResultResource creates a new CallToolResult with an embedded resource
func NewToolResultResource(
	text string,
	resource ResourceContents,
) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
			EmbeddedResource{
				Type:     ContentTypeResource,
				Resource: resource,
			},
		},
	}
}

// NewToolResultError creates a new CallToolResult with an error message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultError(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
		},
		IsError: true,
	}
}

// NewToolResultErrorFromErr creates a new CallToolResult with an error message.
// If an error is provided, its details will be appended to the text message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultErrorFromErr(text string, err error) *CallToolResult {
	if err != nil {
		text = fmt.Sprintf("%s: %v", text, err)
	}
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: text,
			},
		},
		IsError: true,
	}
}

// NewToolResultErrorf creates a new CallToolResult with an error message.
// The error message is formatted using the fmt package.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultErrorf(format string, a ...any) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: ContentTypeText,
				Text: fmt.Sprintf(format, a...),
			},
		},
		IsError: true,
	}
}

// NewListResourcesResult creates a new ListResourcesResult
func NewListResourcesResult(
	resources []Resource,
	nextCursor Cursor,
) *ListResourcesResult {
	return &ListResourcesResult{
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
		Resources: resources,
	}
}

// NewListResourceTemplatesResult creates a new ListResourceTemplatesResult
func NewListResourceTemplatesResult(
	templates []ResourceTemplate,
	nextCursor Cursor,
) *ListResourceTemplatesResult {
	return &ListResourceTemplatesResult{
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
		ResourceTemplates: templates,
	}
}

// NewReadResourceResult creates a new ReadResourceResult with text content
func NewReadResourceResult(text string) *ReadResourceResult {
	return &ReadResourceResult{
		Contents: []ResourceContents{
			TextResourceContents{
				Text: text,
			},
		},
	}
}

// NewListPromptsResult creates a new ListPromptsResult
func NewListPromptsResult(
	prompts []Prompt,
	nextCursor Cursor,
) *ListPromptsResult {
	return &ListPromptsResult{
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
		Prompts: prompts,
	}
}

// NewGetPromptResult creates a new GetPromptResult
func NewGetPromptResult(
	description string,
	messages []PromptMessage,
) *GetPromptResult {
	return &GetPromptResult{
		Description: description,
		Messages:    messages,
	}
}

// NewListToolsResult creates a new ListToolsResult
func NewListToolsResult(tools []Tool, nextCursor Cursor) *ListToolsResult {
	return &ListToolsResult{
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
		Tools: tools,
	}
}

// NewInitializeResult creates a new InitializeResult
func NewInitializeResult(
	protocolVersion string,
	capabilities ServerCapabilities,
	serverInfo Implementation,
	instructions string,
) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities,
		ServerInfo:      serverInfo,
		Instructions:    instructions,
	}
}

// FormatNumberResult
// Helper for formatting numbers in tool results
func FormatNumberResult(value float64) *CallToolResult {
	return NewToolResultText(fmt.Sprintf("%.2f", value))
}

func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func ParseAnnotations(data map[string]any) *Annotations {
	if data == nil {
		return nil
	}
	annotations := &Annotations{}
	if value, ok := data["priority"]; ok {
		annotations.Priority = cast.ToFloat64(value)
	}

	if value, ok := data["audience"]; ok {
		for _, a := range cast.ToStringSlice(value) {
			a := Role(a)
			if a == RoleUser || a == RoleAssistant {
				annotations.Audience = append(annotations.Audience, a)
			}
		}
	}
	return annotations

}

func ExtractMap(data map[string]any, key string) map[string]any {
	if value, ok := data[key]; ok {
		if m, ok := value.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func ParseContent(contentMap map[string]any) (Content, error) {
	contentType := ExtractString(contentMap, "type")

	var annotations *Annotations
	if annotationsMap := ExtractMap(contentMap, "annotations"); annotationsMap != nil {
		annotations = ParseAnnotations(annotationsMap)
	}

	switch contentType {
	case ContentTypeText:
		text := ExtractString(contentMap, "text")
		c := NewTextContent(text)
		c.Annotations = annotations
		return c, nil

	case ContentTypeImage:
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("image data or mimeType is missing")
		}
		c := NewImageContent(data, mimeType)
		c.Annotations = annotations
		return c, nil

	case ContentTypeAudio:
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("audio data or mimeType is missing")
		}
		c := NewAudioContent(data, mimeType)
		c.Annotations = annotations
		return c, nil

	case ContentTypeLink:
		uri := ExtractString(contentMap, "uri")
		name := ExtractString(contentMap, "name")
		description := ExtractString(contentMap, "description")
		mimeType := ExtractString(contentMap, "mimeType")
		if uri == "" || name == "" {
			return nil, fmt.Errorf("resource_link uri or name is missing")
		}
		c := NewResourceLink(uri, name, description, mimeType)
		c.Annotations = annotations
		return c, nil

	case ContentTypeResource:
		resourceMap := ExtractMap(contentMap, "resource")
		if resourceMap == nil {
			return nil, fmt.Errorf("resource is missing")
		}

		resourceContents, err := ParseResourceContents(resourceMap)
		if err != nil {
			return nil, err
		}

		c := NewEmbeddedResource(resourceContents)
		c.Annotations = annotations
		return c, nil
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func ParseGetPromptResult(rawMessage *json.RawMessage) (*GetPromptResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	result := GetPromptResult{}

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = NewMetaFromMap(metaMap)
		}
	}

	description, ok := jsonContent["description"]
	if ok {
		if descriptionStr, ok := description.(string); ok {
			result.Description = descriptionStr
		}
	}

	messages, ok := jsonContent["messages"]
	if ok {
		messagesArr, ok := messages.([]any)
		if !ok {
			return nil, fmt.Errorf("messages is not an array")
		}

		for _, message := range messagesArr {
			messageMap, ok := message.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("message is not an object")
			}

			// Extract role
			roleStr := ExtractString(messageMap, "role")
			if roleStr == "" || (roleStr != string(RoleAssistant) && roleStr != string(RoleUser)) {
				return nil, fmt.Errorf("unsupported role: %s", roleStr)
			}

			// Extract content
			contentMap, ok := messageMap["content"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("content is not an object")
			}

			// Process content
			content, err := ParseContent(contentMap)
			if err != nil {
				return nil, err
			}

			// Append processed message
			result.Messages = append(result.Messages, NewPromptMessage(Role(roleStr), content))

		}
	}

	return &result, nil
}

func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result CallToolResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = NewMetaFromMap(metaMap)
		}
	}

	isError, ok := jsonContent["isError"]
	if ok {
		if isErrorBool, ok := isError.(bool); ok {
			result.IsError = isErrorBool
		}
	}

	contents, ok := jsonContent["content"]
	if !ok {
		return nil, fmt.Errorf("content is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("content is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseContent(contentMap)
		if err != nil {
			return nil, err
		}

		result.Content = append(result.Content, content)
	}

	// Handle structured content
	structuredContent, ok := jsonContent["structuredContent"]
	if ok {
		result.StructuredContent = structuredContent
	}

	return &result, nil
}

func ParseResourceContents(contentMap map[string]any) (ResourceContents, error) {
	uri := ExtractString(contentMap, "uri")
	if uri == "" {
		return nil, fmt.Errorf("resource uri is missing")
	}

	mimeType := ExtractString(contentMap, "mimeType")

	meta := ExtractMap(contentMap, "_meta")

	if _, present := contentMap["_meta"]; present && meta == nil {
		return nil, fmt.Errorf("_meta must be an object")
	}

	if text := ExtractString(contentMap, "text"); text != "" {
		return TextResourceContents{
			Meta:     meta,
			URI:      uri,
			MIMEType: mimeType,
			Text:     text,
		}, nil
	}

	if blob := ExtractString(contentMap, "blob"); blob != "" {
		return BlobResourceContents{
			Meta:     meta,
			URI:      uri,
			MIMEType: mimeType,
			Blob:     blob,
		}, nil
	}

	return nil, fmt.Errorf("unsupported resource type")
}

func ParseReadResourceResult(rawMessage *json.RawMessage) (*ReadResourceResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result ReadResourceResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = NewMetaFromMap(metaMap)
		}
	}

	contents, ok := jsonContent["contents"]
	if !ok {
		return nil, fmt.Errorf("contents is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("contents is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseResourceContents(contentMap)
		if err != nil {
			return nil, err
		}

		result.Contents = append(result.Contents, content)
	}

	return &result, nil
}

func ParseArgument(request CallToolRequest, key string, defaultVal any) any {
	args := request.GetArguments()
	if _, ok := args[key]; !ok {
		return defaultVal
	} else {
		return args[key]
	}
}

// ParseBoolean extracts and converts a boolean parameter from a CallToolRequest.
// If the key is not found in the Arguments map, the defaultValue is returned.
// The function uses cast.ToBool for conversion which handles various string representations
// such as "true", "yes", "1", etc.
func ParseBoolean(request CallToolRequest, key string, defaultValue bool) bool {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToBool(v)
}

// ParseInt64 extracts and converts an int64 parameter from a CallToolRequest.
// If the key is not found in the Arguments map, the defaultValue is returned.
func ParseInt64(request CallToolRequest, key string, defaultValue int64) int64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt64(v)
}

// ParseInt32 extracts and converts an int32 parameter from a CallToolRequest.
func ParseInt32(request CallToolRequest, key string, defaultValue int32) int32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt32(v)
}

// ParseInt16 extracts and converts an int16 parameter from a CallToolRequest.
func ParseInt16(request CallToolRequest, key string, defaultValue int16) int16 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt16(v)
}

// ParseInt8 extracts and converts an int8 parameter from a CallToolRequest.
func ParseInt8(request CallToolRequest, key string, defaultValue int8) int8 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt8(v)
}

// ParseInt extracts and converts an int parameter from a CallToolRequest.
func ParseInt(request CallToolRequest, key string, defaultValue int) int {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToInt(v)
}

// ParseUInt extracts and converts an uint parameter from a CallToolRequest.
func ParseUInt(request CallToolRequest, key string, defaultValue uint) uint {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint(v)
}

// ParseUInt64 extracts and converts an uint64 parameter from a CallToolRequest.
func ParseUInt64(request CallToolRequest, key string, defaultValue uint64) uint64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint64(v)
}

// ParseUInt32 extracts and converts an uint32 parameter from a CallToolRequest.
func ParseUInt32(request CallToolRequest, key string, defaultValue uint32) uint32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint32(v)
}

// ParseUInt16 extracts and converts an uint16 parameter from a CallToolRequest.
func ParseUInt16(request CallToolRequest, key string, defaultValue uint16) uint16 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint16(v)
}

// ParseUInt8 extracts and converts an uint8 parameter from a CallToolRequest.
func ParseUInt8(request CallToolRequest, key string, defaultValue uint8) uint8 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToUint8(v)
}

// ParseFloat32 extracts and converts a float32 parameter from a CallToolRequest.
func ParseFloat32(request CallToolRequest, key string, defaultValue float32) float32 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToFloat32(v)
}

// ParseFloat64 extracts and converts a float64 parameter from a CallToolRequest.
func ParseFloat64(request CallToolRequest, key string, defaultValue float64) float64 {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToFloat64(v)
}

// ParseString extracts and converts a string parameter from a CallToolRequest.
func ParseString(request CallToolRequest, key string, defaultValue string) string {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToString(v)
}

// ParseStringMap extracts and converts a string map parameter from a CallToolRequest.
func ParseStringMap(request CallToolRequest, key string, defaultValue map[string]any) map[string]any {
	v := ParseArgument(request, key, defaultValue)
	return cast.ToStringMap(v)
}

// ToBoolPtr returns a pointer to the given boolean value
func ToBoolPtr(b bool) *bool {
	return &b
}

// GetTextFromContent extracts text from a Content interface that might be a TextContent struct
// or a map[string]any that was unmarshaled from JSON. This is useful when dealing with content
// that comes from different transport layers that may handle JSON differently.
//
// This function uses fallback behavior for non-text content - it returns a string representation
// via fmt.Sprintf for any content that cannot be extracted as text. This is a lossy operation
// intended for convenience in logging and display scenarios.
//
// For strict type validation, use ParseContent() instead, which returns an error for invalid content.
func GetTextFromContent(content any) string {
	switch c := content.(type) {
	case TextContent:
		return c.Text
	case map[string]any:
		// Handle JSON unmarshaled content
		if contentType, exists := c["type"]; exists && contentType == "text" {
			if text, exists := c["text"].(string); exists {
				return text
			}
		}
		return fmt.Sprintf("%v", content)
	case string:
		return c
	default:
		return fmt.Sprintf("%v", content)
	}
}
