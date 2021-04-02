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

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/assets"
	client "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/version"
)

var (
	sliverServerVersion = fmt.Sprintf("v%s", version.FullVersion())
)

func init() {

	// Import
	rootCmd.AddCommand(cmdImport)

	// Version
	rootCmd.AddCommand(cmdVersion)
}

var rootCmd = &cobra.Command{
	Use:   "sliver-client",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Load all necessary configurations (server connection details, TLS security,
		// console configuration, etc.).
		// The configuration is then accessible to all client packages.
		assets.LoadServerConfig()

		// Get a gRPC client connection (Mutual TLS) to the server.
		conn, err := transport.ConnectTLS()
		if err != nil {
			os.Exit(1)
		}

		// Register gRPC services, monitor events, setup the
		// console, logging, prompt/command/completion, etc.
		err = client.Console.Init(conn)
		if err != nil {
			log.Fatal(err)
		}

		// Run the console. Any error is handled internally
		client.Console.Run()
	},
}

// Execute - Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
