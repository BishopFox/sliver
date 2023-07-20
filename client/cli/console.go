package cli

import (
	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/spf13/cobra"
)

// consoleCmd generates the console with required pre/post runners.
func consoleCmd(con *console.SliverClient) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
		RunE: func(cmd *cobra.Command, args []string) error {
			// return console.StartClient(con, nil, nil, true)

			// Bind commands to the app
			server := con.App.Menu(consts.ServerMenu)
			server.SetCommands(command.ServerCommands(con, nil))

			sliver := con.App.Menu(consts.ImplantMenu)
			sliver.SetCommands(command.SliverCommands(con))

			return con.App.Start()
		},
	}

	return consoleCmd
}
