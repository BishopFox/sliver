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

	// CLI dependencies
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	// Teamserver/teamclient dependencies
	"github.com/reeflective/team/server"

	// Sliver Client core, and generic/server-only commands
	clientCommand "github.com/bishopfox/sliver/client/command"
	consoleCmd "github.com/bishopfox/sliver/client/command/console"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/command"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/encoders"
	"github.com/bishopfox/sliver/server/transport"

	// Server-only imports
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
)

// Execute the sliver server binary.
func Execute() {
	// Create a new Sliver Teamserver: the latter is able to serve all remote
	// clients for its users, over any of the available transport stacks (MTLS/TS.
	// Persistent teamserver client listeners are not started by default.
	teamserver, opts, err := transport.NewTeamserver()
	if err != nil {
		panic(err)
	}

	// Use the specific set of dialing options passed by the teamserver,
	// and use them to create an in-memory sliver teamclient, identical
	// in behavior to remote ones.
	// The client has no commands available yet.
	con, err := client.NewSliverClient(opts...)
	if err != nil {
		panic(err)
	}

	// Generate our complete Sliver Framework command-line interface.
	rootCmd := sliverServerCLI(teamserver, con)

	// Run the target Sliver command:
	// Three different examples here, to illustrate.
	//
	// - `sliver generate --os linux` starts the server, ensuring assets are unpacked, etc.
	//    Once ready, the generate command is executed (from the client, passed to the server
	//    via the in-memory RPC, and executed, compiled, then returned to the client).
	//    When the binary exits, an implant is compiled and available client-side (locally here).
	//
	// - `sliver console` starts the console, and everything works like it ever did.
	//    On top of that, you can access and use the entire `teamserver` control commands to
	//    start/close/delete client listeners, create/delete users, manage CAs, show status, etc.
	//
	// - `sliver teamserver serve` is a teamserver-tree specific command, and the teamserver
	//   set above in the code has been given a single hook to register its RPC backend.
	//   The call blocks like your old daemon command, and works _just the same_.
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// sliverServerCLI returns the entire command tree of the Sliver Framework as yielder functions.
// The ready-to-execute command tree (root *cobra.Command) returned is correctly equipped with
// all prerunners needed to connect to remote Sliver teamservers.
// It will also register the appropriate teamclient management commands.
//
// Counterpart of sliver/client/cli.SliverCLI() (not identical: no implant command here).
func sliverServerCLI(team *server.Server, con *client.SliverClient) (root *cobra.Command) {
	teamserverCmds := command.TeamserverCommands(team, con)

	// Generate a single tree instance of server commands:
	// These are used as the primary, one-exec-only CLI of Sliver, and are equipped with
	// a pre-runner ensuring the server and its teamclient are set up and connected.
	server := clientCommand.ServerCommands(con, teamserverCmds)

	root = server()
	root.Use = "sliver-server" // Needed by completion scripts.
	con.IsServer = true

	// Bind the closed-loop console:
	// The console shares the same setup/connection pre-runners as other commands,
	// but the command yielders we pass as arguments don't: this is because we only
	// need one connection for the entire lifetime of the console.
	root.AddCommand(consoleCmd.Command(con, server))

	// The server is also a client of itself, so add our sliver-server
	// binary specific pre-run hooks: assets, encoders, toolchains, etc.
	con.AddPreRuns(preRunServerS(team, con))

	// Pre/post runners and completions.
	clientCommand.BindPreRun(root, con.PreRunConnect)
	clientCommand.BindPostRun(root, con.PostRunDisconnect)

	// Generate the root completion command.
	carapace.Gen(root)

	return root
}

func preRunServerS(teamserver *server.Server, con *client.SliverClient) clientCommand.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		// All commands of the teamserver binary
		// must always have at least these working.
		assets.Setup(false, true)
		encoders.Setup()
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()

		// But we don't start Sliver-specific C2 listeners unless
		// we are being ran in daemon mode, or in the console.
		// We don't always have access to a command, such when
		if cmd != nil {
			if (cmd.Name() == "daemon" && cmd.Parent().Name() == "teamserver") ||
				cmd.Name() == "console" {
				serverConfig := configs.GetServerConfig()
				err := c2.StartPersistentJobs(serverConfig)
				if err != nil {
					con.PrintWarnf("Persistent jobs restart error: %s", err)
				}
			}

		}
		// Let our in-memory teamclient be served.
		return teamserver.Serve(con.Teamclient)
	}
}

// preRunServer is the server-binary-specific pre-run; it ensures that the server
// has everything it needs to perform any client-side command/task.
// func preRunServer(teamserver *server.Server, con *client.SliverClient) func() error {
// 	return func() error {
// 		// Ensure the server has what it needs.
// 		assets.Setup(false, true)
// 		encoders.Setup()
// 		certs.SetupCAs()
// 		certs.SetupWGKeys()
// 		cryptography.AgeServerKeyPair()
// 		cryptography.MinisignServerPrivateKey()
//
// 		// TODO: Move this out of here.
// 		serverConfig := configs.GetServerConfig()
// 		c2.StartPersistentJobs(serverConfig)
//
// 		// Let our in-memory teamclient be served.
// 		return teamserver.Serve(con.Teamclient)
// 	}
// }

// 	preRun := func(cmd *cobra.Command, _ []string) error {
//
// 		// Start persistent implant/c2 jobs (not teamservers)
// 		// serverConfig := configs.GetServerConfig()
// 		// c2.StartPersistentJobs(serverConfig)
//
// 		// Only start the teamservers when the console being
// 		// ran is the console itself: the daemon command will
// 		// start them on its own, since the config is different.
// 		// if cmd.Name() == "console" {
// 		// 	teamserver.ListenerStartPersistents() // Automatically logged errors.
// 		// 	// 	console.StartPersistentJobs(serverConfig) // Old alternative
// 		// }
// 		return nil
// 	}
