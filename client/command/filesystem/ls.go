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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/proto"
)

// LsCmd - List the contents of a remote directory.
func LsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var remotePath string
	if len(args) == 1 {
		remotePath = args[0]
	} else {
		remotePath = "."
	}

	ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    remotePath,
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	if ls.Response != nil && ls.Response.Async {
		con.AddBeaconCallback(ls.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, ls)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintLs(ls, cmd.Flags(), con)
		})
		con.PrintAsyncResponse(ls.Response)
	} else {
		PrintLs(ls, cmd.Flags(), con)
	}
}

// PrintLs - Display an sliverpb.Ls object.
func PrintLs(ls *sliverpb.Ls, flags *pflag.FlagSet, con *console.SliverClient) {
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

	// Extract the flags
	reverseSort, _ := flags.GetBool("reverse")
	sortByTime, _ := flags.GetBool("modified")
	sortBySize, _ := flags.GetBool("size")

	/*
		By default, name sorting is case sensitive.  Upper case entries come before
		lower case ones.  Instead, we will level the playing field and sort
		regardless of case
	*/
	if reverseSort {
		sort.SliceStable(ls.Files, func(i, j int) bool {
			return strings.ToLower(ls.Files[i].Name) > strings.ToLower(ls.Files[j].Name)
		})
	} else {
		sort.SliceStable(ls.Files, func(i, j int) bool {
			return strings.ToLower(ls.Files[i].Name) < strings.ToLower(ls.Files[j].Name)
		})
	}

	/*
		After names are sorted properly, take care of the modified time if the
		user wants to sort by time.  Doing this after sorting by name
		will make the time sorted entries properly sorted by name if times are equal.
	*/

	if sortByTime {
		if reverseSort {
			sort.SliceStable(ls.Files, func(i, j int) bool {
				return ls.Files[i].ModTime > ls.Files[j].ModTime
			})
		} else {
			sort.SliceStable(ls.Files, func(i, j int) bool {
				return ls.Files[i].ModTime < ls.Files[j].ModTime
			})
		}
	} else if sortBySize {
		if reverseSort {
			sort.SliceStable(ls.Files, func(i, j int) bool {
				return ls.Files[i].Size > ls.Files[j].Size
			})
		} else {
			sort.SliceStable(ls.Files, func(i, j int) bool {
				return ls.Files[i].Size < ls.Files[j].Size
			})
		}
	}

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
