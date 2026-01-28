package mcp

import (
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	slivermcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/spf13/cobra"
)

// McpStartCmd starts the local MCP server.
func McpStartCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	rawTransport, _ := cmd.Flags().GetString("transport")
	transport, err := slivermcp.ParseTransport(rawTransport)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	listen, _ := cmd.Flags().GetString("listen")
	name, _ := cmd.Flags().GetString("name")
	version, _ := cmd.Flags().GetString("version")

	cfg := slivermcp.Config{
		Transport:     transport,
		ListenAddress: listen,
		ServerName:    name,
		ServerVersion: version,
	}.WithDefaults()

	msg := `Do you know what prompt injection is and are you an adult?`
	if !settings.IsUserAnAdultWithPrompt(con, msg) {
		con.PrintErrorf("Failed to start MCP server, the user is not qualified to use feature\n")
		return
	}

	if err := slivermcp.Start(cfg, con.Rpc); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Starting MCP server (%s) on %s\n", cfg.Transport, cfg.ListenAddress)
	endpoint, err := cfg.EndpointURL()
	if err == nil {
		con.PrintInfof("Endpoint: %s\n", endpoint)
	}
}
