package exec

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
	"sort"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ExecuteChildrenCmd - List tracked background execute child processes on the implant.
func ExecuteChildrenCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Querying execute children ...", ctrl)
	execChildren, err := con.Rpc.ExecuteChildren(context.Background(), &sliverpb.ExecuteChildrenReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if execChildren.Response != nil && execChildren.Response.Async {
		con.AddBeaconCallback(execChildren.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, execChildren)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintExecuteChildren(execChildren, con)
		})
		con.PrintAsyncResponse(execChildren.Response)
		return
	}

	PrintExecuteChildren(execChildren, con)
}

// PrintExecuteChildren - Render execute children output.
func PrintExecuteChildren(execChildren *sliverpb.ExecuteChildren, con *console.SliverClient) {
	if execChildren == nil || len(execChildren.Children) == 0 {
		con.PrintInfof("No tracked background execute children\n")
		return
	}

	children := append([]*sliverpb.ExecuteChild(nil), execChildren.Children...)
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartTime < children[j].StartTime
	})

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{"pid", "state", "exit", "started", "command", "stdout", "stderr"})

	for _, child := range children {
		state := "running"
		exitCode := "-"
		if child.Exited {
			state = "exited"
			exitCode = fmt.Sprintf("%d", child.ExitCode)
		}

		started := "-"
		if child.StartTime != 0 {
			started = time.Unix(child.StartTime, 0).Local().Format("2006-01-02 15:04:05")
		}

		command := strings.TrimSpace(fmt.Sprintf("%s %s", child.Path, strings.Join(child.Args, " ")))
		tw.AppendRow(table.Row{child.Pid, state, exitCode, started, command, child.Stdout, child.Stderr})
	}

	con.PrintInfof("%s\n", tw.Render())
}
