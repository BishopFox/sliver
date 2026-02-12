package filesystem

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox
	Copyright (C) 2023 Bishop Fox

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
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func HeadCmd(cmd *cobra.Command, con *console.SliverClient, args []string, head bool) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var filePath string
	var fetchBytes bool
	var fetchSize int64
	var operationName string
	var download *sliverpb.Download
	var err error

	if len(args) > 0 {
		filePath = args[0]
	}
	if filePath == "" {
		con.PrintErrorf("Missing parameter: file name\n")
		return
	}

	if cmd.Flags().Changed("bytes") {
		fetchBytes = true
		fetchSize, _ = cmd.Flags().GetInt64("bytes")
		if fetchSize < 0 {
			// Cannot fetch a negative number of bytes
			// Cannot 获取负数字节
			con.PrintErrorf("The number of bytes specified must be positive.")
			return
		}
		if fetchSize == 1 {
			operationName = "byte"
		} else {
			operationName = "bytes"
		}
	} else {
		fetchBytes = false
		fetchSize, _ = cmd.Flags().GetInt64("lines")
		if fetchSize < 0 {
			// Cannot fetch a negative number of lines
			// Cannot 获取负数行
			con.PrintErrorf("The number of lines specified must be positive.")
			return
		}
		if fetchSize == 1 {
			operationName = "line"
		} else {
			operationName = "lines"
		}
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Retrieving %d %s from %s ...", fetchSize, operationName, filePath), ctrl)

	// A tail is a negative head
	// A尾部是负头
	if !head {
		fetchSize *= -1
	}

	if fetchBytes {
		download, err = con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
			Request:          con.ActiveTarget.Request(cmd),
			Path:             filePath,
			MaxBytes:         fetchSize,
			RestrictedToFile: true,
		})
	} else {
		download, err = con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
			Request:          con.ActiveTarget.Request(cmd),
			Path:             filePath,
			MaxLines:         fetchSize,
			RestrictedToFile: true,
		})
	}

	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if download.Response != nil && download.Response.Async {
		con.AddBeaconCallback(download.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, download)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintCat(filePath, download, cmd, con)
		})
		con.PrintAsyncResponse(download.Response)
	} else {
		PrintCat(filePath, download, cmd, con)
	}
}
