package commands

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"time"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func versionCmd(cli *client.Client) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				cli.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		if err := cli.Connect(); err != nil {
			return err
		}

		// Server
		serverVer, err := cli.VersionServer()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Server error: %s\n", err)
		}

		serverVerInfo := fmt.Sprintf("Server v%d.%d.%d - %s - %s/%s\n",
			serverVer.Major, serverVer.Minor, serverVer.Patch, serverVer.Commit,
			serverVer.OS, serverVer.Arch)
		serverCompiledAt := time.Unix(serverVer.CompiledAt, 0)

		fmt.Fprint(cmd.OutOrStdout(), command.Info+serverVerInfo)
		fmt.Fprintf(cmd.OutOrStdout(), "    Compiled at %s\n", serverCompiledAt)
		fmt.Fprintln(cmd.OutOrStdout())

		// Client
		clientVer, err := cli.VersionClient()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Client error: %s\n", err)
			return nil
		}

		clientVerInfo := fmt.Sprintf("Client v%d.%d.%d - %s - %s/%s\n",
			clientVer.Major, clientVer.Minor, clientVer.Patch, clientVer.Commit,
			clientVer.OS, clientVer.Arch)
		clientCompiledAt := time.Unix(clientVer.CompiledAt, 0)

		fmt.Fprint(cmd.OutOrStdout(), command.Info+clientVerInfo)
		fmt.Fprintf(cmd.OutOrStdout(), "    Compiled at %s\n", clientCompiledAt)

		return nil
	}
}
