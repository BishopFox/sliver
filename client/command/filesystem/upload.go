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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
)

// UploadCmd - Upload a file to the remote system
func UploadCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := ""

	localPath := args[0]
	if len(args) > 1 {
		remotePath = args[1]
	}

	// localPath := ctx.Args.String("local-path")
	// remotePath := ctx.Args.String("remote-path")
	isIOC, _ := cmd.Flags().GetBool("ioc")

	if localPath == "" {
		con.PrintErrorf("Missing parameter, see `help upload`\n")
		return
	}

	src, _ := filepath.Abs(localPath)
	_, err := os.Stat(src)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	fileName := filepath.Base(src)

	if remotePath == "" {
		remotePath = fileName
	}

	dst := remotePath

	fileBuf, err := ioutil.ReadFile(src)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	uploadGzip, _ := new(encoders.Gzip).Encode(fileBuf)

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("%s -> %s", src, dst), ctrl)
	upload, err := con.Rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Request:  con.ActiveTarget.Request(cmd),
		Path:     dst,
		Data:     uploadGzip,
		Encoder:  "gzip",
		IsIOC:    isIOC,
		FileName: fileName,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if upload.Response != nil && upload.Response.Async {
		con.AddBeaconCallback(upload.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, upload)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintUpload(upload, con)
		})
		con.PrintAsyncResponse(upload.Response)
	} else {
		PrintUpload(upload, con)
	}
}

// PrintUpload - Print the result of the upload command
func PrintUpload(upload *sliverpb.Upload, con *console.SliverConsoleClient) {
	if upload.Response != nil && upload.Response.Err != "" {
		con.PrintErrorf("%s\n", upload.Response.Err)
		return
	}
	con.PrintInfof("Wrote file to %s\n", upload.Path)
}
