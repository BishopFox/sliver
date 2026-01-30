package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// EnableSampling enables sampling capabilities for the server.
// This allows the server to send sampling requests to clients that support it.
func (s *MCPServer) EnableSampling() {
	s.capabilitiesMu.Lock()
	defer s.capabilitiesMu.Unlock()

	enabled := true
	s.capabilities.sampling = &enabled
}

// RequestSampling sends a sampling request to the client.
// The client must have declared sampling capability during initialization.
func (s *MCPServer) RequestSampling(ctx context.Context, request mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
	session := ClientSessionFromContext(ctx)
	if session == nil {
		return nil, fmt.Errorf("no active session")
	}

	// Check if the session supports sampling requests
	if samplingSession, ok := session.(SessionWithSampling); ok {
		return samplingSession.RequestSampling(ctx, request)
	}

	// Check for inprocess sampling handler in context
	if handler := InProcessSamplingHandlerFromContext(ctx); handler != nil {
		return handler.CreateMessage(ctx, request)
	}

	return nil, fmt.Errorf("session does not support sampling")
}

// SessionWithSampling extends ClientSession to support sampling requests.
type SessionWithSampling interface {
	ClientSession
	RequestSampling(ctx context.Context, request mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error)
}

// inProcessSamplingHandlerKey is the context key for storing inprocess sampling handler
type inProcessSamplingHandlerKey struct{}

// WithInProcessSamplingHandler adds a sampling handler to the context for inprocess clients
func WithInProcessSamplingHandler(ctx context.Context, handler SamplingHandler) context.Context {
	return context.WithValue(ctx, inProcessSamplingHandlerKey{}, handler)
}

// InProcessSamplingHandlerFromContext retrieves the inprocess sampling handler from context
func InProcessSamplingHandlerFromContext(ctx context.Context) SamplingHandler {
	if handler, ok := ctx.Value(inProcessSamplingHandlerKey{}).(SamplingHandler); ok {
		return handler
	}
	return nil
}
