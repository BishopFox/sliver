package mcp

import (
	"context"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const helloToolName = "hello_world"

func newServer(cfg Config) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	helloTool := mcpapi.NewTool(
		helloToolName,
		mcpapi.WithDescription("Basic MCP hello world tool"),
	)
	srv.AddTool(helloTool, helloWorldHandler)
	return srv
}

func helloWorldHandler(_ context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	return mcpapi.NewToolResultText("Hello world"), nil
}
