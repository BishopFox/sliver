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
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// MemfilesRmCmd - Remove a memfile.
// MemfilesRmCmd - Remove 和 memfile.
func MemfilesRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	fdArg := args[0]
	if fdArg == "" {
		con.PrintErrorf("Missing parameter: File Descriptor\n")
		return
	}

	fdInt, err := strconv.ParseInt(fdArg, 0, 64)
	if err != nil {
		con.PrintErrorf("Failed to parse fdArg: %s\n", err)
		return
	}
	memfilesList, err := con.Rpc.MemfilesRm(context.Background(), &sliverpb.MemfilesRmReq{
		Request: con.ActiveTarget.Request(cmd),
		Fd:      fdInt,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if memfilesList.Response != nil && memfilesList.Response.Async {
		con.AddBeaconCallback(memfilesList.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, memfilesList)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRmMemfile(memfilesList, con)
		})
		con.PrintAsyncResponse(memfilesList.Response)
	} else {
		PrintRmMemfile(memfilesList, con)
	}
}

// PrintRmMemfile - Remove a memfile.
// PrintRmMemfile - Remove 和 memfile.
func PrintRmMemfile(memfilesList *sliverpb.MemfilesRm, con *console.SliverClient) {
	if memfilesList.Response != nil && memfilesList.Response.Err != "" {
		con.PrintErrorf("%s\n", memfilesList.Response.Err)
		return
	}
	con.PrintInfof("Removed memfile descriptor: %d\n", memfilesList.Fd)
}
