package mcp

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	clientmcp "github.com/bishopfox/sliver/client/mcp"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the `mcp` command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	defaultConfig := clientmcp.DefaultConfig()

	mcpCmd := &cobra.Command{
		Use:     consts.MCPStr,
		Short:   "Manage the MCP server",
		Long:    help.GetHelpFor([]string{consts.MCPStr}),
		GroupID: consts.NetworkHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			McpCmd(cmd, con, args)
		},
	}

	startCmd := &cobra.Command{
		Use:   consts.StartStr,
		Short: "Start the MCP server",
		Long:  help.GetHelpFor([]string{consts.MCPStr, consts.StartStr}),
		Run: func(cmd *cobra.Command, args []string) {
			McpStartCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, startCmd, func(f *pflag.FlagSet) {
		f.String("transport", string(defaultConfig.Transport), "mcp transport (http or sse)")
		f.String("listen", defaultConfig.ListenAddress, "listen address for the MCP server")
		f.String("name", defaultConfig.ServerName, "server name for MCP initialize")
		f.String("version", defaultConfig.ServerVersion, "server version for MCP initialize")
	})
	flags.BindFlagCompletions(startCmd, func(comp *carapace.ActionMap) {
		(*comp)["transport"] = carapace.ActionValues("http", "sse")
	})
	mcpCmd.AddCommand(startCmd)

	stopCmd := &cobra.Command{
		Use:   consts.StopStr,
		Short: "Stop the MCP server",
		Long:  help.GetHelpFor([]string{consts.MCPStr, consts.StopStr}),
		Run: func(cmd *cobra.Command, args []string) {
			McpStopCmd(cmd, con, args)
		},
	}
	mcpCmd.AddCommand(stopCmd)

	consoleCmd := &cobra.Command{
		Use:     "console",
		Aliases: []string{"repl"},
		Short:   "Open an MCP console session",
		Long:    help.GetHelpFor([]string{consts.MCPStr, "console"}),
		Run: func(cmd *cobra.Command, args []string) {
			McpConsoleCmd(cmd, con, args)
		},
	}
	flags.Bind("", false, consoleCmd, func(f *pflag.FlagSet) {
		f.String("transport", string(defaultConfig.Transport), "mcp transport (http or sse)")
		f.String("url", "", "mcp server URL (defaults to local server settings)")
	})
	flags.BindFlagCompletions(consoleCmd, func(comp *carapace.ActionMap) {
		(*comp)["transport"] = carapace.ActionValues("http", "sse")
	})
	mcpCmd.AddCommand(consoleCmd)

	return []*cobra.Command{mcpCmd}
}
