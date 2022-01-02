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

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// WGSocksListCmd - List WireGuard SOCKS proxies
func WGSocksListCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	socksList, err := con.Rpc.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}
	if socksList.Response != nil && socksList.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", socksList.Response.Err)
		return
	}

	if socksList.Servers != nil {
		if 0 < len(socksList.Servers) {
			tw := table.NewWriter()
			tw.SetStyle(settings.GetTableStyle(con))
			tw.AppendHeader(table.Row{
				"ID",
				"Local Address",
			})
			for _, server := range socksList.Servers {
				tw.AppendRow(table.Row{
					server.ID,
					server.LocalAddr,
				})
			}
			con.Println(tw.Render())
		}
	}

}
