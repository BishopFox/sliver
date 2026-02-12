package privilege

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

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RunAsCmd - Run a command as another user on the remote system
// RunAsCmd - Run 作为远程系统上另一个用户的命令
func RunAsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	domain, _ := cmd.Flags().GetString("domain")
	showWindow, _ := cmd.Flags().GetBool("show-window")
	process, _ := cmd.Flags().GetString("process")
	arguments, _ := cmd.Flags().GetString("args")
	netonly, _ := cmd.Flags().GetBool("net-only")

	if username == "" {
		con.PrintErrorf("Please specify a username\n")
		return
	}

	if process == "" {
		con.PrintErrorf("Please specify a process path\n")
		return
	}

	runAs, err := con.Rpc.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     con.ActiveTarget.Request(cmd),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
		Domain:      domain,
		Password:    password,
		HideWindow:  !showWindow,
		NetOnly:     netonly,
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	name := getName(session, beacon)
	if runAs.Response != nil && runAs.Response.Async {
		con.AddBeaconCallback(runAs.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, runAs)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRunAs(runAs, process, arguments, name, con)
		})
		con.PrintAsyncResponse(runAs.Response)
	} else {
		PrintRunAs(runAs, process, arguments, name, con)
	}
}

// PrintRunAs - Print the result of run as
// PrintRunAs - Print 运行结果
func PrintRunAs(runAs *sliverpb.RunAs, process string, args string, name string, con *console.SliverClient) {
	if runAs.Response != nil && runAs.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", runAs.Response.GetErr())
		return
	}
	con.PrintInfof("Successfully ran %s %s on %s\n", process, args, name)
}

func getName(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Name
	}
	if beacon != nil {
		return beacon.Name
	}
	panic("no session or beacon")
}
