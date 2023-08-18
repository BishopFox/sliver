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
	"encoding/json"
	"fmt"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func importCmd(cli *client.Client) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				cli.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		if 0 < len(args) {
			for _, arg := range args {
				conf, err := cli.ReadConfig(arg)
				if jsonErr, ok := err.(*json.SyntaxError); ok {
					fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"%s\n", jsonErr.Error())
				} else if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"%s\n", err.Error())
					continue
				}

				if err = cli.SaveConfig(conf); err == nil {
					fmt.Fprintln(cmd.OutOrStdout(), command.Info+"Saved new client config in ", cli.ConfigsDir())
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"%s\n", err.Error())
				}
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Missing config file path, see --help")
		}
	}
}
