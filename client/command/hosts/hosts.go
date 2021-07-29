package hosts

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

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"

	"github.com/jedib0t/go-pretty/v6/table"
)

// HostsCmd - Main hosts command
func HostsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	allHosts, err := con.Rpc.Hosts(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	if 0 < len(allHosts.Hosts) {
		con.Printf(hostsTable(allHosts.Hosts, con))
		con.Println()
	} else {
		con.PrintInfof("No hosts\n")
	}
}

func hostsTable(hosts []*clientpb.Host, con *console.SliverConsoleClient) string {
	tw := table.NewWriter()
	tw.AppendHeader(table.Row{"ID", "Hostname", "OS Version", "Sessions"})
	for _, host := range hosts {
		tw.AppendRow(table.Row{
			host.HostUUID,
			host.Hostname,
			host.OSVersion,
			hostSessionNumbers(host.HostUUID, con),
		})
	}
	return tw.Render()
}

func hostSessionNumbers(hostUUID string, con *console.SliverConsoleClient) string {
	hostSessions := SessionsForHost(hostUUID, con)
	if 0 == len(hostSessions) {
		return "None"
	}
	sessionNumbers := []string{}
	for _, hostSession := range hostSessions {
		sessionNumbers = append(sessionNumbers, fmt.Sprintf("%d", hostSession.ID))
	}
	return strings.Join(sessionNumbers, ", ")
}

func SessionsForHost(hostUUID string, con *console.SliverConsoleClient) []*clientpb.Session {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return []*clientpb.Session{}
	}
	hostSessions := []*clientpb.Session{}
	for _, session := range sessions.Sessions {
		if session.UUID == hostUUID {
			hostSessions = append(hostSessions, session)
		}
	}
	return hostSessions
}
