package use

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
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/beacons"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

var ErrNoSelection = errors.New("no selection")

// UseCmd - Change the active session
func UseCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var session *clientpb.Session
	var beacon *clientpb.Beacon
	var err error

	var idArg string
	if len(args) > 0 {
		idArg = args[0]
	}

	// idArg := ctx.Args.String("id")
	if idArg != "" {
		session, beacon, err = SessionOrBeaconByID(idArg, con)
	} else {
		session, beacon, err = SelectSessionOrBeacon(con)
	}
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if session != nil {
		con.PrintInfof("Active session %s (%s)\n", session.Name, session.ID)
		con.ActiveTarget.Set(session, nil)
	} else if beacon != nil {
		con.PrintInfof("Active beacon %s (%s)\n", beacon.Name, beacon.ID)
		con.ActiveTarget.Set(nil, beacon)
	}
}

// SessionOrBeaconByID - Select a session or beacon by ID
func SessionOrBeaconByID(id string, con *console.SliverClient) (*clientpb.Session, *clientpb.Beacon, error) {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	if err == nil {
		for _, session := range sessions.Sessions {
			if strings.HasPrefix(session.ID, id) {
				return session, nil, nil
			}
		}
	}
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	for _, beacon := range beacons.Beacons {
		if strings.HasPrefix(beacon.ID, id) {
			return nil, beacon, nil
		}
	}
	return nil, nil, fmt.Errorf("no session or beacon found with ID %s", id)
}

// SelectSessionOrBeacon - Select a session or beacon
func SelectSessionOrBeacon(con *console.SliverClient) (*clientpb.Session, *clientpb.Beacon, error) {
	// Get and sort sessions
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	sessionsMap := map[string]*clientpb.Session{}
	for _, session := range sessions.GetSessions() {
		sessionsMap[session.ID] = session
	}
	var sessionKeys []string
	for _, session := range sessions.Sessions {
		sessionKeys = append(sessionKeys, session.ID)
	}
	sort.Strings(sessionKeys)

	// Get and sort beacons
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	beaconsMap := map[string]*clientpb.Beacon{}
	for _, beacon := range beacons.Beacons {
		beaconsMap[beacon.ID] = beacon
	}
	beaconKeys := []string{}
	for beaconID := range beaconsMap {
		beaconKeys = append(beaconKeys, beaconID)
	}
	sort.Strings(beaconKeys)

	if len(beaconKeys) == 0 && len(sessionKeys) == 0 {
		return nil, nil, fmt.Errorf("no sessions or beacons 🙁")
	}

	// Render selection table
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	for _, key := range sessionKeys {
		session := sessionsMap[key]
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			"SESSION",
			strings.Split(session.ID, "-")[0],
			session.Name,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
		)
	}
	for _, key := range beaconKeys {
		beacon := beaconsMap[key]
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			"BEACON",
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
		)
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.Select{
		Message: "Select a session or beacon:",
		Options: options,
	}
	selected := ""
	survey.AskOne(prompt, &selected)
	if selected == "" {
		return nil, nil, ErrNoSelection
	}
	for index, option := range options {
		if option == selected {
			if index < len(sessionKeys) {
				return sessionsMap[sessionKeys[index]], nil, nil
			}
			return nil, beaconsMap[beaconKeys[index-len(sessionKeys)]], nil
		}
	}
	return nil, nil, nil
}

// BeaconAndSessionIDCompleter - BeaconAndSessionIDCompleter for beacon / session ids
func BeaconAndSessionIDCompleter(con *console.SliverClient) carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		return action.Invoke(ctx).Merge(
			SessionIDCompleter(con).Invoke(ctx),
			beacons.BeaconIDCompleter(con).Invoke(ctx),
		).ToA()
	}

	return carapace.ActionCallback(comps)
}

// SessionIDCompleter completes session IDs
func SessionIDCompleter(con *console.SliverClient) carapace.Action {
	callback := func(_ carapace.Context) carapace.Action {
		results := make([]string, 0)

		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err == nil {
			for _, s := range sessions.Sessions {
				link := fmt.Sprintf("[%s <- %s]", s.ActiveC2, s.RemoteAddress)
				id := fmt.Sprintf("%s (%d)", s.Name, s.PID)
				userHost := fmt.Sprintf("%s@%s", s.Username, s.Hostname)
				desc := strings.Join([]string{id, userHost, link}, " ")

				results = append(results, s.ID[:8])
				results = append(results, desc)
			}
		}
		return carapace.ActionValuesDescribed(results...).Tag("sessions")
	}

	return carapace.ActionCallback(callback)
}
