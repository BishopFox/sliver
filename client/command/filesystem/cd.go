package filesystem

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// CdCmd - Change directory on the remote system.
// CdCmd - 远程 system. 上的 Change 目录
func CdCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := "."
	if len(args) == 1 {
		filePath = args[0]
	}

	pwd, err := con.Rpc.Cd(context.Background(), &sliverpb.CdReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if pwd.Response != nil && pwd.Response.Async {
		con.AddBeaconCallback(pwd.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, pwd)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintPwd(pwd, con)
		})
		con.PrintAsyncResponse(pwd.Response)
	} else {
		PrintPwd(pwd, con)
	}
}
