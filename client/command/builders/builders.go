package builders

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"sort"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

// BuildersCmd - List external builders.
func BuildersCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	builders, err := con.Rpc.Builders(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s", err)
		return
	}
	if len(builders.Builders) == 0 {
		con.PrintInfof("No external builders connected to server\n")
	} else {
		PrintBuilders(builders.Builders, con)
	}
}

func PrintBuilders(externalBuilders []*clientpb.Builder, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Name", "Operator", "Platform", "CPU", "Available MB", "Report Age", "Availability", "Specs", "Compiler Targets",
	})
	for _, builder := range externalBuilders {
		if builder == nil {
			continue
		}

		targets := []string{}
		for _, target := range builder.Targets {
			targets = append(targets, fmt.Sprintf("%s:%s/%s", target.Format, target.GOOS, target.GOARCH))
		}
		sort.Strings(targets)
		cpu, availableMB, reportAge, availability := builderStatusColumns(builder.GetStatus(), time.Now())

		row := table.Row{
			builder.Name,
			builder.OperatorName,
			fmt.Sprintf("%s/%s", builder.GOOS, builder.GOARCH),
			cpu,
			availableMB,
			reportAge,
			availability,
			builderSpecSupport(builder.GetSpecRefs()),
			strings.Join(targets, "\n"),
		}
		tw.AppendRow(table.Row(row))
	}
	con.Printf("%s\n", tw.Render())
}

// Decimal MB matches the scheduler's memory ordering unit. The status age is
// based on the server's receipt time, so builder clock skew cannot affect it.
func builderStatusColumns(status *clientpb.BuilderStatus, now time.Time) (cpu, availableMB, reportAge, availability string) {
	if status == nil || status.ReportedUnix == 0 {
		return "unknown", "unknown", "never", "unavailable"
	}
	mb := status.AvailableMemoryBytes / 1_000_000
	if status.AvailableMemoryBytes%1_000_000 >= 500_000 {
		mb++
	}
	cpu = fmt.Sprintf("%d", status.EffectiveCPUCores)
	availableMB = fmt.Sprintf("%d", mb)
	base := time.Unix(status.ReportedUnix, 0)
	age := now.Sub(base)
	if age < 0 {
		age = 0
	}
	reportAge = age.Truncate(time.Second).String()
	switch {
	case !status.Online:
		availability = "offline"
	case status.Available:
		availability = "available"
	default:
		availability = "unavailable"
	}
	return
}

func builderSpecSupport(refs []*clientpb.BuildSpecRef) string {
	specs := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		specs = append(specs, fmt.Sprintf("%s@%s %s", ref.Name, ref.Version, ref.Digest))
	}
	sort.Strings(specs)
	if len(specs) == 0 {
		return "none advertised"
	}
	return strings.Join(specs, "\n")
}
