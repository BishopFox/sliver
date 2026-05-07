package client

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// RootsHandler defines the interface for handling roots requests from servers.
// Clients can implement this interface to provide roots list to servers.
type RootsHandler interface {
	// ListRoots handles a list root request from the server and returns the roots list.
	// The implementation should:
	// 1. Validate input against the requested schema
	// 2. Return the appropriate response
	ListRoots(ctx context.Context, request mcp.ListRootsRequest) (*mcp.ListRootsResult, error)
}
