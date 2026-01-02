package server

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
)

var (
	// ErrNoActiveSession is returned when there is no active session in the context
	ErrNoActiveSession = errors.New("no active session")
	// ErrElicitationNotSupported is returned when the session does not support elicitation
	ErrElicitationNotSupported = errors.New("session does not support elicitation")
)

// RequestElicitation sends an elicitation request to the client.
// The client must have declared elicitation capability during initialization.
// The session must implement SessionWithElicitation to support this operation.
func (s *MCPServer) RequestElicitation(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error) {
	session := ClientSessionFromContext(ctx)
	if session == nil {
		return nil, ErrNoActiveSession
	}

	// Check if the session supports elicitation requests
	if elicitationSession, ok := session.(SessionWithElicitation); ok {
		return elicitationSession.RequestElicitation(ctx, request)
	}

	return nil, ErrElicitationNotSupported
}
