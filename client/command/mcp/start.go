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
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm && !settings.IsUserAnAdultWithPrompt(con, msg) {
		con.PrintErrorf("Failed to start MCP server, the user is not qualified to use feature\n")
		con.PrintInfof("Use --yes to skip the confirmation prompt, or toggle it with 'settings autoadult'\n")
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
	status := slivermcp.GetStatus()
	if status.AuthHeader != "" {
		con.PrintInfof("Auth Header: %s\n", status.AuthHeader)
	}
	if status.AuthToken != "" {
		con.PrintInfof("Auth Token: %s\n", status.AuthToken)
	}
	if status.AuthConfigPath != "" {
		con.PrintInfof("Auth Config: %s\n", status.AuthConfigPath)
	}
}
