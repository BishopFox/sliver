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

// RunAsCmd - Run a command as another user on the remote system
func RunAsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	username := ctx.Flags.String("username")
	password := ctx.Flags.String("password")
	domain := ctx.Flags.String("domain")
	showWindow := ctx.Flags.Bool("show-window")
	process := ctx.Flags.String("process")
	arguments := ctx.Flags.String("args")
	netonly := ctx.Flags.Bool("net-only")

	if username == "" {
		con.PrintErrorf("Please specify a username\n")
		return
	}

	if process == "" {
		con.PrintErrorf("Please specify a process path\n")
		return
	}

	runAs, err := con.Rpc.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     con.ActiveTarget.Request(ctx),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
		Domain:      domain,
		Password:    password,
		HideWindow:  !showWindow,
		NetOnly:     netonly,
	})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	name := getName(session, beacon)
	if runAs.Response != nil && runAs.Response.Async {
		con.AddBeaconCallback(runAs.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, runAs)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRunAs(runAs, process, arguments, name, con)
		})
		con.PrintAsyncResponse(runAs.Response)
	} else {
		PrintRunAs(runAs, process, arguments, name, con)
	}

}

// PrintRunAs - Print the result of run as
func PrintRunAs(runAs *sliverpb.RunAs, process string, args string, name string, con *console.SliverConsoleClient) {
	if runAs.Response != nil && runAs.Response.GetErr() != "" {
		con.PrintErrorf("%s\n", runAs.Response.GetErr())
		return
	}
	con.PrintInfof("Successfully ran %s %s on %s\n", process, args, name)
}

func getName(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return session.Name
	}
	if beacon != nil {
		return beacon.Name
	}
	panic("no session or beacon")
}
