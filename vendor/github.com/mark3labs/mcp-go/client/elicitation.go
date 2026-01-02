package client

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ElicitationHandler defines the interface for handling elicitation requests from servers.
// Clients can implement this interface to request additional information from users.
type ElicitationHandler interface {
	// Elicit handles an elicitation request from the server and returns the user's response.
	// The implementation should:
	// 1. Present the request message to the user
	// 2. Validate input against the requested schema
	// 3. Allow the user to accept, decline, or cancel
	// 4. Return the appropriate response
	Elicit(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error)
}
