package reconfig

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
	"context"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"github.com/desertbit/grumble"
)

// ReconfigCmd - Reconfigure metadata about a sessions
func ReconfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var err error
	var reconnectInterval time.Duration
	if ctx.Flags.String("reconnect-interval") != "" {
		reconnectInterval, err = time.ParseDuration(ctx.Flags.String("reconnect-interval"))
		if err != nil {
			con.PrintErrorf("Invalid reconnect interval: %s\n", err)
			return
		}
	}

	var beaconInterval time.Duration
	var beaconJitter time.Duration
	if beacon != nil {
		if ctx.Flags.String("beacon-interval") != "" {
			beaconInterval, err = time.ParseDuration(ctx.Flags.String("beacon-interval"))
			if err != nil {
				con.PrintErrorf("Invalid beacon interval: %s\n", err)
				return
			}
		}
		if ctx.Flags.String("beacon-jitter") != "" {
			beaconJitter, err = time.ParseDuration(ctx.Flags.String("beacon-jitter"))
			if err != nil {
				con.PrintErrorf("Invalid beacon jitter: %s\n", err)
				return
			}
			if beaconInterval == 0 && beaconJitter != 0 {
				con.PrintInfof("Modified beacon jitter will take effect after next check-in\n")
			}
		}
	}

	reconfig, err := con.Rpc.Reconfigure(context.Background(), &sliverpb.ReconfigureReq{
		ReconnectInterval: int64(reconnectInterval),
		BeaconInterval:    int64(beaconInterval),
		BeaconJitter:      int64(beaconJitter),
		Request:           con.ActiveTarget.Request(ctx),
	})

	if err != nil {
		con.PrintWarnf("%s\n", err)
		return
	}
	if reconfig.Response != nil && reconfig.Response.Async {
		con.AddBeaconCallback(reconfig.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, reconfig)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			con.PrintInfof("Reconfigured beacon\n")
		})
		con.PrintAsyncResponse(reconfig.Response)
	} else {
		con.PrintInfof("Reconfiguration complete\n")
	}
}
