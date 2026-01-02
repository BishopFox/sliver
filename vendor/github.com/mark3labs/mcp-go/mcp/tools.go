package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/invopop/jsonschema"
)

var errToolSchemaConflict = errors.New("provide either InputSchema or RawInputSchema, not both")

// ListToolsRequest is sent from the client to request a list of tools the
// server has.
type ListToolsRequest struct {
	PaginatedRequest
	Header http.Header `json:"-"`
}

// ListToolsResult is the server's response to a tools/list request from the
// client.
type ListToolsResult struct {
	PaginatedResult
	Tools []Tool `json:"tools"`
}

// CallToolResult is the server's response to a tool call.
//
// Any errors that originate from the tool SHOULD be reported inside the result
// object, with `isError` set to true, _not_ as an MCP protocol-level error
// response. Otherwise, the LLM would not be able to see that an error occurred
// and self-correct.
//
// However, any errors in _finding_ the tool, an error indicating that the
// server does not support tool calls, or any other exceptional conditions,
// should be reported as an MCP error response.
type CallToolResult struct {
	Result
	Content []Content `json:"content"` // Can be TextContent, ImageContent, AudioContent, or EmbeddedResource
	// Structured content returned as a JSON object in the structuredContent field of a result.
	// For backwards compatibility, a tool that returns structured content SHOULD also return
	// functionally equivalent unstructured content.
	StructuredContent any `json:"structuredContent,omitempty"`
	// Whether the tool call ended in an error.
	//
	// If not set, this is assumed to be false (the call was successful).
	IsError bool `json:"isError,omitempty"`
}

// CallToolRequest is used by the client to invoke a tool provided by the server.
type CallToolRequest struct {
	Request
	Header http.Header    `json:"-"` // HTTP headers from the original request
	Params CallToolParams `json:"params"`
}

type CallToolParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"`
	Meta      *Meta  `json:"_meta,omitempty"`
}

// GetArguments returns the Arguments as map[string]any for backward compatibility
// If Arguments is not a map, it returns an empty map
func (r CallToolRequest) GetArguments() map[string]any {
	if args, ok := r.Params.Arguments.(map[string]any); ok {
		return args
	}
	return nil
}

// GetRawArguments returns the Arguments as-is without type conversion
// This allows users to access the raw arguments in any format
func (r CallToolRequest) GetRawArguments() any {
	return r.Params.Arguments
}

// BindArguments unmarshals the Arguments into the provided struct
// This is useful for working with strongly-typed arguments
func (r CallToolRequest) BindArguments(target any) error {
	if target == nil || reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	// Fast-path: already raw JSON
	if raw, ok := r.Params.Arguments.(json.RawMessage); ok {
		return json.Unmarshal(raw, target)
	}

	data, err := json.Marshal(r.Params.Arguments)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}

	return json.Unmarshal(data, target)
}

// GetString returns a string argument by key, or the default value if not found
func (r CallToolRequest) GetString(key string, defaultValue string) string {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// RequireString returns a string argument by key, or an error if not found or not a string
func (r CallToolRequest) RequireString(key string) (string, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("argument %q is not a string", key)
	}
	return "", fmt.Errorf("required argument %q not found", key)
}

// GetInt returns an int argument by key, or the default value if not found
func (r CallToolRequest) GetInt(key string, defaultValue int) int {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// RequireInt returns an int argument by key, or an error if not found or not convertible to int
func (r CallToolRequest) RequireInt(key string) (int, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i, nil
			}
			return 0, fmt.Errorf("argument %q cannot be converted to int", key)
		default:
			return 0, fmt.Errorf("argument %q is not an int", key)
		}
	}
	return 0, fmt.Errorf("required argument %q not found", key)
}

// GetFloat returns a float64 argument by key, or the default value if not found
func (r CallToolRequest) GetFloat(key string, defaultValue float64) float64 {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return defaultValue
}

