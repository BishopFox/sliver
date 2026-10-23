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
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc"
)

// ConsoleConfigFlagName is the flag used to select a client config file.
const ConsoleConfigFlagName = "config"

// consoleCmd generates the console with required pre/post runners.
func consoleCmd(con *console.SliverClient) *cobra.Command {
	consoleCmd := &cobra.Command{
		Use:   "console",
		Short: "Start the sliver client console",
	}

	consoleCmd.Flags().String(RCFlagName, "", "path to rc script file")
	consoleCmd.RunE, consoleCmd.PersistentPostRunE = consoleRunnerCmd(con, true)
	return consoleCmd
}

func consoleRunnerCmd(con *console.SliverClient, run bool) (pre, post func(cmd *cobra.Command, args []string) error) {
	pre = func(cmd *cobra.Command, _ []string) error {
		if err := applyMultiplayerConnectMode(cmd); err != nil {
			return err
		}

		rcScript, err := ReadRCScript(cmd)
		if err != nil {
			return err
		}

		configPath := ""
		if cmd.Flags().Lookup(ConsoleConfigFlagName) != nil {
			configPath, err = cmd.Flags().GetString(ConsoleConfigFlagName)
			if err != nil {
				return err
			}
		}
		stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
		configKey, config, err := selectConfig(configPath, stdinTTY)
		if err != nil {
			return err
		}

		target := fmt.Sprintf("%s:%d", config.LHost, config.LPort)
		var rpc rpcpb.SliverRPCClient
		var ln *grpc.ClientConn

		// Don't clobber output when simply running an implant command from system shell.
		if run && stdinTTY {
			rpc, ln, err = connectWithSpinner(os.Stdout, target, func(statusFn transport.ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
				return transport.MTLSConnectWithStatus(config, statusFn)
			})
		} else {
			rpc, ln, err = transport.MTLSConnect(config)
		}
		if err != nil {
			return fmt.Errorf("connection to server failed: %w", err)
		}
		if err := console.StartClient(con, rpc, ln, &console.ConnectionDetails{ConfigKey: configKey, Config: config}, command.ServerCommands(con, nil), command.SliverCommands(con), run, rcScript); err != nil {
			_ = con.CloseConnection()
			return err
		}
		return nil
	}

	// Close the RPC connection once exiting
	post = func(_ *cobra.Command, _ []string) error {
		return con.CloseConnection()
	}

	return pre, post
}
