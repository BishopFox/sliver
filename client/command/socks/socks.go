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
	"sort"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// SocksCmd - Display information about tunneled port forward(s)
func SocksCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	socks := core.SocksProxies.List()
	if len(socks) == 0 {
		con.PrintInfof("No socks5 proxies\n")
		return
	}
	sort.Slice(socks[:], func(i, j int) bool {
		return socks[i].ID < socks[j].ID
	})

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
		tw.AppendRow(table.Row{p.ID, p.SessionID, p.BindAddr, p.Username, p.Password})
	}

	con.Printf("%s\n", tw.Render())
}
