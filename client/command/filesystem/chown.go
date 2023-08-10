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

// ChownCmd - Change the owner of a file on the remote file system.
func ChownCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := args[0]

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	uid := args[1]

	if uid == "" {
		con.PrintErrorf("Missing parameter: user id\n")
		return
	}

	gid := args[2]

	if gid == "" {
		con.PrintErrorf("Missing parameter: group id\n")
		return
	}

	recursive, _ := cmd.Flags().GetBool("recursive")

	chown, err := con.Rpc.Chown(context.Background(), &sliverpb.ChownReq{
		Request:   con.ActiveTarget.Request(cmd),
		Path:      filePath,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
	if err != nil {
		con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
		return
	}
	if chown.Response != nil && chown.Response.Async {
		con.AddBeaconCallback(chown.Response, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, chown)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintChown(chown, con)
		})
	} else {
		PrintChown(chown, con)
	}
}

// PrintChown - Print the chown response.
func PrintChown(chown *sliverpb.Chown, con *console.SliverClient) {
	if chown.Response != nil && chown.Response.Err != "" {
		con.PrintErrorf("%s\n", chown.Response.Err)
		return
	}
	con.PrintInfof("%s\n", chown.Path)
}
