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

// RmCmd - Remove a directory from the remote file system
func RmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := ctx.Args.String("path")

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	rm, err := con.Rpc.Rm(context.Background(), &sliverpb.RmReq{
		Request:   con.ActiveTarget.Request(ctx),
		Path:      filePath,
		Recursive: ctx.Flags.Bool("recursive"),
		Force:     ctx.Flags.Bool("force"),
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

// PrintRm - Print the rm response
func PrintRm(rm *sliverpb.Rm, con *console.SliverConsoleClient) {
	if rm.Response != nil && rm.Response.Err != "" {
		con.PrintErrorf("%s\n", rm.Response.Err)
		return
	}
	con.PrintInfof("%s\n", rm.Path)
}
