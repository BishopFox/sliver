package portfwd

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

// PortfwdCmd - Display information about tunneled port forward(s).
// PortfwdCmd - Display 有关隧道端口转发的信息。
func PortfwdCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	PrintPortfwd(con)
}

// PrintPortfwd - Print the port forward(s).
// PrintPortfwd - Print 端口转发。
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
// PortfwdIDCompleter 完成本地 portforwarders. 的 IDs
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
