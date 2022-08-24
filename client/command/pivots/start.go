package pivots

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
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// StartTCPListenerCmd - Start a TCP pivot listener on the remote system
func StartTCPListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	bind := ctx.Flags.String("bind")
	lport := uint16(ctx.Flags.Int("lport"))
	listener, err := con.Rpc.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType_TCP,
		BindAddress: fmt.Sprintf("%s:%d", bind, lport),
		Request:     con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if listener.Response != nil && listener.Response.Err != "" {
		con.PrintErrorf("%s\n", listener.Response.Err)
		return
	}
	con.PrintInfof("Started tcp pivot listener %s with id %d\n", listener.BindAddress, listener.ID)
}

// StartNamedPipeListenerCmd - Start a TCP pivot listener on the remote system
func StartNamedPipeListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	var options []bool
	options = append(options, ctx.Flags.Bool("allow-all"))
	listener, err := con.Rpc.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType_NamedPipe,
		BindAddress: ctx.Flags.String("bind"),
		Request:     con.ActiveTarget.Request(ctx),
		Options:     options,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if listener.Response != nil && listener.Response.Err != "" {
		con.PrintErrorf("%s\n", listener.Response.Err)
		return
	}
	con.PrintInfof("Started named pipe pivot listener %s with id %d\n", listener.BindAddress, listener.ID)
}
