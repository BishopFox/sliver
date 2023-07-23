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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// CdCmd - Change directory on the remote system.
func CdCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	filePath := "."
	if len(args) == 1 {
		filePath = args[0]
	}

	pwd, err := con.Rpc.Cd(context.Background(), &sliverpb.CdReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    filePath,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if pwd.Response != nil && pwd.Response.Async {
		con.AddBeaconCallback(pwd.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, pwd)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintPwd(pwd, con)
		})
		con.PrintAsyncResponse(pwd.Response)
	} else {
		PrintPwd(pwd, con)
	}
}
