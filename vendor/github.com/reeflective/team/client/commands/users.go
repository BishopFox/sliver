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

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func usersCmd(cli *client.Client) func(cmd *cobra.Command, args []string) error {
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
		users, err := cli.Users()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), command.Warn+"Server error: %s\n", err)
		}

		tbl := &table.Table{}
		tbl.SetStyle(command.TableStyle)

		tbl.AppendHeader(table.Row{
			"Name",
			"Status",
			"Last seen",
		})

		for _, user := range users {
			lastSeen := user.LastSeen.Format(time.RFC1123)

			if !user.LastSeen.IsZero() {
				lastSeen = time.Since(user.LastSeen).Round(1 * time.Second).String()
			}

			var status string
			if user.Online {
				status = command.Bold + command.Green + "Online" + command.Normal
			} else {
				status = command.Bold + command.Red + "Offline" + command.Normal
			}

			tbl.AppendRow(table.Row{
				user.Name,
				status,
				lastSeen,
			})
		}

		if len(users) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), tbl.Render())
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), command.Info+"The %s teamserver has no users\n", cli.Name())
		}

		return nil
	}
}
