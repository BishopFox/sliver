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
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// RunAsCmd - Run a command as another user on the remote system
func RunAsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	username := ctx.Flags.String("username")
	process := ctx.Flags.String("process")
	arguments := ctx.Flags.String("args")

	if username == "" {
		con.PrintErrorf("Please specify a username\n")
		return
	}

	if process == "" {
		con.PrintErrorf("Please specify a process path\n")
		return
	}

	runAsResp, err := con.Rpc.RunAs(context.Background(), &sliverpb.RunAsReq{
		Request:     con.ActiveTarget.Request(ctx),
		Username:    username,
		ProcessName: process,
		Args:        arguments,
	})

	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}

	if runAsResp.GetResponse().GetErr() != "" {
		con.PrintErrorf("%s\n", runAsResp.GetResponse().GetErr())
		return
	}

	con.PrintInfof("Successfully ran %s %s on %s\n", process, arguments, session.GetName())
}
