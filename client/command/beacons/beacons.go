package beacons

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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsCmd - Display/interact with beacons
func BeaconsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	killFlag, _ := cmd.Flags().GetString("kill")
	killAll, _ := cmd.Flags().GetBool("kill-all")

	// Handle kill
	if killFlag != "" {
		beacon, err := GetBeacon(con, killFlag)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		err = kill.KillBeacon(beacon, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.Println()
		con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
	}

	if killAll {
		beacons, err := GetBeacons(con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		for _, beacon := range beacons.Beacons {
			err = kill.KillBeacon(beacon, cmd, con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			con.Println()
			con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
		}
	}
	filter, _ := cmd.Flags().GetString("filter")
	var filterRegex *regexp.Regexp
	if filterRe, _ := cmd.Flags().GetString("filter-re"); filterRe != "" {
		var err error
		filterRegex, err = regexp.Compile(filterRe)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
	}

	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	PrintBeacons(beacons.Beacons, filter, filterRegex, con)
}

// PrintBeacons - Display a list of beacons
func PrintBeacons(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) {
	if len(beacons) == 0 {
		con.PrintInfof("No beacons 🙁\n")
		return
	}
	tw := renderBeacons(beacons, filter, filterRegex, con)
	con.Printf("%s\n", tw.Render())
}

func renderBeacons(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient) table.Writer {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 999
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	wideTermWidth := con.Settings.SmallTermWidth < width
	windowsBeaconInList := false
	for _, beacon := range beacons {
		if beacon.OS == "windows" {
			windowsBeaconInList = true
		}
	}
	if wideTermWidth {
		if windowsBeaconInList {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Tasks",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Integrity",
				"Operating System",
				"Locale",
				"Last Check-in",
				"Next Check-in",
			})
		} else {
			tw.AppendHeader(table.Row{
				"ID",
				"Name",
				"Tasks",
				"Transport",
				"Remote Address",
				"Hostname",
				"Username",
				"Process (PID)",
				"Operating System",
				"Locale",
				"Last Check-in",
				"Next Check-in",
			})
		}
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Name",
			"Transport",
			"Hostname",
			"Username",
			"Operating System",
			"Last Check-in",
			"Next Check-in",
		})
	}

	for _, beacon := range beacons {
		color := console.Normal
		activeBeacon := con.ActiveTarget.GetBeacon()
		if activeBeacon != nil && activeBeacon.ID == beacon.ID {
			color = console.Green
		}
		if beacon.Integrity == "" {
			beacon.Integrity = "-"
		}

		// We need a slice of strings so we can apply filters
		var rowEntries []string

		if wideTermWidth {
			rowEntries = []string{
				fmt.Sprintf(color+"%s"+console.Normal, strings.Split(beacon.ID, "-")[0]),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Name),
				fmt.Sprintf(color+"%d/%d"+console.Normal, beacon.TasksCountCompleted, beacon.TasksCount),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Transport),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.RemoteAddress),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Hostname),
				fmt.Sprintf(color+"%s"+console.Normal, strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				fmt.Sprintf(color+"%s (%d)"+console.Normal, beacon.Filename, beacon.PID),
			}

			if windowsBeaconInList {
				rowEntries = append(rowEntries, fmt.Sprintf(color+"%s"+console.Normal, beacon.Integrity))
			}

			rowEntries = append(rowEntries, []string{
				fmt.Sprintf(color+"%s/%s"+console.Normal, beacon.OS, beacon.Arch),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Locale),
				con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), wideTermWidth, false),
				con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), wideTermWidth, true),
			}...)
		} else {
			rowEntries = []string{
				fmt.Sprintf(color+"%s"+console.Normal, strings.Split(beacon.ID, "-")[0]),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Name),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Transport),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Hostname),
				fmt.Sprintf(color+"%s"+console.Normal, strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				fmt.Sprintf(color+"%s/%s"+console.Normal, beacon.OS, beacon.Arch),
				con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), wideTermWidth, false),
				con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), wideTermWidth, true),
			}
		}
		// Build the row struct
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		// Apply filters if any
		if filter == "" && filterRegex == nil {
			tw.AppendRow(row)
		} else {
			for _, rowEntry := range rowEntries {
				if filter != "" {
					if strings.Contains(rowEntry, filter) {
						tw.AppendRow(row)
						break
					}
				}
				if filterRegex != nil {
					if filterRegex.MatchString(rowEntry) {
						tw.AppendRow(row)
						break
					}
				}
			}
		}
	}
	return tw
}
