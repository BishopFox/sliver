package exec

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

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
// ExecuteChildrenCmd - List 跟踪 implant. 上的后台执行子进程
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
// PrintExecuteChildren - Render 执行子进程 output.
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

	con.Printf("%s\n", tw.Render())
}
