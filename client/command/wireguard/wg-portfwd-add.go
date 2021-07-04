package wireguard

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
	"net"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// WGPortFwdAddCmd - Add a new WireGuard port forward
func WGPortFwdAddCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	localPort := ctx.Flags.Int("bind")
	remoteAddr := ctx.Flags.String("remote")
	if remoteAddr == "" {
		con.PrintErrorf("Must specify a remote target host:port")
		return
	}
	remoteHost, remotePort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		con.PrintErrorf("Failed to parse remote target %s\n", err)
		return
	}

	pfwdAdd, err := con.Rpc.WGStartPortForward(context.Background(), &sliverpb.WGPortForwardStartReq{
		LocalPort:     int32(localPort),
		RemoteAddress: remoteAddr,
		Request:       con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	if pfwdAdd.Response != nil && pfwdAdd.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", pfwdAdd.Response.Err)
		return
	}
	con.PrintInfof("Port forwarding %s -> %s:%s\n", pfwdAdd.Forwarder.LocalAddr, remoteHost, remotePort)
}
