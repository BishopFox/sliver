package commands

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// Canaries - List previously generated canaries
type Canaries struct {
	Burned bool `long:"burned" short:"b" description:"include canaries who have sang already"`
}

// Execute - List previously generated canaries
func (c *Canaries) Execute(args []string) (err error) {

	canaries, err := transport.RPC.Canaries(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.RPCError+"Failed to list canaries %s", err)
		return
	}
	if 0 < len(canaries.Canaries) {
		displayCanaries(canaries.Canaries, c.Burned)
	} else {
		fmt.Printf(util.Info + "No canaries in database\n")
	}
	return
}

func displayCanaries(canaries []*clientpb.DNSCanary, burnedOnly bool) {

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "Sliver Name\tDomain\tTriggered\tFirst Trigger\tLatest Trigger\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("Sliver Name")),
		strings.Repeat("=", len("Domain")),
		strings.Repeat("=", len("Triggered")),
		strings.Repeat("=", len("First Trigger")),
		strings.Repeat("=", len("Latest Trigger")),
	)

	lineColors := []string{}
	for _, canary := range canaries {
		if burnedOnly && !canary.Triggered {
			continue
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t\n",
			canary.ImplantName,
			canary.Domain,
			fmt.Sprintf("%v", canary.Triggered),
			canary.FirstTriggered,
			canary.LatestTrigger,
		)
		if canary.Triggered {
			lineColors = append(lineColors, bold+red)
		} else {
			lineColors = append(lineColors, normal)
		}
	}
	table.Flush()

	for index, line := range strings.Split(outputBuf.String(), "\n") {
		if len(line) == 0 {
			continue
		}
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}
}
