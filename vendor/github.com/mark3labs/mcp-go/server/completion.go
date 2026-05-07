package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

type PromptCompletionProvider interface {
	// CompletePromptArgument provides completions for a prompt argument
	CompletePromptArgument(ctx context.Context, promptName string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error)
}

type ResourceCompletionProvider interface {
	// CompleteResourceArgument provides completions for a resource template argument
	CompleteResourceArgument(ctx context.Context, uri string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error)
}

// DefaultCompletionProvider returns no completions (fallback)
type DefaultPromptCompletionProvider struct{}

func (p *DefaultPromptCompletionProvider) CompletePromptArgument(ctx context.Context, promptName string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error) {
	return &mcp.Completion{
		Values: []string{},
	}, nil
}

// DefaultResourceCompletionProvider returns no completions (fallback)
type DefaultResourceCompletionProvider struct{}

func (p *DefaultResourceCompletionProvider) CompleteResourceArgument(ctx context.Context, uri string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error) {
	return &mcp.Completion{
		Values: []string{},
	}, nil
}
