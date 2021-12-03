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
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"

	"github.com/desertbit/grumble"
)

// InfoCmd - Display information about the active session
func InfoCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if session != nil {

		con.Printf(console.Bold+"                ID: %s%d\n", console.Normal, session.ID)
		con.Printf(console.Bold+"              Name: %s%s\n", console.Normal, session.Name)
		con.Printf(console.Bold+"          Hostname: %s%s\n", console.Normal, session.Hostname)
		con.Printf(console.Bold+"              UUID: %s%s\n", console.Normal, session.UUID)
		con.Printf(console.Bold+"          Username: %s%s\n", console.Normal, session.Username)
		con.Printf(console.Bold+"               UID: %s%s\n", console.Normal, session.UID)
		con.Printf(console.Bold+"               GID: %s%s\n", console.Normal, session.GID)
		con.Printf(console.Bold+"               PID: %s%d\n", console.Normal, session.PID)
		con.Printf(console.Bold+"                OS: %s%s\n", console.Normal, session.OS)
		con.Printf(console.Bold+"           Version: %s%s\n", console.Normal, session.Version)
		con.Printf(console.Bold+"              Arch: %s%s\n", console.Normal, session.Arch)
		con.Printf(console.Bold+"    Remote Address: %s%s\n", console.Normal, session.RemoteAddress)
		con.Printf(console.Bold+"         Proxy URL: %s%s\n", console.Normal, session.ProxyURL)
		con.Printf(console.Bold+"Reconnect Interval: %s%d\n", console.Normal, session.ReconnectInterval)
		con.Printf(console.Bold+"          IsDaemon: %s%v\n", console.Normal, session.IsDaemon)

	} else if beacon != nil {

		con.Printf(console.Bold+"                ID: %s%s\n", console.Normal, beacon.ID)
		con.Printf(console.Bold+"              Name: %s%s\n", console.Normal, beacon.Name)
		con.Printf(console.Bold+"          Hostname: %s%s\n", console.Normal, beacon.Hostname)
		con.Printf(console.Bold+"              UUID: %s%s\n", console.Normal, beacon.UUID)
		con.Printf(console.Bold+"          Username: %s%s\n", console.Normal, beacon.Username)
		con.Printf(console.Bold+"               UID: %s%s\n", console.Normal, beacon.UID)
		con.Printf(console.Bold+"               GID: %s%s\n", console.Normal, beacon.GID)
		con.Printf(console.Bold+"               PID: %s%d\n", console.Normal, beacon.PID)
		con.Printf(console.Bold+"                OS: %s%s\n", console.Normal, beacon.OS)
		con.Printf(console.Bold+"           Version: %s%s\n", console.Normal, beacon.Version)
		con.Printf(console.Bold+"              Arch: %s%s\n", console.Normal, beacon.Arch)
		con.Printf(console.Bold+"    Remote Address: %s%s\n", console.Normal, beacon.RemoteAddress)
		con.Printf(console.Bold+"         Proxy URL: %s%s\n", console.Normal, beacon.ProxyURL)
		con.Printf(console.Bold+"          Interval: %s%d\n", console.Normal, beacon.Interval)
		con.Printf(console.Bold+"            Jitter: %s%d\n", console.Normal, beacon.Jitter)

	} else {
		con.PrintErrorf("No target session, see `help %s`\n", consts.InfoStr)
	}
}

// PIDCmd - Get the active session's PID
func PIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

// UIDCmd - Get the active session's UID
func UIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

// GIDCmd - Get the active session's GID
func GIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
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

// WhoamiCmd - Displays the current user of the active session
func WhoamiCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	if session != nil {
		con.Printf("%s\n", session.Username)
	} else if beacon != nil {
		con.Printf("%s\n", beacon.Username)
	}
}
