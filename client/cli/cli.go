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

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

const (
	logFileName = "sliver-client.log"
)

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
	appDir := assets.GetRootAppDir()
	logFile := initLogging(appDir)
	defer logFile.Close()

	rootCmd.TraverseChildren = true

	// Create the console client, without any RPC or commands bound to it yet.
	// This created before anything so that multiple commands can make use of
	// the same underlying command/run infrastructure.
	con := console.NewConsole(false)

	// Import
	rootCmd.AddCommand(importCmd())

	// Version
	rootCmd.AddCommand(cmdVersion)

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

	// No subcommand invoked means starting the console.
	rootCmd.RunE, rootCmd.PostRunE = consoleRunnerCmd(con, true)

	// Completions
	carapace.Gen(rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   "sliver-client",
	Short: "",
	Long:  ``,
}

// Execute - Execute root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}
}
