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
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// WGPortFwdListCmd - List WireGuard port forwards.
func WGPortFwdListCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	fwdList, err := con.Rpc.WGListForwarders(context.Background(), &sliverpb.WGTCPForwardersReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}
	if fwdList.Response != nil && fwdList.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", fwdList.Response.Err)
		return
	}

	if fwdList.Forwarders != nil {
		if len(fwdList.Forwarders) == 0 {
			con.PrintInfof("No port forwards\n")
		} else {
			tw := table.NewWriter()
			tw.SetStyle(settings.GetTableStyle(con))
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Protocol",
				"Port",
			})
			for _, fwd := range fwdList.Forwarders {
				tw.AppendRow(table.Row{
					fwd.ID,
					fwd.LocalAddr,
					fwd.RemoteAddr,
				})
			}
			con.Println(tw.Render())
		}
	}
}
