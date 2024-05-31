package socks

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

// SocksCmd - Display information about tunneled port forward(s).
func SocksCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	socks := core.SocksProxies.List()
	if len(socks) == 0 {
		con.PrintInfof("No socks5 proxies\n")
		return
	}
	sort.Slice(socks, func(i, j int) bool {
		return socks[i].ID < socks[j].ID
	})

	session := con.ActiveTarget.GetSession()

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Session ID",
		"Bind Address",
		"Username",
		"Passwords",
	})
	for _, p := range socks {
		// if we're in an active session, just display socks proxies for the session
		if session != nil && session.ID != p.SessionID {
			continue
		}
		tw.AppendRow(table.Row{p.ID, p.SessionID, p.BindAddr, p.Username, p.Password})
	}

	con.Printf("%s\n", tw.Render())
}

// SocksIDCompleter completes IDs of remote of socks proxy servers.
func SocksIDCompleter(_ *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		socks := core.SocksProxies.List()
		if len(socks) == 0 {
			return carapace.ActionMessage("no active Socks proxies")
		}

		for _, serv := range socks {
			results = append(results, strconv.Itoa(int(serv.ID)))
			results = append(results, fmt.Sprintf("%s [%s] (%s)", serv.BindAddr, serv.Username, serv.SessionID))
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no Socks servers")
		}

		return carapace.ActionValuesDescribed(results...).Tag("socks servers")
	}

	return carapace.ActionCallback(callback)
}
