package tasks

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
		return console.StyleBold.Render(state)
	case "sent":
		return console.StyleBoldOrange.Render(state)
	case "completed":
		return console.StyleBoldGreen.Render(state)
	case "canceled":
		return console.StyleBoldGray.Render(state)
	default:
		return state
	}
}
