package rportfwd

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

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
	"context"
	"fmt"
	"strconv"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// StartRportFwdListenerCmd - Start listener for reverse port forwarding on implant.
// StartRportFwdListenerCmd - Start listener 用于 implant. 上的反向端口转发
func RportFwdListenersCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	rportfwdListeners, err := con.Rpc.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	PrintRportFwdListeners(rportfwdListeners, cmd.Flags(), con)
}

func PrintRportFwdListeners(rportfwdListeners *sliverpb.RportFwdListeners, flags *pflag.FlagSet, con *console.SliverClient) {
	if rportfwdListeners.Response != nil && rportfwdListeners.Response.Err != "" {
		con.PrintErrorf("%s\n", rportfwdListeners.Response.Err)
		return
	}

	if len(rportfwdListeners.Listeners) == 0 {
		con.PrintInfof("No reverse port forwards\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"ID",
		"Remote Address",
		"Bind Address",
	})
	for _, p := range rportfwdListeners.Listeners {
		tw.AppendRow(table.Row{
			p.ID,
			p.ForwardAddress,
			p.BindAddress,
		})
	}
	con.Printf("%s\n", tw.Render())
}

// PortfwdIDCompleter completes IDs of remote portforwarders.
// PortfwdIDCompleter 完成远程 portforwarders. 的 IDs
func PortfwdIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		rportfwdListeners, err := con.Rpc.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
			Request: con.ActiveTarget.Request(con.App.ActiveMenu().Root()),
		})
		if err != nil {
			return carapace.ActionMessage("failed to get remote port forwarders: %s", err.Error())
		}

		for _, fwd := range rportfwdListeners.Listeners {
			results = append(results, strconv.Itoa(int(fwd.ID)))
			faddr := fmt.Sprintf("%s:%d", fwd.ForwardAddress, fwd.ForwardPort)
			laddr := fmt.Sprintf("%s:%d", fwd.BindAddress, fwd.BindPort)
			results = append(results, fmt.Sprintf("%s <- %s", laddr, faddr))
		}

		if len(results) == 0 {
			return carapace.ActionMessage("no remote port forwarders")
		}

		return carapace.ActionValuesDescribed(results...).Tag("remote port forwarders")
	}

	return carapace.ActionCallback(callback)
}
