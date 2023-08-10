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
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

// BeaconsRmCmd - Display/interact with beacons.
func BeaconsRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) > 0 {
		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Failed to get beacons: %s", err)
			return
		}

		for _, arg := range args {
			for _, beac := range beacons.GetBeacons() {
				if strings.HasPrefix(beac.ID, arg) {
					grpcCtx, cancel := con.GrpcContext(cmd)
					_, err = con.Rpc.RmBeacon(grpcCtx, beac)
					if err != nil {
						con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
					}

					cancel()
				}
			}
		}
	} else {
		beacon, err := SelectBeacon(con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		grpcCtx, cancel := con.GrpcContext(cmd)
		defer cancel()
		_, err = con.Rpc.RmBeacon(grpcCtx, beacon)
		if err != nil {
			con.PrintErrorf("%s\n", con.UnwrapServerErr(err))
			return
		}
		con.PrintInfof("Beacon removed (%s)\n", beacon.ID)
	}
}
