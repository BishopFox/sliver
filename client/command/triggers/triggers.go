package triggers

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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

	------------------------------------------------------------------------

	triggers.go -- list all trigger implants (builds with
	IncludeTriggerWake=true). Analogous to how "beacons" lists beacons.
*/

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// TriggersCmd displays an indexed list of all generated trigger implants,
// similar to how "beacons" lists beacons. Filters ImplantBuilds for
// configs with IncludeTriggerWake=true.
func TriggersCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("failed to fetch implant builds: %s\n", err)
		return
	}

	triggerBuilds := sortedTriggerBuilds(builds)
	if len(triggerBuilds) == 0 {
		con.PrintInfof("No trigger implants found. Generate one with 'generate trigger'.\n")
		return
	}

	// Load stored targets for display
	store, _ := LoadTargetStore()

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Index",
		"Name",
		"OS/Arch",
		"Bind Port",
		"C2 Transports",
		"Allowed Clients",
		"Target",
	})

	for i, e := range triggerBuilds {
		config := e.Config

		// Extract port from TriggerWakeBindAddr
		bindPort := ""
		if addr := config.GetTriggerWakeBindAddr(); addr != "" {
			_, portStr, splitErr := net.SplitHostPort(addr)
			if splitErr == nil {
				bindPort = portStr
			} else {
				bindPort = addr // show raw if parsing fails
			}
		}

		// Collect C2 transport schemes
		var schemes []string
		for _, c2 := range config.C2 {
			url := c2.GetURL()
			if idx := strings.Index(url, "://"); idx > 0 {
				scheme := url[:idx]
				// Deduplicate
				found := false
				for _, s := range schemes {
					if s == scheme {
						found = true
						break
					}
				}
				if !found {
					schemes = append(schemes, scheme)
				}
			}
		}

		// Allowed client IDs
		allowedClients := strings.Join(config.GetTriggerWakeAllowedClientIDs(), ", ")
		if allowedClients == "" {
			allowedClients = "(any)"
		}

		// Stored target
		target := store.Targets[e.Name]
		if target == "" {
			target = "(not set)"
		}

		tw.AppendRow(table.Row{
			fmt.Sprintf("%d", i+1),
			e.Name,
			fmt.Sprintf("%s/%s", config.GOOS, config.GOARCH),
			bindPort,
			strings.Join(schemes, ","),
			allowedClients,
			target,
		})
	}

	con.Println(tw.Render())
	con.Println()
}
