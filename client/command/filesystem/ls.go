package filesystem

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
// LsCmd - List 远程 directory. 的内容
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
// PrintLs - Display 和 sliverpb.Ls object.
func PrintLs(ls *sliverpb.Ls, flags *pflag.FlagSet, con *console.SliverClient) {
	if ls.Response != nil && ls.Response.Err != "" {
		con.PrintErrorf("%s\n", ls.Response.Err)
		return
	}

	// Generate metadata to print with the path
	// Generate 元数据与路径一起打印
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
	// Extract 标志
	reverseSort, _ := flags.GetBool("reverse")
	sortByTime, _ := flags.GetBool("modified")
	sortBySize, _ := flags.GetBool("size")

	/*
		By default, name sorting is case sensitive.  Upper case entries come before
		By 默认，名称排序为大小写 sensitive. Upper 大小写条目排在前面
		lower case ones.  Instead, we will level the playing field and sort
		小写 ones. Instead，我们将公平竞争并排序
		regardless of case
		不分情况
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
		After 名称已正确排序，如果修改时间请注意
		user wants to sort by time.  Doing this after sorting by name
		用户希望在按名称排序后按 time. Doing 排序
		will make the time sorted entries properly sorted by name if times are equal.
		如果时间是 equal. ，将使时间排序的条目按名称正确排序
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
