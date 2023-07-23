package tasks

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"sort"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// TasksCmd - Manage beacon tasks.
func TasksCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}
	beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintBeaconTasks(beaconTasks.Tasks, cmd, con)
}

// PrintBeaconTasks - Print beacon tasks.
func PrintBeaconTasks(tasks []*clientpb.BeaconTask, cmd *cobra.Command, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"State",
		"Message Type",
		"Created",
		"Sent",
		"Completed",
	})

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt > tasks[j].CreatedAt
	})

	filterFlag, _ := cmd.Flags().GetString("filter")
	filter := strings.ToLower(filterFlag)
	for _, task := range tasks {
		if filter != "" && !strings.HasPrefix(strings.ToLower(task.Description), filter) {
			continue
		}
		sentAt := time.Unix(task.SentAt, 0).Format(time.RFC1123)
		if time.Unix(task.SentAt, 0).IsZero() {
			sentAt = ""
		}
		completedAt := time.Unix(task.CompletedAt, 0).Format(time.RFC1123)
		if time.Unix(task.CompletedAt, 0).IsZero() {
			completedAt = ""
		}
		tw.AppendRow(table.Row{
			strings.Split(task.ID, "-")[0],
			prettyState(task.State),
			strings.TrimSuffix(task.Description, "Req"),
			time.Unix(task.CreatedAt, 0).Format(time.RFC1123),
			sentAt,
			completedAt,
		})
	}
	overflow, _ := cmd.Flags().GetBool("overflow")
	skipPages, _ := cmd.Flags().GetInt("skip-pages")
	settings.PaginateTable(tw, skipPages, overflow, true, con)
}

func prettyState(state string) string {
	switch strings.ToLower(state) {
	case "pending":
		return console.Bold + state + console.Normal
	case "sent":
		return console.Bold + console.Orange + state + console.Normal
	case "completed":
		return console.Bold + console.Green + state + console.Normal
	case "canceled":
		return console.Bold + console.Gray + state + console.Normal
	default:
		return state
	}
}
