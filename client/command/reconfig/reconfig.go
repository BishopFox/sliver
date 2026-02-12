package reconfig

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

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
	"context"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

// ReconfigCmd - Reconfigure metadata about a sessions.
// 关于 sessions. 的 ReconfigCmd - Reconfigure 元数据
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

	var beaconInterval time.Duration
	var beaconJitter time.Duration
	binterval, _ := cmd.Flags().GetString("beacon-interval")
	bjitter, _ := cmd.Flags().GetString("beacon-jitter")

	if beacon != nil {
		if binterval != "" {
			beaconInterval, err = time.ParseDuration(binterval)
			if err != nil {
				con.PrintErrorf("Invalid beacon interval: %s\n", err)
				return
			}
		}
		if bjitter != "" {
			beaconJitter, err = time.ParseDuration(bjitter)
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
