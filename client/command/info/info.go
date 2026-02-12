package info

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
	"time"

	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// InfoCmd - Display information about the active session.
// InfoCmd - Display 有关活动 session. 的信息
func InfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error

	// Check if we have an active target via 'use'
	// Check 如果我们通过 __PH0__ 有一个活跃目标
	session, beacon := con.ActiveTarget.Get()

	if len(args) > 0 {
		// ID passed via argument takes priority
		// 通过参数传递的 ID 优先
		idArg := args[0]
		session, beacon, err = use.SessionOrBeaconByID(idArg, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	} else if session != nil || beacon != nil {
		currID := ""
		if session != nil {
			currID = session.ID
		} else {
			currID = beacon.ID
		}
		session, beacon, err = use.SessionOrBeaconByID(currID, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	} else {
		if session == nil && beacon == nil {
			session, beacon, err = use.SelectSessionOrBeacon(con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		}
	}

	if session != nil {

		con.Printf("%s %s\n", console.StyleBold.Render("        Session ID:"), session.ID)
		con.Printf("%s %s\n", console.StyleBold.Render("              Name:"), session.Name)
		con.Printf("%s %s\n", console.StyleBold.Render("          Hostname:"), session.Hostname)
		con.Printf("%s %s\n", console.StyleBold.Render("              UUID:"), session.UUID)
		con.Printf("%s %s\n", console.StyleBold.Render("          Username:"), session.Username)
		con.Printf("%s %s\n", console.StyleBold.Render("               UID:"), session.UID)
		con.Printf("%s %s\n", console.StyleBold.Render("               GID:"), session.GID)
		con.Printf("%s %d\n", console.StyleBold.Render("               PID:"), session.PID)
		con.Printf("%s %s\n", console.StyleBold.Render("                OS:"), session.OS)
		con.Printf("%s %s\n", console.StyleBold.Render("           Version:"), session.Version)
		con.Printf("%s %s\n", console.StyleBold.Render("            Locale:"), session.Locale)
		con.Printf("%s %s\n", console.StyleBold.Render("              Arch:"), session.Arch)
		con.Printf("%s %s\n", console.StyleBold.Render("         Active C2:"), session.ActiveC2)
		con.Printf("%s %s\n", console.StyleBold.Render("    Remote Address:"), session.RemoteAddress)
		con.Printf("%s %s\n", console.StyleBold.Render("         Proxy URL:"), session.ProxyURL)
		con.Printf("%s %s\n", console.StyleBold.Render("Reconnect Interval:"), time.Duration(session.ReconnectInterval).String())
		con.Printf("%s %s\n", console.StyleBold.Render("     First Contact:"), con.FormatDateDelta(time.Unix(session.FirstContact, 0), true, false))
		con.Printf("%s %s\n", console.StyleBold.Render("      Last Checkin:"), con.FormatDateDelta(time.Unix(session.LastCheckin, 0), true, false))

	} else if beacon != nil {

		con.Printf("%s %s\n", console.StyleBold.Render("         Beacon ID:"), beacon.ID)
		con.Printf("%s %s\n", console.StyleBold.Render("              Name:"), beacon.Name)
		con.Printf("%s %s\n", console.StyleBold.Render("          Hostname:"), beacon.Hostname)
		con.Printf("%s %s\n", console.StyleBold.Render("              UUID:"), beacon.UUID)
		con.Printf("%s %s\n", console.StyleBold.Render("          Username:"), beacon.Username)
		con.Printf("%s %s\n", console.StyleBold.Render("               UID:"), beacon.UID)
		con.Printf("%s %s\n", console.StyleBold.Render("               GID:"), beacon.GID)
		con.Printf("%s %d\n", console.StyleBold.Render("               PID:"), beacon.PID)
		con.Printf("%s %s\n", console.StyleBold.Render("                OS:"), beacon.OS)
		con.Printf("%s %s\n", console.StyleBold.Render("           Version:"), beacon.Version)
		con.Printf("%s %s\n", console.StyleBold.Render("            Locale:"), beacon.Locale)
		con.Printf("%s %s\n", console.StyleBold.Render("              Arch:"), beacon.Arch)
		con.Printf("%s %s\n", console.StyleBold.Render("         Active C2:"), beacon.ActiveC2)
		con.Printf("%s %s\n", console.StyleBold.Render("    Remote Address:"), beacon.RemoteAddress)
		con.Printf("%s %s\n", console.StyleBold.Render("         Proxy URL:"), beacon.ProxyURL)
		con.Printf("%s %s\n", console.StyleBold.Render("          Interval:"), time.Duration(beacon.Interval).String())
		con.Printf("%s %s\n", console.StyleBold.Render("            Jitter:"), time.Duration(beacon.Jitter).String())
		con.Printf("%s %s\n", console.StyleBold.Render("     First Contact:"), con.FormatDateDelta(time.Unix(beacon.FirstContact, 0), true, false))
		con.Printf("%s %s\n", console.StyleBold.Render("      Last Checkin:"), con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), true, false))
		con.Printf("%s %s\n", console.StyleBold.Render("      Next Checkin:"), con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), true, true))

	} else {
		con.PrintErrorf("No target session, see `help %s`\n", consts.InfoStr)
	}
}

// PIDCmd - Get the active session's PID.
// PIDCmd - Get 活跃 session 的 PID.
func PIDCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if session != nil {
		con.Printf("%d\n", session.PID)
	} else if beacon != nil {
		con.Printf("%d\n", beacon.PID)
	}
}

// UIDCmd - Get the active session's UID.
// UIDCmd - Get 活跃 session 的 UID.
func UIDCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if session != nil {
		con.Printf("%s\n", session.UID)
	} else if beacon != nil {
		con.Printf("%s\n", beacon.UID)
	}
}

// GIDCmd - Get the active session's GID.
// GIDCmd - Get 活跃 session 的 GID.
func GIDCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if session != nil {
		con.Printf("%s\n", session.GID)
	} else if beacon != nil {
		con.Printf("%s\n", beacon.GID)
	}
}

// WhoamiCmd - Displays the current user of the active session.
// WhoamiCmd - Displays 活动 session. 的当前用户
func WhoamiCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var isWin bool
	con.Printf("Logon ID: ")
	if session != nil {
		con.Printf("%s\n", session.Username)
		if session.GetOS() == "windows" {
			isWin = true
		}
	} else if beacon != nil {
		con.Printf("%s\n", beacon.Username)
		if beacon.GetOS() == "windows" {
			isWin = true
		}
	}

	if isWin {
		cto, err := con.Rpc.CurrentTokenOwner(context.Background(), &sliverpb.CurrentTokenOwnerReq{
			Request: con.ActiveTarget.Request(cmd),
		})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}

		if cto.Response != nil && cto.Response.Async {
			con.AddBeaconCallback(cto.Response.TaskID, func(task *clientpb.BeaconTask) {
				err = proto.Unmarshal(task.Response, cto)
				if err != nil {
					con.PrintErrorf("Failed to decode response %s\n", err)
					return
				}
				PrintTokenOwner(cto, con)
			})
			con.PrintAsyncResponse(cto.Response)
		} else {
			PrintTokenOwner(cto, con)
		}
	}
}

func PrintTokenOwner(cto *sliverpb.CurrentTokenOwner, con *console.SliverClient) {
	if cto.Response != nil && cto.Response.Err != "" {
		con.PrintErrorf("%s\n", cto.Response.Err)
		return
	}
	con.PrintInfof("Current Token ID: %s", cto.Output)
}
