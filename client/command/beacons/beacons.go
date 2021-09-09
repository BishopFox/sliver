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

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// BeaconsCmd - Display/interact with beacons
func BeaconsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	PrintBeacons(beacons.Beacons, con)
}

// BeaconsRmCmd - Display/interact with beacons
func BeaconsRmCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacon, err := SelectBeacon(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	_, err = con.Rpc.RmBeacon(context.Background(), beacon)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Beacon removed (%s)\n", beacon.ID)
}

// PrintBeacons - Display a list of beacons
func PrintBeacons(beacons []*clientpb.Beacon, con *console.SliverConsoleClient) {

	if len(beacons) == 0 {
		con.PrintInfof("No beacons 🙁\n")
		return
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle())
	tw.AppendHeader(table.Row{
		"ID",
		"Name",
		"Transport",
		"Remote Address",
		"Hostname",
		"Username",
		"Operating System",
		"Last Check-in",
		"Next Check-in",
	})

	for _, beacon := range beacons {
		next := time.Unix(beacon.NextCheckin, 0).Format(time.RFC1123)
		if time.Unix(beacon.NextCheckin, 0).Before(time.Now()) {
			next = fmt.Sprintf("%s%s%s", console.Bold+console.Red, next, console.Normal)
		}
		tw.AppendRow(table.Row{
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			beacon.Transport,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
			time.Unix(beacon.LastCheckin, 0).Format(time.RFC1123),
			next,
		})
	}
	con.Printf("%s\n", tw.Render())
}
