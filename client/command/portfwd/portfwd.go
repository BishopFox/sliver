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
	"fmt"
	"sort"
	"strconv"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// PortfwdCmd - Display information about tunneled port forward(s).
func PortfwdCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	PrintPortfwd(con)
}

// PrintPortfwd - Print the port forward(s).
func PrintPortfwd(con *console.SliverClient) {
	portfwds := core.Portfwds.List()
	if len(portfwds) == 0 {
		con.PrintInfof("No port forwards\n")
		return
	}
	sort.Slice(portfwds, func(i, j int) bool {
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

// PortfwdIDCompleter completes IDs of local portforwarders.
func PortfwdIDCompleter(_ *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		portfwds := core.Portfwds.List()
		if len(portfwds) == 0 {
			return carapace.ActionMessage("no active local port forwarders")
		}

		for _, fwd := range portfwds {
			results = append(results, strconv.Itoa(int(fwd.ID)))
			results = append(results, fmt.Sprintf("%s (%s)", fwd.BindAddr, fwd.SessionID))
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no local port forwarders")
		}

		return carapace.ActionValuesDescribed(results...).Tag("local port forwarders")
	}

	return carapace.ActionCallback(callback)
}
