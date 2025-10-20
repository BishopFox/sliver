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

	"github.com/gsmith257-cyber/better-sliver-package/client/console"
	"github.com/gsmith257-cyber/better-sliver-package/protobuf/clientpb"
	"github.com/gsmith257-cyber/better-sliver-package/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ReconfigCmd - Reconfigure metadata about a sessions.
func ReconfigCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	var err error
	var reconnectInterval time.Duration
	interval, _ := cmd.Flags().GetString("reconnect-interval")

	if interval != "" {
		reconnectInterval, err = time.ParseDuration(interval)
		if err != nil {
			con.PrintErrorf("Invalid reconnect interval: %s\n", err)
			return
		}
	}

	var BaconInterval time.Duration
	var BaconJitter time.Duration
	binterval, _ := cmd.Flags().GetString("beacon-interval")
	bjitter, _ := cmd.Flags().GetString("beacon-jitter")

	if beacon != nil {
		if binterval != "" {
			BaconInterval, err = time.ParseDuration(binterval)
			if err != nil {
				con.PrintErrorf("Invalid beacon interval: %s\n", err)
				return
			}
		}
		if bjitter != "" {
			BaconJitter, err = time.ParseDuration(bjitter)
			if err != nil {
				con.PrintErrorf("Invalid beacon jitter: %s\n", err)
				return
			}
			if BaconInterval == 0 && BaconJitter != 0 {
				con.PrintInfof("Modified beacon jitter will take effect after next check-in\n")
			}
		}
	}

	reconfig, err := con.Rpc.Reconfigure(context.Background(), &sliverpb.ReconfigureReq{
		ReconnectInterval: int64(reconnectInterval),
		BaconInterval:    int64(BaconInterval),
		BaconJitter:      int64(BaconJitter),
		Request:           con.ActiveTarget.Request(cmd),
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
