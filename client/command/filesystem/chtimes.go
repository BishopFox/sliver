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
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ChtimesCmd - Change the access and modified time of a file on the remote file system.
func ChtimesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	// DateTime layout (https://pkg.go.dev/time)
	layout := "2006-01-02 15:04:05"
	filePath := args[0]

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	atime := args[1]

	if atime == "" {
		con.PrintErrorf("Missing parameter: Last accessed time id\n")
		return
	}

	t_a, err := time.Parse(layout, atime)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	unixAtime := t_a.Unix()

	mtime := args[2]

	if mtime == "" {
		con.PrintErrorf("Missing parameter: Last modified time id\n")
		return
	}

	t_b, err := time.Parse(layout, mtime)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	unixMtime := t_b.Unix()

	chtimes, err := con.Rpc.Chtimes(context.Background(), &sliverpb.ChtimesReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    filePath,
		ATime:   unixAtime,
		MTime:   unixMtime,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if chtimes.Response != nil && chtimes.Response.Async {
		con.AddBeaconCallback(chtimes.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, chtimes)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintChtimes(chtimes, con)
		})
		con.PrintAsyncResponse(chtimes.Response)
	} else {
		PrintChtimes(chtimes, con)
	}
}

// PrintChtimes - Print the Chtimes response.
func PrintChtimes(chtimes *sliverpb.Chtimes, con *console.SliverClient) {
	if chtimes.Response != nil && chtimes.Response.Err != "" {
		con.PrintErrorf("%s\n", chtimes.Response.Err)
		return
	}
	con.PrintInfof("%s\n", chtimes.Path)
}
