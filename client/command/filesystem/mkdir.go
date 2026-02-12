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

// MkdirCmd - Make a remote directory.
// MkdirCmd - Make 远程 directory.
func MkdirCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := args[0]
	// filePath := ctx.Args.String("path")
	// filePath := ctx.Args.String(__PH0__)

	if filePath == "" {
		con.PrintErrorf("Missing parameter: directory name\n")
		return
	}

	mkdir, err := con.Rpc.Mkdir(context.Background(), &sliverpb.MkdirReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if mkdir.Response != nil && mkdir.Response.Async {
		con.AddBeaconCallback(mkdir.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, mkdir)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintMkdir(mkdir, con)
		})
		con.PrintAsyncResponse(mkdir.Response)
	} else {
		PrintMkdir(mkdir, con)
	}
}

// PrintMkdir - Print make directory.
// PrintMkdir - Print 使 directory.
func PrintMkdir(mkdir *sliverpb.Mkdir, con *console.SliverClient) {
	if mkdir.Response != nil && mkdir.Response.Err != "" {
		con.PrintErrorf("%s\n", mkdir.Response.Err)
		return
	}
	con.PrintInfof("%s\n", mkdir.Path)
}
