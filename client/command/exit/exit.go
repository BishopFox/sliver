package exit

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox
	Copyright (C) 2023 Bishop Fox

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
	"os"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// ExitCmd - Exit the console.
// ExitCmd - Exit console.
func ExitCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if con.IsServer {
		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			flushAndExit(con, 1)
		}
		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			flushAndExit(con, 1)
		}
		if 0 < len(sessions.Sessions) || 0 < len(beacons.Beacons) {
			con.Printf("There are %d active sessions and %d active beacons.\n", len(sessions.Sessions), len(beacons.Beacons))
			confirm := false
			forms.Confirm("Are you sure you want to exit?", &confirm)
			if !confirm {
				return
			}
		}
	}
	flushAndExit(con, 0)
}

// Commands returns the `exit` command.
// Commands 返回 __PH0__ command.
func Command(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{{
		Use:   "exit",
		Short: "Exit the program",
		Run: func(cmd *cobra.Command, args []string) {
			ExitCmd(cmd, con, args)
		},
		GroupID: constants.GenericHelpGroup,
	}}
}

func flushAndExit(con *console.SliverClient, code int) {
	if con != nil {
		con.FlushOutput()
	} else {
		os.Stdout.Sync()
	}
	os.Exit(code)
}
