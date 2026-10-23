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
	"path"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

const (
	logFileName = "sliver-client.log"
)

var clientLogFile *os.File

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
	clientLogFile = initLogging(appDir)

	rootCmd.TraverseChildren = true
	rootCmd.Flags().String(RCFlagName, "", "path to rc script file")
	rootCmd.PersistentFlags().String(ConsoleConfigFlagName, "", "path to client config file")
	rootCmd.PersistentFlags().Bool(enableWGFlag, false, "force multiplayer connections through the operator config's WireGuard wrapper")
	rootCmd.PersistentFlags().Bool(disableWGFlag, false, "force multiplayer connections to use direct mTLS")
	rootCmd.MarkFlagsMutuallyExclusive(enableWGFlag, disableWGFlag)

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

	// MCP stdio server.
	rootCmd.AddCommand(mcpCmd(con))

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

type commandExitError struct {
	code int
	err  error
}

func (e *commandExitError) Error() string {
	return e.err.Error()
}

func (e *commandExitError) Unwrap() error {
	return e.err
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := errors.AsType[*commandExitError](err); ok {
		return exitErr.code
	}
	return 1
}

// Execute - Execute root command.
func Execute() {
	defer func() { _ = clientLogFile.Close() }()
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(commandExitCode(err))
	}
}
