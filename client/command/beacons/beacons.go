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
	"github.com/bishopfox/sliver/client/command/targettable"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsCmd - Display/interact with beacons
func BeaconsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	killFlag, _ := cmd.Flags().GetString("kill")
	killAll, _ := cmd.Flags().GetBool("kill-all")
	interact, _ := cmd.Flags().GetString("interact")

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

	if interact != "" {
		beacon, err := GetBeacon(con, interact)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.ActiveTarget.Set(nil, beacon)
		con.PrintInfof("Active beacon %s (%s)\n", beacon.Name, strings.Split(beacon.ID, "-")[0])
		return
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
	return renderBeaconsAtWidth(beacons, filter, filterRegex, con, width, time.Now())
}

// renderBeaconsAtWidth uses one reference time for all candidate layouts,
// preventing a second boundary from changing the table width while it is fitted.
func renderBeaconsAtWidth(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, width int, now time.Time) table.Writer {
	windowsBeaconInList := false
	for _, beacon := range beacons {
		if beacon.OS == "windows" {
			windowsBeaconInList = true
			break
		}
	}

	return targettable.Fit(width, windowsBeaconInList, func(layout targettable.Layout) table.Writer {
		return buildBeaconTable(beacons, filter, filterRegex, con, layout, windowsBeaconInList, now)
	})
}

func buildBeaconTable(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, layout targettable.Layout, windowsBeaconInList bool, now time.Time) table.Writer {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	includeIntegrity := windowsBeaconInList && layout.IncludeIntegrity
	if layout.Compact {
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
	} else {
		header := table.Row{
			"ID",
			"Name",
			"Tasks",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Process (PID)",
		}
		if includeIntegrity {
			header = append(header, "Integrity")
		}
		header = append(header, "Operating System")
		if layout.IncludeLocale {
			header = append(header, "Locale")
		}
		header = append(header, "Last Check-in", "Next Check-in")
		tw.AppendHeader(header)
	}

	activeBeaconID := ""
	if con.ActiveTarget != nil {
		if activeBeacon := con.ActiveTarget.GetBeacon(); activeBeacon != nil {
			activeBeaconID = activeBeacon.ID
		}
	}

	for _, beacon := range beacons {
		integrity := beacon.Integrity
		if integrity == "" {
			integrity = "-"
		}
		username := strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")
		lastCheckin := time.Unix(beacon.LastCheckin, 0)
		nextCheckin := time.Unix(beacon.NextCheckin, 0)
		canonicalEntries := []string{
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			fmt.Sprintf("%d/%d", beacon.TasksCountCompleted, beacon.TasksCount),
			beacon.Transport,
			beacon.RemoteAddress,
			beacon.Hostname,
			username,
			fmt.Sprintf("%s (%d)", beacon.Filename, beacon.PID),
			integrity,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
			beacon.Locale,
			formatBeaconDateDelta(lastCheckin, now, true, false),
			formatBeaconDateDelta(nextCheckin, now, true, false),
		}
		if !beaconMatchesFilter(canonicalEntries, filter, filterRegex) {
			continue
		}

		style := console.StyleNormal
		if activeBeaconID == beacon.ID {
			style = console.StyleGreen
		}

		var rowEntries []string
		if layout.Compact {
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(beacon.Transport),
				style.Render(beacon.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)),
				formatBeaconDateDelta(lastCheckin, now, false, false),
				formatBeaconDateDelta(nextCheckin, now, false, true),
			}
		} else {
			processName := beacon.Filename
			if layout.ProcessBasename {
				processName = targettable.ProcessName(processName)
			}
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(fmt.Sprintf("%d/%d", beacon.TasksCountCompleted, beacon.TasksCount)),
				style.Render(beacon.Transport),
				style.Render(beacon.RemoteAddress),
				style.Render(beacon.Hostname),
				style.Render(username),
				style.Render(fmt.Sprintf("%s (%d)", processName, beacon.PID)),
			}
			if includeIntegrity {
				rowEntries = append(rowEntries, style.Render(integrity))
			}
			rowEntries = append(rowEntries, style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)))
			if layout.IncludeLocale {
				rowEntries = append(rowEntries, style.Render(beacon.Locale))
			}
			rowEntries = append(rowEntries,
				formatBeaconDateDelta(lastCheckin, now, true, false),
				formatBeaconDateDelta(nextCheckin, now, true, true),
			)
		}

		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		tw.AppendRow(row)
	}
	return tw
}

func beaconMatchesFilter(entries []string, filter string, filterRegex *regexp.Regexp) bool {
	if filter == "" && filterRegex == nil {
		return true
	}
	for _, entry := range entries {
		if filter != "" && strings.Contains(entry, filter) {
			return true
		}
		if filterRegex != nil && filterRegex.MatchString(entry) {
			return true
		}
	}
	return false
}

func formatBeaconDateDelta(timestamp time.Time, now time.Time, includeDate bool, color bool) string {
	date := timestamp.Format(time.UnixDate)
	interval := ""
	if timestamp.Before(now) {
		delta := now.Sub(timestamp).Round(time.Second)
		if includeDate {
			interval = fmt.Sprintf("%s (%s ago)", date, delta)
		} else {
			interval = delta.String()
		}
		if color {
			interval = console.StyleBoldRed.Render(interval)
		}
	} else {
		delta := timestamp.Sub(now).Round(time.Second)
		if includeDate {
			interval = fmt.Sprintf("%s (in %s)", date, delta)
		} else {
			interval = delta.String()
		}
		if color {
			interval = console.StyleBoldGreen.Render(interval)
		}
	}
	return interval
}
