package generate

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

func CanariesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	canaries, err := con.Rpc.Canaries(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("Failed to list canaries %s", err)
		return
	}
	if 0 < len(canaries.Canaries) {
		displayCanaries(con, canaries.Canaries, ctx.Flags.Bool("burned"))
	} else {
		con.PrintInfof("No canaries in database\n")
	}
}

func displayCanaries(con *console.SliverConsoleClient, canaries []*clientpb.DNSCanary, burnedOnly bool) {

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
			lineColors = append(lineColors, console.Bold+console.Red)
		} else {
			lineColors = append(lineColors, console.Normal)
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
			con.Printf("%s%s%s\n", lineColor, line, console.Normal)
		} else {
			con.Printf("%s\n", line)
		}
	}
}
