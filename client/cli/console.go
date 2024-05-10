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
	"context"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// consoleCmd generates the console with required pre/post runners.
func consoleCmd(con *console.SliverClient) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
	}

	consoleCmd.RunE, consoleCmd.PersistentPostRunE = consoleRunnerCmd(con, true)
	return consoleCmd
}

func consoleRunnerCmd(con *console.SliverClient, run bool) (pre, post func(cmd *cobra.Command, args []string) error) {
	var ln *grpc.ClientConn

	pre = func(_ *cobra.Command, _ []string) error {

		configs := assets.GetConfigs()
		if len(configs) == 0 {
			fmt.Printf("No config files found at %s (see --help)\n", assets.GetConfigDir())
			return nil
		}
		config := selectConfig()
		if config == nil {
			return nil
		}

		// Don't clobber output when simply running an implant command from system shell.
		if run {
			fmt.Printf("Connecting to %s:%d ...\n", config.LHost, config.LPort)
		}

		var rpc rpcpb.SliverRPCClient
		var err error

		rpc, ln, err = transport.MTLSConnect(config)
		if err != nil {
			fmt.Printf("Connection to server failed %s", err)
			return nil
		}

		// Wait for any connection state changes and exit if the connection is lost.
		go handleConnectionLost(ln)

		return console.StartClient(con, rpc, command.ServerCommands(con, nil), command.SliverCommands(con), run)
	}

	// Close the RPC connection once exiting
	post = func(_ *cobra.Command, _ []string) error {
		if ln != nil {
			return ln.Close()
		}

		return nil
	}

	return pre, post
}

func handleConnectionLost(ln *grpc.ClientConn) {
	currentState := ln.GetState()
	// currentState should be "Ready" when the connection is established.
	if ln.WaitForStateChange(context.Background(), currentState) {
		newState := ln.GetState()
		// newState will be "Idle" if the connection is lost.
		if newState == connectivity.Idle {
			fmt.Println("\nLost connection to server. Exiting now.")
			os.Exit(1)
		}
	}
}
