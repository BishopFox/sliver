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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/crypto/ssh/terminal"
)

// BeaconsCmd - Display/interact with beacons
func BeaconsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	killFlag := ctx.Flags.String("kill")
	killAll := ctx.Flags.Bool("kill-all")

	// Handle kill
	if killFlag != "" {
		beacon, err := GetBeacon(con, killFlag)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		err = kill.KillBeacon(beacon, false, con)
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
			err = kill.KillBeacon(beacon, true, con)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
			con.Println()
			con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
		}
	}

	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	PrintBeacons(beacons.Beacons, con)
}

// PrintBeacons - Display a list of beacons
func PrintBeacons(beacons []*clientpb.Beacon, con *console.SliverConsoleClient) {
	if len(beacons) == 0 {
		con.PrintInfof("No beacons 🙁\n")
		return
	}
	tw := renderBeacons(beacons, con)
	con.Printf("%s\n", tw.Render())
}

func renderBeacons(beacons []*clientpb.Beacon, con *console.SliverConsoleClient) table.Writer {
	width, _, err := terminal.GetSize(0)
	if err != nil {
		width = 999
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	if 182 < width {
		tw.AppendHeader(table.Row{
			"ID",
			"Name",
			"Tasks",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Operating System",
			"Last Check-in",
			"Next Check-in",
		})
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Name",
			"Transport",
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

		nextCheckin := time.Unix(beacon.NextCheckin, 0)
		var next string
		if time.Unix(beacon.NextCheckin, 0).Before(time.Now()) {
			past := time.Now().Sub(nextCheckin)
			next = fmt.Sprintf("%s-%s%s", console.Bold+console.Red, past, console.Normal)
		} else {
			eta := nextCheckin.Sub(time.Now())
			next = fmt.Sprintf("%s%s%s", console.Bold+console.Green, eta, console.Normal)
		}
		if 182 < width {
			tw.AppendRow(table.Row{
				fmt.Sprintf(color+"%s"+console.Normal, strings.Split(beacon.ID, "-")[0]),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Name),
				fmt.Sprintf(color+"%d/%d"+console.Normal, beacon.TasksCountCompleted, beacon.TasksCount),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Transport),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.RemoteAddress),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Hostname),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Username),
				fmt.Sprintf(color+"%s/%s"+console.Normal, beacon.OS, beacon.Arch),
				fmt.Sprintf(color+"%s ago"+console.Normal, time.Now().Sub(time.Unix(beacon.LastCheckin, 0))),
				next,
			})
		} else {
			tw.AppendRow(table.Row{
				fmt.Sprintf(color+"%s"+console.Normal, strings.Split(beacon.ID, "-")[0]),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Name),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Transport),
				fmt.Sprintf(color+"%s"+console.Normal, beacon.Username),
				fmt.Sprintf(color+"%s/%s"+console.Normal, beacon.OS, beacon.Arch),
				fmt.Sprintf(color+"%s ago"+console.Normal, time.Now().Sub(time.Unix(beacon.LastCheckin, 0))),
				next,
			})
		}
	}
	return tw
}
