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

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/version"
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
		serverVer, err := cli.ServerVersion()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Server error: %s\n", err)
		}

		dirty := ""
		if serverVer.Dirty {
			dirty = fmt.Sprintf(" - %sDirty%s", command.Bold, command.Normal)
		}

		serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)
		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)

		// Client
		cdirty := ""
		if version.GitDirty() {
			cdirty = fmt.Sprintf(" - %sDirty%s", command.Bold, command.Normal)
		}

		cliVer := version.Semantic()
		cliSemVer := fmt.Sprintf("%d.%d.%d", cliVer[0], cliVer[1], cliVer[2])
		fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Client v%s - %s%s\n", cliSemVer, version.GitCommit(), cdirty)

		return nil
	}
}
