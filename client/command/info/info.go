package info

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
func InfoCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var err error

	// Check if we have an active target via 'use'
	session, beacon := con.ActiveTarget.Get()

	if len(args) > 0 {
		// ID passed via argument takes priority
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

		con.Printf(console.Bold+"        Session ID: %s%s\n", console.Normal, session.ID)
		con.Printf(console.Bold+"              Name: %s%s\n", console.Normal, session.Name)
		con.Printf(console.Bold+"          Hostname: %s%s\n", console.Normal, session.Hostname)
		con.Printf(console.Bold+"              UUID: %s%s\n", console.Normal, session.UUID)
		con.Printf(console.Bold+"          Username: %s%s\n", console.Normal, session.Username)
		con.Printf(console.Bold+"               UID: %s%s\n", console.Normal, session.UID)
		con.Printf(console.Bold+"               GID: %s%s\n", console.Normal, session.GID)
		con.Printf(console.Bold+"               PID: %s%d\n", console.Normal, session.PID)
		con.Printf(console.Bold+"                OS: %s%s\n", console.Normal, session.OS)
		con.Printf(console.Bold+"           Version: %s%s\n", console.Normal, session.Version)
		con.Printf(console.Bold+"            Locale: %s%s\n", console.Normal, session.Locale)
		con.Printf(console.Bold+"              Arch: %s%s\n", console.Normal, session.Arch)
		con.Printf(console.Bold+"         Active C2: %s%s\n", console.Normal, session.ActiveC2)
		con.Printf(console.Bold+"    Remote Address: %s%s\n", console.Normal, session.RemoteAddress)
		con.Printf(console.Bold+"         Proxy URL: %s%s\n", console.Normal, session.ProxyURL)
		con.Printf(console.Bold+"Reconnect Interval: %s%s\n", console.Normal, time.Duration(session.ReconnectInterval).String())
		con.Printf(console.Bold+"     First Contact: %s%s\n", console.Normal, con.FormatDateDelta(time.Unix(session.FirstContact, 0), true, false))
		con.Printf(console.Bold+"      Last Checkin: %s%s\n", console.Normal, con.FormatDateDelta(time.Unix(session.LastCheckin, 0), true, false))

	} else if beacon != nil {

		con.Printf(console.Bold+"         Beacon ID: %s%s\n", console.Normal, beacon.ID)
		con.Printf(console.Bold+"              Name: %s%s\n", console.Normal, beacon.Name)
		con.Printf(console.Bold+"          Hostname: %s%s\n", console.Normal, beacon.Hostname)
		con.Printf(console.Bold+"              UUID: %s%s\n", console.Normal, beacon.UUID)
		con.Printf(console.Bold+"          Username: %s%s\n", console.Normal, beacon.Username)
		con.Printf(console.Bold+"               UID: %s%s\n", console.Normal, beacon.UID)
		con.Printf(console.Bold+"               GID: %s%s\n", console.Normal, beacon.GID)
		con.Printf(console.Bold+"               PID: %s%d\n", console.Normal, beacon.PID)
		con.Printf(console.Bold+"                OS: %s%s\n", console.Normal, beacon.OS)
		con.Printf(console.Bold+"           Version: %s%s\n", console.Normal, beacon.Version)
		con.Printf(console.Bold+"            Locale: %s%s\n", console.Normal, beacon.Locale)
		con.Printf(console.Bold+"              Arch: %s%s\n", console.Normal, beacon.Arch)
		con.Printf(console.Bold+"         Active C2: %s%s\n", console.Normal, beacon.ActiveC2)
		con.Printf(console.Bold+"    Remote Address: %s%s\n", console.Normal, beacon.RemoteAddress)
		con.Printf(console.Bold+"         Proxy URL: %s%s\n", console.Normal, beacon.ProxyURL)
		con.Printf(console.Bold+"          Interval: %s%s\n", console.Normal, time.Duration(beacon.Interval).String())
		con.Printf(console.Bold+"            Jitter: %s%s\n", console.Normal, time.Duration(beacon.Jitter).String())
		con.Printf(console.Bold+"     First Contact: %s%s\n", console.Normal, con.FormatDateDelta(time.Unix(beacon.FirstContact, 0), true, false))
		con.Printf(console.Bold+"      Last Checkin: %s%s\n", console.Normal, con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), true, false))
		con.Printf(console.Bold+"      Next Checkin: %s%s\n", console.Normal, con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), true, true))

	} else {
		con.PrintErrorf("No target session, see `help %s`\n", consts.InfoStr)
	}
}

// PIDCmd - Get the active session's PID.
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
