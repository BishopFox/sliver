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
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsPruneCmd - Prune stale beacons automatically
func BeaconsPruneCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	duration, _ := cmd.Flags().GetString("duration")
	pruneDuration, err := time.ParseDuration(duration)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Pruning beacons that missed their last checking by %s or more...\n\n", pruneDuration)
	grpcCtx, cancel := con.GrpcContext(cmd)
	defer cancel()
	beacons, err := con.Rpc.GetBeacons(grpcCtx, &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	pruneBeacons := []*clientpb.Beacon{}
	for _, beacon := range beacons.Beacons {
		nextCheckin := time.Unix(beacon.NextCheckin, 0)
		if time.Now().Before(nextCheckin) {
			continue
		}
		delta := time.Since(nextCheckin)
		if pruneDuration <= delta {
			pruneBeacons = append(pruneBeacons, beacon)
		}
	}
	if len(pruneBeacons) == 0 {
		con.PrintInfof("No beacons to prune.\n")
		return
	}
	con.PrintWarnf("The following beacons and their tasks will be removed:\n")
	for index, beacon := range pruneBeacons {
		beacon, err := con.Rpc.GetBeacon(grpcCtx, &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			continue
		}
		con.Printf("\t%d. %s (%s)\n", (index + 1), beacon.Name, beacon.ID)
	}
	con.Println()
	confirm := false
	prompt := &survey.Confirm{Message: "Prune these beacons?"}
	survey.AskOne(prompt, &confirm)
	if !confirm {
		return
	}
	errCount := 0
	for _, beacon := range pruneBeacons {
		_, err := con.Rpc.RmBeacon(grpcCtx, &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			con.PrintErrorf("%s\n", err)
			errCount++
		}
	}
	con.PrintInfof("Pruned %d beacon(s)\n", len(pruneBeacons)-errCount)
}
