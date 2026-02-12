package kill

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
	"errors"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// KillCmd - Kill the active session (not to be confused with TerminateCmd)
// KillCmd - Kill 活跃 session （不要与 TerminateCmd 混淆）
func KillCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	// Confirm with the user, just in case they confused kill with terminate
	// Confirm 与用户，以防他们将终止与终止混淆
	confirm := false
	con.PrintWarnf("WARNING: This will kill the remote implant process\n\n")
	if session != nil {
		_ = forms.Confirm("Kill the active session?", &confirm)
		if !confirm {
			return
		}
		err := KillSession(session, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Killed %s (%s)\n", session.Name, session.ID)
		con.ActiveTarget.Background()
		return
	} else if beacon != nil {
		_ = forms.Confirm("Kill the active beacon?", &confirm)
		if !confirm {
			return
		}
		err := KillBeacon(beacon, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
		con.ActiveTarget.Background()
		return
	}
	con.PrintErrorf("No active session or beacon\n")
}

func KillSession(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) error {
	if session == nil {
		return errors.New("session does not exist")
	}
	timeout, _ := cmd.Flags().GetInt64("timeout")
	force, _ := cmd.Flags().GetBool("force")

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

	_, err := con.Rpc.Kill(context.Background(), &sliverpb.KillReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   timeout,
		},
		Force: force,
	})
	return err
}

func KillBeacon(beacon *clientpb.Beacon, cmd *cobra.Command, con *console.SliverClient) error {
	if beacon == nil {
		return errors.New("session does not exist")
	}

	timeout, _ := cmd.Flags().GetInt64("timeout")
	force, _ := cmd.Flags().GetBool("force")

	_, err := con.Rpc.Kill(context.Background(), &sliverpb.KillReq{
		Request: &commonpb.Request{
			BeaconID: beacon.ID,
			Timeout:  timeout,
		},
		Force: force,
	})
	return err
}
