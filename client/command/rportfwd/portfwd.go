package rportfwd

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
