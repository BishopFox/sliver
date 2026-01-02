package mcp

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const helloToolName = "hello_world"

// SliverMCPServer wraps the MCP server with Sliver RPC access for handlers.
type SliverMCPServer struct {
	Rpc    rpcpb.SliverRPCClient
	server *mcpserver.MCPServer
}

func newServer(cfg Config, rpc rpcpb.SliverRPCClient) *SliverMCPServer {
	base := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	helloTool := mcpapi.NewTool(
		helloToolName,
		mcpapi.WithDescription("Basic MCP hello world tool"),
	)
	srv := &SliverMCPServer{
		Rpc:    rpc,
		server: base,
	}
	srv.server.AddTool(helloTool, srv.helloWorldHandler)
	return srv
}

func (s *SliverMCPServer) helloWorldHandler(_ context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	return mcpapi.NewToolResultText("Hello world"), nil
}
