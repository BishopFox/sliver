package portfwd

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
	"sort"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// PortfwdCmd - Display information about tunneled port forward(s)
func PortfwdCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	PrintPortfwd(con)
}

// PrintPortfwd - Print the port forward(s)
func PrintPortfwd(con *console.SliverConsoleClient) {
	portfwds := core.Portfwds.List()
	if len(portfwds) == 0 {
		con.PrintInfof("No port forwards\n")
		return
	}
	sort.Slice(portfwds[:], func(i, j int) bool {
		return portfwds[i].ID < portfwds[j].ID
	})

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Session ID",
		"Bind Address",
		"Remote Address",
	})
	for _, p := range portfwds {
		tw.AppendRow(table.Row{
			p.ID,
			p.SessionID,
			p.BindAddr,
			p.RemoteAddr,
		})
	}
	con.Printf("%s\n", tw.Render())
}
