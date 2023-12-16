package console

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"os"

	"github.com/reeflective/console"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/reaction"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Command returns the closed-loop Sliver console command.
//
// The latter requires only the set of "server" commands, that is, all commands that
// do not require an active target to run on. This is only because sliver-client/server
// binaries are distinct, and the implant command tree does not care about this, so it
// is always the same in the console.
func Command(con *client.SliverClient, serverCmds console.Commands) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:     "console",
		Short:   "Start the sliver client console",
		GroupID: consts.GenericHelpGroup,
		RunE: func(cmd *cobra.Command, args []string) error {

			// Bind commands to the closed-loop console.
			server := con.App.Menu(consts.ServerMenu)
			server.SetCommands(serverCmds)

			sliver := con.App.Menu(consts.ImplantMenu)
			sliver.SetCommands(command.SliverCommands(con))

			// Reactions
			n, err := reaction.LoadReactions()
			if err != nil && !os.IsNotExist(err) {
				con.PrintErrorf("Failed to load reactions: %s\n", err)
			} else if n > 0 {
				con.PrintInfof("Loaded %d reaction(s) from disk\n", n)
			}

			// Start the console, blocking until player exit.
			return con.StartConsole()
		},
	}

	return consoleCmd
}
