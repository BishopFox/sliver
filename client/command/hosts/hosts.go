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
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"

	"github.com/jedib0t/go-pretty/v6/table"
)

var (
	// ErrNoHosts - No hosts in database
	ErrNoHosts = errors.New("no hosts")
	// ErrNoSelection - No selection made
	ErrNoSelection = errors.New("no selection")
)

// HostsCmd - Main hosts command
func HostsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	allHosts, err := con.Rpc.Hosts(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	if 0 < len(allHosts.Hosts) {
		con.Printf("%s\n", hostsTable(allHosts.Hosts, con))
	} else {
		con.PrintInfof("No hosts\n")
	}
}

func hostsTable(hosts []*clientpb.Host, con *console.SliverConsoleClient) string {
	tw := table.NewWriter()
	tw.SetStyle(table.StyleBold)
	tw.AppendHeader(table.Row{"Hostname", "Operating System", "Sessions", "IOCs", "Extension Data", "ID (Short)"})
	for _, host := range hosts {
		tw.AppendRow(table.Row{
			host.Hostname,
			host.OSVersion,
			hostSessionNumbers(host.HostUUID, con),
			len(host.IOCs),
			len(host.ExtensionData),
			host.HostUUID[:8],
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

// SelectHost - Interactively select a host from the database
func SelectHost(con *console.SliverConsoleClient) (*clientpb.Host, error) {
	allHosts, err := con.Rpc.Hosts(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	// Sort the keys because maps have a randomized order, these keys must be ordered for the selection
	// to work properly since we rely on the index of the user's selection to find the session in the map
	var keys []string
	var hostMap = make(map[string]*clientpb.Host)
	for _, host := range allHosts.Hosts {
		keys = append(keys, host.HostUUID)
		hostMap[host.HostUUID] = host
	}
	sort.Strings(keys)

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	for _, key := range keys {
		host := hostMap[key]
		fmt.Fprintf(table, "%s\t%s\t\n",
			host.Hostname,
			host.OSVersion,
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.Select{
		Message: "Select a host:",
		Options: options,
	}
	selected := ""
	survey.AskOne(prompt, &selected)
	if selected == "" {
		return nil, ErrNoSelection
	}

	// Go from the selected option -> index -> host
	for index, option := range options {
		if option == selected {
			return hostMap[keys[index]], nil
		}
	}
	return nil, ErrNoSelection
}
