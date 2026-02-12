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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// StartTCPListenerCmd - Start a TCP pivot listener on the remote system.
// StartTCPListenerCmd - Start 遥控器 system. 上的 TCP 枢轴 listener
func StartTCPListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	bind, _ := cmd.Flags().GetString("bind")
	lport, _ := cmd.Flags().GetUint16("lport")
	listener, err := con.Rpc.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType_TCP,
		BindAddress: fmt.Sprintf("%s:%d", bind, lport),
		Request:     con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if listener.Response != nil && listener.Response.Err != "" {
		con.PrintErrorf("%s\n", listener.Response.Err)
		return
	}
	con.PrintInfof("Started tcp pivot listener %s with id %d\n", listener.BindAddress, listener.ID)
}

// StartNamedPipeListenerCmd - Start a TCP pivot listener on the remote system.
// StartNamedPipeListenerCmd - Start 遥控器 system. 上的 TCP 枢轴 listener
func StartNamedPipeListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	allowAll, _ := cmd.Flags().GetBool("allow-all")
	bind, _ := cmd.Flags().GetString("bind")

	var options []bool
	options = append(options, allowAll)
	listener, err := con.Rpc.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType_NamedPipe,
		BindAddress: bind,
		Request:     con.ActiveTarget.Request(cmd),
		Options:     options,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if listener.Response != nil && listener.Response.Err != "" {
		con.PrintErrorf("%s\n", listener.Response.Err)
		return
	}
	con.PrintInfof("Started named pipe pivot listener %s with id %d\n", listener.BindAddress, listener.ID)
}
