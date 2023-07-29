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
	"github.com/bishopfox/sliver/client/command"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

func Command(con *client.SliverClient, serverCmds console.Commands) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
		RunE: func(cmd *cobra.Command, args []string) error {
			con.IsCLI = false

			// Bind commands to the closed-loop console.
			server := con.App.Menu(consts.ServerMenu)
			server.SetCommands(serverCmds)

			sliver := con.App.Menu(consts.ImplantMenu)
			sliver.SetCommands(command.SliverCommands(con))

			// Start the console, blocking until player exit.
			return con.StartConsole()
		},
	}

	return consoleCmd
}
