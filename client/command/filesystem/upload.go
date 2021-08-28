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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"

	"github.com/desertbit/grumble"
)

// UploadCmd - Upload a file to the remote system
func UploadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	localPath := ctx.Args.String("local-path")
	remotePath := ctx.Args.String("remote-path")
	isIOC := ctx.Flags.Bool("ioc")

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

	if remotePath == "" {
		fileName := filepath.Base(src)
		remotePath = fileName
	}
	dst := remotePath

	fileBuf, err := ioutil.ReadFile(src)
	uploadGzip := new(encoders.Gzip).Encode(fileBuf)

	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("%s -> %s", src, dst), ctrl)
	upload, err := con.Rpc.Upload(context.Background(), &sliverpb.UploadReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    dst,
		Data:    uploadGzip,
		Encoder: "gzip",
		IsIOC:   isIOC,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Wrote file to %s\n", upload.Path)
	}
}
