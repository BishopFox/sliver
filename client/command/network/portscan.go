package network

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// PortscanCmd - Scan for open ports
func PortscanCmd(ctx *grumble.Context, con *console.SliverConsoleClient) (err error) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	host := ctx.Args.String("host")
	if host == "" {
		con.PrintErrorf("Missing parameter: host\n")
		return
	}

	port := ctx.Args.String("port")
	if port == "" {
		con.PrintErrorf("Missing parameter: port\n")
		return
	}

	threads := ctx.Flags.Int("threads")
	if threads == 0 {
		threads = 32
	}

	ctrl := make(chan bool)
	con.SpinUntil("Scanning ...", ctrl)

	portscan, err := con.Rpc.Portscan(context.Background(), &sliverpb.PortscanReq{
		Request: con.ActiveTarget.Request(ctx),
		Host:	host,
		Port:	port,
		Threads: int32(threads),
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if portscan.Response != nil && portscan.Response.Async {
		con.AddBeaconCallback(portscan.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, portscan)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
		})
		con.PrintAsyncResponse(portscan.Response)
	} else {
		PrintPortscan(portscan, con)
	}
	return
}

// PrintPortscan - Print the portscan response
func PrintPortscan(portscan *sliverpb.Portscan, con *console.SliverConsoleClient) {
	if portscan.Response != nil && portscan.Response.Err != "" {
		con.PrintErrorf("%s\n", portscan.Response.Err)
		return
	}
	con.PrintInfof("Scan results:\n%s\n", portscan.Output)
}
