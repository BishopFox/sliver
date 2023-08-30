package taskmany

/*
	Sliver Implant Framework
	Copyright (C) 2021 Bishop Fox
	Copyright (C) 2023 ActualTrash

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
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// TaskmanyCmd - Task many beacons / sessions
func TaskmanyCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	con.PrintErrorf("Must specify subcommand. See taskmany --help for supported subcommands.\n")
}

// Helper function to wrap grumble commands with taskmany logic
func WrapCommand(c *cobra.Command, con *console.SliverConsoleClient) *cobra.Command {
	wc := &cobra.Command{
		Use:   c.Use,
		Short: c.Short,
		Long:  c.Long,
		Args:  c.Args,
		Run:   wrapFunctionWithTaskmany(con, c.Run),
	}
	wc.Flags().AddFlagSet(c.Flags())
	wc.PersistentFlags().AddFlagSet(c.PersistentFlags())
	return wc
}

// Wrap a function to run it for each beacon / session
func wrapFunctionWithTaskmany(con *console.SliverConsoleClient, f func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		defer con.Println()

		sessions, beacons, err := SelectMultipleBeaconsAndSessions(con)
		if err != nil {
			con.Println()
			con.PrintErrorf("%s\n", err)
			return
		}

		con.Println()

		// Save current active beacon or session
		origSession, origBeacon := con.ActiveTarget.Get()

		nB := 0
		nBSkipped := 0
		for _, b := range beacons {
			if !b.IsDead {
				con.ActiveTarget.Set(nil, b)
				f(cmd, args)
				nB += 1
			} else {
				nBSkipped += 1
			}
		}

		nS := 0
		nSSkipped := 0
		for _, s := range sessions {
			if !s.IsDead {
				con.ActiveTarget.Set(s, nil)
				f(cmd, args)
				nS += 1
			} else {
				nSSkipped += 1
			}
		}

		// Restore active session / beacon
		con.ActiveTarget.Set(origSession, origBeacon)

		con.PrintInfof("Tasked %d sessions and %d beacons >:D\n", nS, nB)
		if nBSkipped > 0 || nSSkipped > 0 {
			con.PrintWarnf("Skipped %d dead sessions and %d dead beacons\n", nSSkipped, nBSkipped)
		}
	}
}

func SelectMultipleBeaconsAndSessions(con *console.SliverConsoleClient) ([]*clientpb.Session, []*clientpb.Beacon, error) {
	// Get and sort sessions
	sessionsObj, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	sessions := sessionsObj.Sessions
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	// Get and sort beacons
	beaconsObj, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	beacons := beaconsObj.Beacons
	sort.Slice(beacons, func(i, j int) bool {
		return beacons[i].ID < beacons[j].ID
	})

	if len(beacons) == 0 && len(sessions) == 0 {
		return nil, nil, fmt.Errorf("no sessions or beacons 🙁")
	}

	// Render selection table
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	sessionOptionMap := map[string]*clientpb.Session{}
	for _, session := range sessions {
		option := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s",
			"SESSION",
			strings.Split(session.ID, "-")[0],
			session.Name,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
		)
		fmt.Fprintf(table, option+"\n")
		o := strings.ReplaceAll(option, "\t", "")
		sessionOptionMap[o] = session
	}

	beaconOptionMap := map[string]*clientpb.Beacon{}
	for _, beacon := range beacons {
		option := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s",
			"BEACON",
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
		)
		fmt.Fprintf(table, option+"\n")
		o := strings.ReplaceAll(option, "\t", "")
		beaconOptionMap[o] = beacon
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	prompt := &survey.MultiSelect{
		Message: "Select sessions and beacons:",
		Options: options,
	}
	selected := []string{}
	survey.AskOne(prompt, &selected)

	if len(selected) == 0 {
		return nil, nil, fmt.Errorf("no sessions or beacons selected 🤔")
	}

	selectedSessions := []*clientpb.Session{}
	selectedBeacons := []*clientpb.Beacon{}
	for _, s := range selected {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		session, ok := sessionOptionMap[s]
		if ok {
			selectedSessions = append(selectedSessions, session)
		}

		beacon, ok := beaconOptionMap[s]
		if ok {
			selectedBeacons = append(selectedBeacons, beacon)
		}
	}

	return selectedSessions, selectedBeacons, nil
}
