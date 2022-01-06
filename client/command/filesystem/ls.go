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
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// LsCmd - List the contents of a remote directory
func LsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := ctx.Args.String("path")
	filter := ""

	/*
		Check to see if the remote path is a filter or contains a filter.
		If the path passes the test to be a filter, then it is a filter
		because paths are not valid filters.
	*/
	if remotePath != "." {

		// Check if the path contains a filter
		// Test on a standardized version of the path (change any \ to /)
		testPath := strings.Replace(remotePath, "\\", "/", -1)
		/*
			Cannot use the path or filepath libraries because the OS
			of the client does not necessarily match the OS of the
			implant
		*/
		lastSeparatorOccurrence := strings.LastIndex(testPath, "/")

		if lastSeparatorOccurrence == -1 {
			// Then this is only a filter
			filter = remotePath
			remotePath = "."
		} else {
			// Then we need to test for a filter on the end of the string

			// The indicies should be the same because we did not change the length of the string
			baseDir := remotePath[:lastSeparatorOccurrence+1]
			potentialFilter := remotePath[lastSeparatorOccurrence+1:]

			_, err := filepath.Match(potentialFilter, "")

			if err == nil {
				// Then we have a filter on the end of the path
				remotePath = baseDir
				filter = potentialFilter
			} else {
				if !strings.HasSuffix(remotePath, "/") {
					remotePath = remotePath + "/"
				}
			}
		}
	}

	ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: con.ActiveTarget.Request(ctx),
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
			PrintLs(ls, ctx.Flags, filter, con)
		})
		con.PrintAsyncResponse(ls.Response)
	} else {
		PrintLs(ls, ctx.Flags, filter, con)
	}
}

// PrintLs - Display an sliverpb.Ls object
func PrintLs(ls *sliverpb.Ls, flags grumble.FlagMap, filter string, con *console.SliverConsoleClient) {
	if ls.Response != nil && ls.Response.Err != "" {
		con.PrintErrorf("%s\n", ls.Response.Err)
		return
	}

	con.Printf("%s\n", ls.Path)
	con.Printf("%s\n", strings.Repeat("=", len(ls.Path)))

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Extract the flags
	reverseSort := flags.Bool("reverse")
	sortByTime := flags.Bool("modified")
	sortBySize := flags.Bool("size")

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
		if filter != "" {
			fileMatch, err := filepath.Match(filter, fileInfo.Name)

			if err != nil {
				/*
					This should not happen because we checked the filter
					before reaching out to the implant.
				*/
				con.PrintErrorf("%s is not a valid filter: %s\n", filter, err)
				break
			}

			if !fileMatch {
				continue
			}
		}

		modTime := time.Unix(fileInfo.ModTime, 0)
		implantLocation := time.FixedZone(ls.Timezone, int(ls.TimezoneOffset))
		modTime = modTime.In(implantLocation)

		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t%s\t<dir>\t%s\n", fileInfo.Mode, fileInfo.Name, modTime.Format(time.RubyDate))
		} else {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", fileInfo.Mode, fileInfo.Name, util.ByteCountBinary(fileInfo.Size), modTime.Format(time.RubyDate))
		}
	}
	table.Flush()
	con.Printf("%s\n", outputBuf.String())
}
