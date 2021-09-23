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
	"os"
	"path"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// DownloadCmd - Download a file from the remote system
func DownloadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	remotePath := ctx.Args.String("remote-path")
	localPath := ctx.Args.String("local-path")

	src := remotePath
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(localPath)
	fi, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		con.PrintErrorf("%s\n", err)
		return
	}
	if err == nil && fi.IsDir() {
		dst = path.Join(dst, fileName)
	}

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("%s -> %s", fileName, dst), ctrl)
	download, err := con.Rpc.Download(context.Background(), &sliverpb.DownloadReq{
		Request: con.ActiveTarget.Request(ctx),
		Path:    remotePath,
	})
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
			PrintDownload(download, ctx, con)
		})
		con.PrintAsyncResponse(download.Response)
	} else {
		PrintDownload(download, ctx, con)
	}
}

// PrintDownload - Print the download response, and save file to disk
func PrintDownload(download *sliverpb.Download, ctx *grumble.Context, con *console.SliverConsoleClient) {
	if download.Response != nil && download.Response.Err != "" {
		con.PrintErrorf("%s\n", download.Response.Err)
		return
	}

	remotePath := ctx.Args.String("remote-path")
	localPath := ctx.Args.String("local-path")

	src := remotePath
	fileName := filepath.Base(src)
	dst, _ := filepath.Abs(localPath)
	fi, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		con.PrintErrorf("%s\n", err)
		return
	}
	if err == nil && fi.IsDir() {
		dst = path.Join(dst, fileName)
	}
	if _, err := os.Stat(dst); err == nil {
		overwrite := false
		prompt := &survey.Confirm{Message: "Overwrite local file?"}
		survey.AskOne(prompt, &overwrite, nil)
		if !overwrite {
			return
		}
	}
	if download.Encoder == "gzip" {
		download.Data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			con.PrintErrorf("Decoding failed %s", err)
			return
		}
	}
	dstFile, err := os.Create(dst)
	if err != nil {
		con.PrintErrorf("Failed to open local file %s: %s\n", dst, err)
		return
	}
	defer dstFile.Close()
	n, err := dstFile.Write(download.Data)
	if err != nil {
		con.PrintErrorf("Failed to write data %v\n", err)
	} else {
		con.PrintInfof("Wrote %d bytes to %s\n", n, dstFile.Name())
	}
}
