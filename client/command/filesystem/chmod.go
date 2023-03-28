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

// ChmodCmd - Change the permissions of a file on the remote file system
func ChmodCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := ctx.Args.String("path")

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	fileMode := ctx.Args.String("mode")

	if fileMode == "" {
		con.PrintErrorf("Missing parameter: file permissions (mode)\n")
		return
	}

	chmod, err := con.Rpc.Chmod(context.Background(), &sliverpb.ChmodReq{
		Request:   con.ActiveTarget.Request(ctx),
		Path:      filePath,
		FileMode:  fileMode,
		Recursive: ctx.Flags.Bool("recursive"),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if chmod.Response != nil && chmod.Response.Async {
		con.AddBeaconCallback(chmod.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, chmod)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintChmod(chmod, con)
		})
		con.PrintAsyncResponse(chmod.Response)
	} else {
		PrintChmod(chmod, con)
	}
}

// PrintChmod - Print the chmod response
func PrintChmod(chmod *sliverpb.Chmod, con *console.SliverConsoleClient) {
	if chmod.Response != nil && chmod.Response.Err != "" {
		con.PrintErrorf("%s\n", chmod.Response.Err)
		return
	}
	con.PrintInfof("%s\n", chmod.Path)
}
