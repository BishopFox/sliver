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

// RmCmd - Remove a directory from the remote file system.
// RmCmd - Remove 远程文件 system. 中的目录
func RmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := args[0]
	// filePath := ctx.Args.String("path")
	// filePath := ctx.Args.String(__PH0__)

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	force, _ := cmd.Flags().GetBool("force")

	rm, err := con.Rpc.Rm(context.Background(), &sliverpb.RmReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      filePath,
		Recursive: recursive,
		Force:     force,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if rm.Response != nil && rm.Response.Async {
		con.AddBeaconCallback(rm.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, rm)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRm(rm, con)
		})
		con.PrintAsyncResponse(rm.Response)
	} else {
		PrintRm(rm, con)
	}
}

// PrintRm - Print the rm response.
// PrintRm - Print rm response.
func PrintRm(rm *sliverpb.Rm, con *console.SliverClient) {
	if rm.Response != nil && rm.Response.Err != "" {
		con.PrintErrorf("%s\n", rm.Response.Err)
		return
	}
	con.PrintInfof("%s\n", rm.Path)
}
