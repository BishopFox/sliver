package cli

import (
	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

// consoleCmd generates the console with required pre/post runners.
func consoleCmd(con *console.SliverClient) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
		RunE: func(cmd *cobra.Command, args []string) error {
			return console.StartClient(con, command.ServerCommands(con, nil), command.SliverCommands(con), true)
		},
	}

	return consoleCmd
}
