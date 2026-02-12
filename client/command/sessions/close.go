package sessions

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// CloseSessionCmd - Close an interactive session but do not kill the remote process.
// CloseSessionCmd - Close 是一个交互式 session 但不杀死远程 process.
func CloseSessionCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Get the active session
	// Get 活跃 session
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		con.PrintErrorf("No active session\n")
		return
	}

	// remove any active socks proxies
	// 删除所有活动的袜子代理
	socks := core.SocksProxies.List()
	if len(socks) != 0 {
		for _, p := range socks {
			if p.SessionID == session.ID {
				core.SocksProxies.Remove(p.ID)
			}
		}
	}

	// Close the session
	// Close session
	_, err := con.Rpc.CloseSession(context.Background(), &sliverpb.CloseSession{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}

	con.ActiveTarget.Set(nil, nil)
}
