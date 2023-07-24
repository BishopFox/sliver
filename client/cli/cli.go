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
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/reeflective/console"
	teamclient "github.com/reeflective/team/client"
	"github.com/reeflective/team/client/commands"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// Execute - Run the sliver-client binary.
func Execute() {
	// Create a client-only (remote TLS-transported connections)
	// Sliver Client, prepared with a working reeflective/teamclient.
	// The teamclient automatically handles remote teamserver configuration
	// prompting/loading and use, as well as other things.
	con := newSliverClient()

	// Prepare the entire Sliver Command-line Interface as yielder functions.
	serverCmds, sliverCmds := getSliverCommands(con)

	// Generate a single tree instance of server commands:
	// These are used as the primary, one-exec-only CLI of Sliver, and are equipped with
	// a pre-runner ensuring the server and its teamclient are set up and connected.
	rootCmd := serverCmds()
	rootCmd.Use = "sliver-client" // Needed by completion scripts.

	// Version
	rootCmd.AddCommand(cmdVersion)

	// Bind the closed-loop console:
	// The console shares the same setup/connection pre-runners as other commands,
	// but the command yielders we pass as arguments don't: this is because we only
	// need one connection for the entire lifetime of the console.
	rootCmd.AddCommand(consoleCmd.Command(con, serverCmds, sliverCmds))

	// Implant.
	// The implant command allows users to run commands on slivers from their
	// system shell. It makes use of pre-runners for connecting to the server
	// and binding sliver commands. These same pre-runners are also used for
	// command completion/filtering purposes.
	rootCmd.AddCommand(implantCmd(con))

	// Pre/post runners and completions.
	command.BindRunners(rootCmd, true, preRunClient(con))
	// command.BindRunners(rootCmd, false, postRunClient(con))

	carapace.Gen(rootCmd)

	// Run the sliver client binary.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// newSliverClient creates a new application teamclient.
// From this teamclient, configured to work with TLS connections
// to remote teamservers, generate a new Sliver Client.
func newSliverClient() *client.SliverClient {
	gTeamclient := transport.NewTeamClient()

	con, opts := client.NewSliverClient(gTeamclient)

	teamclient, err := teamclient.New("sliver", gTeamclient, opts...)
	if err != nil {
		panic(err)
	}

	con.Teamclient = teamclient

	return con
}

// getSliverCommands returns the entire command tree of the Sliver Framework as yielder functions.
func getSliverCommands(con *client.SliverClient) (server, sliver console.Commands) {
	teamclientCmds := func(con *client.SliverClient) []*cobra.Command {
		return []*cobra.Command{
			commands.Generate(con.Teamclient),
		}
	}

	serverCmds := command.ServerCommands(con, teamclientCmds)
	sliverCmds := command.SliverCommands(con)

	return serverCmds, sliverCmds
}

// Before running any CLI entry command, require the Sliver client to connect to a teamserver.
func preRunClient(con *client.SliverClient) func(_ *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// Don't stream console asciicast/logs when using the completion subcommand.
		// We don't use cmd.Root().Find() for this, as it would always trigger the condition true.
		for _, compCmd := range cmd.Root().Commands() {
			if compCmd != nil && compCmd.Name() == "_carapace" && compCmd.CalledAs() != "" {
				con.IsCompleting = true
				break
			}
		}

		return con.Teamclient.Connect()
	}
}

// After running any CLI entry command, correctly disconnect from the server.
func postRunClient(con *client.SliverClient) func(_ *cobra.Command, _ []string) error {
	return func(_ *cobra.Command, _ []string) error {
		return con.Teamclient.Disconnect()
	}
}
