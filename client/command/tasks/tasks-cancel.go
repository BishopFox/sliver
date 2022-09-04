package tasks

import (
	"context"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// TasksCancelCmd - Cancel a beacon task before it's sent to the implant
func TasksCancelCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}

	idArg := ctx.Args.String("id")
	var task *clientpb.BeaconTask
	var err error
	if idArg == "" {
		beaconTasks, err := con.Rpc.GetBeaconTasks(context.Background(), &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		tasks := []*clientpb.BeaconTask{}
		for _, task := range beaconTasks.Tasks {
			if task.State == "pending" {
				tasks = append(tasks, task)
			}
		}
		if len(tasks) == 0 {
			con.PrintErrorf("No pending tasks for beacon\n")
			return
		}

		task, err = SelectBeaconTask(tasks)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Printf(console.UpN+console.Clearln, 1)
	} else {
		task, err = con.Rpc.GetBeaconTaskContent(context.Background(), &clientpb.BeaconTask{ID: idArg})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	if task != nil {
		task, err := con.Rpc.CancelBeaconTask(context.Background(), task)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Task %s canceled\n", task.ID)
	}
}
