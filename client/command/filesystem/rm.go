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
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// RmCmd - Remove a directory from the remote file system.
func RmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := args[0]
	// filePath := ctx.Args.String("path")

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
func PrintRm(rm *sliverpb.Rm, con *console.SliverClient) {
	if rm.Response != nil && rm.Response.Err != "" {
		con.PrintErrorf("%s\n", rm.Response.Err)
		return
	}
	con.PrintInfof("%s\n", rm.Path)
}
