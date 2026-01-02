package mcp

import (
	"context"
	"fmt"
)

// TypedToolHandlerFunc is a function that handles a tool call with typed arguments
type TypedToolHandlerFunc[T any] func(ctx context.Context, request CallToolRequest, args T) (*CallToolResult, error)

// StructuredToolHandlerFunc is a function that handles a tool call with typed arguments and returns structured output
type StructuredToolHandlerFunc[TArgs any, TResult any] func(ctx context.Context, request CallToolRequest, args TArgs) (TResult, error)

// NewTypedToolHandler creates a ToolHandlerFunc that automatically binds arguments to a typed struct
func NewTypedToolHandler[T any](handler TypedToolHandlerFunc[T]) func(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return func(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
		var args T
		if err := request.BindArguments(&args); err != nil {
			return NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
		}
		return handler(ctx, request, args)
	}
}

// NewStructuredToolHandler creates a ToolHandlerFunc that automatically binds arguments to a typed struct
// and returns structured output. It automatically creates both structured and
// text content (from the structured output) for backwards compatibility.
func NewStructuredToolHandler[TArgs any, TResult any](handler StructuredToolHandlerFunc[TArgs, TResult]) func(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	return func(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
		var args TArgs
		if err := request.BindArguments(&args); err != nil {
			return NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
		}

		result, err := handler(ctx, request, args)
		if err != nil {
			return NewToolResultError(fmt.Sprintf("tool execution failed: %v", err)), nil
		}

		return NewToolResultStructuredOnly(result), nil
	}
}
