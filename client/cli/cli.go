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
	"os"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/client/commands"

	"github.com/bishopfox/sliver/client/command"
	sliverConsole "github.com/bishopfox/sliver/client/command/console"
	client "github.com/bishopfox/sliver/client/console"
)

// Execute - Run the sliver client binary.
func Execute() {
	// Create a client-only (remote TLS-transported connections)
	// Sliver Client, prepared with a working reeflective/teamclient.
	// The teamclient automatically handles remote teamserver configuration
	// prompting/loading and use, as well as other things.
	con, err := client.NewSliverClient()
	if err != nil {
		panic(err)
	}

	// Generate our complete Sliver Framework command-line interface.
	rootCmd := SliverCLI(con)

	// Version
	rootCmd.AddCommand(cmdVersion)

	// Run the sliver client binary.
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// SliverCLI returns the entire command tree of the Sliver Framework as yielder functions.
// The ready-to-execute command tree (root *cobra.Command) returned is correctly equipped
// with all prerunners needed to connect to remote Sliver teamservers.
// It will also register the appropriate teamclient management commands.
func SliverCLI(con *client.SliverClient) (root *cobra.Command) {
	teamclientCmds := func(con *client.SliverClient) []*cobra.Command {
		return []*cobra.Command{
			commands.Generate(con.Teamclient),
		}
	}

	// Generate a single tree instance of server commands:
	// These are used as the primary, one-exec-only CLI of Sliver, and are equipped with
	// a pre-runner ensuring the server and its teamclient are set up and connected.
	server := command.ServerCommands(con, teamclientCmds)

	root = server()
	root.Use = "sliver-client" // Needed by completion scripts.

	// Bind the closed-loop console.
	// The console shares the same setup/connection pre-runners as other commands,
	// but the command yielders we pass as arguments don't: this is because we only
	// need one connection for the entire lifetime of the console.
	root.AddCommand(sliverConsole.Command(con, server))

	// Implant.
	// The implant command allows users to run commands on slivers from their
	// system shell. It makes use of pre-runners for connecting to the server
	// and binding sliver commands. These same pre-runners are also used for
	// command completion/filtering purposes.
	root.AddCommand(implantCmd(con, command.SliverCommands(con)))

	// Pre/post runners and completions.
	command.BindPreRun(root, con.PreRunConnect)
	command.BindPostRun(root, con.PostRunDisconnect)

	// Generate the root completion command.
	carapace.Gen(root)

	return root
}
