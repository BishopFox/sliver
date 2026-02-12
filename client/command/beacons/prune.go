package beacons

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsPruneCmd - Prune stale beacons automatically
// BeaconsPruneCmd - Prune 自动失效信标
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
	_ = forms.Confirm("Prune these beacons?", &confirm)
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
