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

	// CLI dependencies
	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	// Teamserver/teamclient dependencies
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"

	// Sliver Client core, and generic/server-only commands
	"github.com/bishopfox/sliver/client/command"
	consoleCmd "github.com/bishopfox/sliver/client/command/console"
	client "github.com/bishopfox/sliver/client/console"
	assetsCmds "github.com/bishopfox/sliver/server/command/assets"
	builderCmds "github.com/bishopfox/sliver/server/command/builder"
	certsCmds "github.com/bishopfox/sliver/server/command/certs"
	"github.com/bishopfox/sliver/server/command/version"
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
	rootCmd, server := sliverServerCLI(teamserver, con)

	// Bind the closed-loop console:
	// The console shares the same setup/connection pre-runners as other commands,
	// but the command yielders we pass as arguments don't: this is because we only
	// need one connection for the entire lifetime of the console.
	rootCmd.AddCommand(consoleCmd.Command(con, server))

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
		fmt.Println(err)
		os.Exit(1)
	}
}

// sliverServerCLI returns the entire command tree of the Sliver Framework as yielder functions.
// The ready-to-execute command tree (root *cobra.Command) returned is correctly equipped with
// all prerunners needed to connect to remote Sliver teamservers.
// It will also register the appropriate teamclient management commands.
//
// Counterpart of sliver/client/cli.SliverCLI() (not identical: no implant command here).
func sliverServerCLI(team *server.Server, con *client.SliverClient) (root *cobra.Command, server console.Commands) {
	teamserverCmds := teamserverCmds(team, con)

	// Generate a single tree instance of server commands:
	// These are used as the primary, one-exec-only CLI of Sliver, and are equipped with
	// a pre-runner ensuring the server and its teamclient are set up and connected.
	server = command.ServerCommands(con, teamserverCmds)

	root = server()
	root.Use = "sliver-server" // Needed by completion scripts.

	// Pre/post runners and completions.
	command.BindRunners(root, true, preRunServer(team, con))
	command.BindRunners(root, false, func(_ *cobra.Command, _ []string) error {
		return con.Disconnect()
	})

	// Generate the root completion command.
	carapace.Gen(root)

	return root, server
}

// Yielder function for server-binary only functions.
func teamserverCmds(teamserver *server.Server, con *client.SliverClient) func(con *client.SliverClient) (cmds []*cobra.Command) {
	return func(con *client.SliverClient) (cmds []*cobra.Command) {
		// Teamserver management
		cmds = append(cmds, commands.Generate(teamserver, con.Teamclient))

		// Sliver-specific
		cmds = append(cmds, version.Commands(con)...)
		cmds = append(cmds, assetsCmds.Commands()...)
		cmds = append(cmds, certsCmds.Commands(con)...)

		// Commands requiring the server to be a remote teamclient.
		cmds = append(cmds, builderCmds.Commands(con, teamserver)...)

		return cmds
	}
}

// preRunServer is the server-binary-specific pre-run; it ensures that the server
// has everything it needs to perform any client-side command/task.
func preRunServer(teamserver *server.Server, con *client.SliverClient) func(_ *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Ensure the server has what it needs.
		assets.Setup(false, true)
		encoders.Setup() // WARN: I added this here after assets.Setup(), but used to be in init. Is it wrong to put it here ?
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()

		// TODO: Move this out of here.
		// serverConfig := configs.GetServerConfig()
		// c2.StartPersistentJobs(serverConfig)

		// Let our in-memory teamclient be served.
		err := teamserver.Serve(con.Teamclient)
		if err != nil {
			return err
		}

		return con.ConnectRun(cmd, args)
	}
}

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
//
// 		console.StartClient(con, nil, nil)
// 		return nil
// 	}
