package server

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
)

var (
	// ErrNoClientSession is returned when there is no active client session in the context
	ErrNoClientSession = errors.New("no active client session")
	// ErrRootsNotSupported is returned when the session does not support roots
	ErrRootsNotSupported = errors.New("session does not support roots")
)

// RequestRoots sends an list roots request to the client.
// The client must have declared roots capability during initialization.
// The session must implement SessionWithRoots to support this operation.
func (s *MCPServer) RequestRoots(ctx context.Context, request mcp.ListRootsRequest) (*mcp.ListRootsResult, error) {
	session := ClientSessionFromContext(ctx)
	if session == nil {
		return nil, ErrNoClientSession
	}

	// Check if the session supports roots requests
	if rootsSession, ok := session.(SessionWithRoots); ok {
		return rootsSession.ListRoots(ctx, request)
	}

	return nil, ErrRootsNotSupported
}
