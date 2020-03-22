package command

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
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

func info(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	var session *clientpb.Session
	if ActiveSession.Session != nil {
		session = ActiveSession.Session
	} else if 0 < len(ctx.Args) {
		session = getSession(ctx.Args[0], rpc)
	}

	if session != nil {
		fmt.Printf(bold+"            ID: %s%d\n", normal, session.ID)
		fmt.Printf(bold+"          Name: %s%s\n", normal, session.Name)
		fmt.Printf(bold+"      Hostname: %s%s\n", normal, session.Hostname)
		fmt.Printf(bold+"      Username: %s%s\n", normal, session.Username)
		fmt.Printf(bold+"           UID: %s%s\n", normal, session.UID)
		fmt.Printf(bold+"           GID: %s%s\n", normal, session.GID)
		fmt.Printf(bold+"           PID: %s%d\n", normal, session.PID)
		fmt.Printf(bold+"            OS: %s%s\n", normal, session.OS)
		fmt.Printf(bold+"       Version: %s%s\n", normal, session.Version)
		fmt.Printf(bold+"          Arch: %s%s\n", normal, session.Arch)
		fmt.Printf(bold+"Remote Address: %s%s\n", normal, session.RemoteAddress)
	} else {
		fmt.Printf(Warn+"No target session, see `help %s`\n", consts.InfoStr)
	}
}

func ping(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ActiveSession.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}
}

func getPID(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ActiveSession.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}
	fmt.Printf("%d\n", ActiveSession.Session.PID)
}

func getUID(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ActiveSession.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSession.Session.UID)
}

func getGID(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ActiveSession.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSession.Session.GID)
}

func whoami(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	if ActiveSession.Session == nil {
		fmt.Printf(Warn + "Please select an active session via `use`\n")
		return
	}
	fmt.Printf("%s\n", ActiveSession.Session.Username)
}
