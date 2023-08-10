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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// MemfilesAddCmd - Add memfile.
func MemfilesAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	memfilesAdd, err := con.Rpc.MemfilesAdd(context.Background(), &sliverpb.MemfilesAddReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	if memfilesAdd.Response != nil && memfilesAdd.Response.Async {
		con.AddBeaconCallback(memfilesAdd.Response, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, memfilesAdd)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintAddMemfile(memfilesAdd, con)
		})
	} else {
		PrintAddMemfile(memfilesAdd, con)
	}
}

// PrintAddMemfile - Print the memfiles response.
func PrintAddMemfile(memfilesAdd *sliverpb.MemfilesAdd, con *console.SliverClient) {
	if memfilesAdd.Response != nil && memfilesAdd.Response.Err != "" {
		con.PrintErrorf("%s\n", memfilesAdd.Response.Err)
		return
	}
	con.PrintInfof("New memfile descriptor: %d\n", memfilesAdd.Fd)
}
