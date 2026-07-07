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
	"github.com/bishopfox/sliver/client/command/output"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconResult represents a single beacon in structured output.
type BeaconResult struct {
	ID                    string `json:"id" yaml:"id"`
	Name                  string `json:"name" yaml:"name"`
	Transport             string `json:"transport" yaml:"transport"`
	RemoteAddress         string `json:"remoteAddress" yaml:"remoteAddress"`
	Hostname              string `json:"hostname" yaml:"hostname"`
	Username              string `json:"username" yaml:"username"`
	Process               string `json:"process" yaml:"process"`
	PID                   uint32 `json:"pid" yaml:"pid"`
	OS                    string `json:"os" yaml:"os"`
	Arch                  string `json:"arch" yaml:"arch"`
	Locale                string `json:"locale" yaml:"locale"`
	Integrity             string `json:"integrity,omitempty" yaml:"integrity,omitempty"`
	LastCheckin           int64  `json:"lastCheckin" yaml:"lastCheckin"`
	NextCheckin           int64  `json:"nextCheckin" yaml:"nextCheckin"`
	TasksCount            uint32 `json:"tasksCount" yaml:"tasksCount"`
	TasksCountCompleted   uint32 `json:"tasksCountCompleted" yaml:"tasksCountCompleted"`
	Active                bool   `json:"active" yaml:"active"`
}

// BeaconListResult represents the beacons list in structured output.
type BeaconListResult struct {
	Beacons []BeaconResult `json:"beacons" yaml:"beacons"`
}

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

	format := output.GetOutputFormat(cmd)
	if format != output.FormatText {
		PrintBeaconsStructured(beacons.Beacons, filter, filterRegex, con, format)
	} else {
		PrintBeacons(beacons.Beacons, filter, filterRegex, con)
	}
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

// PrintBeaconsStructured prints beacons in JSON or YAML format.
func PrintBeaconsStructured(beacons []*clientpb.Beacon, filter string, filterRegex *regexp.Regexp, con *console.SliverClient, format output.OutputFormat) {
	result := BeaconListResult{
		Beacons: make([]BeaconResult, 0),
	}

	for _, beacon := range beacons {
		username := strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")

		// Apply filters
		if filter != "" || filterRegex != nil {
			match := false
			fields := []string{
				beacon.ID, beacon.Name, beacon.Transport, beacon.RemoteAddress,
				beacon.Hostname, username, beacon.Filename, beacon.OS, beacon.Arch,
			}
			if filter != "" {
				for _, field := range fields {
					if strings.Contains(field, filter) {
						match = true
						break
					}
				}
			}
			if !match && filterRegex != nil {
				for _, field := range fields {
					if filterRegex.MatchString(field) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		br := BeaconResult{
			ID:                  beacon.ID,
			Name:                beacon.Name,
			Transport:           beacon.Transport,
			RemoteAddress:       beacon.RemoteAddress,
			Hostname:            beacon.Hostname,
			Username:            username,
			Process:             beacon.Filename,
			PID:                 beacon.PID,
			OS:                  beacon.OS,
			Arch:                beacon.Arch,
			Locale:              beacon.Locale,
			Integrity:           beacon.Integrity,
			LastCheckin:         beacon.LastCheckin,
			NextCheckin:         beacon.NextCheckin,
			TasksCount:          beacon.TasksCount,
			TasksCountCompleted: beacon.TasksCountCompleted,
			Active:              con.ActiveTarget.GetBeacon() != nil && con.ActiveTarget.GetBeacon().ID == beacon.ID,
		}

		result.Beacons = append(result.Beacons, br)
	}

	if err := output.PrintStructured(result, format); err != nil {
		con.PrintErrorf("Failed to format output: %s\n", err)
	}
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
		style := console.StyleNormal
		activeBeacon := con.ActiveTarget.GetBeacon()
		if activeBeacon != nil && activeBeacon.ID == beacon.ID {
			style = console.StyleGreen
		}
		if beacon.Integrity == "" {
			beacon.Integrity = "-"
		}

		// We need a slice of strings so we can apply filters
		var rowEntries []string

		if wideTermWidth {
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(fmt.Sprintf("%d/%d", beacon.TasksCountCompleted, beacon.TasksCount)),
				style.Render(beacon.Transport),
				style.Render(beacon.RemoteAddress),
				style.Render(beacon.Hostname),
				style.Render(strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				style.Render(fmt.Sprintf("%s (%d)", beacon.Filename, beacon.PID)),
			}

			if windowsBeaconInList {
				rowEntries = append(rowEntries, style.Render(beacon.Integrity))
			}

			rowEntries = append(rowEntries, []string{
				style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)),
				style.Render(beacon.Locale),
				con.FormatDateDelta(time.Unix(beacon.LastCheckin, 0), wideTermWidth, false),
				con.FormatDateDelta(time.Unix(beacon.NextCheckin, 0), wideTermWidth, true),
			}...)
		} else {
			rowEntries = []string{
				style.Render(strings.Split(beacon.ID, "-")[0]),
				style.Render(beacon.Name),
				style.Render(beacon.Transport),
				style.Render(beacon.Hostname),
				style.Render(strings.TrimPrefix(beacon.Username, beacon.Hostname+"\\")),
				style.Render(fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch)),
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
