package processes

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

// TerminateCmd - Terminate a process on the remote system
func TerminateCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	pid := ctx.Args.Uint("pid")
	terminated, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
		Request: con.ActiveTarget.Request(ctx),
		Pid:     int32(pid),
		Force:   ctx.Flags.Bool("force"),
	})

	if terminated.Response != nil && terminated.Response.Async {
		con.AddBeaconCallback(terminated.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, terminated)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			if err != nil {
				con.PrintErrorf("%s\n", err)
			} else {
				con.PrintInfof("Process %d has been terminated\n", terminated.Pid)
			}
		})
		con.PrintAsyncResponse(terminated.Response)
	} else {
		if err != nil {
			con.PrintErrorf("%s\n", err)
		} else {
			con.PrintInfof("Process %d has been terminated\n", terminated.Pid)
		}
	}
}
