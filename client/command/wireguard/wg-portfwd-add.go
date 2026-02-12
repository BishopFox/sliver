package wireguard

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
	"net"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// WGPortFwdAddCmd - Add a new WireGuard port forward.
// WGPortFwdAddCmd - Add 一个新的 WireGuard 端口 forward.
func WGPortFwdAddCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.Transport != "wg" {
		con.PrintErrorf("This command is only supported for WireGuard implants")
		return
	}

	localPort, _ := cmd.Flags().GetInt32("bind")
	remoteAddr, _ := cmd.Flags().GetString("remote")
	if remoteAddr == "" {
		con.PrintErrorf("Must specify a remote target host:port")
		return
	}
	remoteHost, remotePort, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		con.PrintErrorf("Failed to parse remote target %s\n", err)
		return
	}

	portfwdAdd, err := con.Rpc.WGStartPortForward(context.Background(), &sliverpb.WGPortForwardStartReq{
		LocalPort:     localPort,
		RemoteAddress: remoteAddr,
		Request:       con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	if portfwdAdd.Response != nil && portfwdAdd.Response.Err != "" {
		con.PrintErrorf("Error: %s\n", portfwdAdd.Response.Err)
		return
	}
	con.PrintInfof("Port forwarding %s -> %s:%s\n", portfwdAdd.Forwarder.LocalAddr, remoteHost, remotePort)
}
