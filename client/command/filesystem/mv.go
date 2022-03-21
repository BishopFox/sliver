package filesystem

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

func MvCmd(ctx *grumble.Context, con *console.SliverConsoleClient) (err error) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	src := ctx.Args.String("src")
	if src == "" {
		con.PrintErrorf("Missing parameter: src\n")
		return
	}

	dst := ctx.Args.String("dst")
	if dst == "" {
		con.PrintErrorf("Missing parameter: dst\n")
		return
	}

	mv, err := con.Rpc.Mv(context.Background(), &sliverpb.MvReq{
		Request: con.ActiveTarget.Request(ctx),
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

// PrintMv - Print the renamed file
func PrintMv(mv *sliverpb.Mv, con *console.SliverConsoleClient) {
	if mv.Response != nil && mv.Response.Err != "" {
		con.PrintErrorf("%s\n", mv.Response.Err)
		return
	}
	con.PrintInfof("%s > %s\n", mv.Src, mv.Dst)
}
