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
	"regexp"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

var portNumberOnlyRegexp = regexp.MustCompile("^[0-9]+$")

// StartRportFwdListenerCmd - Start listener for reverse port forwarding on implant.
// StartRportFwdListenerCmd - Start listener 用于 implant. 上的反向端口转发
func StartRportFwdListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	if session.GetActiveC2() == "dns" {
		con.PrintWarnf("The current C2 is DNS, this is going to be a very slow tunnel!\n")
	}

	bindAddress, _ := cmd.Flags().GetString("bind")
	// Check if the bind address is just a port number, if no host is specified
	// Check 如果绑定地址只是一个端口号，如果没有指定主机
	// we just bind to all interfaces implant-side
	// 我们只需绑定到所有接口 implant__PH0__
	if portNumberOnlyRegexp.MatchString(bindAddress) {
		bindAddress = fmt.Sprintf(":%s", bindAddress)
	}

	forwardAddress, _ := cmd.Flags().GetString("remote")
	// Check if the forward address is just a port number, if no host is specified
	// Check 如果转发地址只是一个端口号，如果没有指定主机
	// we just forward to localhost client-side
	// 我们只是转发到本地主机 client__PH0__
	if portNumberOnlyRegexp.MatchString(forwardAddress) {
		forwardAddress = fmt.Sprintf("127.0.0.1:%s", forwardAddress)
	}
	rportfwdListener, err := con.Rpc.StartRportFwdListener(context.Background(), &sliverpb.RportFwdStartListenerReq{
		Request:        con.ActiveTarget.Request(cmd),
		BindAddress:    bindAddress,
		ForwardAddress: forwardAddress,
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	printStartedRportFwdListener(rportfwdListener, con)
}

func printStartedRportFwdListener(rportfwdListener *sliverpb.RportFwdListener, con *console.SliverClient) {
	if rportfwdListener.Response != nil && rportfwdListener.Response.Err != "" {
		con.PrintErrorf("%s", rportfwdListener.Response.Err)
		return
	}
	con.PrintInfof("Reverse port forwarding %s <- %s\n", rportfwdListener.ForwardAddress, rportfwdListener.BindAddress)
}
