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
	"log"
	"os"
	"path"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/version"
	"github.com/reeflective/team/client/commands"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

const (
	logFileName = "sliver-client.log"
)

var sliverServerVersion = fmt.Sprintf("v%s", version.FullVersion())

// Initialize logging.
func initLogging(appDir string) *os.File {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		panic(fmt.Sprintf("[!] Error opening file: %s", err))
	}
	log.SetOutput(logFile)
	return logFile
}

func init() {
	// Create the console client, without any RPC or commands bound to it yet.
	// This created before anything so that multiple commands can make use of
	// the same underlying command/run infrastructure.
	con := console.NewConsole(false)

	// Teamclient API and commands for remote CLI.
	teamclient := newSliverTeam(con)
	teamclientCmds := commands.Generate(teamclient)

	rootCmd.AddCommand(teamclientCmds)

	// Version
	rootCmd.AddCommand(cmdVersion)

	preRun := func(_ *cobra.Command, _ []string) error {
		return teamclient.Connect()
	}

	rootCmd.PersistentPreRunE = preRun

	// Client console.
	// All commands and RPC connection are generated WITHIN the command RunE():
	// that means there should be no redundant command tree/RPC connections with
	// other command trees below, such as the implant one.
	rootCmd.AddCommand(consoleCmd(con))

	// Implant.
	// The implant command allows users to run commands on slivers from their
	// system shell. It makes use of pre-runners for connecting to the server
	// and binding sliver commands. These same pre-runners are also used for
	// command completion/filtering purposes.
	rootCmd.AddCommand(implantCmd(con))

	// Completions
	comps := carapace.Gen(rootCmd)
	comps.PreRun(func(cmd *cobra.Command, args []string) {
		preRun(cmd, args)
	})
}

var rootCmd = &cobra.Command{
	Use:              "sliver-client",
	Short:            "Client-only Sliver C2 management",
	Long:             ``,
	TraverseChildren: true,
}

// Execute - Execute root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
