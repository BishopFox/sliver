package cursed

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
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

func CursedRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}
	bindPort, err := strconv.Atoi(args[0])
	if err != nil {
		con.PrintErrorf("Failed to parse bind port argument: %s (%s)", args[0], err.Error())
		return
	}

	kill, _ := cmd.Flags().GetBool("kill")
	core.CloseCursedProcessesByBindPort(session.ID, bindPort)
	if kill {
		confirm := false
		err := forms.Confirm("Kill the cursed process?", &confirm)
		if err != nil {
			con.PrintErrorf("%s", err)
			return
		}
		if !confirm {
			con.PrintErrorf("User cancel\n")
			return
		}
		// Get cursed process
		// Get 被诅咒的进程
		var cursedProc *core.CursedProcess
		curses := core.CursedProcessBySessionID(session.ID)
		for _, curse := range curses {
			if curse.BindTCPPort == bindPort {
				cursedProc = curse
			}
		}
		if cursedProc == nil {
			con.PrintErrorf("Failed to find cursed process\n")
			return
		}
		terminateResp, err := con.Rpc.Terminate(context.Background(), &sliverpb.TerminateReq{
			Request: con.ActiveTarget.Request(cmd),
			Pid:     int32(cursedProc.PID),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		if terminateResp.Response != nil && terminateResp.Response.Err != "" {
			con.PrintErrorf("could not terminate the existing process: %s\n", terminateResp.Response.Err)
			return
		}
	}
}
