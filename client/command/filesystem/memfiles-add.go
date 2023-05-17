package filesystem

/*
	Copyright (C) 2023 b0yd

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

// MemfilesAddCmd - Add memfile
func MemfilesAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	memfilesAdd, err := con.Rpc.MemfilesAdd(context.Background(), &sliverpb.MemfilesAddReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if memfilesAdd.Response != nil && memfilesAdd.Response.Async {
		con.AddBeaconCallback(memfilesAdd.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, memfilesAdd)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintAddMemfile(memfilesAdd, con)
		})
		con.PrintAsyncResponse(memfilesAdd.Response)
	} else {
		PrintAddMemfile(memfilesAdd, con)
	}
}

// PrintAddMemfile - Print the memfiles response
func PrintAddMemfile(memfilesAdd *sliverpb.MemfilesAdd, con *console.SliverConsoleClient) {
	if memfilesAdd.Response != nil && memfilesAdd.Response.Err != "" {
		con.PrintErrorf("%s\n", memfilesAdd.Response.Err)
		return
	}
	con.PrintInfof("New memfile descriptor: %d\n", memfilesAdd.Fd)
}
