package filesystem

/*
	Copyright (C) 2023 b0yd
	Copyright (C) 2023 岁

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

// ChownCmd - Change the owner of a file on the remote file system.
// ChownCmd - Change 远程文件 system. 上的文件所有者
func ChownCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := args[0]

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	uid := args[1]

	if uid == "" {
		con.PrintErrorf("Missing parameter: user id\n")
		return
	}

	gid := args[2]

	if gid == "" {
		con.PrintErrorf("Missing parameter: group id\n")
		return
	}

	recursive, _ := cmd.Flags().GetBool("recursive")

	chown, err := con.Rpc.Chown(context.Background(), &sliverpb.ChownReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      filePath,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if chown.Response != nil && chown.Response.Async {
		con.AddBeaconCallback(chown.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, chown)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintChown(chown, con)
		})
		con.PrintAsyncResponse(chown.Response)
	} else {
		PrintChown(chown, con)
	}
}

// PrintChown - Print the chown response.
// PrintChown - Print chown response.
func PrintChown(chown *sliverpb.Chown, con *console.SliverClient) {
	if chown.Response != nil && chown.Response.Err != "" {
		con.PrintErrorf("%s\n", chown.Response.Err)
		return
	}
	con.PrintInfof("%s\n", chown.Path)
}
