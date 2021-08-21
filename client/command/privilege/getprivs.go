package privilege

import (
	"context"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

func GetPrivsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var processName string = "Current Process"

	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	if session.OS != "windows" {
		con.PrintErrorf("Command not supported on this operating system.")
		return
	}

	privs, err := con.Rpc.GetPrivs(context.Background(), &sliverpb.GetPrivsReq{
		Request: con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Response is the Envelope (see RPC API), Err is part of it.
	if privs.Response != nil && privs.Response.Err != "" {
		con.PrintErrorf("NOTE: Information may be incomplete due to an error:\n")
		con.PrintErrorf("%s\n", privs.Response.Err)
	}

	if privs.PrivInfo == nil {
		return
	}

	if privs.ProcessName != "" {
		processName = privs.ProcessName
	}

	// To make things look pretty, figure out the longest name and description
	// for column width
	var nameColumnWidth int = 0
	var descriptionColumnWidth int = 0
	var introWidth int = 34 + len(processName) + len(strconv.Itoa(int(session.PID)))

	for _, entry := range privs.PrivInfo {
		if len(entry.Name) > nameColumnWidth {
			nameColumnWidth = len(entry.Name)
		}
		if len(entry.Description) > descriptionColumnWidth {
			descriptionColumnWidth = len(entry.Description)
		}
	}

	// Give one more space
	nameColumnWidth += 1
	descriptionColumnWidth += 1

	con.Printf("Privilege Information for %s (PID: %d)\n", processName, session.PID)
	con.Println(strings.Repeat("-", introWidth))
	con.Printf("\nProcess Integrity Level: %s\n\n", privs.ProcessIntegrity)
	con.Printf("%-*s\t%-*s\t%s\n", nameColumnWidth, "Name", descriptionColumnWidth, "Description", "Attributes")
	con.Printf("%-*s\t%-*s\t%s\n", nameColumnWidth, "====", descriptionColumnWidth, "===========", "==========")
	for _, entry := range privs.PrivInfo {
		con.Printf("%-*s\t%-*s\t", nameColumnWidth, entry.Name, descriptionColumnWidth, entry.Description)
		if entry.Enabled {
			con.Printf("Enabled")
		} else {
			con.Printf("Disabled")
		}
		if entry.EnabledByDefault {
			con.Printf(", Enabled by Default")
		}
		if entry.Removed {
			con.Printf(", Removed")
		}
		if entry.UsedForAccess {
			con.Printf(", Used for Access")
		}
		con.Printf("\n")
	}
}
