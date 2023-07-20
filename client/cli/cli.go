package cli

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
	"fmt"
	"os"

	"github.com/bishopfox/sliver/client/command"
	consoleCmd "github.com/bishopfox/sliver/client/command/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// Execute - Execute root command.
func Execute() {
	con := newSliverTeam()

	serverCmds := command.ServerCommands(con, nil)()
	serverCmds.Use = "sliver-client"

	// Version
	serverCmds.AddCommand(cmdVersion)

	preRun := func(_ *cobra.Command, _ []string) error {
		return con.Teamclient.Connect()
	}

	serverCmds.PersistentPreRunE = preRun

	postRun := func(_ *cobra.Command, _ []string) error {
		return con.Teamclient.Disconnect()
	}

	// serverCmds.PersistentPostRunE = postRun

	// Client console.
	// All commands and RPC connection are generated WITHIN the command RunE():
	// that means there should be no redundant command tree/RPC connections with
	// other command trees below, such as the implant one.
	consoleServerCmds := command.ServerCommands(con, nil)
	consoleSliverCmds := command.SliverCommands(con)

	serverCmds.AddCommand(consoleCmd.Command(con, consoleServerCmds, consoleSliverCmds))

	// Implant.
	// The implant command allows users to run commands on slivers from their
	// system shell. It makes use of pre-runners for connecting to the server
	// and binding sliver commands. These same pre-runners are also used for
	// command completion/filtering purposes.
	serverCmds.AddCommand(implantCmd(con))

	// Completions
	comps := carapace.Gen(serverCmds)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		preRun(cmd, args)
	})
	comps.PostRun(func(cmd *cobra.Command, args []string) {
		postRun(cmd, args)
	})

	if err := serverCmds.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
