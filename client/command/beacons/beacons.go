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
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

// BeaconsCmd - Display/interact with beacons
func BeaconsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	fmt.Printf("Beacons: %d\n\n", len(beacons.Beacons))

	printBeacons(beacons.Beacons, con)
}

// printBeacons - Display a list of beacons
func printBeacons(beacons []*clientpb.Beacon, con *console.SliverConsoleClient) {
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	// Column Headers
	fmt.Fprintln(table, "ID\tName\tTransport\tRemote Address\tHostname\tUsername\tOperating System\tLast Check-in\tNext Check-in\t")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("ID")),
		strings.Repeat("=", len("Name")),
		strings.Repeat("=", len("Transport")),
		strings.Repeat("=", len("Remote Address")),
		strings.Repeat("=", len("Hostname")),
		strings.Repeat("=", len("Username")),
		strings.Repeat("=", len("Operating System")),
		strings.Repeat("=", len("Last Check-in")),
		strings.Repeat("=", len("Next Check-in")),
	)

	for _, beacon := range beacons {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			beacon.Transport,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
			time.Unix(beacon.LastCheckin, 0).Format(time.RFC1123),
			time.Unix(beacon.NextCheckin, 0).Format(time.RFC1123),
		)
	}
	table.Flush()
	con.Printf("%s", outputBuf.String())
}
