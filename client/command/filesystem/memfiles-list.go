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
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// MemfilesListCmd - List memfiles.
func MemfilesListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	memfilesList, err := con.Rpc.MemfilesList(context.Background(), &sliverpb.MemfilesListReq{
		Request: con.ActiveTarget.Request(cmd),
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
			PrintMemfiles(memfilesList, con)
		})
		con.PrintAsyncResponse(memfilesList.Response)
	} else {
		PrintMemfiles(memfilesList, con)
	}
}

// PrintMemfiles - Display an sliverpb.Ls object.
func PrintMemfiles(ls *sliverpb.Ls, con *console.SliverClient) {
	if ls.Response != nil && ls.Response.Err != "" {
		con.PrintErrorf("%s\n", ls.Response.Err)
		return
	}

	// Generate metadata to print with the path
	numberOfFiles := len(ls.Files)
	var totalSize int64 = 0
	var pathInfo string

	for _, fileInfo := range ls.Files {
		totalSize += fileInfo.Size
	}

	if numberOfFiles == 1 {
		pathInfo = fmt.Sprintf("%s (%d item, %s)", ls.Path, numberOfFiles, util.ByteCountBinary(totalSize))
	} else {
		pathInfo = fmt.Sprintf("%s (%d items, %s)", ls.Path, numberOfFiles, util.ByteCountBinary(totalSize))
	}

	con.Printf("%s\n", pathInfo)
	con.Printf("%s\n", strings.Repeat("=", len(pathInfo)))

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	for _, fileInfo := range ls.Files {
		modTime := time.Unix(fileInfo.ModTime, 0)
		implantLocation := time.FixedZone(ls.Timezone, int(ls.TimezoneOffset))
		modTime = modTime.In(implantLocation)

		owner := ""
		if fileInfo.Uid != "" {
			owner = fileInfo.Uid
		}
		if fileInfo.Gid != "" {
			owner = owner + ":" + fileInfo.Gid + "\t"
		}

		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t%s%s\t<dir>\t%s\n", fileInfo.Mode, owner, fileInfo.Name, modTime.Format(time.RubyDate))
		} else if fileInfo.Link != "" {
			fmt.Fprintf(table, "%s\t%s%s -> %s\t%s\t%s\n", fileInfo.Mode, owner, fileInfo.Name, fileInfo.Link, util.ByteCountBinary(fileInfo.Size), modTime.Format(time.RubyDate))
		} else {
			fmt.Fprintf(table, "%s\t%s%s\t%s\t%s\n", fileInfo.Mode, owner, fileInfo.Name, util.ByteCountBinary(fileInfo.Size), modTime.Format(time.RubyDate))
		}

	}
	table.Flush()
	con.Printf("%s\n", outputBuf.String())
}