// RequireFloat returns a float64 argument by key, or an error if not found or not convertible to float64
func (r CallToolRequest) RequireFloat(key string) (float64, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f, nil
			}
			return 0, fmt.Errorf("argument %q cannot be converted to float64", key)
		default:
			return 0, fmt.Errorf("argument %q is not a float64", key)
		}
	}
	return 0, fmt.Errorf("required argument %q not found", key)
}

// GetBool returns a bool argument by key, or the default value if not found
func (r CallToolRequest) GetBool(key string, defaultValue bool) bool {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	return defaultValue
}

// RequireBool returns a bool argument by key, or an error if not found or not convertible to bool
func (r CallToolRequest) RequireBool(key string) (bool, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b, nil
			}
			return false, fmt.Errorf("argument %q cannot be converted to bool", key)
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		default:
			return false, fmt.Errorf("argument %q is not a bool", key)
		}
	}
	return false, fmt.Errorf("required argument %q not found", key)
}

// GetStringSlice returns a string slice argument by key, or the default value if not found
func (r CallToolRequest) GetStringSlice(key string, defaultValue []string) []string {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []string:
			return v
		case []any:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireStringSlice returns a string slice argument by key, or an error if not found or not convertible to string slice
func (r CallToolRequest) RequireStringSlice(key string) ([]string, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []string:
			return v, nil
		case []any:
			result := make([]string, 0, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					result = append(result, str)
				} else {
					return nil, fmt.Errorf("item %d in argument %q is not a string", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a string slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetIntSlice returns an int slice argument by key, or the default value if not found
func (r CallToolRequest) GetIntSlice(key string, defaultValue []int) []int {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []int:
			return v
		case []any:
			result := make([]int, 0, len(v))
			for _, item := range v {
				switch num := item.(type) {
				case int:
					result = append(result, num)
				case float64:
					result = append(result, int(num))
				case string:
					if i, err := strconv.Atoi(num); err == nil {
						result = append(result, i)
					}
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireIntSlice returns an int slice argument by key, or an error if not found or not convertible to int slice
func (r CallToolRequest) RequireIntSlice(key string) ([]int, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []int:
			return v, nil
		case []any:
			result := make([]int, 0, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case int:
					result = append(result, num)
				case float64:
					result = append(result, int(num))
				case string:
					if i, err := strconv.Atoi(num); err == nil {
						result = append(result, i)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to int", i, key)
					}
				default:
					return nil, fmt.Errorf("item %d in argument %q is not an int", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not an int slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetFloatSlice returns a float64 slice argument by key, or the default value if not found
func (r CallToolRequest) GetFloatSlice(key string, defaultValue []float64) []float64 {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []float64:
			return v
		case []any:
			result := make([]float64, 0, len(v))
			for _, item := range v {
				switch num := item.(type) {
				case float64:
					result = append(result, num)
				case int:
					result = append(result, float64(num))
				case string:
					if f, err := strconv.ParseFloat(num, 64); err == nil {
						result = append(result, f)
					}
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireFloatSlice returns a float64 slice argument by key, or an error if not found or not convertible to float64 slice
func (r CallToolRequest) RequireFloatSlice(key string) ([]float64, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []float64:
			return v, nil
		case []any:
			result := make([]float64, 0, len(v))
			for i, item := range v {
				switch num := item.(type) {
				case float64:
					result = append(result, num)
				case int:
					result = append(result, float64(num))
				case string:
					if f, err := strconv.ParseFloat(num, 64); err == nil {
						result = append(result, f)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to float64", i, key)
					}
				default:
					return nil, fmt.Errorf("item %d in argument %q is not a float64", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a float64 slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// GetBoolSlice returns a bool slice argument by key, or the default value if not found
func (r CallToolRequest) GetBoolSlice(key string, defaultValue []bool) []bool {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []bool:
			return v
		case []any:
			result := make([]bool, 0, len(v))
			for _, item := range v {
				switch b := item.(type) {
				case bool:
					result = append(result, b)
				case string:
					if parsed, err := strconv.ParseBool(b); err == nil {
						result = append(result, parsed)
					}
				case int:
					result = append(result, b != 0)
				case float64:
					result = append(result, b != 0)
				}
			}
			return result
		}
	}
	return defaultValue
}

// RequireBoolSlice returns a bool slice argument by key, or an error if not found or not convertible to bool slice
func (r CallToolRequest) RequireBoolSlice(key string) ([]bool, error) {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case []bool:
			return v, nil
		case []any:
			result := make([]bool, 0, len(v))
			for i, item := range v {
				switch b := item.(type) {
				case bool:
					result = append(result, b)
				case string:
					if parsed, err := strconv.ParseBool(b); err == nil {
						result = append(result, parsed)
					} else {
						return nil, fmt.Errorf("item %d in argument %q cannot be converted to bool", i, key)
					}
				case int:
					result = append(result, b != 0)
				case float64:
					result = append(result, b != 0)
				default:
					return nil, fmt.Errorf("item %d in argument %q is not a bool", i, key)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("argument %q is not a bool slice", key)
		}
	}
	return nil, fmt.Errorf("required argument %q not found", key)
}

// MarshalJSON implements custom JSON marshaling for CallToolResult
func (r CallToolResult) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	// Marshal Meta if present
	if r.Meta != nil {
		m["_meta"] = r.Meta
	}

	// Marshal Content array
	content := make([]any, len(r.Content))
	for i, c := range r.Content {
		content[i] = c
	}
	m["content"] = content

	// Marshal StructuredContent if present
	if r.StructuredContent != nil {
		m["structuredContent"] = r.StructuredContent
	}

	// Marshal IsError if true
	if r.IsError {
		m["isError"] = r.IsError
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements custom JSON unmarshaling for CallToolResult
func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal Meta
	if meta, ok := raw["_meta"]; ok {
		if metaMap, ok := meta.(map[string]any); ok {
			r.Meta = NewMetaFromMap(metaMap)
		}
	}

	// Unmarshal Content array
	if contentRaw, ok := raw["content"]; ok {
		if contentArray, ok := contentRaw.([]any); ok {
			r.Content = make([]Content, len(contentArray))
			for i, item := range contentArray {
				itemBytes, err := json.Marshal(item)
				if err != nil {
					return err
				}
				content, err := UnmarshalContent(itemBytes)
				if err != nil {
					return err
				}
				r.Content[i] = content
			}
		}
	}

	// Unmarshal StructuredContent if present
	if structured, ok := raw["structuredContent"]; ok {
		r.StructuredContent = structured
	}

	// Unmarshal IsError
	if isError, ok := raw["isError"]; ok {
		if isErrorBool, ok := isError.(bool); ok {
			r.IsError = isErrorBool
		}
	}

	return nil
}

// ToolListChangedNotification is an optional notification from the server to
// the client, informing it that the list of tools it offers has changed. This may
// be issued by servers without any previous subscription from the client.
type ToolListChangedNotification struct {
	Notification
}

// Tool represents the definition for a tool the client can call.
type Tool struct {
	// Meta is a metadata object that is reserved by MCP for storing additional information.
	Meta *Meta `json:"_meta,omitempty"`
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema ToolInputSchema `json:"inputSchema"`
	// Alternative to InputSchema - allows arbitrary JSON Schema to be provided
	RawInputSchema json.RawMessage `json:"-"` // Hide this from JSON marshaling
	// A JSON Schema object defining the expected output returned by the tool .
	OutputSchema ToolOutputSchema `json:"outputSchema,omitempty"`
	// Optional JSON Schema defining expected output structure
	RawOutputSchema json.RawMessage `json:"-"` // Hide this from JSON marshaling
	// Optional properties describing tool behavior
	Annotations ToolAnnotation `json:"annotations"`
}

// GetName returns the name of the tool.
func (t Tool) GetName() string {
	return t.Name
}

// MarshalJSON implements the json.Marshaler interface for Tool.
// It handles marshaling either InputSchema or RawInputSchema based on which is set.
func (t Tool) MarshalJSON() ([]byte, error) {
	// Create a map to build the JSON structure
	m := make(map[string]any, 5)

	// Add the name and description
	m["name"] = t.Name
	if t.Description != "" {
		m["description"] = t.Description
	}

	// Determine which input schema to use
	if t.RawInputSchema != nil {
		if t.InputSchema.Type != "" {
			return nil, fmt.Errorf("tool %s has both InputSchema and RawInputSchema set: %w", t.Name, errToolSchemaConflict)
		}
		m["inputSchema"] = t.RawInputSchema
	} else {
		// Use the structured InputSchema
		m["inputSchema"] = t.InputSchema
	}

	// Add output schema if present
	if t.RawOutputSchema != nil {
		if t.OutputSchema.Type != "" {
			return nil, fmt.Errorf("tool %s has both OutputSchema and RawOutputSchema set: %w", t.Name, errToolSchemaConflict)
		}
		m["outputSchema"] = t.RawOutputSchema
	} else if t.OutputSchema.Type != "" { // If no output schema is specified, do not return anything
		m["outputSchema"] = t.OutputSchema
	}

	m["annotations"] = t.Annotations

	// Marshal Meta if present
	if t.Meta != nil {
		m["_meta"] = t.Meta
	}

	return json.Marshal(m)
}

// ToolArgumentsSchema represents a JSON Schema for tool arguments.
type ToolArgumentsSchema struct {
	Defs       map[string]any `json:"$defs,omitempty"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

type ToolInputSchema ToolArgumentsSchema // For retro-compatibility
type ToolOutputSchema ToolArgumentsSchema

// MarshalJSON implements the json.Marshaler interface for ToolInputSchema.
func (tis ToolArgumentsSchema) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m["type"] = tis.Type

	if tis.Defs != nil {
		m["$defs"] = tis.Defs
	}

	// Marshal Properties to '{}' rather than `nil` when its length equals zero
	if tis.Properties != nil {
		m["properties"] = tis.Properties
	}

	if len(tis.Required) > 0 {
		m["required"] = tis.Required
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements the json.Unmarshaler interface for ToolArgumentsSchema.
// It handles both "$defs" (JSON Schema 2019-09+) and "definitions" (JSON Schema draft-07)
// by reading either field and storing it in the Defs field.
func (tis *ToolArgumentsSchema) UnmarshalJSON(data []byte) error {
	// Use a temporary type to avoid infinite recursion
	type Alias ToolArgumentsSchema
	aux := &struct {
		Definitions map[string]any `json:"definitions,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(tis),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If $defs wasn't provided but definitions was, use definitions
	if tis.Defs == nil && aux.Definitions != nil {
		tis.Defs = aux.Definitions
	}

	return nil
}

type ToolAnnotation struct {
	// Human-readable title for the tool
	Title string `json:"title,omitempty"`
	// If true, the tool does not modify its environment
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`
	// If true, the tool may perform destructive updates
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// If true, repeated calls with same args have no additional effect
	IdempotentHint *bool `json:"idempotentHint,omitempty"`
	// If true, tool interacts with external entities
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

// ToolOption is a function that configures a Tool.
// It provides a flexible way to set various properties of a Tool using the functional options pattern.
type ToolOption func(*Tool)

// PropertyOption is a function that configures a property in a Tool's input schema.
// It allows for flexible configuration of JSON Schema properties using the functional options pattern.
type PropertyOption func(map[string]any)

//
// Core Tool Functions
//

// NewTool creates a new Tool with the given name and options.
// The tool will have an object-type input schema with configurable properties.
// Options are applied in order, allowing for flexible tool configuration.
func NewTool(name string, opts ...ToolOption) Tool {
	tool := Tool{
		Name: name,
		InputSchema: ToolInputSchema{
			Type:       "object",
			Properties: make(map[string]any),
			Required:   nil, // Will be omitted from JSON if empty
		},
		Annotations: ToolAnnotation{
			Title:           "",
			ReadOnlyHint:    ToBoolPtr(false),
			DestructiveHint: ToBoolPtr(true),
			IdempotentHint:  ToBoolPtr(false),
			OpenWorldHint:   ToBoolPtr(true),
		},
	}

	for _, opt := range opts {
		opt(&tool)
	}

	return tool
}

// NewToolWithRawSchema creates a new Tool with the given name and a raw JSON
// Schema. This allows for arbitrary JSON Schema to be used for the tool's input
// schema.
//
// NOTE a [Tool] built in such a way is incompatible with the [ToolOption] and
// runtime errors will result from supplying a [ToolOption] to a [Tool] built
// with this function.
func NewToolWithRawSchema(name, description string, schema json.RawMessage) Tool {
	tool := Tool{
		Name:           name,
		Description:    description,
		RawInputSchema: schema,
	}

	return tool
}

// WithDescription adds a description to the Tool.
// The description should provide a clear, human-readable explanation of what the tool does.
func WithDescription(description string) ToolOption {
	return func(t *Tool) {
		t.Description = description
	}
}

// WithInputSchema creates a ToolOption that sets the input schema for a tool.
// It accepts any Go type, usually a struct, and automatically generates a JSON schema from it.
func WithInputSchema[T any]() ToolOption {
	return func(t *Tool) {
		var zero T

		// Generate schema using invopop/jsonschema library
		// Configure reflector to generate clean, MCP-compatible schemas
		reflector := jsonschema.Reflector{
			DoNotReference:            true, // Removes $defs map, outputs entire structure inline
			Anonymous:                 true, // Hides auto-generated Schema IDs
			AllowAdditionalProperties: true, // Removes additionalProperties: false
		}
		schema := reflector.Reflect(zero)

		// Clean up schema for MCP compliance
		schema.Version = "" // Remove $schema field

		// Convert to raw JSON for MCP
		mcpSchema, err := json.Marshal(schema)
		if err != nil {
			// Skip and maintain backward compatibility
			return
		}

		t.InputSchema.Type = ""
		t.RawInputSchema = json.RawMessage(mcpSchema)
	}
}

// WithRawInputSchema sets a raw JSON schema for the tool's input.
// Use this when you need full control over the schema or when working with
// complex schemas that can't be generated from Go types. The jsonschema library
// can handle complex schemas and provides nice extension points, so be sure to
// check that out before using this.
func WithRawInputSchema(schema json.RawMessage) ToolOption {
	return func(t *Tool) {
		t.RawInputSchema = schema
	}
}

// WithOutputSchema creates a ToolOption that sets the output schema for a tool.
// It accepts any Go type, usually a struct, and automatically generates a JSON schema from it.
func WithOutputSchema[T any]() ToolOption {
	return func(t *Tool) {
		var zero T

		// Generate schema using invopop/jsonschema library
		// Configure reflector to generate clean, MCP-compatible schemas
		reflector := jsonschema.Reflector{
			DoNotReference:            true, // Removes $defs map, outputs entire structure inline
			Anonymous:                 true, // Hides auto-generated Schema IDs
			AllowAdditionalProperties: true, // Removes additionalProperties: false
		}
		schema := reflector.Reflect(zero)

		// Clean up schema for MCP compliance
		schema.Version = "" // Remove $schema field

		// Convert to raw JSON for MCP
		mcpSchema, err := json.Marshal(schema)
		if err != nil {
			// Skip and maintain backward compatibility
			return
		}

		// Retrieve the schema from raw JSON
		if err := json.Unmarshal(mcpSchema, &t.OutputSchema); err != nil {
			// Skip and maintain backward compatibility
			return
		}

		// Always set the type to "object" as of the current MCP spec
		// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#output-schema
		t.OutputSchema.Type = "object"
	}
}

// WithRawOutputSchema sets a raw JSON schema for the tool's output.
// Use this when you need full control over the schema or when working with
// complex schemas that can't be generated from Go types. The jsonschema library
// can handle complex schemas and provides nice extension points, so be sure to
// check that out before using this.
func WithRawOutputSchema(schema json.RawMessage) ToolOption {
	return func(t *Tool) {
		t.RawOutputSchema = schema
	}
}

// WithToolAnnotation adds optional hints about the Tool.
func WithToolAnnotation(annotation ToolAnnotation) ToolOption {
	return func(t *Tool) {
		t.Annotations = annotation
	}
}

// WithTitleAnnotation sets the Title field of the Tool's Annotations.
// It provides a human-readable title for the tool.
func WithTitleAnnotation(title string) ToolOption {
	return func(t *Tool) {
		t.Annotations.Title = title
	}
}

// WithReadOnlyHintAnnotation sets the ReadOnlyHint field of the Tool's Annotations.
// If true, it indicates the tool does not modify its environment.
func WithReadOnlyHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.ReadOnlyHint = &value
	}
}

// WithDestructiveHintAnnotation sets the DestructiveHint field of the Tool's Annotations.
// If true, it indicates the tool may perform destructive updates.
func WithDestructiveHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.DestructiveHint = &value
	}
}

// WithIdempotentHintAnnotation sets the IdempotentHint field of the Tool's Annotations.
// If true, it indicates repeated calls with the same arguments have no additional effect.
func WithIdempotentHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.IdempotentHint = &value
	}
}

// WithOpenWorldHintAnnotation sets the OpenWorldHint field of the Tool's Annotations.
// If true, it indicates the tool interacts with external entities.
func WithOpenWorldHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.OpenWorldHint = &value
	}
}

//
// Common Property Options
//

// Description adds a description to a property in the JSON Schema.
// The description should explain the purpose and expected values of the property.
func Description(desc string) PropertyOption {
	return func(schema map[string]any) {
		schema["description"] = desc
	}
}

// Required marks a property as required in the tool's input schema.
// Required properties must be provided when using the tool.
func Required() PropertyOption {
	return func(schema map[string]any) {
		schema["required"] = true
	}
}

// Title adds a display-friendly title to a property in the JSON Schema.
// This title can be used by UI components to show a more readable property name.
func Title(title string) PropertyOption {
	return func(schema map[string]any) {
		schema["title"] = title
	}
}

//
// String Property Options
//

// DefaultString sets the default value for a string property.
// This value will be used if the property is not explicitly provided.
func DefaultString(value string) PropertyOption {
	return func(schema map[string]any) {
		schema["default"] = value
	}
}

// Enum specifies a list of allowed values for a string property.
// The property value must be one of the specified enum values.
func Enum(values ...string) PropertyOption {
	return func(schema map[string]any) {
		schema["enum"] = values
	}
}

// MaxLength sets the maximum length for a string property.
// The string value must not exceed this length.
func MaxLength(max int) PropertyOption {
	return func(schema map[string]any) {
		schema["maxLength"] = max
	}
}

// MinLength sets the minimum length for a string property.
// The string value must be at least this length.
func MinLength(min int) PropertyOption {
	return func(schema map[string]any) {
		schema["minLength"] = min
	}
}

// Pattern sets a regex pattern that a string property must match.
// The string value must conform to the specified regular expression.
func Pattern(pattern string) PropertyOption {
	return func(schema map[string]any) {
		schema["pattern"] = pattern
	}
}

//
// Number Property Options
//

// DefaultNumber sets the default value for a number property.
// This value will be used if the property is not explicitly provided.
func DefaultNumber(value float64) PropertyOption {
	return func(schema map[string]any) {
		schema["default"] = value
	}
}

// Max sets the maximum value for a number property.
// The number value must not exceed this maximum.
func Max(max float64) PropertyOption {
	return func(schema map[string]any) {
		schema["maximum"] = max
	}
}

// Min sets the minimum value for a number property.
// The number value must not be less than this minimum.
func Min(min float64) PropertyOption {
	return func(schema map[string]any) {
		schema["minimum"] = min
	}
}

// MultipleOf specifies that a number must be a multiple of the given value.
// The number value must be divisible by this value.
func MultipleOf(value float64) PropertyOption {
	return func(schema map[string]any) {
		schema["multipleOf"] = value
	}
}

//
// Boolean Property Options
//

// DefaultBool sets the default value for a boolean property.
// This value will be used if the property is not explicitly provided.
func DefaultBool(value bool) PropertyOption {
	return func(schema map[string]any) {
		schema["default"] = value
	}
}

//
// Array Property Options
//

// DefaultArray sets the default value for an array property.
// This value will be used if the property is not explicitly provided.
func DefaultArray[T any](value []T) PropertyOption {
	return func(schema map[string]any) {
		schema["default"] = value
	}
}

//
// Property Type Helpers
//

// WithBoolean adds a boolean property to the tool schema.
// It accepts property options to configure the boolean property's behavior and constraints.
func WithBoolean(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{
			"type": "boolean",
		}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// WithNumber adds a number property to the tool schema.
// It accepts property options to configure the number property's behavior and constraints.
func WithNumber(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{
			"type": "number",
		}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// WithString adds a string property to the tool schema.
// It accepts property options to configure the string property's behavior and constraints.
func WithString(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{
			"type": "string",
		}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// WithObject adds an object property to the tool schema.
// It accepts property options to configure the object property's behavior and constraints.
func WithObject(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// WithArray returns a ToolOption that adds an array-typed property with the given name to a Tool's input schema.
// It applies provided PropertyOption functions to configure the property's schema, moves a `required` flag
// from the property schema into the Tool's InputSchema.Required slice when present, and registers the resulting
// schema under InputSchema.Properties[name].
func WithArray(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{
			"type": "array",
		}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// WithAny adds an input property named name with no predefined JSON Schema type to the Tool's input schema.
// The returned ToolOption applies the provided PropertyOption functions to the property's schema, moves a property-level
// `required` flag into the Tool's InputSchema.Required list if present, and stores the resulting schema under InputSchema.Properties[name].
func WithAny(name string, opts ...PropertyOption) ToolOption {
	return func(t *Tool) {
		schema := map[string]any{}

		for _, opt := range opts {
			opt(schema)
		}

		// Remove required from property schema and add to InputSchema.required
		if required, ok := schema["required"].(bool); ok && required {
			delete(schema, "required")
			t.InputSchema.Required = append(t.InputSchema.Required, name)
		}

		t.InputSchema.Properties[name] = schema
	}
}

// Properties sets the "properties" map for an object schema.
// The returned PropertyOption stores the provided map under the schema's "properties" key.
func Properties(props map[string]any) PropertyOption {
	return func(schema map[string]any) {
		schema["properties"] = props
	}
}

// AdditionalProperties specifies whether additional properties are allowed in the object
// or defines a schema for additional properties
func AdditionalProperties(schema any) PropertyOption {
	return func(schemaMap map[string]any) {
		schemaMap["additionalProperties"] = schema
	}
}

// MinProperties sets the minimum number of properties for an object
func MinProperties(min int) PropertyOption {
	return func(schema map[string]any) {
		schema["minProperties"] = min
	}
}

// MaxProperties sets the maximum number of properties for an object
func MaxProperties(max int) PropertyOption {
	return func(schema map[string]any) {
		schema["maxProperties"] = max
	}
}

// PropertyNames defines a schema for property names in an object
func PropertyNames(schema map[string]any) PropertyOption {
	return func(schemaMap map[string]any) {
		schemaMap["propertyNames"] = schema
	}
}

// Items defines the schema for array items.
// Accepts any schema definition for maximum flexibility.
//
// Example:
//
//	Items(map[string]any{
//	    "type": "object",
//	    "properties": map[string]any{
//	        "name": map[string]any{"type": "string"},
//	        "age": map[string]any{"type": "number"},
//	    },
//	})
//
// For simple types, use ItemsString(), ItemsNumber(), ItemsBoolean() instead.
func Items(schema any) PropertyOption {
	return func(schemaMap map[string]any) {
		schemaMap["items"] = schema
	}
}

// MinItems sets the minimum number of items for an array
func MinItems(min int) PropertyOption {
	return func(schema map[string]any) {
		schema["minItems"] = min
	}
}

// MaxItems sets the maximum number of items for an array
func MaxItems(max int) PropertyOption {
	return func(schema map[string]any) {
		schema["maxItems"] = max
	}
}

// UniqueItems specifies whether array items must be unique
func UniqueItems(unique bool) PropertyOption {
	return func(schema map[string]any) {
		schema["uniqueItems"] = unique
	}
}

// WithStringItems configures an array's items to be of type string.
//
// Supported options: Description(), DefaultString(), Enum(), MaxLength(), MinLength(), Pattern()
// Note: Options like Required() are not valid for item schemas and will be ignored.
//
// Examples:
//
//	mcp.WithArray("tags", mcp.WithStringItems())
//	mcp.WithArray("colors", mcp.WithStringItems(mcp.Enum("red", "green", "blue")))
//	mcp.WithArray("names", mcp.WithStringItems(mcp.MinLength(1), mcp.MaxLength(50)))
//
// Limitations: Only supports simple string arrays. Use Items() for complex objects.
func WithStringItems(opts ...PropertyOption) PropertyOption {
	return func(schema map[string]any) {
		itemSchema := map[string]any{
			"type": "string",
		}

		for _, opt := range opts {
			opt(itemSchema)
		}

		schema["items"] = itemSchema
	}
}

// WithStringEnumItems configures an array's items to be of type string with a specified enum.
// Example:
//
//	mcp.WithArray("priority", mcp.WithStringEnumItems([]string{"low", "medium", "high"}))
//
// Limitations: Only supports string enums. Use WithStringItems(Enum(...)) for more flexibility.
func WithStringEnumItems(values []string) PropertyOption {
	return func(schema map[string]any) {
		schema["items"] = map[string]any{
			"type": "string",
			"enum": values,
		}
	}
}

// WithNumberItems configures an array's items to be of type number.
//
// Supported options: Description(), DefaultNumber(), Min(), Max(), MultipleOf()
// Note: Options like Required() are not valid for item schemas and will be ignored.
//
// Examples:
//
//	mcp.WithArray("scores", mcp.WithNumberItems(mcp.Min(0), mcp.Max(100)))
//	mcp.WithArray("prices", mcp.WithNumberItems(mcp.Min(0)))
//
// Limitations: Only supports simple number arrays. Use Items() for complex objects.
func WithNumberItems(opts ...PropertyOption) PropertyOption {
	return func(schema map[string]any) {
		itemSchema := map[string]any{
			"type": "number",
		}

		for _, opt := range opts {
			opt(itemSchema)
		}

		schema["items"] = itemSchema
	}
}

// WithBooleanItems configures an array's items to be of type boolean.
//
// Supported options: Description(), DefaultBool()
// Note: Options like Required() are not valid for item schemas and will be ignored.
//
// Examples:
//
//	mcp.WithArray("flags", mcp.WithBooleanItems())
//	mcp.WithArray("permissions", mcp.WithBooleanItems(mcp.Description("User permissions")))
//
// Limitations: Only supports simple boolean arrays. Use Items() for complex objects.
func WithBooleanItems(opts ...PropertyOption) PropertyOption {
	return func(schema map[string]any) {
		itemSchema := map[string]any{
			"type": "boolean",
		}

		for _, opt := range opts {
			opt(itemSchema)
		}

		schema["items"] = itemSchema
	}
}
