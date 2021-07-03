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
	"github.com/bishopfox/sliver/protobuf/clientpb"

	"github.com/desertbit/grumble"
)

func InfoCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {

	var session *clientpb.Session
	sessionName := ctx.Args.String("session")
	if con.ActiveSession.GetInteractive() != nil {
		session = con.ActiveSession.GetInteractive()
	} else if sessionName != "" {
		session = con.GetSession(sessionName, con.Rpc)
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
		con.Printf(console.Bold+"     Poll Interval: %s%d\n", console.Normal, session.PollInterval)
		con.Printf(console.Bold+"Reconnect Interval: %s%d\n", console.Normal, session.ReconnectInterval)
	} else {
		con.PrintErrorf("No target session, see `help %s`\n", consts.InfoStr)
	}
}

func PIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	con.Printf("%d\n", session.PID)
}

func UIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	con.Printf("%s\n", session.UID)
}

func GIDCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	con.Printf("%s\n", session.GID)
}

func WhoamiCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	con.Printf("%s\n", session.Username)
}
