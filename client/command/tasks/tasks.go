package tasks

import (
	"context"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// TasksCmd - Manage beacon tasks
func TasksCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}
	beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	PrintBeaconTasks(beaconTasks.Tasks, con)
}

// PrintBeaconTasks - Print beacon tasks
func PrintBeaconTasks(tasks []*clientpb.BeaconTask, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle())
	tw.AppendHeader(table.Row{
		"State",
		"Message Type",
		"Created At",
		"Sent At",
		"Completed At",
	})
	for _, task := range tasks {
		tw.AppendRow(table.Row{
			prettyState(task.State),
			task.Description,
			time.Unix(task.CreatedAt, 0).Format(time.RFC1123),
			time.Unix(task.SentAt, 0).Format(time.RFC1123),
			time.Unix(task.CompletedAt, 0).Format(time.RFC1123),
		})
	}
	con.Printf("%s\n", tw.Render())
}

func prettyState(state string) string {
	switch strings.ToLower(state) {
	case "pending":
		return console.Bold + state + console.Normal
	case "sent":
		return console.Bold + console.Orange + state + console.Normal
	case "completed":
		return console.Bold + console.Green + state + console.Normal
	default:
		return state
	}
}
