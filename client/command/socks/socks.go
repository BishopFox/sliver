package socks

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
// SocksCmd - Display 有关隧道端口转发的信息。
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
		// 如果我们处于活动的 session 中，则只需显示 session 的袜子代理
		if session != nil && session.ID != p.SessionID {
			continue
		}
		tw.AppendRow(table.Row{p.ID, p.SessionID, p.BindAddr, p.Username, p.Password})
	}

	con.Printf("%s\n", tw.Render())
}

// SocksIDCompleter completes IDs of remote of socks proxy servers.
// SocksIDCompleter完成socks代理servers.远程IDs
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
