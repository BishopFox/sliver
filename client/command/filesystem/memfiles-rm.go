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
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// MemfilesRmCmd - Remove a memfile.
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
func PrintRmMemfile(memfilesList *sliverpb.MemfilesRm, con *console.SliverClient) {
	if memfilesList.Response != nil && memfilesList.Response.Err != "" {
		con.PrintErrorf("%s\n", memfilesList.Response.Err)
		return
	}
	con.PrintInfof("Removed memfile descriptor: %d\n", memfilesList.Fd)
}
