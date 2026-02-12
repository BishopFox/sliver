package pivots

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
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// PivotDetailsCmd - Display pivots for all sessions
// 所有会话的 PivotDetailsCmd - Display 枢轴
func PivotDetailsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	pivotListeners, err := con.Rpc.PivotSessionListeners(context.Background(), &sliverpb.PivotListenersReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if pivotListeners.Response != nil && pivotListeners.Response.Err != "" {
		con.PrintErrorf("%s\n", pivotListeners.Response.Err)
		return
	}

	id, _ := cmd.Flags().GetUint32("id")
	if id == uint32(0) {
		selectedListener, err := SelectPivotListener(pivotListeners.Listeners, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		id = selectedListener.ID
	}

	found := false
	for _, listener := range pivotListeners.Listeners {
		if listener.ID == id {
			PrintPivotListenerDetails(listener, con)
			found = true
		}
	}
	if !found {
		con.PrintErrorf("No pivot listener with id %d\n", id)
	}
}

// PrintPivotListenerDetails - Print details of a single pivot listener
// PrintPivotListenerDetails - Print 单枢轴 listener 的详细信息
func PrintPivotListenerDetails(listener *sliverpb.PivotListener, con *console.SliverClient) {
	con.Printf("\n")
	con.Printf("               ID: %d\n", listener.ID)
	con.Printf("         Protocol: %s\n", PivotTypeToString(listener.Type))
	con.Printf("     Bind Address: %s\n", listener.BindAddress)
	con.Printf(" Number of Pivots: %d\n", len(listener.Pivots))
	con.Printf("\n")

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.StyleBold.Render(fmt.Sprintf("%s Pivots", PivotTypeToString(listener.Type))))
	tw.AppendSeparator()
	tw.AppendHeader(table.Row{
		"ID",
		"Remote Address",
	})
	for _, pivotListener := range listener.Pivots {
		tw.AppendRow(table.Row{
			pivotListener.PeerID,
			pivotListener.RemoteAddress,
		})
	}
	con.Printf("%s\n", tw.Render())
}
