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
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// WGSocksStopCmd - Stop a WireGuard SOCKS proxy.
func WGSocksStopCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSession()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	socksID, err := strconv.Atoi(args[0])
	if err != nil {
		con.PrintErrorf("Error converting Socks ID (%s) to int: %s", args[0], err.Error())
		return
	}

	stopReq, err := con.Rpc.WGStopSocks(context.Background(), &sliverpb.WGSocksStopReq{
		ID:      int32(socksID),
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	if stopReq.Response != nil && stopReq.Response.Err != "" {
		con.PrintErrorf("Error: %v\n", stopReq.Response.Err)
		return
	}

	if stopReq.Server != nil {
		con.PrintInfof("Removed socks listener rule %s \n", stopReq.Server.LocalAddr)
	}
}
