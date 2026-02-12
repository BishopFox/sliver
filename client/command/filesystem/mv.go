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

func MvCmd(cmd *cobra.Command, con *console.SliverClient, args []string) (err error) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	src := args[0]
	// src := ctx.Args.String("src")
	// src := ctx.Args.String(__PH0__)
	// if src == "" {
	// 如果 src == __PH0__ {
	// 	con.PrintErrorf("Missing parameter: src\n")
	// 	con.PrintErrorf(__PH0__)
	// 	return
	// 	返回
	// }

	dst := args[1]
	// dst := ctx.Args.String("dst")
	// dst := ctx.Args.String(__PH0__)
	// if dst == "" {
	// 如果 dst == __PH0__ {
	// 	con.PrintErrorf("Missing parameter: dst\n")
	// 	con.PrintErrorf(__PH0__)
	// 	return
	// 	返回
	// }

	mv, err := con.Rpc.Mv(context.Background(), &sliverpb.MvReq{
		Request: con.ActiveTarget.Request(cmd),
		Src:     src,
		Dst:     dst,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	mv.Src, mv.Dst = src, dst

	if mv.Response != nil && mv.Response.Async {
		con.AddBeaconCallback(mv.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, mv)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
		})
		con.PrintAsyncResponse(mv.Response)
	} else {
		PrintMv(mv, con)
	}
	return
}

// PrintMv - Print the renamed file.
// PrintMv - Print 更名为 file.
func PrintMv(mv *sliverpb.Mv, con *console.SliverClient) {
	if mv.Response != nil && mv.Response.Err != "" {
		con.PrintErrorf("%s\n", mv.Response.Err)
		return
	}
	con.PrintInfof("%s > %s\n", mv.Src, mv.Dst)
}
