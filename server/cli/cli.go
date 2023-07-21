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
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/rpc"
	teamserver "github.com/reeflective/team/server"
	teamGrpc "github.com/reeflective/team/transports/grpc/server"
	"google.golang.org/grpc"

	"github.com/reeflective/console"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command"
	client "github.com/bishopfox/sliver/client/console"
	assetsCmds "github.com/bishopfox/sliver/server/command/assets"
	builderCmds "github.com/bishopfox/sliver/server/command/builder"
	certsCmds "github.com/bishopfox/sliver/server/command/certs"

	consoleCmd "github.com/bishopfox/sliver/client/command/console"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/cryptography"
)

// Execute the sliver server binary.
func Execute() {
	// Create a self-serving teamserver:
	// This teamserver creates an in-memory -gRPC-transported- teamclient
	// (which is not yet connected). The server is also able to serve remote
	// clients, although no persistent/network listeners are started by default.
	//
	// This teamclient is used to create a Sliver Client, which functioning
	// is agnostic to its execution mode (one-exec CLI, or closed-loop console).
	// The client has no commands available yet.
	teamserver, con := newSliverServer()

	// Prepare the entire Sliver Command-line Interface as yielder functions:
	// - Server commands are all commands which don't need an active Sliver implant.
	//   These commands include a server-binary-specific set of "teamserver" commands.
	// - Sliver commands are all commands requiring an active target.
	serverCmds, sliverCmds := getSliverCommands(teamserver, con)

	// Generate a single tree instance of server commands:
	// These are used as the primary, one-exec-only CLI of Sliver, and are equiped with
	// a pre-runner ensuring the server and its teamclient are set up and connected.
	rootCmd := serverCmds()
	rootCmd.Use = "sliver-server" // Needed by completion scripts.
	rootCmd.PersistentPreRunE = preRunServer(teamserver, con)

	// Bind additional commands peculiar to the one-exec CLI.
	// NOTE: Down the road these commands should probably stripped of their
	// os.Exit() calls and adapted so that they can be used in the console too.
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(assetsCmds.Commands()...)
	rootCmd.AddCommand(builderCmds.Commands()...)
	rootCmd.AddCommand(certsCmds.Commands()...)

	// Bind the closed-loop console:
	// The console shares the same setup/connection pre-runners as other commands,
	// but the command yielders we pass as arguments don't: this is because we only
	// need one connection for the entire lifetime of the console.
	rootCmd.AddCommand(consoleCmd.Command(con, serverCmds, sliverCmds))

	// Completion setup:
	// Following the same logic as the console command, we only generate and setup
	// completions in our root command tree instance. This setup is not needed in
	// the command yielder functions themselves, because the closed-loop console
	// takes care of its own completion/API interfacing.
	comps := carapace.Gen(rootCmd)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		rootCmd.PersistentPreRunE(cmd, args)
	})
	comps.PostRun(func(cmd *cobra.Command, args []string) {
		con.Teamclient.Disconnect()
	})

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
	//   set above in the init() code has been given a single hook to register its RPC backend.
	//   The call blocks like your old daemon command, and works _just the same_.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// newSliverServer creates a new application teamserver.
// The gRPC listener is hooked with an in-memory teamclient, and the latter
// is passed to our client console package to create a new SliverClient.
func newSliverServer() (*teamserver.Server, *client.SliverClient) {
	//
	// 1) Teamserver ---------
	//

	// NOTE: Teamserver gRPC stack.
	// The teamserver stack is for now contained in a package of the third-party
	// module github.com/reeflective/team:
	// 1) The listener is pre-set with all gRPC transport,auth and middleware logging.
	// 2) This listener could be partially/fully reimplemented within the Sliver repo.
	gTeamserver := teamGrpc.NewListener()

	// NOTE: This might not be needed if 2) above is chosen.
	// The listener obviously works with gRPC servers, so we need to pass
	// a hook for service binding before starting those gRPC HTTP/2 listeners.
	gTeamserver.PostServe(func(grpcServer *grpc.Server) error {
		if grpcServer == nil {
			return errors.New("No gRPC server to use for service")
		}

		rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())

		return nil
	})

	// Here is an import step, where we are given a change to setup
	// the reeflective/teamserver with everything we want: our own
	// database, the application daemon default port, loggers or files,
	// directories, and much more.
	var serverOpts []teamserver.Options
	serverOpts = append(serverOpts,
		teamserver.WithDefaultPort(31337),
		teamserver.WithListener(gTeamserver),
	)

	// Create the application teamserver.
	// Any error is critical, and means we can't work correctly.
	teamserver, err := teamserver.New("sliver", serverOpts...)
	if err != nil {
		log.Fatal(err)
	}

	//
	// 2) Teamclient ---------
	//

	// The gRPC teamserver backend is hooked to produce a single
	// in-memory teamclient RPC/dialer backend. Not encrypted.
	gTeamclient := teamGrpc.NewClientFrom(gTeamserver)

	// Pass the gRPC teamclient backend to our console package,
	// with registers a hook to bind its RPC client and start
	// monitoring events/logs/etc when asked to connect.
	//
	// The options returned are used to dictate to the server
	// how it should configure and run its self-teamclient.
	sliver, opts := client.NewSliverClient(gTeamclient)

	// And let the server create its own teamclient,
	// pass to the Sliver client for normal usage.
	sliver.Teamclient = teamserver.Self(opts...)

	return teamserver, sliver
}

// getSliverCommands wraps the `teamserver` specific commands in a command yielder function, passes those
// server-binary only commands to the main Sliver command yielders, and returns the full, execution-mode
// agnostic Command-Line-Interface for the Sliver Framework.
func getSliverCommands(teamserver *server.Server, con *client.SliverClient) (server, sliver console.Commands) {
	teamserverCmds := func() *cobra.Command {
		return commands.Generate(teamserver, con.Teamclient)
	}

	serverCmds := command.ServerCommands(con, teamserverCmds)
	sliverCmds := command.SliverCommands(con)

	return serverCmds, sliverCmds
}

// preRunServer is the server-binary-specific pre-run; it ensures that the server
// has everything it needs to perform any client-side command/task.
func preRunServer(teamserver *server.Server, con *client.SliverClient) func(_ *cobra.Command, _ []string) error {
	return func(_ *cobra.Command, _ []string) error {
		// Ensure the server has what it needs.
		assets.Setup(false, true)
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()

		// TODO: Move this out of here.
		serverConfig := configs.GetServerConfig()
		c2.StartPersistentJobs(serverConfig)

		// Let our in-memory teamclient be served.
		return teamserver.Serve(con.Teamclient)
	}
}

// 	preRun := func(cmd *cobra.Command, _ []string) error {
// 		// Ensure the server has what it needs
// 		assets.Setup(false, true)
// 		certs.SetupCAs()
// 		certs.SetupWGKeys()
// 		cryptography.AgeServerKeyPair()
// 		cryptography.MinisignServerPrivateKey()
//
// 		// Let our runtime teamclient be served.
// 		if err := teamserver.Serve(teamclient); err != nil {
// 			return err
// 		}
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
