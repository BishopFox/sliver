package transport

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// NewJSONRPCErrorResponse creates a new JSONRPCResponse with an error.
func NewJSONRPCErrorResponse(id mcp.RequestId, code int, message string, data any) *JSONRPCResponse {
	details := mcp.NewJSONRPCErrorDetails(code, message, data)
	return &JSONRPCResponse{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Error:   &details,
	}
}

// NewJSONRPCResultResponse creates a new JSONRPCResponse with a result.
func NewJSONRPCResultResponse(id mcp.RequestId, result json.RawMessage) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}
