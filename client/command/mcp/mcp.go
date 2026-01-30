package mcp

import (
	"time"

	"github.com/bishopfox/sliver/client/console"
	clientmcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/spf13/cobra"
)

// McpCmd prints the current MCP server state.
func McpCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	status := clientmcp.GetStatus()
	state := "stopped"
	if status.Running {
		state = "running"
	}

	con.Printf("Status: %s\n", state)
	con.Printf("Transport: %s\n", status.Config.Transport)
	con.Printf("Listen: %s\n", status.Config.ListenAddress)

	endpoint, err := status.Config.EndpointURL()
	if err == nil {
		con.Printf("Endpoint: %s\n", endpoint)
	} else {
		con.PrintErrorf("Endpoint: %s\n", err)
	}

	if status.Running && !status.StartedAt.IsZero() {
		uptime := time.Since(status.StartedAt).Truncate(time.Second)
		con.Printf("Uptime: %s\n", uptime)
	}
	if status.LastError != "" {
		con.PrintErrorf("Last error: %s\n", status.LastError)
	}
}
