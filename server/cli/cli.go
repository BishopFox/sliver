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

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command"
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

const (

	// Unpack flags
	forceFlagStr = "force"

	// Operator flags
	nameFlagStr  = "name"
	lhostFlagStr = "lhost"
	lportFlagStr = "lport"
	saveFlagStr  = "save"

	// Cert flags
	caTypeFlagStr = "type"
	loadFlagStr   = "load"
)

// Execute - Execute root command
func Execute() {
	// Console interface, started closed-loop or not.
	// Teamserver/client API and commands for remote/local CLI.
	teamserver, con := newSliverTeam()

	// Pre-runners to self-connect
	preRun := func(cmd *cobra.Command, _ []string) error {
		// Ensure the server has what it needs
		assets.Setup(false, true)
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()

		// TODO: Move this out of here.
		serverConfig := configs.GetServerConfig()
		c2.StartPersistentJobs(serverConfig)

		// Let our runtime teamclient be served.
		if err := teamserver.Serve(con.Teamclient); err != nil {
			return err
		}

		return nil
	}

	serverCmds := command.ServerCommands(con, teamserver)()
	serverCmds.Use = "sliver-server"

	serverCmds.PersistentPreRunE = preRun

	serverCmds.AddCommand(versionCmd)
	serverCmds.AddCommand(assetsCmds.Commands()...)
	serverCmds.AddCommand(builderCmds.Commands()...)
	serverCmds.AddCommand(certsCmds.Commands()...)

	// Console
	consoleServerCmds := command.ServerCommands(con, teamserver)
	consoleSliverCmds := command.SliverCommands(con)
	serverCmds.AddCommand(consoleCmd.Command(con, consoleServerCmds, consoleSliverCmds))

	// Completions
	comps := carapace.Gen(serverCmds)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		preRun(cmd, args)
	})
	comps.PostRun(func(cmd *cobra.Command, args []string) {
		con.Teamclient.Disconnect()
	})

	if err := serverCmds.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// func init() {
// 	// Console interface, started closed-loop or not.
// 	con := console.NewConsole(false)
//
// 	// Teamserver/client API and commands for remote/local CLI.
// 	teamserver, teamclient := newSliverTeam(con)
// 	// teamserverCmds := commands.Generate(teamserver, teamclient)
//
// 	con.Teamclient = teamclient
//
// 	// Bind commands to the app
// 	server := con.App.Menu(consts.ServerMenu)
// 	server.SetCommands(command.ServerCommands(con, teamserver))
//
// 	serverCmds := command.ServerCommands(con, teamserver)()
//
// 	// server.Reset()
// 	// sliver := con.App.Menu(consts.ImplantMenu)
// 	// sliver.SetCommands(command.SliverCommands(con))
// 	// rootCmd = server.Command
// 	// rootCmd.Use = "sliver-server"
//
// 	// rootCmd.AddCommand(teamserverCmds)
//
// 	// Pre-runners to self-connect
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
// 	// return nil
//
// 	serverCmds.PersistentPreRunE = preRun
//
// 	serverCmds.AddCommand(consoleCmd(con))
//
// 	// Unpack
// 	unpackCmd.Flags().BoolP(forceFlagStr, "f", false, "Force unpack and overwrite")
// 	serverCmds.AddCommand(unpackCmd)
//
// 	// Certs
// 	cmdExportCA.Flags().StringP(saveFlagStr, "s", "", "save CA to file ...")
// 	cmdExportCA.Flags().StringP(caTypeFlagStr, "t", "", fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
// 	serverCmds.AddCommand(cmdExportCA)
//
// 	cmdImportCA.Flags().StringP(loadFlagStr, "l", "", "load CA from file ...")
// 	cmdImportCA.Flags().StringP(caTypeFlagStr, "t", "", fmt.Sprintf("ca type (%s)", strings.Join(validCATypes(), ", ")))
// 	serverCmds.AddCommand(cmdImportCA)
//
// 	// Builder
// 	// rootCmd.AddCommand(initBuilderCmd())
//
// 	// Version
// 	// rootCmd.AddCommand(versionCmd)
//
// 	// Completions
// 	comps := carapace.Gen(serverCmds)
// 	comps.PreRun(func(cmd *cobra.Command, args []string) {
// 		preRun(cmd, args)
// 	})
// }

// var rootCmd = &cobra.Command{
// 	Use:   "sliver-server",
// 	Short: "",
// 	Long:  ``,
// Run: func(cmd *cobra.Command, args []string) {
// 	// Root command starts the server normally
//
// 	appDir := assets.GetRootAppDir()
// 	logFile := initConsoleLogging(appDir)
// 	defer logFile.Close()
//
// 	defer func() {
// 		if r := recover(); r != nil {
// 			log.Printf("panic:\n%s", debug.Stack())
// 			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
// 			os.Exit(99)
// 		}
// 	}()
//
// 	assets.Setup(false, true)
// 	certs.SetupCAs()
// 	certs.SetupWGKeys()
// 	cryptography.AgeServerKeyPair()
// 	cryptography.MinisignServerPrivateKey()
//
// 	serverConfig := configs.GetServerConfig()
// 	c2.StartPersistentJobs(serverConfig)
// 	console.StartPersistentJobs(serverConfig)
// 	if serverConfig.DaemonMode {
// 		daemon.Start(daemon.BlankHost, daemon.BlankPort)
// 	} else {
// 		os.Args = os.Args[:1] // Hide cli from grumble console
// 		console.Start()
// 	}
// },
// }
