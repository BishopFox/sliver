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

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// WGSocksListCmd - List WireGuard SOCKS proxies.
func WGSocksListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	socksList, err := con.Rpc.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: con.ActiveTarget.Request(cmd),
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

// SocksIDCompleter IDs of WireGuard socks servers.
func SocksIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		socksList, err := con.Rpc.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
			Request: con.ActiveTarget.Request(con.App.ActiveMenu().Root()),
		})
		if err != nil {
			return carapace.ActionMessage("failed to get Wireguard Socks servers: %s", err.Error())
		}

		for _, serv := range socksList.Servers {
			results = append(results, strconv.Itoa(int(serv.ID)))
			results = append(results, serv.LocalAddr)
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no Wireguard Socks servers")
		}

		return carapace.ActionValuesDescribed(results...).Tag("wireguard socks servers")
	}

	return carapace.ActionCallback(callback)
}
